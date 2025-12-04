#!/bin/bash
set +e

# Создание базы данных cashapp, если её нет
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE cashapp;
EOSQL

echo "База данных cashapp готова!"
