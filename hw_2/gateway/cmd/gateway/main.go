package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gateway/internal/api"
	gatewayauth "gateway/internal/auth"
	gatewaygrpc "gateway/internal/grpc"
)


var mockServices *MockServices

func main() {
	mockMode := os.Getenv("MOCK_MODE") == "true"

	if mockMode {
		fmt.Println("Gateway service started on :8080 (mock mode)")

		mockServices = NewMockServices()

		router := api.NewRouterWithMocks(mockServices)
		if err := http.ListenAndServe(":8080", router); err != nil {
			log.Fatalf("Server stopped: %v", err)
		}
	} else {
		fmt.Println("Gateway service started on :8080 (production mode with gRPC)")
		ctx := context.Background()
		
		ledgerClient, ledgerCloseFn, err := gatewaygrpc.NewClient(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка подключения к Ledger gRPC: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := ledgerCloseFn(); err != nil {
				fmt.Fprintf(os.Stderr, "Ошибка при закрытии Ledger gRPC соединения: %v\n", err)
			}
		}()

		var authClient *gatewayauth.Client
		var authCloseFn func() error
		maxAuthRetries := 10
		for i := 0; i < maxAuthRetries; i++ {
			client, closeFn, err := gatewayauth.NewClient(ctx)
			if err == nil {
				authClient = client
				authCloseFn = closeFn
				fmt.Printf("Successfully connected to Auth service (attempt %d/%d)\n", i+1, maxAuthRetries)
				break
			}
			if i < maxAuthRetries-1 {
				fmt.Printf("Failed to connect to Auth (attempt %d/%d), retrying...\n", i+1, maxAuthRetries)
				time.Sleep(2 * time.Second)
			} else {
				fmt.Fprintf(os.Stderr, "Ошибка подключения к Auth gRPC после %d попыток: %v\n", maxAuthRetries, err)
				os.Exit(1)
			}
		}
		defer func() {
			if authCloseFn != nil {
				if err := authCloseFn(); err != nil {
					fmt.Fprintf(os.Stderr, "Ошибка при закрытии Auth gRPC соединения: %v\n", err)
				}
			}
		}()

		router := api.NewRouter(ledgerClient, authClient)
		if err := http.ListenAndServe(":8080", router); err != nil {
			log.Fatalf("Server stopped: %v", err)
		}
	}
}

type MockServices struct {
	users map[string]string
	budgets map[string][]api.BudgetResponse
	transactions map[string][]api.TransactionResponse
	jwtTokens map[string]string
}

func NewMockServices() *MockServices {
	return &MockServices{
		users: make(map[string]string),
		budgets: make(map[string][]api.BudgetResponse),
		transactions: make(map[string][]api.TransactionResponse),
		jwtTokens: make(map[string]string),
	}
}

func (m *MockServices) Register(email, password, name string) (string, error) {
	if _, exists := m.users[email]; exists {
		return "", fmt.Errorf("user already exists")
	}
	userID := fmt.Sprintf("user_%d", len(m.users)+1)
	m.users[email] = userID
	m.budgets[userID] = []api.BudgetResponse{}
	m.transactions[userID] = []api.TransactionResponse{}
	return userID, nil
}

func (m *MockServices) Login(email, password string) (string, error) {
	userID, exists := m.users[email]
	if !exists {
		return "", fmt.Errorf("user not found")
	}
	token := fmt.Sprintf("jwt_token_%s", userID)
	m.jwtTokens[token] = userID
	return token, nil
}

func (m *MockServices) ValidateToken(token string) (string, error) {
	userID, exists := m.jwtTokens[token]
	if !exists {
		return "", fmt.Errorf("invalid token")
	}
	return userID, nil
}

func (m *MockServices) SetBudget(userID string, category string, limit float64) error {
	budgets := m.budgets[userID]
	for i, b := range budgets {
		if b.Category == category {
			budgets = append(budgets[:i], budgets[i+1:]...)
			break
		}
	}
	budget := api.BudgetResponse{
		Category:  category,
		Limit:     limit,
		Remaining: limit,
		Period:    "2025-01",
	}
	budgets = append(budgets, budget)
	m.budgets[userID] = budgets
	return nil
}

func (m *MockServices) ListBudgets(userID string) []api.BudgetResponse {
	return m.budgets[userID]
}

func (m *MockServices) AddTransaction(userID string, amount float64, category, description, date string) (api.TransactionResponse, error) {
	budgets := m.budgets[userID]
	var budgetLimit float64
	for _, b := range budgets {
		if b.Category == category {
			budgetLimit = b.Limit
			break
		}
	}

	var spent float64
	for _, t := range m.transactions[userID] {
		if t.Category == category {
			spent += t.Amount
		}
	}

	if spent + amount > budgetLimit && budgetLimit > 0 {
		return api.TransactionResponse{}, fmt.Errorf("budget exceeded")
	}

	transaction := api.TransactionResponse{
		ID:          len(m.transactions[userID]) + 1,
		Amount:      amount,
		Category:    category,
		Description: description,
		Date:        date,
	}

	m.transactions[userID] = append(m.transactions[userID], transaction)
	return transaction, nil
}

func (m *MockServices) ListTransactions(userID string) []api.TransactionResponse {
	return m.transactions[userID]
}

func (m *MockServices) GetReportSummary(userID, from, to string) (map[string]interface{}, error) {
	transactions := m.transactions[userID]
	budgets := m.budgets[userID]

	categoryTotals := make(map[string]float64)
	for _, t := range transactions {
		categoryTotals[t.Category] += t.Amount
	}

	var summaries []map[string]interface{}
	var totalExpenses float64
	var totalBudget float64

	for category, total := range categoryTotals {
		summaries = append(summaries, map[string]interface{}{
			"category": category,
			"total":    total,
		})
		totalExpenses += total
	}

	for _, b := range budgets {
		totalBudget += b.Limit
	}

	budgetUsagePercent := 0.0
	if totalBudget > 0 {
		budgetUsagePercent = (totalExpenses / totalBudget) * 100
	}

	return map[string]interface{}{
		"summaries":            summaries,
		"total_expenses":       totalExpenses,
		"total_budget":         totalBudget,
		"budget_usage_percent": budgetUsagePercent,
	}, nil
}

func (m *MockServices) ImportTransactionsBulk(userID string, transactions []api.CreateTransactionRequest) (map[string]interface{}, error) {
	var accepted, rejected int
	var errors []map[string]interface{}

	for i, req := range transactions {
		_, err := m.AddTransaction(userID, req.Amount, req.Category, req.Description, req.Date)
		if err != nil {
			rejected++
			errors = append(errors, map[string]interface{}{
				"index": i,
				"error": err.Error(),
			})
		} else {
			accepted++
		}
	}

	return map[string]interface{}{
		"accepted": accepted,
		"rejected": rejected,
		"errors":   errors,
	}, nil
}