-- Migration: Drop technical specifications tables
-- Description: Drops all tables related to technical specifications

-- Drop tables in reverse order (child tables first)
DROP TABLE IF EXISTS missing_errors CASCADE;
DROP TABLE IF EXISTS invalid_errors CASCADE;
DROP TABLE IF EXISTS versions CASCADE;
DROP TABLE IF EXISTS technical_specifications CASCADE;