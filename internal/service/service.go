package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rongwang/COMP90018-server/internal/models"
	"github.com/rongwang/COMP90018-server/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// Service defines all the business logic operations
type Service interface {
	// Authentication
	SignUp(ctx context.Context, req models.SignUpRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)

	// Ledger operations
	CreateLedger(ctx context.Context, userID string, req models.CreateLedgerRequest) (*models.LedgerResponse, error)
	DeleteLedger(ctx context.Context, userID, ledgerID string) error

	// Ledger changes
	SubmitLedgerChange(ctx context.Context, userID, ledgerID string, req models.LedgerChangeRequest) (*models.LedgerChangeResponse, error)
	GetLedgerChanges(ctx context.Context, userID, ledgerID string, fromSeq, toSeq int64) (*models.GetLedgerChangesResponse, error)
	GetLatestSequenceNumber(ctx context.Context, userID, ledgerID string) (*models.SequenceNumberResponse, error)

	// Ledger sharing
	AddUserToLedger(ctx context.Context, userID, ledgerID string, req models.AddUserToLedgerRequest) (*models.AddUserResponse, error)
}

// DefaultService implements the Service interface
type DefaultService struct {
	repo          repository.Repository
	jwtSecret     []byte
	tokenDuration time.Duration
}

// NewDefaultService creates a new DefaultService
func NewDefaultService(repo repository.Repository, jwtSecret string) Service {
	return &DefaultService{
		repo:          repo,
		jwtSecret:     []byte(jwtSecret),
		tokenDuration: 24 * time.Hour, // 24 hours token validity
	}
}

// Authentication methods
func (s *DefaultService) SignUp(ctx context.Context, req models.SignUpRequest) (*models.AuthResponse, error) {
	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("error checking user existence: %w", err)
	}

	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	// Create the user
	user := &models.User{
		ID:       uuid.New().String(),
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashedPassword),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	return &models.AuthResponse{
		Status: "success",
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
	}, nil
}

