-- Migration: Make suppliers.tax_id nullable and allow suppliers to be registered with only name and phone.
-- Safe and idempotent migration for PostgreSQL.

ALTER TABLE suppliers ALTER COLUMN tax_id DROP NOT NULL;
