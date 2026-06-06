-- PostgreSQL Schema for POS System (English Version)
-- Translated from original partner model

-- Drop tables if they exist (in reverse dependency order)
DROP VIEW IF EXISTS v_latest_product_purchases;
DROP VIEW IF EXISTS v_full_catalog;

DROP TABLE IF EXISTS addition_details;
DROP TABLE IF EXISTS sale_details;
DROP TABLE IF EXISTS sales;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS purchase_details;
DROP TABLE IF EXISTS purchases;
DROP TABLE IF EXISTS suppliers;
DROP TABLE IF EXISTS inventories;
DROP TABLE IF EXISTS allowed_additions;
DROP TABLE IF EXISTS product_recipes;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS units_of_measure;

-- Drop function if exists
DROP FUNCTION IF EXISTS update_updated_at_column();

-- =============================================================================
-- FUNCTIONS & TRIGGERS FOR TIMESTAMP UPDATES
-- =============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- =============================================================================
-- CORE TABLES
-- =============================================================================

-- units_of_measure (Spanish: unidades_medida)
CREATE TABLE units_of_measure (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL, -- e.g. Kilogram, Unit, Box, Litre
    abbreviation VARCHAR(10) NOT NULL, -- e.g. kg, und, cj, lt
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- categories (Spanish: categorias)
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL, -- e.g. Beverages, Fruit, Ice Cream
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- products (Spanish: productos)
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    category_id INT NOT NULL,
    unit_of_measure_id INT NOT NULL,
    sell_price DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    average_cost DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    is_for_sale BOOLEAN NOT NULL DEFAULT TRUE,
    requires_recipe BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories(id),
    CONSTRAINT fk_products_unit FOREIGN KEY (unit_of_measure_id) REFERENCES units_of_measure(id)
);

