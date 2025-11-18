package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrBudgetExceeded сигнализирует о нарушении лимита бюджета.
	ErrBudgetExceeded = errors.New("budget exceeded")
	// ErrNotFound сигнализирует об отсутствии запрашиваемой сущности.
	ErrNotFound = errors.New("not found")
)

// Validatable описывает сущности, которые могут быть провалидированы.
type Validatable interface {
	Validate() error
}

// Transaction представляет финансовую транзакцию.
type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string
}

// Validate проверяет корректность полей транзакции.
func (tx Transaction) Validate() error {
	if tx.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0, got %.2f", tx.Amount)
	}
	if tx.Category == "" {
		return errors.New("category must not be empty")
	}
	if tx.Date == "" {
		return errors.New("date must not be empty")
	}
	return nil
}

// Budget представляет бюджет на категорию.
type Budget struct {
	Category  string  `json:"category"`
	Limit     float64 `json:"limit"`
	Remaining float64 `json:"remaining"`
	Period    string  `json:"period"`
}

// Validate проверяет корректность бюджетной записи.
func (b Budget) Validate() error {
	if b.Category == "" {
		return errors.New("category must not be empty")
	}
	if b.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0, got %.2f", b.Limit)
	}
	return nil
}

// ReportSummary описывает сумму расходов по категории за период.
type ReportSummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// CheckValid — вспомогательная функция для демонстрации полиморфизма.
func CheckValid(v Validatable) error {
	return v.Validate()
}

