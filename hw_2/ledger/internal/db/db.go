package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config описывает настройки подключения к базе данных.
type Config struct {
	DSN string
}

// New создаёт новое подключение к базе данных, выполняет Ping и настраивает пул соединений.
func New(ctx context.Context, cfg Config) (*sql.DB, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = getDSNFromEnv()
	}

	instance, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть соединение с БД: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := instance.PingContext(pingCtx); err != nil {
		instance.Close()
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}

	instance.SetMaxOpenConns(10)
	instance.SetMaxIdleConns(5)
	instance.SetConnMaxLifetime(5 * time.Minute)

	return instance, nil
}

// Close аккуратно закрывает соединение с базой данных.
func Close(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

// getDSNFromEnv формирует строку подключения к базе данных из переменных окружения.
func getDSNFromEnv() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "5432")
	user := getEnvOrDefault("DB_USER", "postgres")
	password := getEnvOrDefault("DB_PASS", "postgres")
	dbname := getEnvOrDefault("DB_NAME", "cashapp")
	sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

