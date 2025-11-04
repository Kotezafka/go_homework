package ledger

import (
	"strings"
	"testing"
)

// TestTransaction_Validate тестирует валидацию транзакций
func TestTransaction_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		transaction Transaction
		wantError bool
	}{
		{
			name: "валидная транзакция",
			transaction: Transaction{
				Amount:   1000.50,
				Category: "еда",
				Date:     "2025-01-15",
			},
			wantError: false,
		},
		{
			name: "нулевая сумма",
			transaction: Transaction{
				Amount:   0,
				Category: "еда",
				Date:     "2025-01-15",
			},
			wantError: true,
		},
		{
			name: "отрицательная сумма",
			transaction: Transaction{
				Amount:   -100.0,
				Category: "еда",
				Date:     "2025-01-15",
			},
			wantError: true,
		},
		{
			name: "пустая категория",
			transaction: Transaction{
				Amount:   500.0,
				Category: "",
				Date:     "2025-01-15",
			},
			wantError: true,
		},
		{
			name: "положительная сумма с валидной категорией",
			transaction: Transaction{
				Amount:   2500.75,
				Category: "транспорт",
				Date:     "2025-01-15",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.transaction.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("Transaction.Validate() ожидалась ошибка, но получен nil")
				}
			} else {
				if err != nil {
					t.Errorf("Transaction.Validate() неожиданная ошибка: %v", err)
				}
			}
		})
	}
}

// TestBudget_Validate тестирует валидацию бюджетов
func TestBudget_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		budget    Budget
		wantError bool
	}{
		{
			name: "положительный лимит",
			budget: Budget{
				Category: "еда",
				Limit:    5000.0,
				Period:   "2025-01",
			},
			wantError: false,
		},
		{
			name: "нулевой лимит",
			budget: Budget{
				Category: "еда",
				Limit:    0,
				Period:   "2025-01",
			},
			wantError: true,
		},
		{
			name: "отрицательный лимит",
			budget: Budget{
				Category: "еда",
				Limit:    -100.0,
				Period:   "2025-01",
			},
			wantError: true,
		},
		{
			name: "пустая категория",
			budget: Budget{
				Category: "",
				Limit:    5000.0,
				Period:   "2025-01",
			},
			wantError: true,
		},
		{
			name: "валидный бюджет с большим лимитом",
			budget: Budget{
				Category: "развлечения",
				Limit:    10000.50,
				Period:   "2025-01",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.budget.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("Budget.Validate() ожидалась ошибка, но получен nil")
				}
			} else {
				if err != nil {
					t.Errorf("Budget.Validate() неожиданная ошибка: %v", err)
				}
			}
		})
	}
}

// TestBudgetLimitEnforcement тестирует проверку лимитов бюджета при добавлении транзакций
func TestBudgetLimitEnforcement(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	t.Run("транзакция в пределах лимита", func(t *testing.T) {
		Reset()
		
		// Устанавливаем бюджет
		budget := Budget{Category: "еда", Limit: 5000.0, Period: "2025-01"}
		if err := SetBudget(budget); err != nil {
			t.Fatalf("Не удалось установить бюджет: %v", err)
		}

		// Добавляем транзакцию в пределах лимита
		tx := Transaction{
			Amount:   3000.0,
			Category: "еда",
			Date:     "2025-01-15",
		}

		err := AddTransaction(tx)
		if err != nil {
			t.Errorf("AddTransaction() неожиданная ошибка: %v", err)
		}

		// Проверяем, что транзакция добавлена
		transactions := ListTransactions()
		if len(transactions) != 1 {
			t.Errorf("Ожидалась 1 транзакция, получено: %d", len(transactions))
		}
	})

	t.Run("транзакция превышает лимит", func(t *testing.T) {
		Reset()
		
		// Устанавливаем бюджет
		budget := Budget{Category: "еда", Limit: 5000.0, Period: "2025-01"}
		if err := SetBudget(budget); err != nil {
			t.Fatalf("Не удалось установить бюджет: %v", err)
		}

		// Добавляем первую транзакцию
		tx1 := Transaction{
			Amount:   3000.0,
			Category: "еда",
			Date:     "2025-01-15",
		}
		if err := AddTransaction(tx1); err != nil {
			t.Fatalf("Не удалось добавить первую транзакцию: %v", err)
		}

		// Пытаемся добавить транзакцию, которая превысит лимит
		initialCount := len(ListTransactions())
		tx2 := Transaction{
			Amount:   2500.0, // 3000 + 2500 = 5500 > 5000
			Category: "еда",
			Date:     "2025-01-16",
		}

		err := AddTransaction(tx2)
		if err == nil {
			t.Error("AddTransaction() ожидалась ошибка 'budget exceeded', но получен nil")
		}

		// Проверяем, что транзакция не была добавлена
		transactions := ListTransactions()
		if len(transactions) != initialCount {
			t.Errorf("Количество транзакций изменилось после ошибки: было %d, стало %d", initialCount, len(transactions))
		}
	})

	t.Run("транзакция точно на лимите", func(t *testing.T) {
		Reset()
		
		// Устанавливаем бюджет
		budget := Budget{Category: "еда", Limit: 5000.0, Period: "2025-01"}
		if err := SetBudget(budget); err != nil {
			t.Fatalf("Не удалось установить бюджет: %v", err)
		}

		// Добавляем транзакцию точно на лимит
		tx := Transaction{
			Amount:   5000.0,
			Category: "еда",
			Date:     "2025-01-15",
		}

		err := AddTransaction(tx)
		if err != nil {
			t.Errorf("AddTransaction() неожиданная ошибка при транзакции на лимит: %v", err)
		}

		// Проверяем, что транзакция добавлена
		transactions := ListTransactions()
		if len(transactions) != 1 {
			t.Errorf("Ожидалась 1 транзакция, получено: %d", len(transactions))
		}
	})
}

