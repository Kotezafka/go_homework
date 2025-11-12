#!/bin/bash

# Полный тест всех требований задания через API

echo "=========================================="
echo "ПОЛНЫЙ ТЕСТ ВСЕХ ТРЕБОВАНИЙ ЗАДАНИЯ"
echo "=========================================="
echo ""

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Настройка переменных
export DATABASE_URL="postgres://postgres_user:postgres_password@localhost:5430/cashapp?sslmode=disable"
export REDIS_ADDR=localhost:6379

BASE_URL="http://localhost:8080"

# Важно: Ledger сервис должен быть запущен с теми же переменными окружения!
# Используйте START_LEDGER.sh или установите переменные перед запуском:
# export DATABASE_URL="postgres://postgres_user:postgres_password@localhost:5430/cashapp?sslmode=disable"
# export REDIS_ADDR=localhost:6379

# Проверка доступности сервисов
echo -e "${BLUE}=== Проверка доступности сервисов ===${NC}"
echo ""

if ! curl -s "$BASE_URL/ping" > /dev/null 2>&1; then
    echo -e "${RED}✗ Gateway не доступен на $BASE_URL${NC}"
    echo "Запустите Gateway: cd gateway && go run main.go"
    exit 1
fi

echo -e "${GREEN}✓ Gateway доступен${NC}"
echo ""

# Проверка подключения к базе данных
echo -e "${BLUE}=== Проверка подключения к базе данных ===${NC}"
if ! psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${RED}✗ Не удалось подключиться к базе данных${NC}"
    echo "Проверьте:"
    echo "  1. PostgreSQL запущен и доступен на порту 5430"
    echo "  2. База данных 'cashapp' создана"
    echo "  3. Пользователь 'postgres_user' с паролем 'postgres_password' существует"
    echo "  4. Ledger сервис запущен с правильной DATABASE_URL:"
    echo "     export DATABASE_URL=\"$DATABASE_URL\""
    echo "     cd ledger && go run ./cmd/ledger"
    echo ""
    echo "Текущая DATABASE_URL: $DATABASE_URL"
    exit 1
fi
echo -e "${GREEN}✓ Подключение к базе данных успешно${NC}"
echo ""

# Функция для проверки ответа
check_response() {
    local method=$1
    local url=$2
    local data=$3
    local expected_status=$4
    local description=$5
    
    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" 2>&1)
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data" 2>&1)
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        echo -e "${GREEN}✓${NC} $description (HTTP $http_code)"
        echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
        echo ""
        return 0
    else
        echo -e "${RED}✗${NC} $description (ожидался $expected_status, получен $http_code)"
        echo "Ответ: $body"
        echo ""
        return 1
    fi
}

PASSED=0
FAILED=0

# Тест 1: Создание бюджета
echo -e "${BLUE}=== Тест 1: Создание бюджета ===${NC}"
if check_response "POST" "$BASE_URL/api/budgets" \
    '{"category":"еда","limit":5000}' \
    "201" \
    "POST /api/budgets - создание бюджета"; then
    ((PASSED++))
else
    ((FAILED++))
fi

# Тест 2: Проверка бюджета в БД
echo -e "${BLUE}=== Тест 2: Проверка бюджета в БД ===${NC}"
budget_count=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM budgets WHERE category='еда';" 2>/dev/null | tr -d ' ')
if [ "$budget_count" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} Бюджет сохранён в БД (найдено записей: $budget_count)"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Бюджет не найден в БД"
    ((FAILED++))
fi
echo ""

# Тест 3: Создание транзакции в пределах лимита
echo -e "${BLUE}=== Тест 3: Создание транзакции в пределах лимита ===${NC}"
if check_response "POST" "$BASE_URL/api/transactions" \
    '{"amount":1500,"category":"еда","description":"Обед","date":"2025-01-15"}' \
    "201" \
    "POST /api/transactions - транзакция в пределах лимита"; then
    ((PASSED++))
else
    ((FAILED++))
fi

# Тест 4: Проверка транзакции в БД
echo -e "${BLUE}=== Тест 4: Проверка транзакции в БД ===${NC}"
tx_count=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM expenses WHERE category='еда' AND amount=1500;" 2>/dev/null | tr -d ' ')
if [ "$tx_count" -gt 0 ]; then
    echo -e "${GREEN}✓${NC} Транзакция сохранена в БД (найдено записей: $tx_count)"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Транзакция не найдена в БД"
    ((FAILED++))
fi
echo ""

# Тест 5: Попытка превысить лимит
echo -e "${BLUE}=== Тест 5: Попытка превысить лимит ===${NC}"
initial_count=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM expenses;" 2>/dev/null | tr -d ' ')

