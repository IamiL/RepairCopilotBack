-- Migration: Drop error feedback table
-- Description: Removes error feedback table and enum type

DROP TABLE IF EXISTS error_feedback;
DROP TYPE IF EXISTS error_type;