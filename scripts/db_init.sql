-- Drop database if exists (comment out in production)
DROP DATABASE IF EXISTS billapp;

-- Create database
CREATE DATABASE billapp;

-- The rest of the schema creation needs to be in a separate file
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Create ledgers table
CREATE TABLE IF NOT EXISTS ledgers (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    currency VARCHAR(3) NOT NULL,
    created_by VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Create ledger_users table (for ledger sharing)
CREATE TABLE IF NOT EXISTS ledger_users (
    ledger_id VARCHAR(36) NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permissions VARCHAR(10) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    PRIMARY KEY (ledger_id, user_id)
);

-- Create ledger_changes table
CREATE TABLE IF NOT EXISTS ledger_changes (
    id VARCHAR(36) PRIMARY KEY,
    ledger_id VARCHAR(36) NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    sequence_number BIGINT NOT NULL,
    sql_statement TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    base_sequence_number BIGINT NOT NULL,
    UNIQUE (ledger_id, sequence_number)
);

-- Create ledger_sequences table to track sequence numbers
CREATE TABLE IF NOT EXISTS ledger_sequences (
    ledger_id VARCHAR(36) PRIMARY KEY REFERENCES ledgers(id) ON DELETE CASCADE,
    current_sequence BIGINT NOT NULL DEFAULT 0
);

-- Create a trigger to initialize sequence when a new ledger is created
CREATE OR REPLACE FUNCTION init_ledger_sequence()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO ledger_sequences (ledger_id, current_sequence) VALUES (NEW.id, 0);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER create_ledger_sequence
    AFTER INSERT ON ledgers
    FOR EACH ROW
    EXECUTE FUNCTION init_ledger_sequence();

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_ledger_changes_ledger_id ON ledger_changes(ledger_id);
CREATE INDEX IF NOT EXISTS idx_ledger_changes_ledger_seq ON ledger_changes(ledger_id, sequence_number);
