package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ledger"
)

func TestBudgetsAPI(t *testing.T) {
	t.Run("POST /api/budgets - валидный бюджет", func(t *testing.T) {
		svc := newFakeLedgerService()
		router := NewRouter(svc)

		reqBody := CreateBudgetRequest{Category: "еда", Limit: 5000.0}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		if res.Code != http.StatusCreated {
			t.Fatalf("ожидался статус 201, получен %d", res.Code)
		}

		var response BudgetResponse
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
			t.Fatalf("не удалось распарсить ответ: %v", err)
		}

		if response.Category != "еда" || response.Limit != 5000 {
			t.Errorf("неожиданный ответ %+v", response)
		}
	})

	t.Run("POST /api/budgets - невалидный JSON", func(t *testing.T) {
		router := NewRouter(newFakeLedgerService())

		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("ожидался 400, получен %d", res.Code)
		}
	})

	t.Run("POST /api/budgets - нулевой лимит", func(t *testing.T) {
		router := NewRouter(newFakeLedgerService())

		reqBody := CreateBudgetRequest{Category: "еда", Limit: 0}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("ожидался 400, получен %d", res.Code)
		}
	})

	t.Run("GET /api/budgets - список бюджетов", func(t *testing.T) {
		svc := newFakeLedgerService()
		router := NewRouter(svc)

		reqBody := CreateBudgetRequest{Category: "еда", Limit: 5000.0}
		body, _ := json.Marshal(reqBody)
		createReq := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
		createReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(httptest.NewRecorder(), createReq)

		req := httptest.NewRequest(http.MethodGet, "/api/budgets", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		if res.Code != http.StatusOK {
			t.Fatalf("ожидался 200, получен %d", res.Code)
		}

		var budgets []BudgetResponse
		if err := json.Unmarshal(res.Body.Bytes(), &budgets); err != nil {
			t.Fatalf("не удалось распарсить ответ: %v", err)
		}

		if len(budgets) == 0 {
			t.Fatal("ожидался хотя бы один бюджет")
		}
	})
}

func TestTransactionsAPI(t *testing.T) {
	t.Run("ok - создание и получение транзакции", func(t *testing.T) {
		svc := newFakeLedgerService()
		router := NewRouter(svc)

		createBudget(router, CreateBudgetRequest{Category: "еда", Limit: 5000})

		txReq := CreateTransactionRequest{
			Amount:      1500,
			Category:    "еда",
			Description: "Обед",
			Date:        "2025-01-15",
		}
		body, _ := json.Marshal(txReq)
		req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		if res.Code != http.StatusCreated {
			t.Fatalf("ожидался 201, получен %d", res.Code)
		}

		req = httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
		res = httptest.NewRecorder()
		router.ServeHTTP(res, req)

		var txs []TransactionResponse
		if err := json.Unmarshal(res.Body.Bytes(), &txs); err != nil {
			t.Fatalf("не удалось распарсить ответ: %v", err)
		}

		if len(txs) != 1 || txs[0].Category != "еда" {
			t.Fatalf("неожиданный список транзакций: %+v", txs)
		}
	})

	t.Run("exceeded - превышение бюджета", func(t *testing.T) {
		svc := newFakeLedgerService()
		router := NewRouter(svc)

		createBudget(router, CreateBudgetRequest{Category: "еда", Limit: 5000})
		createTransaction(router, CreateTransactionRequest{Amount: 3000, Category: "еда", Date: "2025-01-15"})

		req := httptest.NewRequest(http.MethodPost, "/api/transactions",
			bytes.NewBufferString(`{"amount":2500,"category":"еда","date":"2025-01-16"}`))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		if res.Code != http.StatusConflict {
			t.Fatalf("ожидался 409, получен %d", res.Code)
		}
	})

	t.Run("bad_json - некорректный JSON", func(t *testing.T) {
		router := NewRouter(newFakeLedgerService())
		req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBufferString("bad"))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("ожидался 400, получен %d", res.Code)
		}
	})

	t.Run("validation_error - невалидная транзакция", func(t *testing.T) {
		router := NewRouter(newFakeLedgerService())
		reqBody := CreateTransactionRequest{Amount: 0, Category: "еда"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("ожидался 400, получен %d", res.Code)
		}
	})
}

func createBudget(router http.Handler, payload CreateBudgetRequest) {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), req)
}

func createTransaction(router http.Handler, payload CreateTransactionRequest) {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(httptest.NewRecorder(), req)
}

type fakeLedgerService struct {
	budgets      map[string]ledger.Budget
	transactions []ledger.Transaction
}

func newFakeLedgerService() *fakeLedgerService {
	return &fakeLedgerService{
		budgets: make(map[string]ledger.Budget),
	}
}

func (f *fakeLedgerService) SetBudget(_ context.Context, budget ledger.Budget) (ledger.Budget, error) {
	f.budgets[budget.Category] = budget
	return budget, nil
}

func (f *fakeLedgerService) ListBudgets(_ context.Context) ([]ledger.Budget, error) {
	result := make([]ledger.Budget, 0, len(f.budgets))
	for _, b := range f.budgets {
		b.Remaining = b.Limit - f.spentForCategory(b.Category)
		result = append(result, b)
	}
	return result, nil
}

func (f *fakeLedgerService) AddTransaction(_ context.Context, tx ledger.Transaction) (ledger.Transaction, error) {
	if tx.Amount <= 0 || tx.Category == "" {
		return ledger.Transaction{}, fmt.Errorf("invalid transaction")
	}

	if budget, ok := f.budgets[tx.Category]; ok {
		if f.spentForCategory(tx.Category)+tx.Amount > budget.Limit {
			return ledger.Transaction{}, ledger.ErrBudgetExceeded
		}
	}

	tx.ID = len(f.transactions) + 1
	f.transactions = append(f.transactions, tx)
	return tx, nil
}

func (f *fakeLedgerService) ListTransactions(_ context.Context) ([]ledger.Transaction, error) {
	cpy := make([]ledger.Transaction, len(f.transactions))
	copy(cpy, f.transactions)
	return cpy, nil
}

func (f *fakeLedgerService) GetReportSummary(_ context.Context, _, _ string) ([]ledger.ReportSummary, error) {
	summary := make(map[string]float64)
	for _, tx := range f.transactions {
		summary[tx.Category] += tx.Amount
	}
	result := make([]ledger.ReportSummary, 0, len(summary))
	for category, total := range summary {
		result = append(result, ledger.ReportSummary{Category: category, Total: total})
	}
	return result, nil
}

func (f *fakeLedgerService) spentForCategory(category string) float64 {
	var total float64
	for _, tx := range f.transactions {
		if tx.Category == category {
			total += tx.Amount
		}
	}
	return total
}

