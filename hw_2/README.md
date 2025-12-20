# Gateway & Ledger Services
# Redis Кеширование в Ledger Service

## Описание задания

**Задача:** Подключить Redis как ключ-значение кеш с TTL для результатов отчёта и списков бюджетов в микросервисе Ledger.

**Цель:** Оптимизировать производительность системы аналитики личных трат за счёт кеширования часто запрашиваемых данных (отчёты и списки бюджетов).

## Реализация

### 1. Архитектура решения

Кеширование реализовано в сервисе Ledger через интерфейс `cache.Client`

### 2. Структура кода

```
ledger/
├── internal/
│   ├── cache/
│   │   └── cache.go          # Реализация Redis клиента
│   ├── service/
│   │   └── service.go        # Использование кеша для бюджетов и отчётов
│   └── app/
│       └── app.go            # Инициализация кеша при старте сервиса
```

### 3. Пакет кеша (`ledger/internal/cache`)

#### Интерфейс Client

```go
type Client interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, keys ...string) error
    Flush(ctx context.Context) error
    Close() error
}
```

#### Реализация Redis клиента

**Файл:** `ledger/internal/cache/cache.go`

**Конфигурация:**
- `REDIS_ADDR` (по умолчанию: `localhost:6379`) — адрес Redis сервера
- `REDIS_DB` (по умолчанию: `0`) — номер базы данных Redis
- `REDIS_PASSWORD` (опционально) — пароль для подключения

**Пример инициализации:**
```go
cacheClient, err := cache.New(ctx)
if err != nil {
    log.Fatalf("Failed to initialize cache: %v", err)
}
defer cacheClient.Close()
```

### 4. Кеширование списков бюджетов

**Реализация:** `ledger/internal/service/service.go`

#### Метод `ListBudgets()`

**Стратегия кеширования:**
1. **Проверка кеша** — попытка загрузить данные из Redis
2. **Cache Hit** — если данные найдены, возвращаются из кеша
3. **Cache Miss** — если данных нет:
   - Загрузка из PostgreSQL
   - Расчёт остатка бюджета (Limit - потраченная сумма)
   - Сохранение в Redis с TTL 30 секунд
   - Возврат данных клиенту

**Ключ кеша:** `budgets:all:<userID>`

**TTL:** 30 секунд (`defaultBudgetsTTL`)

**Сериализация:** JSON (массив объектов `Budget`)

**Пример кода:**
```go
func (s *ledgerService) ListBudgets(ctx context.Context, userID string) ([]domain.Budget, error) {
    // Попытка загрузить из кеша
    if budgets, ok, err := s.loadBudgetsFromCache(ctx, userID); err == nil && ok {
        return budgets, nil
    }

    // Загрузка из БД
    budgets, err := s.budgets.List(ctx, userID)
    if err != nil {
        return nil, err
    }

    // Расчёт остатков
    for i := range budgets {
        spent, _ := s.expenses.SumByCategory(ctx, userID, budgets[i].Category)
        budgets[i].Remaining = budgets[i].Limit - spent
    }

    // Сохранение в кеш
    s.saveBudgetsToCache(ctx, userID, budgets)
    return budgets, nil
}
```

#### Инвалидация кеша

**Метод `SetBudget()`** — при создании/обновлении бюджета кеш инвалидируется:

```go
func (s *ledgerService) SetBudget(ctx context.Context, userID string, budget domain.Budget) (domain.Budget, error) {
    // ... валидация и сохранение в БД ...
    
    // Инвалидация кеша
    s.invalidateBudgetsCache(ctx, userID)
    return budget, nil
}
```

**Ключ для удаления:** `budgets:all:<userID>`

### 5. Кеширование результатов отчёта

**Реализация:** `ledger/internal/service/service.go`

#### Метод `GetReportSummary()`

