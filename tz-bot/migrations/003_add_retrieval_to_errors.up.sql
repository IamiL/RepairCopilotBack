-- Migration: Add retrieval field to error tables
-- Description: Adds retrieval field to store array of texts from LLM retrieval process

-- Add retrieval field to invalid_errors table (store as TEXT array)
ALTER TABLE invalid_errors 
ADD COLUMN retrieval TEXT[];

-- Add retrieval field to missing_errors table (store as TEXT array)
ALTER TABLE missing_errors 
ADD COLUMN retrieval TEXT[];