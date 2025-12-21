package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"ledger/internal/cache"
	"ledger/internal/domain"
	"ledger/shared"
)

const (
	budgetsCacheKey   = "budgets:all"
	defaultBudgetsTTL = 30 * time.Second
	defaultReportTTL  = 30 * time.Second
)


type Service interface {
	SetBudget(ctx context.Context, userID string, budget domain.Budget) (domain.Budget, error)
	ListBudgets(ctx context.Context, userID string) ([]domain.Budget, error)
	AddTransaction(ctx context.Context, userID string, tx domain.Transaction) (domain.Transaction, error)
	ListTransactions(ctx context.Context, userID string) ([]domain.Transaction, error)
	GetReportSummary(ctx context.Context, userID, from, to string) ([]domain.ReportSummary, error)
	ImportTransactionsBulk(ctx context.Context, userID string, txs []domain.Transaction, workers int) (shared.BulkImportResult, error)
	ImportCSV(ctx context.Context, userID string, csvData string) (shared.BulkImportResult, error)
	ExportCSV(ctx context.Context, userID string, from, to string) (string, error)
}

type ledgerService struct {
	budgets  domain.BudgetRepository
	expenses domain.ExpenseRepository
	cache    cache.Client
	now      func() time.Time
}

func New(
	budgetRepo domain.BudgetRepository,
	expenseRepo domain.ExpenseRepository,
	cacheClient cache.Client,
	now func() time.Time,
) Service {
	if now == nil {
		now = time.Now
	}

	return &ledgerService{
		budgets:  budgetRepo,
		expenses: expenseRepo,
		cache:    cacheClient,
		now:      now,
	}
}

func (s *ledgerService) SetBudget(ctx context.Context, userID string, budget domain.Budget) (domain.Budget, error) {
	if err := budget.Validate(); err != nil {
		return domain.Budget{}, fmt.Errorf("validate budget: %w", err)
	}
	if budget.Period == "" {
		budget.Period = s.currentPeriod()
	}

	if err := s.budgets.Upsert(ctx, userID, budget); err != nil {
		return domain.Budget{}, fmt.Errorf("save budget: %w", err)
	}

	s.invalidateBudgetsCache(ctx, userID)
	return budget, nil
}

func (s *ledgerService) ListBudgets(ctx context.Context, userID string) ([]domain.Budget, error) {
	if budgets, ok, err := s.loadBudgetsFromCache(ctx, userID); err == nil && ok {
		return budgets, nil
	}

	budgets, err := s.budgets.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	for i := range budgets {
		spent, err := s.expenses.SumByCategory(ctx, userID, budgets[i].Category)
		if err != nil {
			return nil, fmt.Errorf("sum expenses for category %s: %w", budgets[i].Category, err)
		}
		budgets[i].Remaining = budgets[i].Limit - spent
		if budgets[i].Remaining < 0 {
			budgets[i].Remaining = 0
		}
		if budgets[i].Period == "" {
			budgets[i].Period = s.currentPeriod()
		}
	}

	s.saveBudgetsToCache(ctx, userID, budgets)
	return budgets, nil
}

func (s *ledgerService) AddTransaction(ctx context.Context, userID string, tx domain.Transaction) (domain.Transaction, error) {
	if err := tx.Validate(); err != nil {
		return domain.Transaction{}, fmt.Errorf("validate transaction: %w", err)
	}

	if err := s.ensureBudgetLimit(ctx, userID, tx); err != nil {
		return domain.Transaction{}, err
	}

	created, err := s.expenses.Create(ctx, userID, tx)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("create transaction: %w", err)
	}

	s.invalidateBudgetsCache(ctx, userID)
	return created, nil
}

func (s *ledgerService) ListTransactions(ctx context.Context, userID string) ([]domain.Transaction, error) {
	transactions, err := s.expenses.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}

	return transactions, nil
}

