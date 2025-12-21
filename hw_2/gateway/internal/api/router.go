package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"ledger"
)

type MockServicesInterface interface {
	Register(email, password, name string) (string, error)
	Login(email, password string) (string, error)
	ValidateToken(token string) (string, error)
	SetBudget(userID string, category string, limit float64) error
	ListBudgets(userID string) []BudgetResponse
	AddTransaction(userID string, amount float64, category, description, date string) (TransactionResponse, error)
	ListTransactions(userID string) []TransactionResponse
	GetReportSummary(userID, from, to string) (map[string]interface{}, error)
	ImportTransactionsBulk(userID string, transactions []CreateTransactionRequest) (map[string]interface{}, error)
}

type AuthClient interface {
	Register(ctx context.Context, email, password, name string) (string, error)
	Login(ctx context.Context, email, password string) (string, error)
	ValidateToken(ctx context.Context, token string) (string, error)
}

func NewRouter(svc ledger.Service, authClient AuthClient) http.Handler {
	h := handler{svc: svc}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/transactions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.handleCreateTransaction(w, r)
		case http.MethodGet:
			h.handleListTransactions(w, r)
		default:
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	apiMux.HandleFunc("/transactions/bulk", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		h.handleImportBulk(w, r)
	})

	apiMux.HandleFunc("/budgets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.handleCreateBudget(w, r)
		case http.MethodGet:
			h.handleListBudgets(w, r)
		default:
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})

	apiMux.HandleFunc("/reports/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		h.handleGetReportSummary(w, r)
	})

	rootMux := http.NewServeMux()
	rootMux.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Name     string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorResponse(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}
		if req.Email == "" || req.Password == "" || req.Name == "" {
			errorResponse(w, http.StatusBadRequest, "email, password and name are required")
			return
		}
		userID, err := authClient.Register(r.Context(), req.Email, req.Password, req.Name)
		if err != nil {
			errorResponse(w, http.StatusConflict, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"user_id":  userID,
			"message": "user registered successfully",
		})
	})
	rootMux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errorResponse(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}
		if req.Email == "" || req.Password == "" {
			errorResponse(w, http.StatusBadRequest, "email and password are required")
			return
		}
		token, err := authClient.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			errorResponse(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"token": token})
	})
	rootMux.Handle("/api/", jwtMiddleware(authClient, timeoutMiddleware(2*time.Second, loggingMiddleware(http.StripPrefix("/api", apiMux)))))
	rootMux.HandleFunc("/ping", handlePing)

	return rootMux
}

type handler struct {
	svc ledger.Service
}

func (h handler) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

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

	created, err := h.svc.AddTransaction(r.Context(), userID, tx)
	if err != nil {
		if errors.Is(err, ledger.ErrBudgetExceeded) {
			errorResponse(w, http.StatusConflict, "budget exceeded")
			return
		}
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, ToTransactionResponse(created))
}

func (h handler) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	txs, err := h.svc.ListTransactions(r.Context(), userID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	responses := make([]TransactionResponse, 0, len(txs))
	for _, tx := range txs {
		responses = append(responses, ToTransactionResponse(tx))
	}

	writeJSON(w, http.StatusOK, responses)
}

func (h handler) handleCreateBudget(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

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

	created, err := h.svc.SetBudget(r.Context(), userID, budget)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, ToBudgetResponse(created))
}

func (h handler) handleListBudgets(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	budgets, err := h.svc.ListBudgets(r.Context(), userID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	responses := make([]BudgetResponse, 0, len(budgets))
	for _, b := range budgets {
		responses = append(responses, ToBudgetResponse(b))
	}

	writeJSON(w, http.StatusOK, responses)
}

func (h handler) handleGetReportSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		errorResponse(w, http.StatusBadRequest, "Parameters 'from' and 'to' are required (format: YYYY-MM-DD)")
		return
	}

	summary, err := h.svc.GetReportSummary(r.Context(), userID, from, to)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			errorResponse(w, 504, "request timeout or cancelled")
			return
		}
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	budgets, err := h.svc.ListBudgets(r.Context(), userID)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	var totalExpenses, totalBudget float64
	for _, s := range summary {
		totalExpenses += s.Total
	}
	for _, b := range budgets {
		totalBudget += b.Limit
	}

	budgetUsagePercent := 0.0
	if totalBudget > 0 {
		budgetUsagePercent = (totalExpenses / totalBudget) * 100
	}

	response := map[string]interface{}{
		"summaries":            summary,
		"total_expenses":       totalExpenses,
		"total_budget":         totalBudget,
		"budget_usage_percent": budgetUsagePercent,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h handler) handleImportBulk(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var reqs []CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON format (array expected)")
		return
	}
	if len(reqs) == 0 {
		errorResponse(w, http.StatusBadRequest, "Empty payload")
		return
	}
	txs := make([]ledger.Transaction, len(reqs))
	for i, req := range reqs {
		txs[i] = req.ToDomainTransaction()
	}
	workers := 4
	if n := r.URL.Query().Get("workers"); n != "" {
		if parsed, err := strconv.Atoi(n); err == nil && parsed > 0 && parsed <= 32 {
			workers = parsed
		}
	}
	result, err := h.svc.ImportTransactionsBulk(r.Context(), userID, txs, workers)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			errorResponse(w, 504, "request timeout or cancelled")
		} else {
			errorResponse(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, ToBulkImportResultDTO(result))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "pong")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("[LOG] %s %s took %v\n", r.Method, r.URL.Path, time.Since(start))
	})
}

