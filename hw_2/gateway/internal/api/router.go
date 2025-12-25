package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"ledger"
)

// NewRouter конструирует HTTP-маршрутизатор поверх сервисного интерфейса.
func NewRouter(svc ledger.Service) http.Handler {
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
	rootMux.Handle("/api/", userIDMiddleware(timeoutMiddleware(2*time.Second, loggingMiddleware(http.StripPrefix("/api", apiMux)))))
	rootMux.HandleFunc("/ping", handlePing)

	return rootMux
}

type handler struct {
	svc ledger.Service
}

func (h handler) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
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

	created, err := h.svc.AddTransaction(r.Context(), tx)
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
	txs, err := h.svc.ListTransactions(r.Context())
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

	created, err := h.svc.SetBudget(r.Context(), budget)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, ToBudgetResponse(created))
}

func (h handler) handleListBudgets(w http.ResponseWriter, r *http.Request) {
	budgets, err := h.svc.ListBudgets(r.Context())
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
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		errorResponse(w, http.StatusBadRequest, "Parameters 'from' and 'to' are required (format: YYYY-MM-DD)")
		return
	}

	summary, err := h.svc.GetReportSummary(r.Context(), from, to)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			errorResponse(w, 504, "request timeout or cancelled")
			return
		}
		errorResponse(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (h handler) handleImportBulk(w http.ResponseWriter, r *http.Request) {
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
	result, err := h.svc.ImportTransactionsBulk(r.Context(), txs, workers)
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

// userIDMiddleware кладёт userId в context, чтобы Ledger мог делать user-scoped cache keys.
// Источник: заголовок `X-User-ID` (или `X-User-Id`).
func userIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = r.Header.Get("X-User-Id")
		}
		if userID == "" {
			errorResponse(w, http.StatusUnauthorized, "missing X-User-ID header")
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "userId", userID))
		next.ServeHTTP(w, r)
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

