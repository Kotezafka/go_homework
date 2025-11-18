package main

import (
	"context"
	"fmt"
	"os"

	"ledger"
)

func main() {
	fmt.Println("Сервис учёта запущен")

	ctx := context.Background()
	svc, closeFn, err := ledger.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := closeFn(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка при закрытии ресурсов: %v\n", err)
		}
	}()

	// Демонстрация простого использования сервиса (можно заменить на gRPC позднее).
	if _, err := svc.ListBudgets(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при получении бюджетов: %v\n", err)
	}

	fmt.Println("Ledger сервис готов к работе. Ожидание запросов от Gateway...")
	select {}
}

