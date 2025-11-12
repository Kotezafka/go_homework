package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client глобальная переменная для хранения клиента Redis
var Client *redis.Client

// InitCache инициализирует подключение к Redis
func InitCache() error {
	addr := getEnvOrDefault("REDIS_ADDR", "localhost:6379")
	db, _ := strconv.Atoi(getEnvOrDefault("REDIS_DB", "0"))
	password := os.Getenv("REDIS_PASSWORD")

	Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       db,
		Password: password,
	})

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("не удалось подключиться к Redis: %w", err)
	}

	return nil
}

// getEnvOrDefault возвращает значение переменной окружения или значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Close закрывает соединение с Redis
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

