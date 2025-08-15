-- Migration: Add analysis metrics to versions table
-- Description: Adds allRubs, allTokens, and inspectionTime fields to versions table

ALTER TABLE versions 
ADD COLUMN all_rubs DOUBLE PRECISION,
ADD COLUMN all_tokens BIGINT,
ADD COLUMN inspection_time BIGINT; -- storing as nanoseconds for time.Duration

-- Add comments for clarity
COMMENT ON COLUMN versions.all_rubs IS 'Total cost in rubles for the analysis';
COMMENT ON COLUMN versions.all_tokens IS 'Total number of tokens used in the analysis';
COMMENT ON COLUMN versions.inspection_time IS 'Time taken for inspection in nanoseconds';