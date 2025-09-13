package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID        string    `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	Name      string    `db:"name" json:"name"`
	Password  string    `db:"password" json:"-"` // Password hash, not returned in JSON
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Ledger represents a ledger owned by users
type Ledger struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Currency    string    `db:"currency" json:"currency"`
	CreatedBy   string    `db:"created_by" json:"createdBy"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// LedgerUser represents the relationship between users and ledgers (for sharing)
type LedgerUser struct {
	LedgerID    string    `db:"ledger_id" json:"ledgerId"`
	UserID      string    `db:"user_id" json:"userId"`
	Permissions string    `db:"permissions" json:"permissions"` // "read" or "write"
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

// LedgerChange represents a change made to a ledger
type LedgerChange struct {
	ID              string    `db:"id" json:"id"`
	LedgerID        string    `db:"ledger_id" json:"ledgerId"`
	UserID          string    `db:"user_id" json:"userId"`
	SequenceNumber  int64     `db:"sequence_number" json:"sequenceNumber"`
	SQLStatement    string    `db:"sql_statement" json:"sqlStatement"`
	Timestamp       time.Time `db:"timestamp" json:"timestamp"`
	BaseSequenceNum int64     `db:"base_sequence_number" json:"baseSequenceNumber"`
}
