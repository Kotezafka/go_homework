package main

import (
	"fmt"
	"log"
	"net/http"

	"gateway/internal/api"
)

func main() {
	fmt.Println("Gateway service started on :8080 (mock mode)")

	mockServices := NewMockServices()

	router := api.NewRouterWithMocks(mockServices)

	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}