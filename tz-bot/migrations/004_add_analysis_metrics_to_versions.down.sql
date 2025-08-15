-- Migration: Remove analysis metrics from versions table
-- Description: Removes allRubs, allTokens, and inspectionTime fields from versions table

ALTER TABLE versions 
DROP COLUMN IF EXISTS all_rubs,
DROP COLUMN IF EXISTS all_tokens,
DROP COLUMN IF EXISTS inspection_time;