-- +goose Up
-- +goose StatementBegin
CREATE TABLE budgets (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    category TEXT NOT NULL,
    limit_amount NUMERIC(14,2) NOT NULL CHECK (limit_amount > 0)
);

CREATE UNIQUE INDEX budgets_user_id_category_key ON budgets(user_id, category);

CREATE TABLE expenses (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    amount NUMERIC(14,2) NOT NULL CHECK (amount <> 0),
    category TEXT NOT NULL,
    description TEXT,
    date DATE NOT NULL
);

CREATE INDEX idx_expenses_user_category_date ON expenses(user_id, category, date);
CREATE INDEX idx_expenses_user_date ON expenses(user_id, date DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_expenses_user_date;
DROP INDEX IF EXISTS idx_expenses_user_category_date;
DROP INDEX IF EXISTS budgets_user_id_category_key;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS budgets;
-- +goose StatementEnd
