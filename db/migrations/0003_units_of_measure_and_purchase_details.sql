-- Migration: Units of measure fixes & purchase line conversion attributes.
-- Safe and idempotent migration for PostgreSQL.

-- 1. Fix typo 'Vulto' -> 'Bulto' in units_of_measure if present
UPDATE units_of_measure
SET name = 'Bulto', abbreviation = 'bl'
WHERE name ILIKE 'Vulto';

-- 2. Add standard units of measure if they don't already exist
INSERT INTO units_of_measure (name, abbreviation)
SELECT 'Gramo', 'g'
WHERE NOT EXISTS (SELECT 1 FROM units_of_measure WHERE name ILIKE 'Gramo');

INSERT INTO units_of_measure (name, abbreviation)
SELECT 'Mililitro', 'ml'
WHERE NOT EXISTS (SELECT 1 FROM units_of_measure WHERE name ILIKE 'Mililitro');

INSERT INTO units_of_measure (name, abbreviation)
SELECT 'Litro', 'lt'
WHERE NOT EXISTS (SELECT 1 FROM units_of_measure WHERE name ILIKE 'Litro');

INSERT INTO units_of_measure (name, abbreviation)
SELECT 'Bulto', 'bl'
WHERE NOT EXISTS (SELECT 1 FROM units_of_measure WHERE name ILIKE 'Bulto');

-- 3. Enhance purchase_details table with purchase_unit_id and conversion_factor
ALTER TABLE purchase_details
ADD COLUMN IF NOT EXISTS purchase_unit_id INT NULL,
ADD COLUMN IF NOT EXISTS conversion_factor DECIMAL(12, 4) NOT NULL DEFAULT 1.0000;

-- 4. Add foreign key constraint for purchase_unit_id if not present
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_purchase_details_purchase_unit'
    ) THEN
        ALTER TABLE purchase_details
        ADD CONSTRAINT fk_purchase_details_purchase_unit
        FOREIGN KEY (purchase_unit_id) REFERENCES units_of_measure(id);
    END IF;
END $$;
