package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB глобальная переменная для хранения подключения к базе данных
var DB *sql.DB

// InitDB инициализирует подключение к базе данных
// Приоритет: DATABASE_URL > отдельные переменные > значения по умолчанию
func InitDB() error {
	dsn := getDSN()

	var err error
	DB, err = sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("не удалось открыть соединение с БД: %w", err)
	}

	// Проверяем подключение
	if err := DB.Ping(); err != nil {
		DB.Close()
		return fmt.Errorf("не удалось подключиться к БД: %w", err)
	}

	// Настройка пула соединений
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	return nil
}

// getDSN формирует строку подключения к базе данных
func getDSN() string {
	// Приоритет 1: DATABASE_URL
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	// Приоритет 2: отдельные переменные окружения
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "5432")
	user := getEnvOrDefault("DB_USER", "postgres")
	password := getEnvOrDefault("DB_PASS", "postgres")
	dbname := getEnvOrDefault("DB_NAME", "cashapp")
	sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

	// Формируем DSN
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)

	return dsn
}

// getEnvOrDefault возвращает значение переменной окружения или значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Close закрывает соединение с базой данных
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

