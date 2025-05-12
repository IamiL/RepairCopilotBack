CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS messages (
                                         id uuid PRIMARY KEY,
                                        user_id BIGINT NOT NULL,
                                         text TEXT NOT NULL
)