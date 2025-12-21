package ledger

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"ledger/internal/app"
	"ledger/internal/domain"
	"ledger/internal/service"
)

type (
	Transaction = domain.Transaction
	Budget = domain.Budget
	ReportSummary = domain.ReportSummary
	Validatable = domain.Validatable
)

var (
	ErrBudgetExceeded = domain.ErrBudgetExceeded

	testService Service
	testOnce    sync.Once
	testMu      sync.Mutex
)

type Service = service.Service

func New(ctx context.Context) (Service, func() error, error) {
	return app.NewService(ctx)
}

func getTestService() Service {
	testOnce.Do(func() {
		budgetRepo := &testBudgetRepo{store: make(map[string]domain.Budget)}
		expenseRepo := &testExpenseRepo{items: make([]domain.Transaction, 0)}
		testService = service.New(budgetRepo, expenseRepo, nil, nil)
	})
	return testService
}

func Reset() {
	testMu.Lock()
	defer testMu.Unlock()
	testService = nil
	testOnce = sync.Once{}
	budgetRepo := &testBudgetRepo{store: make(map[string]domain.Budget)}
	expenseRepo := &testExpenseRepo{items: make([]domain.Transaction, 0)}
	testService = service.New(budgetRepo, expenseRepo, nil, nil)
}

func AddTransaction(tx Transaction) error {
	ctx := context.Background()
	testMu.Lock()
	svc := getTestService()
	testMu.Unlock()
	_, err := svc.AddTransaction(ctx, "test", tx)
	return err
}

func ListTransactions() ([]Transaction, error) {
	ctx := context.Background()
	testMu.Lock()
	svc := getTestService()
	testMu.Unlock()
	return svc.ListTransactions(ctx, "test")
}

func SetBudget(b Budget) error {
	ctx := context.Background()
	testMu.Lock()
	svc := getTestService()
	testMu.Unlock()
	_, err := svc.SetBudget(ctx, "test", b)
	return err
}

func Budgets() ([]Budget, error) {
	ctx := context.Background()
	testMu.Lock()
	svc := getTestService()
	testMu.Unlock()
	return svc.ListBudgets(ctx, "test")
}

type testBudgetRepo struct {
	store map[string]domain.Budget
	mu    sync.Mutex
}

func (r *testBudgetRepo) Upsert(_ context.Context, userID string, budget domain.Budget) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[budget.Category] = budget
	return nil
}

func (r *testBudgetRepo) GetByCategory(_ context.Context, userID, category string) (domain.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.store[category]; ok {
		return b, nil
	}
	return domain.Budget{}, domain.ErrNotFound
}

func (r *testBudgetRepo) List(_ context.Context, userID string) ([]domain.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]domain.Budget, 0, len(r.store))
	for _, b := range r.store {
		result = append(result, b)
	}
	return result, nil
}

type testExpenseRepo struct {
	items []domain.Transaction
	mu    sync.Mutex
}

func (r *testExpenseRepo) Create(_ context.Context, userID string, tx domain.Transaction) (domain.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tx.ID = len(r.items) + 1
	r.items = append(r.items, tx)
	return tx, nil
}

func (r *testExpenseRepo) List(_ context.Context, userID string) ([]domain.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]domain.Transaction, len(r.items))
	copy(cp, r.items)
	return cp, nil
}

func (r *testExpenseRepo) SumByCategory(_ context.Context, userID, category string) (float64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var total float64
	for _, tx := range r.items {
		if tx.Category == category {
			total += tx.Amount
		}
	}
	return total, nil
}

func (r *testExpenseRepo) Summary(_ context.Context, userID, from, to string) ([]domain.ReportSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	agg := map[string]float64{}
	for _, tx := range r.items {
		agg[tx.Category] += tx.Amount
	}
	result := make([]domain.ReportSummary, 0, len(agg))
	for category, total := range agg {
		result = append(result, domain.ReportSummary{Category: category, Total: total})
	}
	return result, nil
}

func (r *testExpenseRepo) ImportCSV(_ context.Context, userID string, csvData string) ([]domain.Transaction, error) {
	return []domain.Transaction{}, nil
}

func (r *testExpenseRepo) ExportCSV(_ context.Context, userID string, from, to string) (string, error) {
	return "amount,category,description,date\n", nil
}

func CheckValid(v Validatable) error {
	return domain.CheckValid(v)
}

func LoadBudgets(ctx context.Context, svc Service, reader io.Reader) error {
	var payload []Budget
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return err
	}

	for _, budget := range payload {
		if _, err := svc.SetBudget(ctx, "test", budget); err != nil {
			return err
		}
	}

	return nil
}
