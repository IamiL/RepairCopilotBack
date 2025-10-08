-- Откат миграции
DROP TABLE IF EXISTS user_states;
DROP TABLE IF EXISTS telegram_users;
DROP TYPE IF EXISTS user_state_enum;