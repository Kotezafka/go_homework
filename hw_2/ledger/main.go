package main

import (
	"fmt"
	"time"
)

// Transaction представляет финансовую транзакцию
type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string
}

var transactions []Transaction

// AddTransaction добавляет транзакцию в хранилище
func AddTransaction(tx Transaction) error {
	if tx.Amount == 0 {
		return fmt.Errorf("transaction amount cannot be zero")
	}
	tx.ID = len(transactions) + 1
	transactions = append(transactions, tx)
	return nil
}

// ListTransactions возвращает копию списка транзакций
func ListTransactions() []Transaction {
	cpy := make([]Transaction, len(transactions))
	copy(cpy, transactions)
	return cpy
}

func main() {
	fmt.Println("Ledger service started")

	t1 := Transaction{
		Amount:      1500.50,
		Category:    "Food",
		Description: "Groceries at supermarket",
		Date:        time.Now().Format("2006-01-02"),
	}

	t2 := Transaction{
		Amount:      3000.00,
		Category:    "Transport",
		Description: "Taxi ride to airport",
		Date:        time.Now().Format("2006-01-02"),
	}

	t3 := Transaction{
		Amount:      0.00,
		Category:    "Entertainment",
		Description: "Movie ticket",
		Date:        time.Now().Format("2006-01-02"),
	}

	// Добавляем транзакции
	if err := AddTransaction(t1); err != nil {
		fmt.Printf("Error adding transaction 1: %v\n", err)
	} else {
		fmt.Println("Transaction 1 added successfully")
	}

	if err := AddTransaction(t2); err != nil {
		fmt.Printf("Error adding transaction 2: %v\n", err)
	} else {
		fmt.Println("Transaction 2 added successfully")
	}

	if err := AddTransaction(t3); err != nil {
		fmt.Printf("Error adding transaction 3: %v\n", err)
	} else {
		fmt.Println("Transaction 3 added successfully")
	}

	fmt.Println("\n--- All Transactions ---")
	for _, tx := range ListTransactions() {
		fmt.Printf("ID: %d | Amount: %.2f | Category: %s | Description: %s | Date: %s\n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date)
	}
}