package domain

import "context"

type BudgetRepository interface {
	Upsert(ctx context.Context, userID string, budget Budget) error
	GetByCategory(ctx context.Context, userID, category string) (Budget, error)
	List(ctx context.Context, userID string) ([]Budget, error)
}

type ExpenseRepository interface {
	Create(ctx context.Context, userID string, tx Transaction) (Transaction, error)
	List(ctx context.Context, userID string) ([]Transaction, error)
	SumByCategory(ctx context.Context, userID, category string) (float64, error)
	Summary(ctx context.Context, userID, from, to string) ([]ReportSummary, error)
	ImportCSV(ctx context.Context, userID string, csvData string) ([]Transaction, error)
	ExportCSV(ctx context.Context, userID string, from, to string) (string, error)
}