type respWrapper struct {
	http.ResponseWriter
	wroteHeader atomic.Bool
}
func (rw *respWrapper) WriteHeader(code int) {
	if rw.wroteHeader.CompareAndSwap(false, true) {
		rw.ResponseWriter.WriteHeader(code)
	}
}
func (rw *respWrapper) Write(b []byte) (int, error) {
	rw.wroteHeader.CompareAndSwap(false, true)
	return rw.ResponseWriter.Write(b)
}


func timeoutMiddleware(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		resp := &respWrapper{ResponseWriter: w}
		done := make(chan struct{})
		reqWithCtx := r.WithContext(ctx)

		go func() {
			next.ServeHTTP(resp, reqWithCtx)
			close(done)
		}()

		select {
		case <-ctx.Done():
			if !resp.wroteHeader.Load() {
				errorResponse(resp, 504, "request timeout or cancelled")
			}
		case <-done:
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func jwtMiddleware(authClient AuthClient, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldSkipAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			errorResponse(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			errorResponse(w, http.StatusUnauthorized, "Invalid authorization format")
			return
		}

		token := parts[1]

		userID, err := authClient.ValidateToken(r.Context(), token)
		if err != nil {
			errorResponse(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		if userID == "" {
			errorResponse(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func shouldSkipAuth(path string) bool {
	return false
}


func NewRouterWithMocks(mockServices MockServicesInterface) http.Handler {
	h := mockHandler{services: mockServices}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/auth/register", h.handleRegister)
	apiMux.HandleFunc("/auth/login", h.handleLogin)
	apiMux.HandleFunc("/api/budgets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.handleCreateBudget(w, r)
		case http.MethodGet:
			h.handleListBudgets(w, r)
		default:
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})
	apiMux.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.handleCreateTransaction(w, r)
		case http.MethodGet:
			h.handleListTransactions(w, r)
		default:
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	})
	apiMux.HandleFunc("/api/transactions/bulk", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		h.handleImportBulk(w, r)
	})
	apiMux.HandleFunc("/api/reports/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		h.handleGetReportSummary(w, r)
	})

	rootMux := http.NewServeMux()
	rootMux.Handle("/api/", mockJWTAuthMiddleware(mockServices)(apiMux))
	rootMux.HandleFunc("/ping", handlePing)

	return rootMux
}

type mockHandler struct {
	services MockServicesInterface
}

func mockJWTAuthMiddleware(services MockServicesInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/auth/register" || r.URL.Path == "/auth/login" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				errorResponse(w, http.StatusUnauthorized, "Authorization header required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				errorResponse(w, http.StatusUnauthorized, "Invalid authorization format")
				return
			}

			token := parts[1]
			userID, err := services.ValidateToken(token)
			if err != nil {
				errorResponse(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *mockHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	userID, err := h.services.Register(req.Email, req.Password, req.Name)
	if err != nil {
		errorResponse(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":  userID,
		"message": "user registered successfully",
	})
}

func (h *mockHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	token, err := h.services.Login(req.Email, req.Password)
	if err != nil {
		errorResponse(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":     token,
		"user_id":   "mock-user-id",
		"expires_at": "2025-12-31T23:59:59Z",
	})
}

func (h *mockHandler) handleCreateBudget(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	err := h.services.SetBudget(userID, req.Category, req.Limit)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, BudgetResponse{
		Category:  req.Category,
		Limit:     req.Limit,
		Remaining: req.Limit,
		Period:    "2025-01",
	})
}

func (h *mockHandler) handleListBudgets(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	budgets := h.services.ListBudgets(userID)
	writeJSON(w, http.StatusOK, budgets)
}

func (h *mockHandler) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	transaction, err := h.services.AddTransaction(userID, req.Amount, req.Category, req.Description, req.Date)
	if err != nil {
		if strings.Contains(err.Error(), "budget exceeded") {
			errorResponse(w, http.StatusConflict, "budget exceeded")
		} else {
			errorResponse(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, transaction)
}

func (h *mockHandler) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	transactions := h.services.ListTransactions(userID)
	writeJSON(w, http.StatusOK, transactions)
}

func (h *mockHandler) handleGetReportSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		errorResponse(w, http.StatusBadRequest, "Parameters 'from' and 'to' are required")
		return
	}

	summary, err := h.services.GetReportSummary(userID, from, to)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (h *mockHandler) handleImportBulk(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		errorResponse(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var reqs []CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON format (array expected)")
		return
	}

	result, err := h.services.ImportTransactionsBulk(userID, reqs)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

