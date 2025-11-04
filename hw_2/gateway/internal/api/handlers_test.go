package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"ledger"
)

// setupTestRouter создаёт тестовый роутер, аналогичный main.go
func setupTestRouter() *http.ServeMux {
	router := http.NewServeMux()

	apiRouter := http.NewServeMux()
	apiRouter.HandleFunc("/transactions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateTransaction(w, r)
		case http.MethodGet:
			handleListTransactions(w, r)
		default:
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})
	apiRouter.HandleFunc("/budgets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreateBudget(w, r)
		case http.MethodGet:
			handleListBudgets(w, r)
		default:
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	router.Handle("/api/", http.StripPrefix("/api", apiRouter))
	return router
}

// Вспомогательные функции для тестов (копии из main.go)
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	tx := req.ToDomainTransaction()

	if err := tx.Validate(); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := ledger.AddTransaction(tx); err != nil {
		if strings.Contains(err.Error(), "Превышен бюджет для категории") {
			errorResponse(w, http.StatusConflict, "budget exceeded")
			return
		}
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	txs := ledger.ListTransactions()
	if len(txs) == 0 {
		errorResponse(w, http.StatusInternalServerError, "Failed to retrieve created transaction")
		return
	}
	createdTx := txs[len(txs)-1]

	resp := ToTransactionResponse(createdTx)
	writeJSON(w, http.StatusCreated, resp)
}

func handleListTransactions(w http.ResponseWriter, r *http.Request) {
	txs := ledger.ListTransactions()
	var responses []TransactionResponse
	for _, tx := range txs {
		responses = append(responses, ToTransactionResponse(tx))
	}

	writeJSON(w, http.StatusOK, responses)
}

func handleCreateBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	budget := req.ToDomainBudget()

	if err := budget.Validate(); err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := ledger.SetBudget(budget); err != nil {
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	resp := ToBudgetResponse(budget)
	writeJSON(w, http.StatusCreated, resp)
}

func handleListBudgets(w http.ResponseWriter, r *http.Request) {
	var budgets []BudgetResponse
	for _, b := range ledger.Budgets() {
		budgets = append(budgets, ToBudgetResponse(b))
	}

	writeJSON(w, http.StatusOK, budgets)
}

// TestBudgetsAPI тестирует API для работы с бюджетами
func TestBudgetsAPI(t *testing.T) {
	ledger.Reset()
	t.Cleanup(func() { ledger.Reset() })

	router := setupTestRouter()

	t.Run("POST /api/budgets - валидный бюджет", func(t *testing.T) {
		ledger.Reset()

		reqBody := CreateBudgetRequest{
			Category: "еда",
			Limit:    5000.0,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Ожидался статус 201, получен: %d", w.Code)
		}

		// Проверяем Content-Type
		contentType := w.Header().Get("Content-Type")
		expectedContentType := "application/json; charset=utf-8"
		if contentType != expectedContentType {
			t.Errorf("Ожидался Content-Type %s, получен: %s", expectedContentType, contentType)
		}

		// Проверяем JSON ответ
		var response BudgetResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if response.Category != "еда" {
			t.Errorf("Ожидалась категория 'еда', получена: %s", response.Category)
		}
		if response.Limit != 5000.0 {
			t.Errorf("Ожидался лимит 5000.0, получен: %.2f", response.Limit)
		}
	})

	t.Run("POST /api/budgets - невалидный JSON", func(t *testing.T) {
		ledger.Reset()

		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Ожидался статус 400, получен: %d", w.Code)
		}

		var errorResp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if errorResp["error"] == "" {
			t.Error("Ожидалось поле 'error' в ответе")
		}
	})

	t.Run("POST /api/budgets - нулевой лимит", func(t *testing.T) {
		ledger.Reset()

		reqBody := CreateBudgetRequest{
			Category: "еда",
			Limit:    0,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Ожидался статус 400, получен: %d", w.Code)
		}

		var errorResp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if errorResp["error"] == "" {
			t.Error("Ожидалось поле 'error' в ответе")
		}
	})

	t.Run("GET /api/budgets - список бюджетов", func(t *testing.T) {
		ledger.Reset()

		// Создаём бюджет через API
		reqBody := CreateBudgetRequest{
			Category: "еда",
			Limit:    5000.0,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Получаем список бюджетов
		req = httptest.NewRequest(http.MethodGet, "/api/budgets", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Ожидался статус 200, получен: %d", w.Code)
		}

		var budgets []BudgetResponse
		if err := json.Unmarshal(w.Body.Bytes(), &budgets); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if len(budgets) == 0 {
			t.Error("Ожидался хотя бы один бюджет в списке")
		}

		found := false
		for _, b := range budgets {
			if b.Category == "еда" && b.Limit == 5000.0 {
				found = true
				break
			}
		}

		if !found {
			t.Error("Не найден созданный бюджет в списке")
		}
	})
}

