-- +goose Up
-- +goose StatementBegin
-- создаем таблицу логов
CREATE TABLE IF NOT EXISTS action_logs (
    id SERIAL PRIMARY KEY,
    action VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    type INTEGER
);

-- Индекс для быстрого поиска по пользователю и дате
CREATE INDEX idx_action_logs_user_id_created_at ON action_logs(user_id, created_at DESC);

-- Индекс для общего поиска по дате
CREATE INDEX idx_action_logs_created_at ON action_logs(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- удаляем индексы и таблицу
DROP INDEX IF EXISTS idx_action_logs_created_at;
DROP INDEX IF EXISTS idx_action_logs_user_id_created_at;
DROP TABLE IF EXISTS action_logs;
-- +goose StatementEnd