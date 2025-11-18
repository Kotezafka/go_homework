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
	// Transaction — публичная проекция доменной транзакции.
	Transaction = domain.Transaction
	// Budget — публичная проекция доменного бюджета.
	Budget = domain.Budget
	// ReportSummary — проекция сводного отчёта.
	ReportSummary = domain.ReportSummary
	// Validatable — контракт для валидации сущностей.
	Validatable = domain.Validatable
)

var (
	// ErrBudgetExceeded возвращается при нарушении лимита бюджета.
	ErrBudgetExceeded = domain.ErrBudgetExceeded

	// testService используется для обратной совместимости с тестами
	testService Service
	testOnce    sync.Once
	testMu      sync.Mutex
)

// Service описывает публичный интерфейс приложения Ledger.
type Service = service.Service

// New создаёт сервис и возвращает функцию для освобождения ресурсов.
func New(ctx context.Context) (Service, func() error, error) {
	return app.NewService(ctx)
}

// getTestService возвращает in-memory сервис для тестов
func getTestService() Service {
	testOnce.Do(func() {
		budgetRepo := &testBudgetRepo{store: make(map[string]domain.Budget)}
		expenseRepo := &testExpenseRepo{items: make([]domain.Transaction, 0)}
		testService = service.New(budgetRepo, expenseRepo, nil, nil)
	})
	return testService
}

// Reset очищает данные (используется в тестах)
func Reset() {
	testMu.Lock()
	defer testMu.Unlock()
	testService = nil
	testOnce = sync.Once{}
	// Создаем новый чистый сервис
	budgetRepo := &testBudgetRepo{store: make(map[string]domain.Budget)}
	expenseRepo := &testExpenseRepo{items: make([]domain.Transaction, 0)}
	testService = service.New(budgetRepo, expenseRepo, nil, nil)
}

// AddTransaction добавляет транзакцию (для обратной совместимости с тестами)
func AddTransaction(tx Transaction) error {
	ctx := context.Background()
	testMu.Lock()
	svc := getTestService()
	testMu.Unlock()
	_, err := svc.AddTransaction(ctx, tx)
	return err
}

// ListTransactions возвращает список транзакций (для обратной совместимости с тестами)
func ListTransactions() ([]Transaction, error) {
	ctx := context.Background()
	testMu.Lock()
	svc := getTestService()
	testMu.Unlock()
	return svc.ListTransactions(ctx)
}

// SetBudget устанавливает бюджет (для обратной совместимости с тестами)
func SetBudget(b Budget) error {
	ctx := context.Background()
	testMu.Lock()
	svc := getTestService()
	testMu.Unlock()
	_, err := svc.SetBudget(ctx, b)
	return err
}

// Budgets возвращает список бюджетов (для обратной совместимости с тестами)
func Budgets() ([]Budget, error) {
	ctx := context.Background()
	testMu.Lock()
	svc := getTestService()
	testMu.Unlock()
	return svc.ListBudgets(ctx)
}

// testBudgetRepo - in-memory репозиторий для тестов
type testBudgetRepo struct {
	store map[string]domain.Budget
	mu    sync.Mutex
}

func (r *testBudgetRepo) Upsert(_ context.Context, budget domain.Budget) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[budget.Category] = budget
	return nil
}

func (r *testBudgetRepo) GetByCategory(_ context.Context, category string) (domain.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.store[category]; ok {
		return b, nil
	}
	return domain.Budget{}, domain.ErrNotFound
}

func (r *testBudgetRepo) List(_ context.Context) ([]domain.Budget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]domain.Budget, 0, len(r.store))
	for _, b := range r.store {
		result = append(result, b)
	}
	return result, nil
}

// testExpenseRepo - in-memory репозиторий для тестов
type testExpenseRepo struct {
	items []domain.Transaction
	mu    sync.Mutex
}

func (r *testExpenseRepo) Create(_ context.Context, tx domain.Transaction) (domain.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	tx.ID = len(r.items) + 1
	r.items = append(r.items, tx)
	return tx, nil
}

func (r *testExpenseRepo) List(_ context.Context) ([]domain.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]domain.Transaction, len(r.items))
	copy(cp, r.items)
	return cp, nil
}

func (r *testExpenseRepo) SumByCategory(_ context.Context, category string) (float64, error) {
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

func (r *testExpenseRepo) Summary(_ context.Context, _, _ string) ([]domain.ReportSummary, error) {
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

// CheckValid делегирует валидацию сущности домену.
func CheckValid(v Validatable) error {
	return domain.CheckValid(v)
}

// LoadBudgets читает бюджеты из JSON и добавляет их через сервис.
func LoadBudgets(ctx context.Context, svc Service, reader io.Reader) error {
	var payload []Budget
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		return err
	}

	for _, budget := range payload {
		if _, err := svc.SetBudget(ctx, budget); err != nil {
			return err
		}
	}

	return nil
}
