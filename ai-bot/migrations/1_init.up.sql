CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS messages (
    id uuid PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    from_bot BOOLEAN NOT NULL,
    text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS chats (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    is_finished BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY,
    active_chat uuid NOT NULL,
    is_registered BOOLEAN NOT NULL,
    login varchar(16) NULL,
    pass_hash varchar(100) NULL,
    is_worker BOOLEAN NULL,
    department_id uuid NULL,
    is_admin uuid NULL
);

CREATE TABLE IF NOT EXISTS departments(
    id uuid PRIMARY KEY,
    name text NOT NULL
)