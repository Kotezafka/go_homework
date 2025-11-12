# Многоэтапная сборка для оптимизации размера образа
FROM golang:1.23-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git

# Установка goose для миграций
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Рабочая директория
WORKDIR /app

# Копирование go.work и go.mod файлов для кеширования зависимостей
COPY go.work ./
COPY ledger/go.mod ledger/go.sum ./ledger/
COPY gateway/go.mod ./gateway/

# Загрузка зависимостей
WORKDIR /app/ledger
RUN go mod download

WORKDIR /app/gateway
RUN go mod download

# Копирование всего кода
WORKDIR /app
COPY ledger/ ./ledger/
COPY gateway/ ./gateway/

# Сборка gateway приложения
WORKDIR /app/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gateway main.go

# Финальный образ
FROM alpine:latest

# Установка необходимых инструментов
RUN apk --no-cache add ca-certificates postgresql-client redis netcat-openbsd curl wget

# Копирование goose из builder
COPY --from=builder /go/bin/goose /usr/local/bin/goose

# Создание рабочей директории
WORKDIR /app

# Копирование скомпилированного приложения
COPY --from=builder /app/gateway/gateway .

# Копирование миграций
COPY --from=builder /app/ledger/migrations ./migrations

# Копирование скриптов
COPY docker-entrypoint.sh /docker-entrypoint.sh
COPY docker-check.sh /docker-check.sh

# Переменные окружения по умолчанию
ENV DATABASE_URL="postgres://postgres_user:postgres_password@postgres:5432/cashapp?sslmode=disable"
ENV REDIS_ADDR="redis:6379"

# Установка прав на скрипты
RUN chmod +x /docker-entrypoint.sh /docker-check.sh

EXPOSE 8080

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./gateway"]

