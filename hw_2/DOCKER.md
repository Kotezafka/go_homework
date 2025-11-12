# Запуск приложения через Docker

## Быстрый старт

1. **Сборка и запуск всех сервисов:**
   ```bash
   docker-compose up --build
   ```

2. **Запуск в фоновом режиме:**
   ```bash
   docker-compose up -d --build
   ```

3. **Проверка работы:**
   ```bash
   curl http://localhost:8080/ping
   ```

## Остановка

```bash
docker-compose down
```

Для удаления всех данных (включая базу данных):
```bash
docker-compose down -v
```

## Просмотр логов

Все сервисы:
```bash
docker-compose logs -f
```

Только gateway:
```bash
docker-compose logs -f gateway
```

## Структура

- **PostgreSQL** - порт 5431 (внутри контейнера 5432)
- **Redis** - порт 6380 (внутри контейнера 6379)
- **Gateway** - порт 8080

**Примечание:** Порты изменены, чтобы не конфликтовать с существующими контейнерами. Внутри Docker сети контейнеры общаются напрямую по внутренним портам.

## Переменные окружения

Можно переопределить через `docker-compose.yml` или `.env` файл:

```yaml
environment:
  DATABASE_URL: "postgres://user:pass@postgres:5432/cashapp?sslmode=disable"
  REDIS_ADDR: "redis:6379"
```

## Что происходит при запуске

1. PostgreSQL и Redis запускаются и ждут готовности
2. Gateway ждёт готовности зависимостей
3. Автоматически создаётся база данных `cashapp` (если её нет)
4. Применяются миграции из `ledger/migrations/`
5. Запускается Gateway сервис на порту 8080

## Отладка

Войти в контейнер gateway:
```bash
docker-compose exec gateway sh
```

Проверить подключение к PostgreSQL:
```bash
docker-compose exec gateway psql "$DATABASE_URL" -c "SELECT 1;"
```

Проверить подключение к Redis:
```bash
docker-compose exec gateway redis-cli -h redis ping
```

## Проверка системы внутри Docker

При запуске контейнера автоматически выполняется проверка системы (если `CHECK_ENABLED=true`):

```bash
# Проверка включена по умолчанию в docker-compose.yml
docker-compose up -d
```

Проверка включает:
- ✅ Подключение к PostgreSQL
- ✅ Подключение к Redis  
- ✅ Статус миграций (goose status)
- ✅ Существование таблиц budgets и expenses
- ✅ Структура таблиц
- ✅ Доступность данных в БД

**Ручной запуск проверки:**

```bash
# Запустить проверку внутри контейнера
docker-compose exec gateway /docker-check.sh
```

**Настройка проверки:**

В `docker-compose.yml` можно настроить:
- `CHECK_ENABLED=true` - включить/выключить проверку при старте
- `CHECK_REQUIRED=true` - остановить контейнер, если проверка не пройдена

## Запуск тестов

### API тесты (интеграционные)

После запуска контейнеров можно запустить полный набор API тестов:

```bash
./docker-test.sh
```

Или вручную:
```bash
# Убедитесь, что сервисы запущены
docker-compose up -d

# Запустите тесты
./test_all_requirements.sh
```

### Go unit тесты

Go unit тесты нужно запускать на хосте (не в контейнере):

```bash
# Тесты ledger
cd ledger && go test -v ./...

# Тесты gateway
cd gateway && go test -v ./...
```

