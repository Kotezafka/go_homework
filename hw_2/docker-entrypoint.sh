#!/bin/sh
set -e

echo "Ожидание готовности PostgreSQL..."
until pg_isready -h postgres -p 5432 -U postgres_user -d postgres > /dev/null 2>&1; do
  echo "PostgreSQL недоступен - ожидание..."
  sleep 1
done

echo "PostgreSQL готов!"

# Создание базы данных cashapp, если её нет
PGPASSWORD=postgres_password psql -h postgres -U postgres_user -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'cashapp'" | grep -q 1 || \
PGPASSWORD=postgres_password psql -h postgres -U postgres_user -d postgres -c "CREATE DATABASE cashapp;"

echo "База данных cashapp готова!"

echo "Ожидание готовности Redis..."
until redis-cli -h redis ping > /dev/null 2>&1; do
  echo "Redis недоступен - ожидание..."
  sleep 1
done

echo "Redis готов!"

echo "Применение миграций..."
goose -dir migrations postgres "$DATABASE_URL" up || {
  echo "Ошибка при применении миграций"
  exit 1
}

echo "Миграции применены успешно!"

# Запуск проверки системы (если переменная CHECK_ENABLED=true)
if [ "$CHECK_ENABLED" = "true" ]; then
  echo ""
  echo "=========================================="
  echo "Запуск проверки системы..."
  echo "=========================================="
  /docker-check.sh || {
    echo "Проверка системы не пройдена"
    if [ "$CHECK_REQUIRED" = "true" ]; then
      exit 1
    fi
  }
  echo ""
fi

echo "Запуск приложения..."
exec "$@"

