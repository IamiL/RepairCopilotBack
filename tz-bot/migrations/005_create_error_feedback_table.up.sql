-- Migration: Create error feedback table
-- Description: Creates table for storing user feedback on invalid and missing errors

-- Create enum type for error types
CREATE TYPE error_type AS ENUM ('invalid', 'missing');

CREATE TABLE IF NOT EXISTS error_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version_id UUID NOT NULL REFERENCES versions(id) ON DELETE CASCADE,
    error_id UUID NOT NULL,
    error_type error_type NOT NULL,
    is_good_error BOOLEAN NOT NULL,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_error_feedback_version_id ON error_feedback(version_id);
CREATE INDEX IF NOT EXISTS idx_error_feedback_error_id ON error_feedback(error_id);
CREATE INDEX IF NOT EXISTS idx_error_feedback_error_type ON error_feedback(error_type);
CREATE INDEX IF NOT EXISTS idx_error_feedback_is_good_error ON error_feedback(is_good_error);
CREATE INDEX IF NOT EXISTS idx_error_feedback_created_at ON error_feedback(created_at);

-- Add comments for clarity
COMMENT ON TABLE error_feedback IS 'Stores user feedback on error analysis quality';
COMMENT ON COLUMN error_feedback.version_id IS 'ID of the version this feedback belongs to';
COMMENT ON COLUMN error_feedback.error_id IS 'ID from either invalid_errors or missing_errors table';
COMMENT ON COLUMN error_feedback.error_type IS 'Type of error: invalid or missing';
COMMENT ON COLUMN error_feedback.is_good_error IS 'True if error analysis is good, false if bad';
COMMENT ON COLUMN error_feedback.comment IS 'Optional user comment about the error analysis';