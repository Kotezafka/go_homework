package ledger

import (
	"context"
	"errors"
	"testing"
	"time"

	"ledger/internal/domain"
	internalservice "ledger/internal/service"
)

func TestTransactionValidate(t *testing.T) {
	t.Parallel()

	valid := Transaction{Amount: 100, Category: "food", Date: "2025-01-01"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("ожидалась валидная транзакция: %v", err)
	}

	invalid := []Transaction{
		{Amount: 0, Category: "food", Date: "2025-01-01"},
		{Amount: 10, Category: "", Date: "2025-01-01"},
		{Amount: 10, Category: "food", Date: ""},
	}

	for _, tx := range invalid {
		if err := tx.Validate(); err == nil {
			t.Fatalf("Validate() ожидалась ошибка для %#v", tx)
				}
			}
}

func TestBudgetValidate(t *testing.T) {
	t.Parallel()

	valid := Budget{Category: "food", Limit: 5000}
	if err := valid.Validate(); err != nil {
		t.Fatalf("ожидалась валидность бюджета: %v", err)
	}

	invalid := []Budget{
		{Category: "", Limit: 100},
		{Category: "food", Limit: 0},
	}

	for _, b := range invalid {
		if err := b.Validate(); err == nil {
			t.Fatalf("Validate() ожидалась ошибка для %#v", b)
				}
			}
}

func TestServiceAddTransactionBudgetLimit(t *testing.T) {
	ctx := context.Background()
	budgetRepo := newFakeBudgetRepo()
	expenseRepo := newFakeExpenseRepo()

	svc := internalservice.New(budgetRepo, expenseRepo, nil, func() time.Time {
		return time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	})

	budget := Budget{Category: "еда", Limit: 5000}
	if _, err := svc.SetBudget(ctx, budget); err != nil {
		t.Fatalf("SetBudget() ошибка: %v", err)
		}

	if _, err := svc.AddTransaction(ctx, Transaction{Amount: 3000, Category: "еда", Date: "2025-01-10"}); err != nil {
		t.Fatalf("AddTransaction() неожиданная ошибка: %v", err)
	}

	_, err := svc.AddTransaction(ctx, Transaction{Amount: 2500, Category: "еда", Date: "2025-01-11"})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("ожидалась ErrBudgetExceeded, получено: %v", err)
	}
	}

// --- тестовые репозитории ---

type fakeBudgetRepo struct {
	store map[string]domain.Budget
}

func newFakeBudgetRepo() *fakeBudgetRepo {
	return &fakeBudgetRepo{store: make(map[string]domain.Budget)}
}

func (r *fakeBudgetRepo) Upsert(_ context.Context, budget domain.Budget) error {
	r.store[budget.Category] = budget
	return nil
}

func (r *fakeBudgetRepo) GetByCategory(_ context.Context, category string) (domain.Budget, error) {
	if b, ok := r.store[category]; ok {
		return b, nil
	}
	return domain.Budget{}, domain.ErrNotFound
}

func (r *fakeBudgetRepo) List(_ context.Context) ([]domain.Budget, error) {
	result := make([]domain.Budget, 0, len(r.store))
	for _, b := range r.store {
		result = append(result, b)
	}
	return result, nil
}

type fakeExpenseRepo struct {
	items []domain.Transaction
}

func newFakeExpenseRepo() *fakeExpenseRepo {
	return &fakeExpenseRepo{}
}

func (r *fakeExpenseRepo) Create(_ context.Context, tx domain.Transaction) (domain.Transaction, error) {
	tx.ID = len(r.items) + 1
	r.items = append(r.items, tx)
	return tx, nil
}

func (r *fakeExpenseRepo) List(_ context.Context) ([]domain.Transaction, error) {
	cp := make([]domain.Transaction, len(r.items))
	copy(cp, r.items)
	return cp, nil
}

func (r *fakeExpenseRepo) SumByCategory(_ context.Context, category string) (float64, error) {
	var total float64
	for _, tx := range r.items {
		if tx.Category == category {
			total += tx.Amount
		}
	}
	return total, nil
}

func (r *fakeExpenseRepo) Summary(_ context.Context, _, _ string) ([]domain.ReportSummary, error) {
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

