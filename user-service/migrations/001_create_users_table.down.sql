-- Migration rollback: Drop users table
-- Description: Removes the users table and related objects

-- Drop trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_users_is_admin2;
DROP INDEX IF EXISTS idx_users_is_admin1;
DROP INDEX IF EXISTS idx_users_login;

-- Drop table
DROP TABLE IF EXISTS users;