response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/transactions" \
    -H "Content-Type: application/json" \
    -d '{"amount":4000,"category":"еда","description":"Ужин","date":"2025-01-15"}' 2>&1)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" = "409" ]; then
    echo -e "${GREEN}✓${NC} Попытка превысить лимит вернула 409 Conflict"
    echo "Ответ: $body"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Ожидался 409, получен $http_code"
    echo "Ответ: $body"
    ((FAILED++))
fi

# Проверка, что транзакция НЕ добавлена
final_count=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM expenses;" 2>/dev/null | tr -d ' ')
if [ "$final_count" = "$initial_count" ]; then
    echo -e "${GREEN}✓${NC} Транзакция не добавлена в БД (количество не изменилось)"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Транзакция была добавлена в БД (количество изменилось: $initial_count -> $final_count)"
    ((FAILED++))
fi
echo ""

# Тест 6: Список транзакций из БД
echo -e "${BLUE}=== Тест 6: Список транзакций из БД ===${NC}"
if check_response "GET" "$BASE_URL/api/transactions" "" "200" "GET /api/transactions - список из БД"; then
    ((PASSED++))
else
    ((FAILED++))
fi

# Тест 7: Список бюджетов из БД
echo -e "${BLUE}=== Тест 7: Список бюджетов из БД ===${NC}"
if check_response "GET" "$BASE_URL/api/budgets" "" "200" "GET /api/budgets - список из БД"; then
    ((PASSED++))
else
    ((FAILED++))
fi

# Тест 8: Проверка сохранности данных после перезапуска
echo -e "${BLUE}=== Тест 8: Проверка сохранности данных ===${NC}"
echo "Создаём дополнительную транзакцию..."
check_response "POST" "$BASE_URL/api/transactions" \
    '{"amount":500,"category":"транспорт","description":"Такси","date":"2025-01-16"}' \
    "201" \
    "Создание транзакции для проверки сохранности" > /dev/null

tx_before=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM expenses;" 2>/dev/null | tr -d ' ')
budget_before=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM budgets;" 2>/dev/null | tr -d ' ')

echo "Транзакций в БД: $tx_before"
echo "Бюджетов в БД: $budget_before"
echo -e "${YELLOW}⚠${NC} Перезапустите Ledger и Gateway, затем проверьте данные снова"
echo ""

# Тест 9: Проверка кеша отчётов
echo -e "${BLUE}=== Тест 9: Проверка кеша отчётов ===${NC}"
echo "Первый запрос (cache miss - из БД):"
time1=$(date +%s%N)
response1=$(curl -s "$BASE_URL/api/reports/summary?from=2025-01-01&to=2025-01-31")
time1_end=$(date +%s%N)
duration1=$((($time1_end - $time1) / 1000000))

echo "Время выполнения: ${duration1}ms"
echo "$response1" | python3 -m json.tool 2>/dev/null || echo "$response1"
echo ""

echo "Второй запрос (cache hit - из Redis):"
time2=$(date +%s%N)
response2=$(curl -s "$BASE_URL/api/reports/summary?from=2025-01-01&to=2025-01-31")
time2_end=$(date +%s%N)
duration2=$((($time2_end - $time2) / 1000000))

echo "Время выполнения: ${duration2}ms"

if [ "$duration2" -lt "$duration1" ]; then
    echo -e "${GREEN}✓${NC} Второй запрос быстрее (кеш работает)"
    ((PASSED++))
else
    echo -e "${YELLOW}⚠${NC} Время выполнения похоже (возможно, кеш не работает или запросы слишком быстрые)"
fi

if [ "$response1" = "$response2" ]; then
    echo -e "${GREEN}✓${NC} Ответы идентичны (кеш работает корректно)"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Ответы различаются"
    ((FAILED++))
fi
echo ""

# Тест 10: Валидация (нулевая сумма)
echo -e "${BLUE}=== Тест 10: Валидация (нулевая сумма) ===${NC}"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/transactions" \
    -H "Content-Type: application/json" \
    -d '{"amount":0,"category":"еда","description":"Тест","date":"2025-01-15"}' 2>&1)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" = "400" ]; then
    echo -e "${GREEN}✓${NC} Валидация работает (400 Bad Request для нулевой суммы)"
    echo "Ответ: $body"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} Ожидался 400, получен $http_code"
    ((FAILED++))
fi
echo ""

echo "=========================================="
echo "ИТОГИ ТЕСТИРОВАНИЯ"
echo "=========================================="
echo -e "${GREEN}Пройдено:${NC} $PASSED"
echo -e "${RED}Провалено:${NC} $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ Все тесты пройдены!${NC}"
    exit 0
else
    echo -e "${RED}✗ Некоторые тесты не пройдены${NC}"
    exit 1
fi

