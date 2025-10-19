package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Validatable определяет контракт для проверки валидности данных структуры
type Validatable interface {
	Validate() error
}

// Transaction представляет финансовую транзакцию
type Transaction struct {
	ID          int
	Amount      float64
	Category    string
	Description string
	Date        string
}

// Budget представляет бюджет на категорию
type Budget struct {
	Category   string  `json:"category"`
	Limit      float64 `json:"limit"`
	Remaining  float64 `json:"remaining"`
	Period     string  `json:"period"`
}

var transactions []Transaction
var budgets = make(map[string]Budget)

// Validate реализует интерфейс Validatable для Transaction
func (tx Transaction) Validate() error {
	if tx.Amount <= 0 {
		return fmt.Errorf("Сумма транзакции должна быть положительной, получено: %.2f", tx.Amount)
	}
	if tx.Category == "" {
		return fmt.Errorf("Категория транзакции не может быть пустой")
	}
	return nil
}

// Validate реализует интерфейс Validatable для Budget
func (b Budget) Validate() error {
	if b.Limit <= 0 {
		return fmt.Errorf("Лимит бюджета должен быть положительным числом, получено: %.2f", b.Limit)
	}
	if b.Category == "" {
		return fmt.Errorf("Категория бюджета не может быть пустой")
	}
	return nil
}

// AddTransaction добавляет транзакцию в хранилище после валидации
func AddTransaction(tx Transaction) error {
	// Валидация перед добавлением
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("Невалидная транзакция: %w", err)
	}

	// Проверяем наличие бюджета для категории
	if budget, exists := budgets[tx.Category]; exists {
		if tx.Amount > budget.Remaining {
			currentSum := budget.Limit - budget.Remaining
			return fmt.Errorf("Превышен бюджет для категории '%s' (период %s): limit %.2f, current sum %.2f, adding %.2f", tx.Category, budget.Period, budget.Limit, currentSum, tx.Amount)
		}
		budget.Remaining -= tx.Amount
		budgets[tx.Category] = budget
	}

	// Добавляем транзакцию
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

// SetBudget добавляет или обновляет бюджет для категории после валидации
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return fmt.Errorf("Невалидный бюджет: %w", err)
	}

	b.Remaining = b.Limit
	budgets[b.Category] = b
	return nil
}

// LoadBudgets читает бюджеты из JSON-потока
func LoadBudgets(r io.Reader) error {
	decoder := json.NewDecoder(r)
	var budgetList []Budget

	err := decoder.Decode(&budgetList)
	if err != nil {
		return fmt.Errorf("Failed to decode budgets: %w", err)
	}

	for _, b := range budgetList {
		if err := SetBudget(b); err != nil {
			return fmt.Errorf("Ошибка при установке бюджета '%s': %w", b.Category, err)
		}
	}

	return nil
}

func getCurrentPeriod() string {
	return time.Now().Format("2006-01")
}

// CheckValid — демонстрационная функция, использующая полиморфизм через интерфейс Validatable
func CheckValid(v Validatable) error {
	return v.Validate()
}

func main() {
	fmt.Println("Сервис учёта запущен\n")

	currentPeriod := getCurrentPeriod()

	// Инициализация бюджетов через SetBudget с указанием периода
	// Убедимся, что они проходят валидацию
	if err := SetBudget(Budget{Category: "Food", Limit: 5000.0, Period: currentPeriod}); err != nil {
		fmt.Printf("Ошибка при установке бюджета 'Food': %v\n", err)
	}
	if err := SetBudget(Budget{Category: "Transport", Limit: 2000.0, Period: currentPeriod}); err != nil {
		fmt.Printf("Ошибка при установке бюджета 'Transport': %v\n", err)
	}
	if err := SetBudget(Budget{Category: "Entertainment", Limit: 1000.0, Period: currentPeriod}); err != nil {
		fmt.Printf("Ошибка при установке бюджета 'Entertainment': %v\n", err)
	}

	// Загрузка бюджетов из файла
	file, err := os.Open("budgets.json")
	if err != nil {
		fmt.Printf("Warning: файл budgets.json не найден или недоступен: %v\n\n", err)
	} else {
		defer file.Close()
		reader := io.Reader(file)
		err = LoadBudgets(reader)
		if err != nil {
			fmt.Printf("Error: ошибка при загрузке бюджетов из файла: %v\n\n", err)
		} else {
			fmt.Println("Бюджеты успешно загружены из файла budgets.json\n\n")
		}
	}

	t1 := Transaction{
		Amount:      1500.50,
		Category:    "Food",
		Description: "Продукты в супермаркете",
		Date:        time.Now().Format("2006-01-02"),
	}

	t2 := Transaction{
		Amount:      3000.00,
		Category:    "Transport",
		Description: "Поездка на такси в аэропорт",
		Date:        time.Now().Format("2006-01-02"),
	}

	t3 := Transaction{
		Amount:      0.00,
		Category:    "Entertainment",
		Description: "Билет в кино",
		Date:        time.Now().Format("2006-01-02"),
	}

	t4 := Transaction{
		Amount:      4000.00,
		Category:    "Food",
		Description: "Ужин в ресторане",
		Date:        time.Now().Format("2006-01-02"),
	}

	t5 := Transaction{
		Amount:      500.00,
		Category:    "Food",
		Description: "Кофе и перекусы",
		Date:        time.Now().Format("2006-01-02"),
	}

	// Демонстрация полиморфизма через интерфейс Validatable
	fmt.Println("\nТестирование интерфейса Validatable:")
	testValidatables := []Validatable{
		t1,
		Budget{Category: "Test", Limit: 100.0},
		t3,
		Budget{Category: "", Limit: 0},
		t2,
		t5,
	}

	for i, v := range testValidatables {
		fmt.Printf("Тест %d: ", i+1)
		err := CheckValid(v)
		if err != nil {
			fmt.Printf("❌ Ошибка валидации: %v\n", err)
		} else {
			fmt.Printf("✅ Валидно\n")
		}
	}

	// Добавляем транзакции и обрабатываем ошибки
	testCases := []struct {
		name string
		tx   Transaction
	}{
		{"Корректная транзакция (еда)", t1},
		{"Транзакция по транспорту (превышает лимит)", t2},
		{"Транзакция с нулевой суммой", t3},
		{"Транзакция, превышающая бюджет (еда)", t4},
		{"Дополнительная транзакция по еде", t5},
	}

	for _, tc := range testCases {
		fmt.Printf("\nTesting: %s\n", tc.name)
		err := AddTransaction(tc.tx)
		if err != nil {
			fmt.Printf("❌ Error: ошибка при добавлении транзакции: %v\n", err)
		} else {
			fmt.Printf("✅ Транзакция успешно добавлена: ID=%d, Amount=%.2f, Category=%s\n", tc.tx.ID, tc.tx.Amount, tc.tx.Category)
		}
	}

	// Выводим список всех успешных транзакций
	fmt.Println("\n\nВсе сохранённые транзакции")
	for _, tx := range ListTransactions() {
		fmt.Printf("ID: %d | Amount: %.2f | Category: %s | Description: %s | Date: %s\n",
			tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date)
	}

	// Выводим текущие бюджеты
	fmt.Println("\nТекущие бюджеты")
	for cat, budget := range budgets {
		fmt.Printf("Category: %s | Initial Limit: %.2f | Remaining: %.2f | Period: %s\n", cat, budget.Limit, budget.Remaining, budget.Period)
	}
}