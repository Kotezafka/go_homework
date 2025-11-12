-- +goose Up
-- +goose StatementBegin
CREATE TABLE budgets (
    id SERIAL PRIMARY KEY,
    category TEXT UNIQUE NOT NULL,
    limit_amount NUMERIC(14,2) NOT NULL CHECK (limit_amount > 0)
);

CREATE TABLE expenses (
    id SERIAL PRIMARY KEY,
    amount NUMERIC(14,2) NOT NULL CHECK (amount <> 0),
    category TEXT NOT NULL,
    description TEXT,
    date DATE NOT NULL
);

-- Опциональный индекс по (category, date)
CREATE INDEX idx_expenses_category_date ON expenses(category, date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_expenses_category_date;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS budgets;
-- +goose StatementEnd