CREATE TRIGGER update_products_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- product_recipes (Spanish: recetas_productos)
CREATE TABLE product_recipes (
    parent_product_id INT NOT NULL,
    child_product_id INT NOT NULL,
    quantity DECIMAL(10, 4) NOT NULL DEFAULT 1.0000,
    PRIMARY KEY (parent_product_id, child_product_id),
    
    CONSTRAINT fk_recipes_parent FOREIGN KEY (parent_product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_recipes_child FOREIGN KEY (child_product_id) REFERENCES products(id)
);

-- allowed_additions (Spanish: adiciones_permitidas)
CREATE TABLE allowed_additions (
    main_product_id INT NOT NULL,
    addition_product_id INT NOT NULL,
    additional_price DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    PRIMARY KEY (main_product_id, addition_product_id),
    
    CONSTRAINT fk_additions_main FOREIGN KEY (main_product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_additions_addition FOREIGN KEY (addition_product_id) REFERENCES products(id)
);

-- inventories (Spanish: inventarios)
CREATE TABLE inventories (
    product_id INT PRIMARY KEY,
    current_stock DECIMAL(12, 4) NOT NULL DEFAULT 0.0000,
    minimum_stock DECIMAL(12, 4) NOT NULL DEFAULT 0.0000,
    maximum_stock DECIMAL(12, 4) NOT NULL DEFAULT 0.0000,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_inventaries_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

CREATE TRIGGER update_inventories_updated_at
BEFORE UPDATE ON inventories
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- suppliers (Spanish: proveedores)
CREATE TABLE suppliers (
    id SERIAL PRIMARY KEY,
    tax_id VARCHAR(20) NOT NULL UNIQUE, -- NIT/Cedula
    company_name VARCHAR(100) NOT NULL, -- Razon Social
    contact_name VARCHAR(100) NULL,
    phone VARCHAR(20) NULL,
    email VARCHAR(100) NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- purchases (Spanish: compras)
CREATE TABLE purchases (
    id SERIAL PRIMARY KEY,
    supplier_id INT NOT NULL,
    purchase_date TIMESTAMPTZ NOT NULL,
    invoice_number VARCHAR(50) NULL,
    total_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    notes TEXT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_purchases_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers(id)
);

-- purchase_details (Spanish: detalles_compras)
CREATE TABLE purchase_details (
    id SERIAL PRIMARY KEY,
    purchase_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity_purchased DECIMAL(12, 4) NOT NULL,
    unit_cost DECIMAL(12, 2) NOT NULL,
    
    CONSTRAINT fk_purchase_details_purchase FOREIGN KEY (purchase_id) REFERENCES purchases(id) ON DELETE CASCADE,
    CONSTRAINT fk_purchase_details_product FOREIGN KEY (product_id) REFERENCES products(id)
);

-- users (Spanish: usuarios)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(100) NOT NULL,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'Cajero', -- Administrator, Cajero
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- sales (Spanish: ventas)
CREATE TABLE sales (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    sale_date TIMESTAMPTZ NOT NULL,
    total_amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
    payment_method VARCHAR(30) NOT NULL, -- Cash, Card, etc.
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_sales_user FOREIGN KEY (user_id) REFERENCES users(id)
);

-- sale_details (Spanish: detalles_ventas)
CREATE TABLE sale_details (
    id SERIAL PRIMARY KEY,
    sale_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity DECIMAL(10, 2) NOT NULL DEFAULT 1.00,
    historical_unit_price DECIMAL(12, 2) NOT NULL,
    subtotal DECIMAL(12, 2) NOT NULL,
    
    CONSTRAINT fk_sale_details_sale FOREIGN KEY (sale_id) REFERENCES sales(id) ON DELETE CASCADE,
    CONSTRAINT fk_sale_details_product FOREIGN KEY (product_id) REFERENCES products(id)
);

-- addition_details (Spanish: detalles_adiciones)
CREATE TABLE addition_details (
    id SERIAL PRIMARY KEY,
    sale_detail_id INT NOT NULL,
    addition_product_id INT NOT NULL,
    quantity DECIMAL(10, 2) NOT NULL DEFAULT 1.00,
    historical_addition_price DECIMAL(12, 2) NOT NULL,
    
    CONSTRAINT fk_addition_details_sale_detail FOREIGN KEY (sale_detail_id) REFERENCES sale_details(id) ON DELETE CASCADE,
    CONSTRAINT fk_addition_details_product FOREIGN KEY (addition_product_id) REFERENCES products(id)
);

-- =============================================================================
-- VIEWS
-- =============================================================================

CREATE OR REPLACE VIEW v_full_catalog AS
SELECT 
    p.id AS product_id,
    p.name AS product,
    c.name AS category,
    u.abbreviation AS unit,
    p.sell_price,
    p.average_cost,
    p.is_for_sale,
    p.requires_recipe,
    p.created_at
FROM products p
JOIN categories c ON p.category_id = c.id
JOIN units_of_measure u ON p.unit_of_measure_id = u.id;

CREATE OR REPLACE VIEW v_latest_product_purchases AS
SELECT 
    pd.product_id,
    p.name AS product,
    u.name AS unit_of_measure,
    s.company_name AS supplier,
    pur.purchase_date AS latest_purchase_date,
    pur.invoice_number,
    pd.quantity_purchased,
    pd.unit_cost
FROM purchase_details pd
JOIN purchases pur ON pd.purchase_id = pur.id
JOIN products p ON pd.product_id = p.id
JOIN units_of_measure u ON p.unit_of_measure_id = u.id
JOIN suppliers s ON pur.supplier_id = s.id
WHERE pur.purchase_date = (
    SELECT MAX(pur2.purchase_date)
    FROM purchase_details pd2
    JOIN purchases pur2 ON pd2.purchase_id = pur2.id
    WHERE pd2.product_id = pd.product_id
);

-- =============================================================================
-- INDEXES
-- =============================================================================
CREATE INDEX idx_products_name ON products (name);
CREATE INDEX idx_products_created_at ON products (created_at);
CREATE INDEX idx_purchase_details_product_id ON purchase_details (product_id);
CREATE INDEX idx_sales_sale_date ON sales (sale_date);
CREATE INDEX idx_suppliers_company_name ON suppliers (company_name);
