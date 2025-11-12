package main

import (
	"fmt"
	"os"
	"ledger"
)

func main() {
	fmt.Println("Сервис учёта запущен")

	// Инициализация подключений к БД и Redis
	if err := ledger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации: %v\n", err)
		os.Exit(1)
	}
	defer ledger.Shutdown()

	fmt.Println("Успешно подключено к базе данных")
	fmt.Println("Успешно подключено к Redis")
	fmt.Println("Ledger сервис готов к работе. Ожидание запросов от Gateway...")

	// Держим сервис запущенным
	select {} // Блокируем выполнение, чтобы сервис не завершился
}

