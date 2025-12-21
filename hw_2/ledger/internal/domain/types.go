package domain

import (
	"errors"
	"fmt"
)

var (
	ErrBudgetExceeded = errors.New("budget exceeded")
	ErrNotFound = errors.New("not found")
)

type Validatable interface {
	Validate() error
}

type Transaction struct {
	ID          int     `json:"id"`
	UserID      string  `json:"user_id"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

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

type Budget struct {
	UserID    string  `json:"user_id"`
	Category  string  `json:"category"`
	Limit     float64 `json:"limit"`
	Remaining float64 `json:"remaining"`
	Period    string  `json:"period"`
}

func (b Budget) Validate() error {
	if b.Category == "" {
		return errors.New("category must not be empty")
	}
	if b.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0, got %.2f", b.Limit)
	}
	return nil
}

type ReportSummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

func CheckValid(v Validatable) error {
	return v.Validate()
}

