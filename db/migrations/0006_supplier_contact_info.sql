-- Migration 0006: widen suppliers.phone for comma-separated numbers and add address.
-- Safe and idempotent migration for PostgreSQL.

ALTER TABLE suppliers ALTER COLUMN phone TYPE VARCHAR(255);
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS address VARCHAR(255) NULL;
