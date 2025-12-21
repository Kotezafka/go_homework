# Gateway & Ledger Services
# Микросервисное приложение для учёта финансов

Домашнее задание включает два сервиса: HTTP-шлюз (`gateway`) и бизнес-логику (`ledger`).
Приложение для личного учёта финансов с интеграцией Google Таблиц. Включает микросервисную архитектуру с аутентификацией, управлением транзакциями и бюджетами, а также импортом/экспортом данных.

## Архитектура

### Микросервисы
- **Gateway** - HTTP API шлюз (порт 8080)
- **Auth** - сервис аутентификации с JWT (gRPC порт 50051)
- **Ledger** - бизнес-логика финансов (gRPC порт 50052)
- **PostgreSQL** - база данных
- **Redis** - кеширование

### Структура сервисов

```
auth/
├── cmd/auth/             # Точка входа auth сервиса
├── internal/
│   ├── domain/           # Доменные сущности пользователей
│   ├── repository/pg/    # PostgreSQL репозиторий пользователей
│   └── service/          # Бизнес-логика аутентификации
├── proto/                # gRPC proto файлы
└── pkg/auth/             # gRPC сервер

ledger/
├── cmd/ledger/           # Точка входа ledger сервиса
├── internal/
│   ├── domain/           # Доменные сущности и интерфейсы
│   ├── repository/pg/    # PostgreSQL репозитории
│   ├── service/          # Бизнес-логика финансов
│   ├── app/              # Инициализация зависимостей
│   ├── db/               # Подключение к БД
│   └── cache/            # Redis клиент
├── proto/                # gRPC proto файлы
└── pkg/ledger/           # gRPC сервер

gateway/
├── cmd/gateway/          # Точка входа HTTP API
├── internal/api/         # HTTP обработчики и DTO
└── Dockerfile            # Контейнеризация
```

### Коммуникация
Сервисы общаются через gRPC:
- Gateway <-> Auth (аутентификация)
- Gateway <-> Ledger (финансовые операции)


## API Использование

### JWT Авторизация

Все прикладные эндпоинты защищены JWT-авторизацией. Для доступа к API необходимо:

1. **Зарегистрировать пользователя**
2. **Получить JWT токен** через логин
3. **Использовать токен** в заголовке `Authorization: Bearer <token>`

### Пример работы с защищенным API

```bash
# 1. Регистрация пользователя
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123",
    "name": "John Doe"
  }'

# 2. Вход и получение токена
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "securepassword123"
  }' | jq -r '.token')

# 3. Использование токена для создания бюджета
curl -X POST http://localhost:8080/api/budgets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "category": "еда",
    "limit": 5000.0
  }'

# 4. Получение списка бюджетов
curl -X GET http://localhost:8080/api/budgets \
  -H "Authorization: Bearer $TOKEN"

# 5. Создание транзакции
curl -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "amount": 1000.0,
    "category": "еда",
    "description": "Покупка продуктов",
    "date": "2025-01-15"
  }'

# 6. Получение отчета
curl -X GET "http://localhost:8080/api/reports/summary?from=2025-01-01&to=2025-01-31" \
  -H "Authorization: Bearer $TOKEN"
```

### Smoke test

```bash
cd hw_2

docker compose up -d --build

# 1. Миграции БД
go run github.com/pressly/goose/v3/cmd/goose@latest -dir ./auth/migrations postgres \
  "postgres://postgres_user:postgres_password@localhost:5434/postgres?sslmode=disable" up

go run github.com/pressly/goose/v3/cmd/goose@latest -dir ./ledger/migrations postgres \
  "postgres://postgres_user:postgres_password@localhost:5434/cashapp?sslmode=disable" up

# 2. Ping
curl -s http://localhost:8080/ping && echo

# 3. Проверка, что /api/* защищены (должно быть 401)
curl -s -i http://localhost:8080/api/budgets | head -10

# 4. Регистрация/логин и извлечение токена
curl -s -X POST http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"u1@example.com","password":"pass12345","name":"User One"}' && echo

TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"u1@example.com","password":"pass12345"}' \
  | sed -n 's/.*\"token\":\"\\([^\"]*\\)\".*/\\1/p')
echo "TOKEN=$TOKEN"

# 5. Бюджеты
curl -s -X POST http://localhost:8080/api/budgets \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"category":"еда","limit":5000}' && echo

curl -s http://localhost:8080/api/budgets -H "Authorization: Bearer $TOKEN" && echo

# 6. Транзакции
curl -s -X POST http://localhost:8080/api/transactions \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"amount":1000,"category":"еда","description":"Покупка","date":"2025-01-15"}' && echo

curl -s http://localhost:8080/api/transactions -H "Authorization: Bearer $TOKEN" && echo

# 7. Превышение бюджета (должно быть 409)
curl -s -i -X POST http://localhost:8080/api/transactions \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"amount":5000,"category":"еда","description":"Слишком много","date":"2025-01-16"}' | head -20

# 8. Отчёт + кеш
curl -s "http://localhost:8080/api/reports/summary?from=2025-01-01&to=2025-01-31" \
  -H "Authorization: Bearer $TOKEN" && echo

curl -s "http://localhost:8080/api/reports/summary?from=2025-01-01&to=2025-01-31" \
  -H "Authorization: Bearer $TOKEN" && echo

docker compose logs ledger --tail=50 | grep -E "\\[cache\\]"

# 9. Bulk import
curl -s -X POST "http://localhost:8080/api/transactions/bulk?workers=4" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '[{"amount":10,"category":"еда","description":"a","date":"2025-01-10"},{"amount":0,"category":"еда","description":"bad","date":"2025-01-10"},{"amount":6000,"category":"еда","description":"too big","date":"2025-01-11"}]' \
  && echo
```

### Переменные окружения

**Gateway:**
- `MOCK_MODE` - режим работы (true/false)
- `LEDGER_ADDR` - адрес Ledger gRPC сервиса (по умолчанию: `ledger:50052`)
- `AUTH_ADDR` - адрес Auth gRPC сервиса (по умолчанию: `auth:50051`)

**Ledger:**
- `DATABASE_URL` - строка подключения к PostgreSQL
- `REDIS_ADDR` - адрес Redis сервера

**Auth:**
- `DATABASE_URL` - строка подключения к PostgreSQL
- `JWT_SECRET` - секретный ключ для подписи JWT токенов

## Запуск

### Docker Compose

```bash
# Запуск всех сервисов
docker-compose up -d

# Просмотр логов
docker-compose logs -f

# Остановка
docker-compose down
```