**Стратегия кеширования:**
1. **Проверка кеша** — попытка загрузить отчёт из Redis
2. **Cache Hit** — если отчёт найден, возвращается из кеша
3. **Cache Miss** — если отчёта нет:
   - Конкурентный расчёт сумм по категориям через PostgreSQL
   - Агрегация: `SELECT COALESCE(SUM(amount),0) FROM expenses WHERE category = $1 AND date >= $2 AND date <= $3`
   - Сохранение результата в Redis с TTL 30 секунд
   - Возврат отчёта клиенту

**Ключ кеша:** `report:summary:<from>:<to>` (даты в формате `YYYY-MM-DD`)

**TTL:** 30 секунд (`defaultReportTTL`)

**Сериализация:** JSON (массив объектов `ReportSummary`)

**Пример кода:**
```go
func (s *ledgerService) GetReportSummary(ctx context.Context, userID, from, to string) ([]domain.ReportSummary, error) {
    // Попытка загрузить из кеша
    if summary, ok, err := s.loadReportFromCache(ctx, from, to); err == nil && ok {
        return summary, nil
    }

    // Конкурентный расчёт отчёта
    // ... вычисление через горутины ...

    // Сохранение в кеш
    s.saveReportToCache(ctx, from, to, summary)
    return summary, nil
}
```

**Формирование ключа:**
```go
func reportCacheKey(from, to string) string {
    return fmt.Sprintf("report:summary:%s:%s", from, to)
}
```


## Запуск Redis через Docker Compose

```yaml
redis:
  image: redis:7-alpine
  container_name: cashapp-redis
  ports:
    - "6379:6379"
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 5s
    timeout: 3s
    retries: 5
```

Запуск:
```bash
docker-compose up -d redis
```


В `docker-compose.yml` для сервиса `ledger`:

```yaml
ledger:
  environment:
    DATABASE_URL: "postgres://postgres_user:postgres_password@db:5432/cashapp?sslmode=disable"
    REDIS_ADDR: "redis:6379"
    REDIS_DB: "0"
    # REDIS_PASSWORD: "optional_password"
```


## Использование

### Проверка работы кеша

1. **Запустите сервисы:**
```bash
docker-compose up -d
```

2. **Создайте бюджет:**
```bash
curl -X POST http://localhost:8080/api/budgets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"category":"еда","limit":5000}'
```

3. **Получите список бюджетов (первый запрос — из БД, второй — из кеша):**
```bash
# Первый запрос — cache miss, данные из БД
curl http://localhost:8080/api/budgets \
  -H "Authorization: Bearer <token>"

# Второй запрос в течение 30 секунд — cache hit, данные из Redis
curl http://localhost:8080/api/budgets \
  -H "Authorization: Bearer <token>"
```

4. **Получите отчёт (проверка кеширования отчётов):**
```bash
# Первый запрос — cache miss
curl "http://localhost:8080/api/reports/summary?from=2025-01-01&to=2025-01-31" \
  -H "Authorization: Bearer <token>"

# Второй запрос в течение 30 секунд — cache hit
curl "http://localhost:8080/api/reports/summary?from=2025-01-01&to=2025-01-31" \
  -H "Authorization: Bearer <token>"
```

### Мониторинг Redis

Подключитесь к Redis и проверьте ключи:

```bash
# Через Docker
docker exec -it cashapp-redis redis-cli

# В Redis CLI
KEYS *
GET "budgets:all:test-user-id"
GET "report:summary:2025-01-01:2025-01-31"
TTL "budgets:all:test-user-id"  # Проверка оставшегося времени жизни
```


## Тестирование

### Ручная проверка

1. Убедитесь, что Redis запущен:
```bash
docker ps | grep redis
```

2. Проверьте логи Ledger сервиса:
```bash
docker-compose logs ledger | grep -i redis
```

3. Проверьте подключение через Redis CLI:
```bash
docker exec -it cashapp-redis redis-cli ping
# Ожидаемый ответ: PONG
```

### Автоматические тесты

Для тестирования кеша можно использовать mock-реализацию интерфейса `cache.Client` или тестовый Redis контейнер