func (s *DefaultService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	// Get the user
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	if user == nil {
		return nil, errors.New("invalid email or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("error generating token: %w", err)
	}

	return &models.AuthResponse{
		Status:    "success",
		UserID:    user.ID,
		Token:     token,
		ExpiresIn: int(s.tokenDuration.Seconds()),
	}, nil
}

// Ledger operations
func (s *DefaultService) CreateLedger(
	ctx context.Context,
	userID string,
	req models.CreateLedgerRequest,
) (*models.LedgerResponse, error) {
	// Create the ledger
	ledger := &models.Ledger{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Currency:    req.Currency,
		CreatedBy:   userID,
	}

	if err := s.repo.CreateLedger(ctx, ledger); err != nil {
		return nil, fmt.Errorf("error creating ledger: %w", err)
	}

	return &models.LedgerResponse{
		Status:                "success",
		LedgerID:              ledger.ID,
		Name:                  ledger.Name,
		CreatedAt:             ledger.CreatedAt.Format(time.RFC3339),
		InitialSequenceNumber: 0, // New ledgers start with sequence 0
	}, nil
}

func (s *DefaultService) DeleteLedger(ctx context.Context, userID, ledgerID string) error {
	// Check if ledger exists
	ledger, err := s.repo.GetLedger(ctx, ledgerID)
	if err != nil {
		return fmt.Errorf("error getting ledger: %w", err)
	}

	if ledger == nil {
		return errors.New("ledger not found")
	}

	// Check if user has permission to delete the ledger (must be the creator)
	if ledger.CreatedBy != userID {
		return errors.New("you don't have permission to delete this ledger")
	}

	// Delete the ledger
	if err := s.repo.DeleteLedger(ctx, ledgerID); err != nil {
		return fmt.Errorf("error deleting ledger: %w", err)
	}

	return nil
}

// Ledger changes
func (s *DefaultService) SubmitLedgerChange(
	ctx context.Context,
	userID string,
	ledgerID string,
	req models.LedgerChangeRequest,
) (*models.LedgerChangeResponse, error) {
	// Check if user has write permission
	hasAccess, err := s.repo.CheckLedgerAccess(ctx, ledgerID, userID, "write")
	if err != nil {
		return nil, fmt.Errorf("error checking ledger access: %w", err)
	}

	if !hasAccess {
		return nil, errors.New("you don't have write permission for this ledger")
	}

	// Get the latest sequence number
	latestSeq, err := s.repo.GetLatestSequenceNumber(ctx, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("error getting latest sequence number: %w", err)
	}

	// Create the ledger change
	change := &models.LedgerChange{
		ID:              uuid.New().String(),
		LedgerID:        ledgerID,
		UserID:          userID,
		SQLStatement:    req.SQLStatement,
		BaseSequenceNum: latestSeq, // Use the latest sequence as base
		Timestamp:       time.Now().UTC(),
		SequenceNumber:  latestSeq + 1, // Increment by 1
	}

	// Add the change with the pre-assigned sequence number
	if err := s.repo.AddLedgerChange(ctx, change); err != nil {
		return nil, fmt.Errorf("error adding ledger change: %w", err)
	}

	return &models.LedgerChangeResponse{
		Status:                 "success",
		AssignedSequenceNumber: change.SequenceNumber,
		Timestamp:              change.Timestamp.Format(time.RFC3339),
	}, nil
}

func (s *DefaultService) GetLedgerChanges(
	ctx context.Context,
	userID string,
	ledgerID string,
	fromSeq int64,
	toSeq int64,
) (*models.GetLedgerChangesResponse, error) {
	// Check if user has read permission
	hasAccess, err := s.repo.CheckLedgerAccess(ctx, ledgerID, userID, "read")
	if err != nil {
		return nil, fmt.Errorf("error checking ledger access: %w", err)
	}

	if !hasAccess {
		return nil, errors.New("you don't have access to this ledger")
	}

	// Get the changes
	changes, err := s.repo.GetLedgerChangesBySequenceRange(ctx, ledgerID, fromSeq, toSeq)
	if err != nil {
		return nil, fmt.Errorf("error getting ledger changes: %w", err)
	}

	// Get the latest sequence number
	latestSeq, err := s.repo.GetLatestSequenceNumber(ctx, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("error getting latest sequence number: %w", err)
	}

	return &models.GetLedgerChangesResponse{
		Status:               "success",
		LedgerID:             ledgerID,
		Changes:              changes,
		LatestSequenceNumber: latestSeq,
	}, nil
}

// Ledger sharing
func (s *DefaultService) AddUserToLedger(
	ctx context.Context,
	userID string,
	ledgerID string,
	req models.AddUserToLedgerRequest,
) (*models.AddUserResponse, error) {
	// Check if the requesting user has write permission
	hasAccess, err := s.repo.CheckLedgerAccess(ctx, ledgerID, userID, "write")
	if err != nil {
		return nil, fmt.Errorf("error checking ledger access: %w", err)
	}

	if !hasAccess {
		return nil, errors.New("you don't have permission to add users to this ledger")
	}

	// Get the user to add by email
	userToAdd, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	if userToAdd == nil {
		return nil, errors.New("user not found")
	}

	// Create the ledger user relationship
	ledgerUser := &models.LedgerUser{
		LedgerID:    ledgerID,
		UserID:      userToAdd.ID,
		Permissions: req.Permissions,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.repo.AddUserToLedger(ctx, ledgerUser); err != nil {
		return nil, fmt.Errorf("error adding user to ledger: %w", err)
	}

	return &models.AddUserResponse{
		Status:      "success",
		Message:     "User added to ledger successfully",
		UserID:      userToAdd.ID,
		Email:       userToAdd.Email,
		Permissions: req.Permissions,
	}, nil
}

// GetLatestSequenceNumber retrieves the latest sequence number for a ledger
func (s *DefaultService) GetLatestSequenceNumber(
	ctx context.Context,
	userID string,
	ledgerID string,
) (*models.SequenceNumberResponse, error) {
	// Check if user has read permission
	hasAccess, err := s.repo.CheckLedgerAccess(ctx, ledgerID, userID, "read")
	if err != nil {
		return nil, fmt.Errorf("error checking ledger access: %w", err)
	}

	if !hasAccess {
		return nil, errors.New("you don't have access to this ledger")
	}

	// Get the latest sequence number
	latestSeq, err := s.repo.GetLatestSequenceNumber(ctx, ledgerID)
	if err != nil {
		return nil, fmt.Errorf("error getting latest sequence number: %w", err)
	}

	return &models.SequenceNumberResponse{
		Status:               "success",
		LedgerID:             ledgerID,
		LatestSequenceNumber: latestSeq,
	}, nil
}

// Helper methods
func (s *DefaultService) generateJWT(user *models.User) (string, error) {
	expirationTime := time.Now().Add(s.tokenDuration)

	claims := jwt.MapClaims{
		"sub": user.ID, // subject
		"exp": expirationTime.Unix(),
		"iat": time.Now().Unix(), // issued at
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