// TestTransactionsAPI тестирует API для работы с транзакциями
func TestTransactionsAPI(t *testing.T) {
	ledger.Reset()
	t.Cleanup(func() { ledger.Reset() })

	router := setupTestRouter()

	t.Run("ok - создание и получение транзакции", func(t *testing.T) {
		ledger.Reset()

		// Сначала создаём бюджет
		budgetReq := CreateBudgetRequest{
			Category: "еда",
			Limit:    5000.0,
		}
		body, _ := json.Marshal(budgetReq)
		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Создаём транзакцию
		txReq := CreateTransactionRequest{
			Amount:      1500.0,
			Category:    "еда",
			Description: "Обед",
			Date:        "2025-01-15",
		}
		body, _ = json.Marshal(txReq)
		req = httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Ожидался статус 201, получен: %d", w.Code)
		}

		var txResponse TransactionResponse
		if err := json.Unmarshal(w.Body.Bytes(), &txResponse); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if txResponse.Amount != 1500.0 {
			t.Errorf("Ожидалась сумма 1500.0, получена: %.2f", txResponse.Amount)
		}
		if txResponse.Category != "еда" {
			t.Errorf("Ожидалась категория 'еда', получена: %s", txResponse.Category)
		}
		if txResponse.Date != "2025-01-15" {
			t.Errorf("Ожидалась дата '2025-01-15', получена: %s", txResponse.Date)
		}

		// Получаем список транзакций
		req = httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Ожидался статус 200, получен: %d", w.Code)
		}

		var transactions []TransactionResponse
		if err := json.Unmarshal(w.Body.Bytes(), &transactions); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if len(transactions) != 1 {
			t.Errorf("Ожидалась 1 транзакция, получено: %d", len(transactions))
		}

		if transactions[0].ID != txResponse.ID {
			t.Errorf("ID транзакции не совпадает")
		}
	})

	t.Run("exceeded - превышение бюджета", func(t *testing.T) {
		ledger.Reset()

		// Создаём бюджет
		budgetReq := CreateBudgetRequest{
			Category: "еда",
			Limit:    5000.0,
		}
		body, _ := json.Marshal(budgetReq)
		req := httptest.NewRequest(http.MethodPost, "/api/budgets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Добавляем первую транзакцию
		txReq1 := CreateTransactionRequest{
			Amount:      3000.0,
			Category:    "еда",
			Description: "Обед",
			Date:        "2025-01-15",
		}
		body, _ = json.Marshal(txReq1)
		req = httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Пытаемся добавить транзакцию, которая превысит лимит
		initialTxs := ledger.ListTransactions()
		initialCount := len(initialTxs)

		txReq2 := CreateTransactionRequest{
			Amount:      2500.0, // 3000 + 2500 = 5500 > 5000
			Category:    "еда",
			Description: "Ужин",
			Date:        "2025-01-16",
		}
		body, _ = json.Marshal(txReq2)
		req = httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("Ожидался статус 409, получен: %d", w.Code)
		}

		var errorResp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if errorResp["error"] != "budget exceeded" {
			t.Errorf("Ожидалась ошибка 'budget exceeded', получена: %s", errorResp["error"])
		}

		// Проверяем, что транзакция не была добавлена
		finalTxs := ledger.ListTransactions()
		finalCount := len(finalTxs)
		if finalCount != initialCount {
			t.Errorf("Количество транзакций изменилось после ошибки: было %d, стало %d", initialCount, finalCount)
		}
	})

	t.Run("bad_json - некорректный JSON", func(t *testing.T) {
		ledger.Reset()

		req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Ожидался статус 400, получен: %d", w.Code)
		}

		var errorResp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if !strings.Contains(errorResp["error"], "Invalid JSON") {
			t.Errorf("Ожидалась ошибка 'Invalid JSON format', получена: %s", errorResp["error"])
		}
	})

	t.Run("validation_error - невалидная транзакция", func(t *testing.T) {
		ledger.Reset()

		// Транзакция с нулевой суммой
		txReq := CreateTransactionRequest{
			Amount:      0,
			Category:    "еда",
			Description: "Тест",
			Date:        "2025-01-15",
		}
		body, _ := json.Marshal(txReq)
		req := httptest.NewRequest(http.MethodPost, "/api/transactions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Ожидался статус 400, получен: %d", w.Code)
		}

		var errorResp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
			t.Fatalf("Не удалось распарсить JSON ответ: %v", err)
		}

		if errorResp["error"] == "" {
			t.Error("Ожидалось поле 'error' в ответе")
		}
	})
}

