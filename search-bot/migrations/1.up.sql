CREATE TABLE IF NOT EXISTS chats
(
    ID            UUID PRIMARY KEY,
    user_id       UUID,
    is_processing BOOLEAN,
    created_at    TIMESTAMP NOT NULL,
    updated_at    TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS users
(
    ID                      UUID PRIMARY KEY,
--     login                   varchar(255),
--     pass_hash               text,
--     first_name              varchar(255),
--     last_name               varchar(255),
--     email                   varchar(255),
--     confirmation_code       varchar(10),
--     is_confirmed            BOOLEAN,
--     is_admin                BOOLEAN,
--     admin_level             INTEGER,
--     last_visit_at           TIMESTAMP,
    messages_per_day        INTEGER,
    messages_left_for_today INTEGER,
    created_at              TIMESTAMP NOT NULL,
    updated_at              TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS messages
(
    id            UUID PRIMARY KEY,
    chat_id       UUID,
    role          VARCHAR(10),
    content       TEXT,
    nesting_level INTEGER,
    created_at    TIMESTAMP NOT NULL,
    updated_at    TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS actions
(
    id         UUID PRIMARY KEY,
    type       INTEGER,
    user_id    UUID,
    message    TEXT,
    created_at TIMESTAMP
);