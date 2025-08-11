-- Migration: Remove retrieval field from error tables
-- Description: Removes retrieval field from invalid_errors and missing_errors tables

-- Remove retrieval field from invalid_errors table
ALTER TABLE invalid_errors 
DROP COLUMN IF EXISTS retrieval;

-- Remove retrieval field from missing_errors table  
ALTER TABLE missing_errors 
DROP COLUMN IF EXISTS retrieval;