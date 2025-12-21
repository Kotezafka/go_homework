package pg

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"ledger/internal/domain"
)

type ExpenseRepository struct {
	db *sql.DB
}

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) DB() *sql.DB {
	return r.db
}

func (r *ExpenseRepository) Create(ctx context.Context, userID string, tx domain.Transaction) (domain.Transaction, error) {
	const query = `
INSERT INTO expenses(user_id, amount, category, description, date)
VALUES ($1, $2, $3, $4, $5)
RETURNING id`

	if err := r.db.QueryRowContext(ctx, query, userID, tx.Amount, tx.Category, tx.Description, tx.Date).Scan(&tx.ID); err != nil {
		return domain.Transaction{}, fmt.Errorf("pg: insert expense: %w", err)
	}

	return tx, nil
}

func (r *ExpenseRepository) List(ctx context.Context, userID string) ([]domain.Transaction, error) {
	const query = `
SELECT id, user_id, amount, category, description, date
FROM expenses
WHERE user_id = $1
ORDER BY date DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("pg: list expenses: %w", err)
	}
	defer rows.Close()

	var transactions []domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Amount, &tx.Category, &tx.Description, &tx.Date); err != nil {
			return nil, fmt.Errorf("pg: scan expense: %w", err)
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pg: iterate expenses: %w", err)
	}

	return transactions, nil
}

func (r *ExpenseRepository) SumByCategory(ctx context.Context, userID, category string) (float64, error) {
	const query = `SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE user_id = $1 AND category = $2`

	var spent float64
	if err := r.db.QueryRowContext(ctx, query, userID, category).Scan(&spent); err != nil {
		return 0, fmt.Errorf("pg: sum expenses by category: %w", err)
	}

	return spent, nil
}

func (r *ExpenseRepository) Summary(ctx context.Context, userID, from, to string) ([]domain.ReportSummary, error) {
	const query = `
SELECT category, COALESCE(SUM(amount), 0) AS total
FROM expenses
WHERE user_id = $1 AND date >= $2 AND date <= $3
GROUP BY category
ORDER BY category`

	rows, err := r.db.QueryContext(ctx, query, userID, from, to)
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

func (r *ExpenseRepository) ImportCSV(ctx context.Context, userID string, csvData string) ([]domain.Transaction, error) {
	reader := csv.NewReader(strings.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV: %w", err)
	}

	if len(records) == 0 {
		return []domain.Transaction{}, nil
	}

	startIndex := 0
	if len(records[0]) >= 4 && strings.ToLower(records[0][0]) == "amount" {
		startIndex = 1
	}

	var transactions []domain.Transaction
	for i := startIndex; i < len(records); i++ {
		record := records[i]
		if len(record) < 4 {
			continue
		}

		amount, err := strconv.ParseFloat(strings.TrimSpace(record[0]), 64)
		if err != nil {
			continue
		}

		tx := domain.Transaction{
			UserID:      userID,
			Amount:      amount,
			Category:    strings.TrimSpace(record[1]),
			Description: strings.TrimSpace(record[2]),
			Date:        strings.TrimSpace(record[3]),
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (r *ExpenseRepository) ExportCSV(ctx context.Context, userID string, from, to string) (string, error) {
	const query = `
SELECT amount, category, description, date
FROM expenses
WHERE user_id = $1 AND date >= $2 AND date <= $3
ORDER BY date DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return "", fmt.Errorf("pg: export CSV: %w", err)
	}
	defer rows.Close()

	var records [][]string
	records = append(records, []string{"amount", "category", "description", "date"})

	for rows.Next() {
		var amount float64
		var category, description, date string
		if err := rows.Scan(&amount, &category, &description, &date); err != nil {
			return "", fmt.Errorf("pg: scan for CSV export: %w", err)
		}
		records = append(records, []string{
			fmt.Sprintf("%.2f", amount),
			category,
			description,
			date,
		})
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("pg: iterate for CSV export: %w", err)
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	if err := writer.WriteAll(records); err != nil {
		return "", fmt.Errorf("write CSV: %w", err)
	}

	return buf.String(), nil
}