// TestAddTransaction_WithoutBudget тестирует добавление транзакции без установленного бюджета
func TestAddTransaction_WithoutBudget(t *testing.T) {
	t.Parallel()
	Reset()
	t.Cleanup(func() { Reset() })

	tx := Transaction{
		Amount:   1000.0,
		Category: "категория_без_бюджета",
		Date:     "2025-01-15",
	}

	err := AddTransaction(tx)
	if err != nil {
		t.Errorf("AddTransaction() неожиданная ошибка для транзакции без бюджета: %v", err)
	}

	transactions := ListTransactions()
	if len(transactions) != 1 {
		t.Errorf("Ожидалась 1 транзакция, получено: %d", len(transactions))
	}

	if transactions[0].ID != 1 {
		t.Errorf("Ожидался ID=1, получен: %d", transactions[0].ID)
	}
}

// TestListTransactions тестирует получение списка транзакций
func TestListTransactions(t *testing.T) {
	t.Parallel()
	Reset()
	t.Cleanup(func() { Reset() })

	// Добавляем несколько транзакций
	tx1 := Transaction{Amount: 100.0, Category: "еда", Date: "2025-01-15"}
	tx2 := Transaction{Amount: 200.0, Category: "транспорт", Date: "2025-01-16"}
	tx3 := Transaction{Amount: 300.0, Category: "еда", Date: "2025-01-17"}

	AddTransaction(tx1)
	AddTransaction(tx2)
	AddTransaction(tx3)

	transactions := ListTransactions()
	if len(transactions) != 3 {
		t.Errorf("Ожидалось 3 транзакции, получено: %d", len(transactions))
	}

	// Проверяем, что это копия, а не ссылка на оригинал
	if len(transactions) != len(ListTransactions()) {
		t.Error("ListTransactions() должен возвращать копию")
	}
}

// TestBudgets тестирует получение списка бюджетов
func TestBudgets(t *testing.T) {
	t.Parallel()
	Reset()
	t.Cleanup(func() { Reset() })

	// Сначала проверяем, что список пуст
	budgets := Budgets()
	if len(budgets) != 0 {
		t.Errorf("Ожидался пустой список бюджетов, получено: %d", len(budgets))
	}

	// Устанавливаем несколько бюджетов
	budget1 := Budget{Category: "еда", Limit: 5000.0, Period: "2025-01"}
	budget2 := Budget{Category: "транспорт", Limit: 2000.0, Period: "2025-01"}
	budget3 := Budget{Category: "развлечения", Limit: 1000.0, Period: "2025-01"}

	SetBudget(budget1)
	SetBudget(budget2)
	SetBudget(budget3)

	budgets = Budgets()
	if len(budgets) != 3 {
		t.Errorf("Ожидалось 3 бюджета, получено: %d", len(budgets))
	}

	// Проверяем, что все бюджеты присутствуют
	categories := make(map[string]bool)
	for _, b := range budgets {
		categories[b.Category] = true
	}

	if !categories["еда"] || !categories["транспорт"] || !categories["развлечения"] {
		t.Error("Не все бюджеты присутствуют в списке")
	}
}

