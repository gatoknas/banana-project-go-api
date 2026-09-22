-- Migration: add Cafeteria product category.
-- This is additive and safe to run against an existing database
-- (it does NOT drop or modify any existing tables or duplicate entries).

INSERT INTO categories (name)
SELECT 'Cafeteria'
WHERE NOT EXISTS (
    SELECT 1 FROM categories WHERE name = 'Cafeteria'
);
