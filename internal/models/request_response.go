package models

// Request models
type SignUpRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateLedgerRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Currency    string `json:"currency" binding:"required"`
}

type LedgerChangeRequest struct {
	SQLStatement string `json:"sqlStatement" binding:"required"`
}

type AddUserToLedgerRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Permissions string `json:"permissions" binding:"required,oneof=read write"`
}

// Response models
type AuthResponse struct {
	Status    string `json:"status"`
	UserID    string `json:"userId,omitempty"`
	Email     string `json:"email,omitempty"`
	Name      string `json:"name,omitempty"`
	Token     string `json:"token,omitempty"`
	ExpiresIn int    `json:"expiresIn,omitempty"`
}

type LedgerResponse struct {
	Status                string `json:"status"`
	LedgerID              string `json:"ledgerId,omitempty"`
	Name                  string `json:"name,omitempty"`
	CreatedAt             string `json:"createdAt,omitempty"`
	InitialSequenceNumber int64  `json:"initialSequenceNumber,omitempty"`
}

type LedgerChangeResponse struct {
	Status                 string `json:"status"`
	AssignedSequenceNumber int64  `json:"assignedSequenceNumber,omitempty"`
	Timestamp              string `json:"timestamp,omitempty"`
}

type GetLedgerChangesResponse struct {
	Status               string         `json:"status"`
	LedgerID             string         `json:"ledgerId"`
	Changes              []LedgerChange `json:"changes"`
	LatestSequenceNumber int64          `json:"latestSequenceNumber"`
}

type AddUserResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	UserID      string `json:"userId,omitempty"`
	Email       string `json:"email,omitempty"`
	Permissions string `json:"permissions,omitempty"`
}

type SequenceNumberResponse struct {
	Status               string `json:"status"`
	LedgerID             string `json:"ledgerId"`
	LatestSequenceNumber int64  `json:"latestSequenceNumber"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