// TestSetBudget тестирует установку и обновление бюджета
func TestSetBudget(t *testing.T) {
	t.Parallel()
	Reset()
	t.Cleanup(func() { Reset() })

	t.Run("установка нового бюджета", func(t *testing.T) {
		Reset()

		budget := Budget{Category: "еда", Limit: 5000.0, Period: "2025-01"}
		err := SetBudget(budget)
		if err != nil {
			t.Errorf("SetBudget() неожиданная ошибка: %v", err)
		}

		budgets := Budgets()
		if len(budgets) != 1 {
			t.Errorf("Ожидался 1 бюджет, получено: %d", len(budgets))
		}

		if budgets[0].Remaining != 5000.0 {
			t.Errorf("Ожидался Remaining=5000.0, получен: %.2f", budgets[0].Remaining)
		}
	})

	t.Run("обновление существующего бюджета", func(t *testing.T) {
		Reset()

		// Устанавливаем первый бюджет
		budget1 := Budget{Category: "еда", Limit: 5000.0, Period: "2025-01"}
		SetBudget(budget1)

		// Добавляем транзакцию, чтобы изменить Remaining
		tx := Transaction{Amount: 1000.0, Category: "еда", Date: "2025-01-15"}
		AddTransaction(tx)

		// Обновляем бюджет
		budget2 := Budget{Category: "еда", Limit: 8000.0, Period: "2025-01"}
		err := SetBudget(budget2)
		if err != nil {
			t.Errorf("SetBudget() неожиданная ошибка при обновлении: %v", err)
		}

		budgets := Budgets()
		if len(budgets) != 1 {
			t.Errorf("Ожидался 1 бюджет, получено: %d", len(budgets))
		}

		// Remaining должен быть сброшен на новый Limit
		if budgets[0].Remaining != 8000.0 {
			t.Errorf("Ожидался Remaining=8000.0 после обновления, получен: %.2f", budgets[0].Remaining)
		}
	})

	t.Run("установка невалидного бюджета", func(t *testing.T) {
		Reset()

		budget := Budget{Category: "", Limit: 0, Period: "2025-01"}
		err := SetBudget(budget)
		if err == nil {
			t.Error("SetBudget() ожидалась ошибка для невалидного бюджета, но получен nil")
		}
	})
}

// TestLoadBudgets тестирует загрузку бюджетов из JSON
func TestLoadBudgets(t *testing.T) {
	t.Parallel()
	Reset()
	t.Cleanup(func() { Reset() })

	t.Run("валидный JSON", func(t *testing.T) {
		Reset()

		jsonData := `[
			{"category": "еда", "limit": 5000.0, "remaining": 5000.0, "period": "2025-01"},
			{"category": "транспорт", "limit": 2000.0, "remaining": 2000.0, "period": "2025-01"}
		]`

		err := LoadBudgets(strings.NewReader(jsonData))
		if err != nil {
			t.Errorf("LoadBudgets() неожиданная ошибка: %v", err)
		}

		budgets := Budgets()
		if len(budgets) != 2 {
			t.Errorf("Ожидалось 2 бюджета, получено: %d", len(budgets))
		}
	})

	t.Run("невалидный JSON", func(t *testing.T) {
		Reset()

		jsonData := `invalid json`

		err := LoadBudgets(strings.NewReader(jsonData))
		if err == nil {
			t.Error("LoadBudgets() ожидалась ошибка для невалидного JSON, но получен nil")
		}
	})

	t.Run("JSON с невалидным бюджетом", func(t *testing.T) {
		Reset()

		jsonData := `[
			{"category": "еда", "limit": 5000.0, "remaining": 5000.0, "period": "2025-01"},
			{"category": "", "limit": 0, "remaining": 0, "period": "2025-01"}
		]`

		err := LoadBudgets(strings.NewReader(jsonData))
		if err == nil {
			t.Error("LoadBudgets() ожидалась ошибка для невалидного бюджета, но получен nil")
		}
	})
}

