package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rongwang/COMP90018-server/internal/models"
)

// Repository interface defines the methods that any repository implementation must satisfy
type Repository interface {
	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)

	// Ledger operations
	CreateLedger(ctx context.Context, ledger *models.Ledger) error
	DeleteLedger(ctx context.Context, ledgerID string) error
	GetLedger(ctx context.Context, ledgerID string) (*models.Ledger, error)
	GetUserLedgers(ctx context.Context, userID string) ([]models.Ledger, error)

	// Ledger change operations
	AddLedgerChange(ctx context.Context, change *models.LedgerChange) error
	GetLedgerChangesBySequenceRange(ctx context.Context, ledgerID string, fromSeq, toSeq int64) ([]models.LedgerChange, error)
	GetLatestSequenceNumber(ctx context.Context, ledgerID string) (int64, error)

	// Ledger sharing operations
	AddUserToLedger(ctx context.Context, ledgerUser *models.LedgerUser) error
	CheckLedgerAccess(ctx context.Context, ledgerID, userID string, requiredPermission string) (bool, error)
	GetLedgerUsers(ctx context.Context, ledgerID string) ([]models.LedgerUser, error)
}

// PostgresRepository implements the Repository interface using PostgreSQL
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// GetDB returns the underlying database connection
func (r *PostgresRepository) GetDB() *sqlx.DB {
	return r.db
}

// User repository methods
func (r *PostgresRepository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, name, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	// Generate a new UUID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Name, user.Password, user.CreatedAt, user.UpdatedAt)

	return err
}

func (r *PostgresRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT * FROM users WHERE email = $1`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, err
	}

	return &user, nil
}

func (r *PostgresRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT * FROM users WHERE id = $1`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, err
	}

	return &user, nil
}

// Ledger repository methods
func (r *PostgresRepository) CreateLedger(ctx context.Context, ledger *models.Ledger) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()

	query := `
		INSERT INTO ledgers (id, name, description, currency, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	// Generate a new UUID if not provided
	if ledger.ID == "" {
		ledger.ID = uuid.New().String()
	}

	now := time.Now().UTC()
	ledger.CreatedAt = now
	ledger.UpdatedAt = now

	_, err = tx.ExecContext(ctx, query,
		ledger.ID, ledger.Name, ledger.Description, ledger.Currency,
		ledger.CreatedBy, ledger.CreatedAt, ledger.UpdatedAt)

	if err != nil {
		return err
	}

	// Add the creator as a user with write permissions
	ledgerUser := &models.LedgerUser{
		LedgerID:    ledger.ID,
		UserID:      ledger.CreatedBy,
		Permissions: "write",
		CreatedAt:   now,
	}

	err = r.addUserToLedgerTx(ctx, tx, ledgerUser)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) DeleteLedger(ctx context.Context, ledgerID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()

	// Delete ledger users first (due to foreign key constraint)
	_, err = tx.ExecContext(ctx, `DELETE FROM ledger_users WHERE ledger_id = $1`, ledgerID)
	if err != nil {
		return err
	}

	// Delete ledger changes
	_, err = tx.ExecContext(ctx, `DELETE FROM ledger_changes WHERE ledger_id = $1`, ledgerID)
	if err != nil {
		return err
	}

	// Delete the ledger
	_, err = tx.ExecContext(ctx, `DELETE FROM ledgers WHERE id = $1`, ledgerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetLedger(ctx context.Context, ledgerID string) (*models.Ledger, error) {
	query := `SELECT * FROM ledgers WHERE id = $1`

	var ledger models.Ledger
	err := r.db.GetContext(ctx, &ledger, query, ledgerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Ledger not found
		}
		return nil, err
	}

	return &ledger, nil
}

func (r *PostgresRepository) GetUserLedgers(ctx context.Context, userID string) ([]models.Ledger, error) {
	query := `
		SELECT l.* FROM ledgers l
		JOIN ledger_users lu ON l.id = lu.ledger_id
		WHERE lu.user_id = $1
	`

	var ledgers []models.Ledger
	err := r.db.SelectContext(ctx, &ledgers, query, userID)
	if err != nil {
		return nil, err
	}

	return ledgers, nil
}

// Ledger change repository methods
func (r *PostgresRepository) AddLedgerChange(ctx context.Context, change *models.LedgerChange) error {
	// Start a regular transaction - no need for serializable since we're using a dedicated sequence table
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()

	// Get and increment the sequence number atomically
	var nextSeq int64
	err = tx.QueryRowContext(ctx,
		`UPDATE ledger_sequences 
		SET current_sequence = current_sequence + 1 
		WHERE ledger_id = $1 
		RETURNING current_sequence`,
		change.LedgerID).Scan(&nextSeq)
	if err != nil {
		return err
	}

	change.SequenceNumber = nextSeq

	// Generate a new UUID if not provided
	if change.ID == "" {
		change.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if change.Timestamp.IsZero() {
		change.Timestamp = time.Now().UTC()
	}

	// Insert the change with the next sequence number
	query := `
		INSERT INTO ledger_changes (id, ledger_id, user_id, sequence_number, sql_statement, timestamp, base_sequence_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.ExecContext(ctx, query,
		change.ID, change.LedgerID, change.UserID, change.SequenceNumber,
		change.SQLStatement, change.Timestamp, change.BaseSequenceNum)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetLedgerChangesBySequenceRange(
	ctx context.Context,
	ledgerID string,
	fromSeq,
	toSeq int64,
) ([]models.LedgerChange, error) {
	query := `
		SELECT * FROM ledger_changes 
		WHERE ledger_id = $1 AND sequence_number >= $2
	`

	args := []interface{}{ledgerID, fromSeq}

	// Add toSeq condition if provided
	if toSeq > 0 {
		query += ` AND sequence_number <= $3`
		args = append(args, toSeq)
	}

	query += ` ORDER BY sequence_number ASC`

	var changes []models.LedgerChange
	err := r.db.SelectContext(ctx, &changes, query, args...)
	if err != nil {
		return nil, err
	}

	return changes, nil
}

func (r *PostgresRepository) GetLatestSequenceNumber(ctx context.Context, ledgerID string) (int64, error) {
	query := `SELECT current_sequence FROM ledger_sequences WHERE ledger_id = $1`

	var seqNum int64
	err := r.db.GetContext(ctx, &seqNum, query, ledgerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil // Return 0 if no sequence exists yet
		}
		return 0, err
	}

	return seqNum, nil
}

