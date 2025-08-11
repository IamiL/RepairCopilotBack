-- Migration: Add order number to invalid errors
-- Description: Adds order_number column to invalid_errors table to track the sequential number of each error in the array

ALTER TABLE invalid_errors 
ADD COLUMN order_number INTEGER NOT NULL DEFAULT 0;

-- Update existing records to have order numbers based on their current order
-- This ensures compatibility with existing data
WITH ordered_errors AS (
    SELECT 
        id,
        ROW_NUMBER() OVER (PARTITION BY version_id ORDER BY created_at, error_id) - 1 as new_order_number
    FROM invalid_errors
)
UPDATE invalid_errors 
SET order_number = oe.new_order_number
FROM ordered_errors oe 
WHERE invalid_errors.id = oe.id;

-- Create index for performance on order_number queries
CREATE INDEX IF NOT EXISTS idx_invalid_errors_version_id_order_number ON invalid_errors(version_id, order_number);