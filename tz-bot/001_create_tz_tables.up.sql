-- Migration: Create technical specifications tables
-- Description: Creates tables for storing technical specifications and their versions with errors

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create technical_specifications table
CREATE TABLE IF NOT EXISTS technical_specifications (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create versions table
CREATE TABLE IF NOT EXISTS versions (
    id UUID PRIMARY KEY,
    technical_specification_id UUID NOT NULL REFERENCES technical_specifications(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    original_file_id VARCHAR(255) NOT NULL,
    out_html TEXT NOT NULL,
    css TEXT NOT NULL,
    checked_file_id VARCHAR(255) NOT NULL
);

-- Create InvalidErrors table
CREATE TABLE IF NOT EXISTS invalid_errors (
    id UUID PRIMARY KEY,
    version_id UUID NOT NULL REFERENCES versions(id) ON DELETE CASCADE,
    error_id INTEGER NOT NULL,
    error_id_str VARCHAR(255) NOT NULL,
    group_id VARCHAR(255) NOT NULL,
    error_code VARCHAR(255) NOT NULL,
    quote TEXT NOT NULL,
    analysis TEXT NOT NULL,
    critique TEXT NOT NULL,
    verification TEXT NOT NULL,
    suggested_fix TEXT NOT NULL,
    rationale TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create MissingErrors table
CREATE TABLE IF NOT EXISTS missing_errors (
    id UUID PRIMARY KEY,
    version_id UUID NOT NULL REFERENCES versions(id) ON DELETE CASCADE,
    error_id INTEGER NOT NULL,
    error_id_str VARCHAR(255) NOT NULL,
    group_id VARCHAR(255) NOT NULL,
    error_code VARCHAR(255) NOT NULL,
    analysis TEXT NOT NULL,
    critique TEXT NOT NULL,
    verification TEXT NOT NULL,
    suggested_fix TEXT NOT NULL,
    rationale TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_technical_specifications_user_id ON technical_specifications(user_id);
CREATE INDEX IF NOT EXISTS idx_technical_specifications_created_at ON technical_specifications(created_at);

CREATE INDEX IF NOT EXISTS idx_versions_technical_specification_id ON versions(technical_specification_id);
CREATE INDEX IF NOT EXISTS idx_versions_version_number ON versions(version_number);
CREATE INDEX IF NOT EXISTS idx_versions_created_at ON versions(created_at);

CREATE INDEX IF NOT EXISTS idx_invalid_errors_version_id ON invalid_errors(version_id);
CREATE INDEX IF NOT EXISTS idx_invalid_errors_group_id ON invalid_errors(group_id);
CREATE INDEX IF NOT EXISTS idx_invalid_errors_error_code ON invalid_errors(error_code);

CREATE INDEX IF NOT EXISTS idx_missing_errors_version_id ON missing_errors(version_id);
CREATE INDEX IF NOT EXISTS idx_missing_errors_group_id ON missing_errors(group_id);
CREATE INDEX IF NOT EXISTS idx_missing_errors_error_code ON missing_errors(error_code);

-- Create unique constraint for version numbers per technical specification
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_version_per_spec ON versions(technical_specification_id, version_number);