// Ledger sharing repository methods
// addUserToLedgerTx is a helper method that adds a user to a ledger within an existing transaction
func (r *PostgresRepository) addUserToLedgerTx(ctx context.Context, tx *sql.Tx, ledgerUser *models.LedgerUser) error {
	// Check if entry already exists
	var exists bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM ledger_users WHERE ledger_id = $1 AND user_id = $2)`,
		ledgerUser.LedgerID, ledgerUser.UserID).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		// Update the permissions if the user is already added
		query := `UPDATE ledger_users SET permissions = $1 WHERE ledger_id = $2 AND user_id = $3`
		_, err = tx.ExecContext(ctx, query,
			ledgerUser.Permissions, ledgerUser.LedgerID, ledgerUser.UserID)
	} else {
		// Add the user to the ledger
		query := `INSERT INTO ledger_users (ledger_id, user_id, permissions, created_at) VALUES ($1, $2, $3, $4)`

		if ledgerUser.CreatedAt.IsZero() {
			ledgerUser.CreatedAt = time.Now().UTC()
		}

		_, err = tx.ExecContext(ctx, query,
			ledgerUser.LedgerID, ledgerUser.UserID, ledgerUser.Permissions, ledgerUser.CreatedAt)
	}

	return err
}

func (r *PostgresRepository) AddUserToLedger(ctx context.Context, ledgerUser *models.LedgerUser) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()

	err = r.addUserToLedgerTx(ctx, tx, ledgerUser)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresRepository) CheckLedgerAccess(
	ctx context.Context,
	ledgerID string,
	userID string,
	requiredPermission string,
) (bool, error) {
	query := `SELECT permissions FROM ledger_users WHERE ledger_id = $1 AND user_id = $2`

	var permission string
	err := r.db.GetContext(ctx, &permission, query, ledgerID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // No access
		}
		return false, err
	}

	// If write permission is required, check if user has write permission
	// If read permission is required, both read and write permissions are sufficient
	if requiredPermission == "write" {
		return permission == "write", nil
	}

	return true, nil // User has access
}

func (r *PostgresRepository) GetLedgerUsers(ctx context.Context, ledgerID string) ([]models.LedgerUser, error) {
	query := `SELECT * FROM ledger_users WHERE ledger_id = $1`

	var ledgerUsers []models.LedgerUser
	err := r.db.SelectContext(ctx, &ledgerUsers, query, ledgerID)
	if err != nil {
		return nil, err
	}

	return ledgerUsers, nil
}
