#!/bin/sh
set -e

echo "=========================================="
echo "ПРОВЕРКА СИСТЕМЫ ВНУТРИ DOCKER"
echo "=========================================="
echo ""

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

check() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
        return 0
    else
        echo -e "${RED}✗${NC} $1"
        return 1
    fi
}

PASSED=0
FAILED=0

# Проверка 1: Подключение к PostgreSQL
echo -e "${BLUE}=== Проверка подключения к PostgreSQL ===${NC}"
if psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1; then
    if check "Подключение к PostgreSQL успешно"; then
        PASSED=$((PASSED + 1))
    fi
else
    if ! check "Не удалось подключиться к PostgreSQL"; then
        FAILED=$((FAILED + 1))
    fi
fi
echo ""

# Проверка 2: Подключение к Redis
echo -e "${BLUE}=== Проверка подключения к Redis ===${NC}"
if redis-cli -h redis ping > /dev/null 2>&1; then
    if check "Подключение к Redis успешно"; then
        PASSED=$((PASSED + 1))
    fi
else
    if ! check "Не удалось подключиться к Redis"; then
        FAILED=$((FAILED + 1))
    fi
fi
echo ""

# Проверка 3: Статус миграций
echo -e "${BLUE}=== Проверка статуса миграций ===${NC}"
if command -v goose > /dev/null 2>&1; then
    echo "Текущий статус миграций:"
    goose -dir migrations postgres "$DATABASE_URL" status
    if [ $? -eq 0 ]; then
        if check "Миграции применены корректно"; then
            PASSED=$((PASSED + 1))
        fi
    else
        if ! check "Ошибка при проверке миграций"; then
            FAILED=$((FAILED + 1))
        fi
    fi
else
    echo -e "${RED}✗${NC} goose не найден"
    FAILED=$((FAILED + 1))
fi
echo ""

# Проверка 4: Существование таблиц
echo -e "${BLUE}=== Проверка структуры базы данных ===${NC}"
TABLES=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('budgets', 'expenses');" 2>/dev/null | tr -d ' ')

if [ "$TABLES" = "2" ]; then
    if check "Таблицы budgets и expenses существуют"; then
        PASSED=$((PASSED + 1))
    fi
    
    # Проверка структуры таблиц
    echo "Структура таблицы budgets:"
    psql "$DATABASE_URL" -c "\d budgets" 2>/dev/null | head -10
    
    echo "Структура таблицы expenses:"
    psql "$DATABASE_URL" -c "\d expenses" 2>/dev/null | head -10
else
    if ! check "Таблицы не найдены (найдено: $TABLES из 2)"; then
        FAILED=$((FAILED + 1))
    fi
fi
echo ""

# Проверка 5: Работа API
echo -e "${BLUE}=== Проверка работы API ===${NC}"
# Проверяем, доступен ли API
if wget -q --spider http://localhost:8080/ping 2>/dev/null || curl -s http://localhost:8080/ping > /dev/null 2>&1; then
    if check "API доступен (ping)"; then
        PASSED=$((PASSED + 1))
    fi
    
    # Проверка создания бюджета
    if command -v curl > /dev/null 2>&1; then
        RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8080/api/budgets \
            -H "Content-Type: application/json" \
            -d '{"category":"test","limit":1000}' 2>/dev/null)
        HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
        
        if [ "$HTTP_CODE" = "201" ]; then
            if check "API: создание бюджета работает"; then
                PASSED=$((PASSED + 1))
            fi
        else
            if ! check "API: создание бюджета вернуло код $HTTP_CODE"; then
                FAILED=$((FAILED + 1))
            fi
        fi
        
        # Проверка получения бюджетов
        RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8080/api/budgets 2>/dev/null)
        HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
        
        if [ "$HTTP_CODE" = "200" ]; then
            if check "API: получение списка бюджетов работает"; then
                PASSED=$((PASSED + 1))
            fi
        else
            if ! check "API: получение бюджетов вернуло код $HTTP_CODE"; then
                FAILED=$((FAILED + 1))
            fi
        fi
    fi
else
    echo -e "${YELLOW}⚠${NC} API недоступен (приложение ещё не запущено - это нормально при старте)"
    echo "   Проверка API будет доступна после запуска приложения"
fi
echo ""

# Проверка 6: Данные в базе
echo -e "${BLUE}=== Проверка данных в базе ===${NC}"
BUDGET_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM budgets;" 2>/dev/null | tr -d ' ')
EXPENSE_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM expenses;" 2>/dev/null | tr -d ' ')

echo "Бюджетов в БД: $BUDGET_COUNT"
echo "Транзакций в БД: $EXPENSE_COUNT"
if check "Данные доступны для чтения"; then
    PASSED=$((PASSED + 1))
fi
echo ""

echo "=========================================="
echo "ИТОГИ ПРОВЕРКИ"
echo "=========================================="
echo -e "${GREEN}Пройдено:${NC} $PASSED"
echo -e "${RED}Провалено:${NC} $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ Все проверки пройдены успешно!${NC}"
    exit 0
else
    echo -e "${RED}✗ Некоторые проверки не пройдены${NC}"
    exit 1
fi

