package ledger

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"ledger/internal/cache"
	"ledger/internal/db"
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

// Удалены глобальные переменные - теперь используем БД

// Init инициализирует подключения к БД и Redis
// Должна быть вызвана перед использованием функций ledger
func Init() error {
	// Инициализация подключения к базе данных
	if err := db.InitDB(); err != nil {
		return fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	// Инициализация подключения к Redis
	if err := cache.InitCache(); err != nil {
		db.Close()
		return fmt.Errorf("ошибка подключения к Redis: %w", err)
	}

	return nil
}

// Shutdown закрывает подключения к БД и Redis
func Shutdown() error {
	var errs []error
	if err := cache.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := db.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("ошибки при закрытии подключений: %v", errs)
	}
	return nil
}

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

// AddTransaction добавляет транзакцию в БД после валидации и проверки лимита
func AddTransaction(tx Transaction) error {
	// Валидация перед добавлением
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("Невалидная транзакция: %w", err)
	}

	// Проверяем наличие бюджета для категории
	var limitAmount float64
	limitQuery := `SELECT limit_amount FROM budgets WHERE category = $1`
	err := db.DB.QueryRow(limitQuery, tx.Category).Scan(&limitAmount)
	
	if err == nil {
		// Бюджет найден - проверяем лимит
		// Считаем уже потраченную сумму
		var spent float64
		spentQuery := `SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE category = $1`
		db.DB.QueryRow(spentQuery, tx.Category).Scan(&spent)

		// Проверяем, не превысит ли новая транзакция лимит
		if spent+tx.Amount > limitAmount {
			return errors.New("budget exceeded")
		}
	} else if err != sql.ErrNoRows {
		// Ошибка при запросе к БД (не "no rows")
		return fmt.Errorf("ошибка при проверке бюджета: %w", err)
	}
	// Если бюджета нет (sql.ErrNoRows) - разрешаем добавление без лимита

	// Вставляем транзакцию в БД
	insertQuery := `INSERT INTO expenses(amount, category, description, date) 
					VALUES($1, $2, $3, $4) RETURNING id`
	err = db.DB.QueryRow(insertQuery, tx.Amount, tx.Category, tx.Description, tx.Date).Scan(&tx.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении транзакции в БД: %w", err)
	}

	// Инвалидируем кеш списка бюджетов (так как Remaining изменился)
	ctx := context.Background()
	cache.Client.Del(ctx, "budgets:all")

	return nil
}

// ListTransactions возвращает список всех транзакций из БД
func ListTransactions() ([]Transaction, error) {
	query := `SELECT id, amount, category, description, date 
			  FROM expenses 
			  ORDER BY date DESC, id DESC`
	
	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении транзакций из БД: %w", err)
	}
	defer rows.Close()

	var result []Transaction
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Category, &tx.Description, &tx.Date); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании транзакции: %w", err)
		}
		result = append(result, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по транзакциям: %w", err)
	}

	return result, nil
}

// Budgets возвращает список всех бюджетов из БД
func Budgets() ([]Budget, error) {
	ctx := context.Background()
	
	// Проверяем кеш
	cached, err := cache.Client.Get(ctx, "budgets:all").Result()
	if err == nil {
		var budgets []Budget
		if err := json.Unmarshal([]byte(cached), &budgets); err == nil {
			return budgets, nil
		}
	}

	// Читаем из БД
	query := `SELECT category, limit_amount FROM budgets ORDER BY category`
	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении бюджетов из БД: %w", err)
	}
	defer rows.Close()

	var result []Budget
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.Category, &b.Limit); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании бюджета: %w", err)
		}
		// Вычисляем Remaining (limit - потраченная сумма)
		var spent float64
		spentQuery := `SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE category = $1`
		db.DB.QueryRow(spentQuery, b.Category).Scan(&spent)
		b.Remaining = b.Limit - spent
		b.Period = getCurrentPeriod()
		result = append(result, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по бюджетам: %w", err)
	}

	// Сохраняем в кеш на 30 секунд
	if data, err := json.Marshal(result); err == nil {
		cache.Client.Set(ctx, "budgets:all", data, 30*time.Second)
	}

	return result, nil
}

