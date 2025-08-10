-- Migration: Create users table
-- Description: Creates the users table with UUID primary key and admin flags

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    login VARCHAR(255) NOT NULL,
    pass_hash BYTEA NOT NULL,
    is_admin1 BOOLEAN NOT NULL DEFAULT FALSE,
    is_admin2 BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on login for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_login ON users(login);

-- Create index on admin flags for admin queries
CREATE INDEX IF NOT EXISTS idx_users_is_admin1 ON users(is_admin1) WHERE is_admin1 = TRUE;
CREATE INDEX IF NOT EXISTS idx_users_is_admin2 ON users(is_admin2) WHERE is_admin2 = TRUE;