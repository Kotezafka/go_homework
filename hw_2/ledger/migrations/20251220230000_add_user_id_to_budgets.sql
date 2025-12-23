-- +goose Up
-- +goose StatementBegin

-- Проверяем, существует ли колонка user_id в таблице budgets
DO $$
BEGIN
    -- Если колонки user_id нет, пересоздаём таблицу
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_schema = 'public' 
        AND table_name = 'budgets' 
        AND column_name = 'user_id'
    ) THEN
        -- Сохраняем данные если они есть (временная таблица)
        CREATE TEMP TABLE budgets_backup AS SELECT * FROM budgets;
        
        -- Удаляем старую таблицу
        DROP TABLE IF EXISTS budgets CASCADE;
        
        -- Создаём таблицу с правильной структурой
        CREATE TABLE budgets (
            id SERIAL PRIMARY KEY,
            user_id UUID NOT NULL,
            category TEXT NOT NULL,
            limit_amount NUMERIC(14,2) NOT NULL CHECK (limit_amount > 0),
            UNIQUE(user_id, category)
        );
        
        -- Восстанавливаем данные с временным user_id (если были данные)
        -- Используем gen_random_uuid() для старых записей
        INSERT INTO budgets (user_id, category, limit_amount)
        SELECT gen_random_uuid(), category, limit_amount
        FROM budgets_backup;
        
        -- Удаляем временную таблицу
        DROP TABLE IF EXISTS budgets_backup;
        
        RAISE NOTICE 'Таблица budgets пересоздана с колонкой user_id';
    ELSE
        RAISE NOTICE 'Колонка user_id уже существует в таблице budgets';
    END IF;
END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Откат: удаляем колонку user_id
ALTER TABLE budgets DROP CONSTRAINT IF EXISTS budgets_user_category_unique;
ALTER TABLE budgets DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