// SetBudget добавляет или обновляет бюджет для категории после валидации
func SetBudget(b Budget) error {
	if err := b.Validate(); err != nil {
		return fmt.Errorf("Невалидный бюджет: %w", err)
	}

	// UPSERT бюджета в БД
	query := `INSERT INTO budgets(category, limit_amount) 
			  VALUES($1, $2) 
			  ON CONFLICT(category) DO UPDATE SET limit_amount = EXCLUDED.limit_amount`
	
	_, err := db.DB.Exec(query, b.Category, b.Limit)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении бюджета в БД: %w", err)
	}

	// Инвалидируем кеш списка бюджетов
	ctx := context.Background()
	cache.Client.Del(ctx, "budgets:all")

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

// ReportSummary представляет сводку по категориям за период
type ReportSummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

// GetReportSummary возвращает сводку расходов по категориям за период с кешированием
func GetReportSummary(ctx context.Context, from, to string) ([]ReportSummary, error) {
	// Формируем ключ кеша
	cacheKey := fmt.Sprintf("report:summary:%s:%s", from, to)

	// Проверяем кеш
	cached, err := cache.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		var summary []ReportSummary
		if err := json.Unmarshal([]byte(cached), &summary); err == nil {
			return summary, nil
		}
	}

	// Вычисляем сводку из БД
	query := `SELECT category, COALESCE(SUM(amount), 0) as total 
			  FROM expenses 
			  WHERE date >= $1 AND date <= $2 
			  GROUP BY category 
			  ORDER BY category`
	
	rows, err := db.DB.Query(query, from, to)
	if err != nil {
		return nil, fmt.Errorf("ошибка при вычислении сводки: %w", err)
	}
	defer rows.Close()

	var result []ReportSummary
	for rows.Next() {
		var rs ReportSummary
		if err := rows.Scan(&rs.Category, &rs.Total); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании сводки: %w", err)
		}
		result = append(result, rs)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по сводке: %w", err)
	}

	// Сохраняем в кеш на 30 секунд
	if data, err := json.Marshal(result); err == nil {
		cache.Client.Set(ctx, cacheKey, data, 30*time.Second)
	}

	return result, nil
}

// Reset очищает внутренние коллекции транзакций и бюджетов (используется в тестах)
// Теперь очищает БД и кеш
func Reset() {
	// Очищаем БД
	db.DB.Exec("DELETE FROM expenses")
	db.DB.Exec("DELETE FROM budgets")
	
	// Очищаем кеш
	ctx := context.Background()
	cache.Client.FlushDB(ctx)
}

// Демонстрационная функция main (для тестирования)
// Для запуска сервиса используйте main.go
func demoMain() {
	fmt.Println("Сервис учёта запущен (демо режим)")

	// Инициализация подключения к базе данных
	if err := db.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка подключения к базе данных: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Успешно подключено к базе данных")

	// Инициализация подключения к Redis
	if err := cache.InitCache(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка подключения к Redis: %v\n", err)
		os.Exit(1)
	}
	defer cache.Close()

	fmt.Println("Успешно подключено к Redis")

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
			fmt.Println("Бюджеты успешно загружены из файла budgets.json")
			fmt.Println()
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
	txs, err := ListTransactions()
	if err != nil {
		fmt.Printf("Ошибка при получении транзакций: %v\n", err)
	} else {
		for _, tx := range txs {
			fmt.Printf("ID: %d | Amount: %.2f | Category: %s | Description: %s | Date: %s\n",
				tx.ID, tx.Amount, tx.Category, tx.Description, tx.Date)
		}
	}

	// Выводим текущие бюджеты
	fmt.Println("\nТекущие бюджеты")
	budgets, err := Budgets()
	if err != nil {
		fmt.Printf("Ошибка при получении бюджетов: %v\n", err)
	} else {
		for _, budget := range budgets {
			fmt.Printf("Category: %s | Initial Limit: %.2f | Remaining: %.2f | Period: %s\n", budget.Category, budget.Limit, budget.Remaining, budget.Period)
		}
	}
}