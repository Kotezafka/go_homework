package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"gateway/internal/api"
	"ledger"
)

func main() {
	ctx := context.Background()
	svc, closeFn, err := ledger.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации Ledger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := closeFn(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка при закрытии Ledger: %v\n", err)
		}
	}()

	fmt.Println("Gateway service started on :8080")
	if err := http.ListenAndServe(":8080", api.NewRouter(svc)); err != nil {
		fmt.Fprintf(os.Stderr, "Server stopped: %v\n", err)
	}
}
