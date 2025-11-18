package pg

import (
	"context"
	"database/sql"
	"fmt"

	"ledger/internal/domain"
)

// ExpenseRepository реализует доменный интерфейс ExpenseRepository.
type ExpenseRepository struct {
	db *sql.DB
}

// NewExpenseRepository конструирует ExpenseRepository.
func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) Create(ctx context.Context, tx domain.Transaction) (domain.Transaction, error) {
	const query = `
INSERT INTO expenses(amount, category, description, date)
VALUES ($1, $2, $3, $4)
RETURNING id`

	if err := r.db.QueryRowContext(ctx, query, tx.Amount, tx.Category, tx.Description, tx.Date).Scan(&tx.ID); err != nil {
		return domain.Transaction{}, fmt.Errorf("pg: insert expense: %w", err)
	}

	return tx, nil
}

func (r *ExpenseRepository) List(ctx context.Context) ([]domain.Transaction, error) {
	const query = `
SELECT id, amount, category, description, date
FROM expenses
ORDER BY date DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("pg: list expenses: %w", err)
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Category, &tx.Description, &tx.Date); err != nil {
			return nil, fmt.Errorf("pg: scan expense: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pg: iterate expenses: %w", err)
	}

	return transactions, nil
}

func (r *ExpenseRepository) SumByCategory(ctx context.Context, category string) (float64, error) {
	const query = `SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE category = $1`

	var spent float64
	if err := r.db.QueryRowContext(ctx, query, category).Scan(&spent); err != nil {
		return 0, fmt.Errorf("pg: sum expenses by category: %w", err)
	}

	return spent, nil
}

func (r *ExpenseRepository) Summary(ctx context.Context, from, to string) ([]domain.ReportSummary, error) {
	const query = `
SELECT category, COALESCE(SUM(amount), 0) AS total
FROM expenses
WHERE date >= $1 AND date <= $2
GROUP BY category
ORDER BY category`

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("pg: report summary: %w", err)
	}
	defer rows.Close()

	var summary []domain.ReportSummary
	for rows.Next() {
		var item domain.ReportSummary
		if err := rows.Scan(&item.Category, &item.Total); err != nil {
			return nil, fmt.Errorf("pg: scan report summary: %w", err)
		}
		summary = append(summary, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pg: iterate report summary: %w", err)
	}

	return summary, nil
}

