-- Migration: Remove order number from invalid errors
-- Description: Removes order_number column from invalid_errors table

-- Drop the index first
DROP INDEX IF EXISTS idx_invalid_errors_version_id_order_number;

-- Drop the column
ALTER TABLE invalid_errors 
DROP COLUMN IF EXISTS order_number;