package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ledger/internal/cache"
	"ledger/internal/domain"
)

const (
	budgetsCacheKey   = "budgets:all"
	defaultBudgetsTTL = 30 * time.Second
	defaultReportTTL  = 30 * time.Second
)

// Service описывает бизнес-операции Ledger.
type Service interface {
	SetBudget(ctx context.Context, budget domain.Budget) (domain.Budget, error)
	ListBudgets(ctx context.Context) ([]domain.Budget, error)
	AddTransaction(ctx context.Context, tx domain.Transaction) (domain.Transaction, error)
	ListTransactions(ctx context.Context) ([]domain.Transaction, error)
	GetReportSummary(ctx context.Context, from, to string) ([]domain.ReportSummary, error)
}

type ledgerService struct {
	budgets  domain.BudgetRepository
	expenses domain.ExpenseRepository
	cache    cache.Client
	now      func() time.Time
}

// New создаёт реализацию сервиса.
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

func (s *ledgerService) SetBudget(ctx context.Context, budget domain.Budget) (domain.Budget, error) {
	if err := budget.Validate(); err != nil {
		return domain.Budget{}, fmt.Errorf("validate budget: %w", err)
	}
	if budget.Period == "" {
		budget.Period = s.currentPeriod()
	}

	if err := s.budgets.Upsert(ctx, budget); err != nil {
		return domain.Budget{}, fmt.Errorf("save budget: %w", err)
	}

	s.invalidateBudgetsCache(ctx)
	return budget, nil
}

func (s *ledgerService) ListBudgets(ctx context.Context) ([]domain.Budget, error) {
	if budgets, ok, err := s.loadBudgetsFromCache(ctx); err == nil && ok {
		return budgets, nil
	}

	budgets, err := s.budgets.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range budgets {
		spent, err := s.expenses.SumByCategory(ctx, budgets[i].Category)
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

	s.saveBudgetsToCache(ctx, budgets)
	return budgets, nil
}

func (s *ledgerService) AddTransaction(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	if err := tx.Validate(); err != nil {
		return domain.Transaction{}, fmt.Errorf("validate transaction: %w", err)
	}

	if err := s.ensureBudgetLimit(ctx, tx); err != nil {
		return domain.Transaction{}, err
	}

	created, err := s.expenses.Create(ctx, tx)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("create transaction: %w", err)
	}

	s.invalidateBudgetsCache(ctx)
	return created, nil
}

func (s *ledgerService) ListTransactions(ctx context.Context) ([]domain.Transaction, error) {
	transactions, err := s.expenses.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}

	return transactions, nil
}

func (s *ledgerService) GetReportSummary(ctx context.Context, from, to string) ([]domain.ReportSummary, error) {
	if from == "" || to == "" {
		return nil, errors.New("from and to must be provided")
	}

	if summary, ok, err := s.loadReportFromCache(ctx, from, to); err == nil && ok {
		return summary, nil
	}

	summary, err := s.expenses.Summary(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("report summary: %w", err)
	}

	s.saveReportToCache(ctx, from, to, summary)
	return summary, nil
}

func (s *ledgerService) ensureBudgetLimit(ctx context.Context, tx domain.Transaction) error {
	budget, err := s.budgets.GetByCategory(ctx, tx.Category)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get budget for category %s: %w", tx.Category, err)
	}

	spent, err := s.expenses.SumByCategory(ctx, tx.Category)
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

func (s *ledgerService) invalidateBudgetsCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(ctx, budgetsCacheKey)
}

func (s *ledgerService) loadBudgetsFromCache(ctx context.Context) ([]domain.Budget, bool, error) {
	if s.cache == nil {
		return nil, false, nil
	}

	data, err := s.cache.Get(ctx, budgetsCacheKey)
	if err != nil || data == "" {
		return nil, false, err
	}

	var budgets []domain.Budget
	if err := json.Unmarshal([]byte(data), &budgets); err != nil {
		return nil, false, err
	}
	return budgets, true, nil
}

func (s *ledgerService) saveBudgetsToCache(ctx context.Context, budgets []domain.Budget) {
	if s.cache == nil {
		return
	}

	data, err := json.Marshal(budgets)
	if err != nil {
		return
	}

	_ = s.cache.Set(ctx, budgetsCacheKey, data, defaultBudgetsTTL)
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

