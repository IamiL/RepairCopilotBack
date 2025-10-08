-- Создание таблицы пользователей Telegram
CREATE TABLE IF NOT EXISTS telegram_users (
    id SERIAL PRIMARY KEY,
    tg_user_id BIGINT NOT NULL UNIQUE,
    user_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создание индекса для быстрого поиска по tg_user_id
CREATE INDEX IF NOT EXISTS idx_telegram_users_tg_user_id ON telegram_users(tg_user_id);

-- Создание индекса для быстрого поиска по user_id
CREATE INDEX IF NOT EXISTS idx_telegram_users_user_id ON telegram_users(user_id);

-- Создание типа для состояний пользователя
CREATE TYPE user_state_enum AS ENUM (
    'unauthorized',
    'awaiting_login',
    'awaiting_password',
    'authorized',
    'in_chat'
);

-- Создание таблицы состояний пользователей
CREATE TABLE IF NOT EXISTS user_states (
    id SERIAL PRIMARY KEY,
    tg_user_id BIGINT NOT NULL UNIQUE REFERENCES telegram_users(tg_user_id) ON DELETE CASCADE,
    state user_state_enum NOT NULL DEFAULT 'unauthorized',
    login_attempt VARCHAR(255),
    current_chat_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Создание индекса для быстрого поиска по tg_user_id
CREATE INDEX IF NOT EXISTS idx_user_states_tg_user_id ON user_states(tg_user_id);

-- Создание индекса для быстрого поиска по состоянию
CREATE INDEX IF NOT EXISTS idx_user_states_state ON user_states(state);