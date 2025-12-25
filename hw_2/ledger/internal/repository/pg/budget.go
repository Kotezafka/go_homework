package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ledger/internal/domain"
)


type BudgetRepository struct {
	db *sql.DB
}

func NewBudgetRepository(db *sql.DB) *BudgetRepository {
	return &BudgetRepository{db: db}
}

func (r *BudgetRepository) Upsert(ctx context.Context, budget domain.Budget) error {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("pg: upsert budget: %w", err)
	}
	const query = `
INSERT INTO budgets(user_id, category, limit_amount)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount`

	if _, err := r.db.ExecContext(ctx, query, userID, budget.Category, budget.Limit); err != nil {
		return fmt.Errorf("pg: upsert budget: %w", err)
	}

	return nil
}

func (r *BudgetRepository) GetByCategory(ctx context.Context, category string) (domain.Budget, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return domain.Budget{}, fmt.Errorf("pg: get budget by category: %w", err)
	}

	const query = `SELECT category, limit_amount FROM budgets WHERE user_id = $1 AND category = $2`

	var budget domain.Budget
	if err := r.db.QueryRowContext(ctx, query, userID, category).Scan(&budget.Category, &budget.Limit); err != nil {
		if err == sql.ErrNoRows {
			return domain.Budget{}, domain.ErrNotFound
		}
		return domain.Budget{}, fmt.Errorf("pg: get budget by category: %w", err)
	}

	return budget, nil
}

func (r *BudgetRepository) List(ctx context.Context) ([]domain.Budget, error) {
	userID, err := userIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("pg: list budgets: %w", err)
	}

	const query = `SELECT category, limit_amount FROM budgets WHERE user_id = $1 ORDER BY category`

	rows, err := r.db.QueryContext(ctx, query, userID)
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

func userIDFromContext(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("missing user id in context")
	}
	for _, k := range []any{"userID", "userId", "user_id"} {
		if v := ctx.Value(k); v != nil {
			if s, ok := v.(string); ok && s != "" {
				return s, nil
			}
		}
	}
	return "", errors.New("missing user id in context")
}

