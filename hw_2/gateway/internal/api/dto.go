package api

import (
	"ledger/shared"
	"ledger"
	"time"
)

// DTO для создания транзакции
type CreateTransactionRequest struct {
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

// DTO для ответа о транзакции
type TransactionResponse struct {
	ID          int     `json:"id"`
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
}

// DTO для создания/обновления бюджета
type CreateBudgetRequest struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
}

// DTO для ответа о бюджете
type BudgetResponse struct {
	Category   string  `json:"category"`
	Limit      float64 `json:"limit"`
	Remaining  float64 `json:"remaining"`
	Period     string  `json:"period"`
}

// BulkImportResultDTO - результат batch-импорта для ответа Gateway
type BulkImportResultDTO struct {
	Accepted int                    `json:"accepted"`
	Rejected int                    `json:"rejected"`
	Errors   []BulkImportErrorDTO   `json:"errors"`
}

type BulkImportErrorDTO struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

// Преобразует DTO в доменную модель
func (req *CreateTransactionRequest) ToDomainTransaction() ledger.Transaction {
	return ledger.Transaction{
		Amount:      req.Amount,
		Category:    req.Category,
		Description: req.Description,
		Date:        req.Date,
	}
}

// Преобразует доменную модель в DTO
func ToTransactionResponse(tx ledger.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Category:    tx.Category,
		Description: tx.Description,
		Date:        tx.Date,
	}
}

// Преобразует DTO в доменную модель
func (req *CreateBudgetRequest) ToDomainBudget() ledger.Budget {
	return ledger.Budget{
		Category:  req.Category,
		Limit:     req.Limit,
		Remaining: req.Limit,
		Period:    time.Now().Format("2006-01"),
	}
}

// Преобразует доменную модель в DTO
func ToBudgetResponse(b ledger.Budget) BudgetResponse {
	return BudgetResponse{
		Category:   b.Category,
		Limit:      b.Limit,
		Remaining:  b.Remaining,
		Period:     b.Period,
	}
}

// Маппинг из domain к DTO для ответа в API
func ToBulkImportResultDTO(r shared.BulkImportResult) BulkImportResultDTO {
	dto := BulkImportResultDTO{
		Accepted: r.Accepted,
		Rejected: r.Rejected,
	}
	for _, e := range r.Errors {
		dto.Errors = append(dto.Errors, BulkImportErrorDTO{
			Index: e.Index,
			Error: e.Error,
		})
	}
	return dto
}