// TestCheckValid тестирует функцию CheckValid с интерфейсом Validatable
func TestCheckValid(t *testing.T) {
	t.Parallel()
	Reset()
	t.Cleanup(func() { Reset() })

	t.Run("валидная транзакция", func(t *testing.T) {
		tx := Transaction{Amount: 1000.0, Category: "еда", Date: "2025-01-15"}
		err := CheckValid(tx)
		if err != nil {
			t.Errorf("CheckValid() неожиданная ошибка для валидной транзакции: %v", err)
		}
	})

	t.Run("невалидная транзакция", func(t *testing.T) {
		tx := Transaction{Amount: 0, Category: "еда", Date: "2025-01-15"}
		err := CheckValid(tx)
		if err == nil {
			t.Error("CheckValid() ожидалась ошибка для невалидной транзакции, но получен nil")
		}
	})

	t.Run("валидный бюджет", func(t *testing.T) {
		budget := Budget{Category: "еда", Limit: 5000.0, Period: "2025-01"}
		err := CheckValid(budget)
		if err != nil {
			t.Errorf("CheckValid() неожиданная ошибка для валидного бюджета: %v", err)
		}
	})

	t.Run("невалидный бюджет", func(t *testing.T) {
		budget := Budget{Category: "", Limit: 0, Period: "2025-01"}
		err := CheckValid(budget)
		if err == nil {
			t.Error("CheckValid() ожидалась ошибка для невалидного бюджета, но получен nil")
		}
	})
}

// TestAddTransaction_ValidationError тестирует обработку ошибок валидации
func TestAddTransaction_ValidationError(t *testing.T) {
	t.Parallel()
	Reset()
	t.Cleanup(func() { Reset() })

	t.Run("транзакция с нулевой суммой", func(t *testing.T) {
		Reset()

		tx := Transaction{Amount: 0, Category: "еда", Date: "2025-01-15"}
		err := AddTransaction(tx)
		if err == nil {
			t.Error("AddTransaction() ожидалась ошибка для транзакции с нулевой суммой, но получен nil")
		}

		transactions := ListTransactions()
		if len(transactions) != 0 {
			t.Errorf("Транзакция не должна быть добавлена, но получено: %d транзакций", len(transactions))
		}
	})

	t.Run("транзакция с пустой категорией", func(t *testing.T) {
		Reset()

		tx := Transaction{Amount: 1000.0, Category: "", Date: "2025-01-15"}
		err := AddTransaction(tx)
		if err == nil {
			t.Error("AddTransaction() ожидалась ошибка для транзакции с пустой категорией, но получен nil")
		}

		transactions := ListTransactions()
		if len(transactions) != 0 {
			t.Errorf("Транзакция не должна быть добавлена, но получено: %d транзакций", len(transactions))
		}
	})
}

// TestAddTransaction_MultipleTransactions тестирует добавление нескольких транзакций
func TestAddTransaction_MultipleTransactions(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	budget := Budget{Category: "еда", Limit: 10000.0, Period: "2025-01"}
	SetBudget(budget)

	// Добавляем несколько транзакций
	for i := 0; i < 5; i++ {
		tx := Transaction{
			Amount:   100.0 * float64(i+1),
			Category: "еда",
			Date:     "2025-01-15",
		}
		err := AddTransaction(tx)
		if err != nil {
			t.Fatalf("Не удалось добавить транзакцию %d: %v", i+1, err)
		}
	}

	transactions := ListTransactions()
	if len(transactions) != 5 {
		t.Errorf("Ожидалось 5 транзакций, получено: %d", len(transactions))
	}

	// Проверяем, что ID установлены правильно
	for i, tx := range transactions {
		expectedID := i + 1
		if tx.ID != expectedID {
			t.Errorf("Ожидался ID=%d для транзакции %d, получен: %d", expectedID, i+1, tx.ID)
		}
	}
}

// TestAddTransaction_BudgetRemainingUpdate тестирует обновление Remaining при добавлении транзакций
func TestAddTransaction_BudgetRemainingUpdate(t *testing.T) {
	Reset()
	t.Cleanup(func() { Reset() })

	budget := Budget{Category: "еда", Limit: 5000.0, Period: "2025-01"}
	SetBudget(budget)

	// Добавляем транзакцию
	tx1 := Transaction{Amount: 2000.0, Category: "еда", Date: "2025-01-15"}
	AddTransaction(tx1)

	budgets := Budgets()
	if len(budgets) != 1 {
		t.Fatalf("Ожидался 1 бюджет, получено: %d", len(budgets))
	}

	if budgets[0].Remaining != 3000.0 {
		t.Errorf("Ожидался Remaining=3000.0, получен: %.2f", budgets[0].Remaining)
	}

	// Добавляем ещё одну транзакцию
	tx2 := Transaction{Amount: 1500.0, Category: "еда", Date: "2025-01-16"}
	AddTransaction(tx2)

	budgets = Budgets()
	if budgets[0].Remaining != 1500.0 {
		t.Errorf("Ожидался Remaining=1500.0 после второй транзакции, получен: %.2f", budgets[0].Remaining)
	}
}

