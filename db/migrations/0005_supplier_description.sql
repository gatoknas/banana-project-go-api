-- Migration 0005: Add optional description notes column to suppliers table.
-- Safe and idempotent migration for PostgreSQL.

ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS description TEXT NULL;
