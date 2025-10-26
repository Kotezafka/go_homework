package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"ledger"
	"gateway/internal/api"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		fmt.Printf("[LOG] %s %s took %v\n", r.Method, r.URL.Path, duration)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request) {

	var req api.CreateTransactionRequest
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

	resp := api.ToTransactionResponse(createdTx)
	writeJSON(w, http.StatusCreated, resp)
}

func handleListTransactions(w http.ResponseWriter, r *http.Request) {

	txs := ledger.ListTransactions()
	var responses []api.TransactionResponse
	for _, tx := range txs {
		responses = append(responses, api.ToTransactionResponse(tx))
	}

	writeJSON(w, http.StatusOK, responses)
}

func handleCreateBudget(w http.ResponseWriter, r *http.Request) {

	var req api.CreateBudgetRequest
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

	resp := api.ToBudgetResponse(budget)
	writeJSON(w, http.StatusCreated, resp)
}

func handleListBudgets(w http.ResponseWriter, r *http.Request) {
	var budgets []api.BudgetResponse
	for _, b := range ledger.Budgets() {
		budgets = append(budgets, api.ToBudgetResponse(b))
	}

	writeJSON(w, http.StatusOK, budgets)
}


func main() {
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

	router.Handle("/api/", http.StripPrefix("/api", loggingMiddleware(apiRouter)))

	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "pong")
	})

	fmt.Println("Gateway service started on :8080")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}