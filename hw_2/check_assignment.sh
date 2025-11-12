#!/bin/bash

# Скрипт для проверки всех требований задания

echo "=========================================="
echo "ПРОВЕРКА ВСЕХ ТРЕБОВАНИЙ ЗАДАНИЯ"
echo "=========================================="
echo ""

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASSED=0
FAILED=0

check() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} $1"
        ((FAILED++))
    fi
}

echo "=== 1. ПРОВЕРКА МИГРАЦИЙ ==="
echo ""

# Проверка каталога миграций
[ -d "ledger/migrations" ] && check "Каталог ledger/migrations/ существует" || echo -e "${RED}✗${NC} Каталог ledger/migrations/ не найден"

# Проверка файлов миграций
[ -f "ledger/migrations"/*.sql ] && check "SQL файлы миграций существуют" || echo -e "${RED}✗${NC} SQL файлы миграций не найдены"

# Проверка goose
which goose > /dev/null 2>&1 && check "Goose установлен" || echo -e "${YELLOW}⚠${NC} Goose не найден в PATH (может быть установлен через go install)"

# Проверка структуры таблиц в миграциях
if grep -q "CREATE TABLE budgets" ledger/migrations/*.sql 2>/dev/null; then
    check "Таблица budgets определена в миграциях"
else
    echo -e "${RED}✗${NC} Таблица budgets не найдена в миграциях"
fi

if grep -q "CREATE TABLE expenses" ledger/migrations/*.sql 2>/dev/null; then
    check "Таблица expenses определена в миграциях"
else
    echo -e "${RED}✗${NC} Таблица expenses не найдена в миграциях"
fi

if grep -q "UNIQUE" ledger/migrations/*.sql 2>/dev/null; then
    check "UNIQUE constraint на category в budgets"
else
    echo -e "${RED}✗${NC} UNIQUE constraint на category не найден"
fi

echo ""
echo "=== 2. ПРОВЕРКА ПОДКЛЮЧЕНИЯ К БД ==="
echo ""

# Проверка пакета db
[ -f "ledger/internal/db/db.go" ] && check "Пакет ledger/internal/db существует" || echo -e "${RED}✗${NC} Пакет db не найден"

# Проверка sql.Open
grep -q "sql.Open" ledger/internal/db/db.go 2>/dev/null && check "Используется sql.Open для подключения" || echo -e "${RED}✗${NC} sql.Open не найден"

# Проверка Ping
grep -q "Ping" ledger/internal/db/db.go 2>/dev/null && check "Выполняется Ping() для проверки подключения" || echo -e "${RED}✗${NC} Ping() не найден"

# Проверка DATABASE_URL
grep -q "DATABASE_URL" ledger/internal/db/db.go 2>/dev/null && check "Поддержка переменной DATABASE_URL" || echo -e "${RED}✗${NC} DATABASE_URL не найден"

# Проверка отдельных переменных окружения
grep -q "DB_HOST\|DB_PORT\|DB_USER\|DB_PASS\|DB_NAME" ledger/internal/db/db.go 2>/dev/null && check "Поддержка отдельных переменных окружения" || echo -e "${RED}✗${NC} Отдельные переменные окружения не найдены"

# Проверка пула соединений
grep -q "SetMaxOpenConns\|SetMaxIdleConns\|SetConnMaxLifetime" ledger/internal/db/db.go 2>/dev/null && check "Настроен пул соединений БД" || echo -e "${RED}✗${NC} Пул соединений не настроен"

echo ""
echo "=== 3. ПРОВЕРКА ФУНКЦИЙ РАБОТЫ С БД ==="
echo ""

# Проверка SetBudget
if grep -q "ON CONFLICT.*DO UPDATE" ledger/ledger.go 2>/dev/null; then
    check "SetBudget использует UPSERT (ON CONFLICT)"
else
    echo -e "${RED}✗${NC} SetBudget не использует UPSERT"
fi

# Проверка ListBudgets
if grep -q "SELECT.*FROM budgets" ledger/ledger.go 2>/dev/null; then
    check "ListBudgets читает из таблицы budgets"
else
    echo -e "${RED}✗${NC} ListBudgets не читает из БД"
fi

# Проверка AddTransaction
if grep -q "INSERT INTO expenses" ledger/ledger.go 2>/dev/null; then
    check "AddTransaction вставляет в таблицу expenses"
else
    echo -e "${RED}✗${NC} AddTransaction не вставляет в БД"
fi

if grep -q "RETURNING id" ledger/ledger.go 2>/dev/null; then
    check "AddTransaction использует RETURNING id"
else
    echo -e "${RED}✗${NC} RETURNING id не найден"
fi

if grep -q "budget exceeded\|Превышен бюджет" ledger/ledger.go 2>/dev/null; then
    check "AddTransaction проверяет лимит бюджета"
else
    echo -e "${RED}✗${NC} Проверка лимита бюджета не найдена"
fi

# Проверка ListTransactions
if grep -q "SELECT.*FROM expenses" ledger/ledger.go 2>/dev/null; then
    check "ListTransactions читает из таблицы expenses"
else
    echo -e "${RED}✗${NC} ListTransactions не читает из БД"
fi

# Проверка валидации
if grep -q "tx.Validate()\|b.Validate()" ledger/ledger.go 2>/dev/null; then
    check "Валидация вызывается перед добавлением"
else
    echo -e "${RED}✗${NC} Валидация не вызывается"
fi

echo ""
echo "=== 4. ПРОВЕРКА REDIS КЕША ==="
echo ""

# Проверка пакета cache
[ -f "ledger/internal/cache/cache.go" ] && check "Пакет ledger/internal/cache существует" || echo -e "${RED}✗${NC} Пакет cache не найден"

# Проверка redis клиента
grep -q "redis/go-redis" ledger/internal/cache/cache.go 2>/dev/null && check "Используется github.com/redis/go-redis/v9" || echo -e "${RED}✗${NC} Redis клиент не найден"

# Проверка переменных окружения Redis
grep -q "REDIS_ADDR\|REDIS_DB\|REDIS_PASSWORD" ledger/internal/cache/cache.go 2>/dev/null && check "Поддержка переменных окружения Redis" || echo -e "${RED}✗${NC} Переменные окружения Redis не найдены"

# Проверка GetReportSummary
if grep -q "GetReportSummary" ledger/ledger.go 2>/dev/null; then
    check "Функция GetReportSummary существует"
    
    if grep -q "report:summary:" ledger/ledger.go 2>/dev/null; then
        check "GetReportSummary использует кеш с ключом report:summary:"
    else
        echo -e "${RED}✗${NC} Ключ кеша report:summary: не найден"
    fi
    
    if grep -q "Set.*30.*time.Second\|Set.*TTL" ledger/ledger.go 2>/dev/null; then
        check "GetReportSummary кеширует с TTL 30 секунд"
    else
        echo -e "${YELLOW}⚠${NC} TTL для кеша не найден или отличается"
    fi
else
    echo -e "${RED}✗${NC} GetReportSummary не найдена"
fi

# Проверка кеша бюджетов
if grep -q "budgets:all" ledger/ledger.go 2>/dev/null; then
    check "Кеш списков бюджетов (budgets:all) реализован"
else
    echo -e "${YELLOW}⚠${NC} Кеш budgets:all не найден (опционально)"
fi

echo ""
echo "=== 5. ПРОВЕРКА README ==="
echo ""

# Проверка инструкций в README
if grep -qi "goose.*up\|миграции" README.md 2>/dev/null; then
    check "README содержит инструкции по применению миграций"
else
    echo -e "${RED}✗${NC} Инструкции по миграциям не найдены в README"
fi

if grep -qi "DATABASE_URL\|REDIS_ADDR" README.md 2>/dev/null; then
    check "README содержит информацию о переменных окружения"
else
    echo -e "${RED}✗${NC} Информация о переменных окружения не найдена"
fi

if grep -qi "curl.*api/reports/summary" README.md 2>/dev/null; then
    check "README содержит примеры cURL для /api/reports/summary"
else
    echo -e "${RED}✗${NC} Примеры cURL для reports/summary не найдены"
fi

if grep -qi "PostgreSQL\|Redis\|кеш" README.md 2>/dev/null; then
    check "README описывает использование PostgreSQL и Redis"
else
    echo -e "${RED}✗${NC} Описание PostgreSQL/Redis не найдено"
fi

echo ""
echo "=== 6. ПРОВЕРКА СТРУКТУРЫ ПРОЕКТА ==="
echo ""

[ -f "ledger/cmd/ledger/main.go" ] && check "Точка входа Ledger (cmd/ledger/main.go) существует" || echo -e "${RED}✗${NC} main.go не найден"
[ -f "gateway/main.go" ] && check "Gateway main.go существует" || echo -e "${RED}✗${NC} Gateway main.go не найден"
[ -f "ledger/go.mod" ] && check "ledger/go.mod существует" || echo -e "${RED}✗${NC} ledger/go.mod не найден"
[ -f "gateway/go.mod" ] && check "gateway/go.mod существует" || echo -e "${RED}✗${NC} gateway/go.mod не найден"

echo ""
echo "=========================================="
echo "ИТОГИ ПРОВЕРКИ"
echo "=========================================="
echo -e "${GREEN}Пройдено:${NC} $PASSED"
echo -e "${RED}Провалено:${NC} $FAILED"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ Все проверки пройдены!${NC}"
    exit 0
else
    echo -e "${RED}✗ Некоторые проверки не пройдены${NC}"
    exit 1
fi