func (s *ledgerService) GetReportSummary(ctx context.Context, userID, from, to string) ([]domain.ReportSummary, error) {
	if from == "" || to == "" {
		return nil, errors.New("from and to must be provided")
	}

	if summary, ok, err := s.loadReportFromCache(ctx, from, to); err == nil && ok {
		log.Printf("[cache] HIT: report:summary:%s:%s", from, to)
		return summary, nil
	}
	log.Printf("[cache] MISS: report:summary:%s:%s", from, to)

	categories, err := s.getCategoriesForPeriod(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		result  = make(map[string]float64)
		errOnce sync.Once
		calcErr error
	)

	heartbeatCtx, heartbeatCancel := context.WithCancel(ctx)
	defer heartbeatCancel()
	go func() {
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				log.Println("[report] concurrent summary calculation in progress...")
			}
		}
	}()

	wg.Add(len(categories))
	for _, cat := range categories {
		cat := cat
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}
			
			var total float64
			row := s.expensesDB().QueryRowContext(ctx, "SELECT COALESCE(SUM(amount),0) FROM expenses WHERE category = $1 AND date >= $2 AND date <= $3", cat, from, to)
			if err := row.Scan(&total); err != nil {
				errOnce.Do(func() { calcErr = fmt.Errorf("scan sum for %s: %w", cat, err) })
				return
			}
			mu.Lock()
			result[cat] = total
			mu.Unlock()
		}()
	}

	wg.Wait()
	heartbeatCancel()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if calcErr != nil {
		return nil, calcErr
	}

	summary := make([]domain.ReportSummary, 0, len(result))
	var totalExpenses float64
	for k, v := range result {
		summary = append(summary, domain.ReportSummary{Category: k, Total: v})
		totalExpenses += v
	}

	budgets, err := s.budgets.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get budgets for totals: %w", err)
	}

	var totalBudget float64
	for _, b := range budgets {
		totalBudget += b.Limit
	}

	s.saveReportToCache(ctx, from, to, summary)

	return summary, nil
}

func (s *ledgerService) ensureBudgetLimit(ctx context.Context, userID string, tx domain.Transaction) error {
	budget, err := s.budgets.GetByCategory(ctx, userID, tx.Category)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get budget for category %s: %w", tx.Category, err)
	}

	spent, err := s.expenses.SumByCategory(ctx, userID, tx.Category)
	if err != nil {
		return fmt.Errorf("sum spent for category %s: %w", tx.Category, err)
	}

	if spent+tx.Amount > budget.Limit {
		return domain.ErrBudgetExceeded
	}

	return nil
}

func (s *ledgerService) currentPeriod() string {
	return s.now().Format("2006-01")
}

func (s *ledgerService) invalidateBudgetsCache(ctx context.Context, userID string) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(ctx, budgetsCacheKey+":"+userID)
}

func (s *ledgerService) loadBudgetsFromCache(ctx context.Context, userID string) ([]domain.Budget, bool, error) {
	if s.cache == nil {
		return nil, false, nil
	}

	cacheKey := budgetsCacheKey + ":" + userID
	data, err := s.cache.Get(ctx, cacheKey)
	if err != nil || data == "" {
		return nil, false, err
	}

	var budgets []domain.Budget
	if err := json.Unmarshal([]byte(data), &budgets); err != nil {
		return nil, false, err
	}
	return budgets, true, nil
}

func (s *ledgerService) saveBudgetsToCache(ctx context.Context, userID string, budgets []domain.Budget) {
	if s.cache == nil {
		return
	}

	data, err := json.Marshal(budgets)
	if err != nil {
		return
	}

	cacheKey := budgetsCacheKey + ":" + userID
	_ = s.cache.Set(ctx, cacheKey, data, defaultBudgetsTTL)
}

func (s *ledgerService) loadReportFromCache(ctx context.Context, from, to string) ([]domain.ReportSummary, bool, error) {
	if s.cache == nil {
		return nil, false, nil
	}

	cacheKey := reportCacheKey(from, to)
	data, err := s.cache.Get(ctx, cacheKey)
	if err != nil || data == "" {
		return nil, false, err
	}

	var summary []domain.ReportSummary
	if err := json.Unmarshal([]byte(data), &summary); err != nil {
		return nil, false, err
	}

	return summary, true, nil
}

