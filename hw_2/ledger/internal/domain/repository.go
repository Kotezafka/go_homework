package domain

import "context"

// BudgetRepository описывает поведение хранилища бюджетов,
// необходимое бизнес-логике.
type BudgetRepository interface {
	Upsert(ctx context.Context, budget Budget) error
	GetByCategory(ctx context.Context, category string) (Budget, error)
	List(ctx context.Context) ([]Budget, error)
}

// ExpenseRepository описывает операции с расходами.
type ExpenseRepository interface {
	Create(ctx context.Context, tx Transaction) (Transaction, error)
	List(ctx context.Context) ([]Transaction, error)
	SumByCategory(ctx context.Context, category string) (float64, error)
	Summary(ctx context.Context, from, to string) ([]ReportSummary, error)
}

