#!/bin/bash
set -e

echo "=========================================="
echo "Автоматическая проверка проекта"
echo "=========================================="
echo ""


GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'


check_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
        return 0
    else
        echo -e "${RED}✗${NC} $2"
        return 1
    fi
}

# Очистка предыдущих запусков
echo "1. Очистка предыдущих запусков..."
docker compose down -v > /dev/null 2>&1 || true
check_status $? "Контейнеры остановлены и удалены"

# Запуск сервисов
echo ""
echo "2. Запуск сервисов (db, redis, gateway)..."
docker compose up -d db redis
check_status $? "Сервисы db и redis запущены"

# Ожидание готовности db и redis
echo ""
echo "3. Ожидание готовности сервисов..."
timeout=60
elapsed=0
while [ $elapsed -lt $timeout ]; do
    if docker compose ps db redis | grep -q "healthy"; then
        if docker compose ps db redis | grep -q "healthy" && [ $(docker compose ps db redis | grep -c "healthy") -eq 2 ]; then
            break
        fi
    fi
    sleep 2
    elapsed=$((elapsed + 2))
done

if [ $elapsed -ge $timeout ]; then
    echo -e "${RED}✗${NC} Таймаут ожидания готовности сервисов"
    docker compose ps
    exit 1
fi
check_status 0 "Сервисы готовы"

# Проверка создания базы данных
echo ""
echo "4. Проверка базы данных cashapp..."
sleep 2
DB_EXISTS=$(docker compose exec -T db psql -U postgres_user -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='cashapp'" 2>/dev/null || echo "0")
if [ "$DB_EXISTS" != "1" ]; then
    echo -e "${YELLOW}⚠${NC} База cashapp не найдена, создание..."
    docker compose exec -T db psql -U postgres_user -d postgres -c "CREATE DATABASE cashapp;" > /dev/null 2>&1
fi
check_status 0 "База данных cashapp готова"

# Применение миграций
echo ""
echo "5. Применение миграций..."
docker run --rm --network hw_2_cashapp-network \
    -v "$(pwd)/ledger/migrations:/migrations" \
    -e DATABASE_URL="postgres://postgres_user:postgres_password@db:5432/cashapp?sslmode=disable" \
    golang:1.23-alpine sh -c "go install github.com/pressly/goose/v3/cmd/goose@latest > /dev/null 2>&1 && /go/bin/goose -dir /migrations postgres \"\$DATABASE_URL\" up" > /dev/null 2>&1
check_status $? "Миграции применены"

# Проверка таблиц
TABLES=$(docker compose exec -T db psql -U postgres_user -d cashapp -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('budgets', 'expenses');" 2>/dev/null || echo "0")
if [ "$TABLES" = "2" ]; then
    check_status 0 "Таблицы созданы (budgets, expenses)"
else
    check_status 1 "Таблицы не найдены"
fi

# Сборка и запуск gateway
echo ""
echo "6. Сборка и запуск gateway..."
docker compose build gateway > /dev/null 2>&1
check_status $? "Gateway собран"

docker compose up -d gateway > /dev/null 2>&1
check_status $? "Gateway запущен"

# Ожидание готовности gateway
echo ""
echo "7. Ожидание готовности gateway..."
timeout=30
elapsed=0
while [ $elapsed -lt $timeout ]; do
    if curl -s http://localhost:8080/ping > /dev/null 2>&1; then
        break
    fi
    sleep 1
    elapsed=$((elapsed + 1))
done

if [ $elapsed -ge $timeout ]; then
    echo -e "${RED}✗${NC} Gateway не отвечает на /ping"
    docker compose logs gateway --tail 20
    exit 1
fi
check_status 0 "Gateway готов"

# Тестирование эндпоинтов
echo ""
echo "8. Тестирование API эндпоинтов..."
echo ""

# GET /ping
RESPONSE=$(curl -s http://localhost:8080/ping 2>&1)
if [ "$RESPONSE" = "pong" ]; then
    check_status 0 "GET /ping"
else
    check_status 1 "GET /ping (ожидалось 'pong', получено '$RESPONSE')"
fi

# POST /api/budgets
RESPONSE=$(curl -s -X POST http://localhost:8080/api/budgets \
    -H "Content-Type: application/json" \
    -d '{"category":"тест","limit":1000}' 2>&1)
if echo "$RESPONSE" | grep -q '"category":"тест"'; then
    check_status 0 "POST /api/budgets"
else
    check_status 1 "POST /api/budgets"
fi

# GET /api/budgets
RESPONSE=$(curl -s http://localhost:8080/api/budgets 2>&1)
if echo "$RESPONSE" | grep -q '"category"'; then
    check_status 0 "GET /api/budgets"
else
    check_status 1 "GET /api/budgets"
fi

# POST /api/transactions
RESPONSE=$(curl -s -X POST http://localhost:8080/api/transactions \
    -H "Content-Type: application/json" \
    -d '{"amount":100,"category":"тест","description":"тестовая транзакция","date":"2025-12-03"}' 2>&1)
if echo "$RESPONSE" | grep -q '"id"'; then
    check_status 0 "POST /api/transactions"
else
    check_status 1 "POST /api/transactions"
fi

# GET /api/transactions
RESPONSE=$(curl -s http://localhost:8080/api/transactions 2>&1)
if echo "$RESPONSE" | grep -q '"id"'; then
    check_status 0 "GET /api/transactions"
else
    check_status 1 "GET /api/transactions"
fi

# POST /api/transactions/bulk
RESPONSE=$(curl -s -X POST http://localhost:8080/api/transactions/bulk \
    -H "Content-Type: application/json" \
    -d '[{"amount":50,"category":"тест","description":"bulk1","date":"2025-12-03"},{"amount":75,"category":"тест","description":"bulk2","date":"2025-12-03"}]' 2>&1)
if echo "$RESPONSE" | grep -q '"accepted"'; then
    check_status 0 "POST /api/transactions/bulk"
else
    check_status 1 "POST /api/transactions/bulk"
fi

# Проверка превышения бюджета
RESPONSE=$(curl -s -X POST http://localhost:8080/api/transactions \
    -H "Content-Type: application/json" \
    -d '{"amount":5000,"category":"тест","description":"превышение","date":"2025-12-03"}' 2>&1)
if echo "$RESPONSE" | grep -q '"error":"budget exceeded"'; then
    check_status 0 "Проверка бюджета (превышение лимита)"
else
    check_status 1 "Проверка бюджета"
fi

# Итоговая статистика
echo ""
echo "=========================================="
echo "Итоговая статистика"
echo "=========================================="

BUDGETS_COUNT=$(docker compose exec -T db psql -U postgres_user -d cashapp -tAc "SELECT COUNT(*) FROM budgets;" 2>/dev/null || echo "0")
EXPENSES_COUNT=$(docker compose exec -T db psql -U postgres_user -d cashapp -tAc "SELECT COUNT(*) FROM expenses;" 2>/dev/null || echo "0")

echo "Бюджетов в БД: $BUDGETS_COUNT"
echo "Транзакций в БД: $EXPENSES_COUNT"
echo ""

# Статус контейнеров
echo "Статус контейнеров:"
docker compose ps --format "table {{.Name}}\t{{.Status}}" | grep -E "NAME|cashapp"

echo ""
echo -e "${GREEN}=========================================="
echo "Все проверки завершены!"
echo "==========================================${NC}"