func (s *ledgerService) saveReportToCache(ctx context.Context, from, to string, summary []domain.ReportSummary) {
	if s.cache == nil {
		return
	}

	data, err := json.Marshal(summary)
	if err != nil {
		return
	}

	_ = s.cache.Set(ctx, reportCacheKey(from, to), data, defaultReportTTL)
}

func reportCacheKey(from, to string) string {
	return fmt.Sprintf("report:summary:%s:%s", from, to)
}


func (s *ledgerService) getCategoriesForPeriod(ctx context.Context, userID, from, to string) ([]string, error) {
	cats := map[string]struct{}{}

	budgets, _ := s.budgets.List(ctx, userID)
	for _, b := range budgets {
		cats[b.Category] = struct{}{}
	}

	if db := s.expensesDB(); db != nil {
		const q = "SELECT DISTINCT category FROM expenses WHERE user_id = $1 AND date >= $2 AND date <= $3"
		rows, err := db.QueryContext(ctx, q, userID, from, to)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var c string
				if err := rows.Scan(&c); err == nil {
					cats[c] = struct{}{}
				}
			}
		}
	}
	list := make([]string, 0, len(cats))
	for cat := range cats {
		list = append(list, cat)
	}
	return list, nil
}


func (s *ledgerService) expensesDB() *sql.DB {
	type dbGetter interface{ DB() *sql.DB }
	if exp, ok := s.expenses.(dbGetter); ok {
		return exp.DB()
	}
	return nil
}


type BulkImportResult struct {
	Accepted int                `json:"accepted"`
	Rejected int                `json:"rejected"`
	Errors   []BulkImportError  `json:"errors"`
}

type BulkImportError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

func (s *ledgerService) ImportTransactionsBulk(ctx context.Context, userID string, txs []domain.Transaction, workers int) (shared.BulkImportResult, error) {
	type job struct {
		Index int
		Tx    domain.Transaction
	}
	type result struct {
		Index int
		Ok    bool
		Err   error
	}
	var (
		jobs    = make(chan job, len(txs))
		results = make(chan result, len(txs))
	)

	for idx, tx := range txs {
		jobs <- job{Index: idx, Tx: tx}
	}
	close(jobs)

	var wg sync.WaitGroup
	workerCount := workers
	if workerCount <= 0 {
		workerCount = 4
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if err := j.Tx.Validate(); err != nil {
					results <- result{Index: j.Index, Ok: false, Err: err}
					continue
				}
				err := s.ensureBudgetLimit(ctx, userID, j.Tx)
				if err != nil {
					results <- result{Index: j.Index, Ok: false, Err: err}
					continue
				}
				_, err = s.expenses.Create(ctx, userID, j.Tx)
				if err != nil {
					results <- result{Index: j.Index, Ok: false, Err: err}
				} else {
					results <- result{Index: j.Index, Ok: true, Err: nil}
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var bulkResult shared.BulkImportResult
	for r := range results {
		if ctx.Err() != nil {
			return bulkResult, ctx.Err()
		}
		if r.Ok {
			bulkResult.Accepted++
		} else {
			bulkResult.Rejected++
			if r.Err != nil {
				bulkResult.Errors = append(bulkResult.Errors, shared.BulkImportError{Index: r.Index, Error: r.Err.Error()})
			}
		}
	}

	return bulkResult, ctx.Err()
}

func (s *ledgerService) ImportCSV(ctx context.Context, userID string, csvData string) (shared.BulkImportResult, error) {
	transactions, err := s.expenses.ImportCSV(ctx, userID, csvData)
	if err != nil {
		return shared.BulkImportResult{}, fmt.Errorf("import CSV: %w", err)
	}

	return s.ImportTransactionsBulk(ctx, userID, transactions, 4)
}

func (s *ledgerService) ExportCSV(ctx context.Context, userID string, from, to string) (string, error) {
	return s.expenses.ExportCSV(ctx, userID, from, to)
}

