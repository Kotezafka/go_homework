package pg

import (
	"context"
	"database/sql"
	"fmt"

	"ledger/internal/domain"
)

// BudgetRepository реализует доменный интерфейс BudgetRepository
// поверх PostgreSQL.
type BudgetRepository struct {
	db *sql.DB
}

// NewBudgetRepository создаёт репозиторий бюджетов.
func NewBudgetRepository(db *sql.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

func (r *BudgetRepository) Upsert(ctx context.Context, budget domain.Budget) error {
	const query = `
INSERT INTO budgets(category, limit_amount)
VALUES ($1, $2)
ON CONFLICT (category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount`

	if _, err := r.db.ExecContext(ctx, query, budget.Category, budget.Limit); err != nil {
		return fmt.Errorf("pg: upsert budget: %w", err)
	}

	return nil
}

func (r *BudgetRepository) GetByCategory(ctx context.Context, category string) (domain.Budget, error) {
	const query = `SELECT category, limit_amount FROM budgets WHERE category = $1`

	var budget domain.Budget
	if err := r.db.QueryRowContext(ctx, query, category).Scan(&budget.Category, &budget.Limit); err != nil {
		if err == sql.ErrNoRows {
			return domain.Budget{}, domain.ErrNotFound
		}
		return domain.Budget{}, fmt.Errorf("pg: get budget by category: %w", err)
	}

	return budget, nil
}

func (r *BudgetRepository) List(ctx context.Context) ([]domain.Budget, error) {
	const query = `SELECT category, limit_amount FROM budgets ORDER BY category`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("pg: list budgets: %w", err)
	}
	defer rows.Close()

	var budgets []domain.Budget
	for rows.Next() {
		var b domain.Budget
		if err := rows.Scan(&b.Category, &b.Limit); err != nil {
			return nil, fmt.Errorf("pg: scan budget: %w", err)
		}
		budgets = append(budgets, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pg: iterate budgets: %w", err)
	}

	return budgets, nil
}

