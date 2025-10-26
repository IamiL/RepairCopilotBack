-- +goose Up
-- +goose StatementBegin
CREATE
EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create technical_specifications table
CREATE TABLE IF NOT EXISTS technical_specifications
(
    id
    UUID
    PRIMARY
    KEY,
    name
    VARCHAR
(
    255
) NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
                             );
-- Create versions table
CREATE TABLE IF NOT EXISTS versions
(
    id
    UUID
    PRIMARY
    KEY,
    technical_specification_id
    UUID
    NOT
    NULL
    REFERENCES
    technical_specifications
(
    id
) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP
  WITH TIME ZONE NOT NULL,
      original_file_id VARCHAR (255) NOT NULL,
    out_html TEXT NOT NULL,
    css TEXT NOT NULL,
    checked_file_id VARCHAR
(
    255
) NOT NULL,
    original_file_size BIGINT,
    number_of_errors BIGINT,
    status TEXT,
    progress INTEGER,
    all_rubs DOUBLE PRECISION,
    all_tokens BIGINT,
    inspection_time BIGINT
    );

CREATE TABLE IF NOT EXISTS llm_cache (
                                         id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    messages_hash VARCHAR(64) NOT NULL UNIQUE,
    response_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS errors
(
    id                   UUID PRIMARY KEY,
    version_id           UUID,
    group_id             TEXT,
    error_code           TEXT,
    order_number         INTEGER,
    name                 TEXT,
    description          TEXT,
    detector             TEXT,
    preliminary_notes    TEXT,
    overall_critique     TEXT,
    verdict              TEXT,
    process_analysis     TEXT,
    process_critique     TEXT,
    process_verification TEXT,
    process_retrieval    TEXT[], -- array of strings
    instances            JSONB   -- storing instances as JSON
);

CREATE TABLE IF NOT EXISTS invalid_instances (
                                                 id UUID PRIMARY KEY,
                                                 html_id INTEGER NOT NULL,
                                                 error_id UUID NOT NULL,
                                                 quote TEXT,
                                                 rationale TEXT,
                                                 suggested_fix TEXT,
                                                 original_quote TEXT,
                                                 quote_lines TEXT[], -- array of strings
                                                 until_the_end_of_sentence BOOLEAN,
                                                 start_line_number INTEGER,
                                                 end_line_number INTEGER,
                                                 system_comment TEXT,
                                                 order_number INTEGER,
                                                 feedback_exists BOOLEAN,
                                                 feedback_mark BOOLEAN,
                                                 feedback_comment TEXT,
                                                 feedback_user UUID,
                                                 feedback_created_at TIMESTAMP,
                                                 feedback_verification_exists BOOLEAN,
                                                 feedback_verification_mark BOOLEAN,
                                                 feedback_verification_comment TEXT,
                                                 feedback_verification_user UUID
);

CREATE TABLE IF NOT EXISTS missing_instances (
                                                 id UUID PRIMARY KEY,
                                                 html_id INTEGER NOT NULL,
                                                 error_id UUID NOT NULL,
                                                 suggested_fix TEXT,
                                                 rationale TEXT,
                                                 feedback_exists BOOLEAN,
                                                 feedback_mark BOOLEAN,
                                                 feedback_comment TEXT,
                                                 feedback_user UUID,
                                                 feedback_created_at TIMESTAMP,
                                                 feedback_verification_exists BOOLEAN,
                                                 feedback_verification_mark BOOLEAN,
                                                 feedback_verification_comment TEXT,
                                                 feedback_verification_user UUID
);

ALTER TABLE versions
ADD COLUMN html_from_word_parser text,
ADD COLUMN html_with_placeholder text,
ADD COLUMN html_paragraphs text,
ADD COLUMN markdown_from_markdown_service text,
ADD COLUMN html_with_ids_from_markdown_service text,
ADD COLUMN mappings_from_markdown_service jsonb,
ADD COLUMN promts_from_promt_builder jsonb,
ADD COLUMN group_reports_from_llm jsonb,
ADD COLUMN html_paragraphs_with_wrapped_errors text,
ADD COLUMN report jsonb;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_llm_cache_messages_hash ON llm_cache(messages_hash);
CREATE INDEX IF NOT EXISTS idx_llm_cache_created_at ON llm_cache(created_at);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_technical_specifications_user_id ON technical_specifications (user_id);
CREATE INDEX IF NOT EXISTS idx_technical_specifications_created_at ON technical_specifications (created_at);

CREATE INDEX IF NOT EXISTS idx_versions_technical_specification_id ON versions (technical_specification_id);
CREATE INDEX IF NOT EXISTS idx_versions_version_number ON versions (version_number);
CREATE INDEX IF NOT EXISTS idx_versions_created_at ON versions (created_at);

CREATE INDEX IF NOT EXISTS idx_invalid_errors_version_id ON invalid_errors (version_id);
CREATE INDEX IF NOT EXISTS idx_invalid_errors_group_id ON invalid_errors (group_id);
CREATE INDEX IF NOT EXISTS idx_invalid_errors_error_code ON invalid_errors (error_code);

CREATE INDEX IF NOT EXISTS idx_missing_errors_version_id ON missing_errors (version_id);
CREATE INDEX IF NOT EXISTS idx_missing_errors_group_id ON missing_errors (group_id);
CREATE INDEX IF NOT EXISTS idx_missing_errors_error_code ON missing_errors (error_code);

-- Create index for performance on order_number queries
CREATE INDEX IF NOT EXISTS idx_invalid_errors_version_id_order_number ON invalid_errors(version_id, order_number);

-- Create unique constraint for version numbers per technical specification
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_version_per_spec ON versions (technical_specification_id, version_number);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS missing_errors CASCADE;
DROP TABLE IF EXISTS invalid_errors CASCADE;
DROP TABLE IF EXISTS versions CASCADE;
DROP TABLE IF EXISTS technical_specifications CASCADE;
-- +goose StatementEnd