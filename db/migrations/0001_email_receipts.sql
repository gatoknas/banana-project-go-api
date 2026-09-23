-- Migration: add email_receipts table for Gmail bank-receipt ingestion.
-- This is additive and safe to run against an existing database
-- (it does NOT drop or modify any existing tables).

CREATE TABLE IF NOT EXISTS email_receipts (
    id SERIAL PRIMARY KEY,
    message_id VARCHAR(255) NOT NULL UNIQUE, -- Gmail message id, used for idempotent upserts
    sender VARCHAR(255),
    subject TEXT,
    received_at TIMESTAMPTZ,
    transaction_date TIMESTAMPTZ,
    amount DECIMAL(12, 2),
    currency VARCHAR(10) DEFAULT 'COP',
    payer VARCHAR(255),
    bank VARCHAR(100),
    reference VARCHAR(255),
    transaction_number VARCHAR(255),
    payment_method VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'imported', -- imported / error
    parse_error TEXT,
    raw_body TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_receipts_received_at ON email_receipts (received_at);
CREATE INDEX IF NOT EXISTS idx_email_receipts_transaction_date ON email_receipts (transaction_date);
