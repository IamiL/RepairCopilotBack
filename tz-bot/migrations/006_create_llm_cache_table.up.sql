-- Migration: Create LLM cache table
-- Description: Creates table for caching LLM requests and responses based on message hash

CREATE TABLE IF NOT EXISTS llm_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    messages_hash VARCHAR(64) NOT NULL UNIQUE,
    response_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_llm_cache_messages_hash ON llm_cache(messages_hash);
CREATE INDEX IF NOT EXISTS idx_llm_cache_created_at ON llm_cache(created_at);

-- Add comments for clarity
COMMENT ON TABLE llm_cache IS 'Stores cached LLM requests and responses';
COMMENT ON COLUMN llm_cache.messages_hash IS 'SHA-256 hash of the messages array for cache lookup';
COMMENT ON COLUMN llm_cache.response_data IS 'Cached response data from LLM API in JSON format';
COMMENT ON COLUMN llm_cache.created_at IS 'When the cache entry was created';
COMMENT ON COLUMN llm_cache.updated_at IS 'When the cache entry was last updated';