DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_schema = 'public' 
        AND table_name = 'budgets' 
        AND column_name = 'user_id'
    ) THEN
        CREATE TEMP TABLE budgets_backup AS SELECT * FROM budgets;
        
        DROP TABLE IF EXISTS budgets CASCADE;
        
        CREATE TABLE budgets (
            id SERIAL PRIMARY KEY,
            user_id UUID NOT NULL,
            category TEXT NOT NULL,
            limit_amount NUMERIC(14,2) NOT NULL CHECK (limit_amount > 0),
            UNIQUE(user_id, category)
        );
        
        INSERT INTO budgets (user_id, category, limit_amount)
        SELECT gen_random_uuid(), category, limit_amount
        FROM budgets_backup;
        
        DROP TABLE IF EXISTS budgets_backup;
        
        RAISE NOTICE 'Таблица budgets пересоздана с колонкой user_id';
    ELSE
        RAISE NOTICE 'Колонка user_id уже существует в таблице budgets';
    END IF;
END $$;

ALTER TABLE budgets DROP CONSTRAINT IF EXISTS budgets_user_category_unique;
ALTER TABLE budgets DROP COLUMN IF EXISTS user_id;
