-- +goose Up
-- +goose StatementBegin
-- создаем таблицу пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    login VARCHAR(255) NOT NULL,
    pass_hash TEXT NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    last_visit_at TIMESTAMP,
    is_admin1 BOOLEAN NOT NULL,
    is_admin2 BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    is_confirmed BOOLEAN,
    confirmation_code TEXT,
    inspections_per_day INTEGER,
    inspections_for_today INTEGER,
    inspections_left_for_today INTEGER,
    inspections_count INTEGER
    );

-- Create index on login for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_login ON users(login);

-- Create index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Create index on admin flags for admin queries
CREATE INDEX IF NOT EXISTS idx_users_is_admin1 ON users(is_admin1) WHERE is_admin1 = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_is_admin2 ON users(is_admin2) WHERE is_admin2 = TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- удаляем индексы и таблицу
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_users_is_admin2;
DROP INDEX IF EXISTS idx_users_is_admin1;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_login;

-- Drop table
DROP TABLE IF EXISTS users;
-- +goose StatementEnd