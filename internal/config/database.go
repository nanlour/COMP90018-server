package config

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// SetupDatabase initializes the database connection
func SetupDatabase(cfg *Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.Database.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// createTables creates the necessary tables in the database
func createTables(db *sqlx.DB) error {
	// Create users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(36) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			password VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Create ledgers table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ledgers (
			id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			currency VARCHAR(3) NOT NULL,
			created_by VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Create ledger_users table (for ledger sharing)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ledger_users (
			ledger_id VARCHAR(36) NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			permissions VARCHAR(10) NOT NULL,
			created_at TIMESTAMP NOT NULL,
			PRIMARY KEY (ledger_id, user_id)
		)
	`)
	if err != nil {
		return err
	}

	// Create ledger_changes table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ledger_changes (
			id VARCHAR(36) PRIMARY KEY,
			ledger_id VARCHAR(36) NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
			user_id VARCHAR(36) NOT NULL REFERENCES users(id),
			sequence_number BIGINT NOT NULL,
			sql_statement TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			base_sequence_number BIGINT NOT NULL,
			UNIQUE (ledger_id, sequence_number)
		)
	`)
	if err != nil {
		return err
	}

	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_ledger_changes_ledger_id ON ledger_changes(ledger_id)",
		"CREATE INDEX IF NOT EXISTS idx_ledger_changes_ledger_seq ON ledger_changes(ledger_id, sequence_number)",
	}

	for _, idx := range indexes {
		_, err = db.Exec(idx)
		if err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
			// Don't return error here, indexes are not critical
		}
	}

	return nil
}
