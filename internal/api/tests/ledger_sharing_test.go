package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/rongwang/COMP90018-server/internal/api/testutils"
	"github.com/rongwang/COMP90018-server/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestLedgerSharing(t *testing.T) {
	testCtx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(testCtx)

	// First, create a test ledger
	createLedgerReq := models.CreateLedgerRequest{
		Name:        "Shared Ledger",
		Description: "A test ledger for sharing tests",
		Currency:    "USD",
	}

	w := testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/ledgers",
		createLedgerReq,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	var ledgerResponse models.LedgerResponse
	err := json.Unmarshal(w.Body.Bytes(), &ledgerResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, ledgerResponse.LedgerID)
	ledgerID := ledgerResponse.LedgerID

	// Create another user to share with
	signupReq := models.SignUpRequest{
		Email:    "shareuser@example.com",
		Password: "Password123",
		Name:     "Share User",
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/auth/signup",
		signupReq,
		nil,
	)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Test adding user to ledger
	shareReq := models.AddUserToLedgerRequest{
		Email:       "shareuser@example.com",
		Permissions: "read",
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		fmt.Sprintf("/api/ledgers/%s/users", ledgerID),
		shareReq,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var shareResponse models.AddUserResponse
	err = json.Unmarshal(w.Body.Bytes(), &shareResponse)
	assert.NoError(t, err)
	assert.Equal(t, "success", shareResponse.Status)
	assert.Equal(t, "shareuser@example.com", shareResponse.Email)
	assert.Equal(t, "read", shareResponse.Permissions)

	// Login as the shared user
	loginReq := models.LoginRequest{
		Email:    "shareuser@example.com",
		Password: "Password123",
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/auth/login",
		loginReq,
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var loginResponse models.AuthResponse
	err = json.Unmarshal(w.Body.Bytes(), &loginResponse)
	assert.NoError(t, err)
	sharedUserToken := loginResponse.Token

	// Test that the shared user can access the ledger's changes
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodGet,
		fmt.Sprintf("/api/ledgers/%s/sequence", ledgerID),
		nil,
		testutils.AuthHeaders(sharedUserToken),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test that the shared user with read permission cannot submit changes
	changeReq := models.LedgerChangeRequest{
		SQLStatement: "INSERT INTO entries (id, amount, description) VALUES ('unauthorized', 100.00, 'Unauthorized')",
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		fmt.Sprintf("/api/ledgers/%s/changes", ledgerID),
		changeReq,
		testutils.AuthHeaders(sharedUserToken),
	)

	assert.Equal(t, http.StatusForbidden, w.Code)

	// Update permissions to write
	shareReq.Permissions = "write"
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		fmt.Sprintf("/api/ledgers/%s/users", ledgerID),
		shareReq,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	// Now shared user should be able to submit changes
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		fmt.Sprintf("/api/ledgers/%s/changes", ledgerID),
		changeReq,
		testutils.AuthHeaders(sharedUserToken),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test sharing with non-existent user
	invalidShareReq := models.AddUserToLedgerRequest{
		Email:       "nonexistent@example.com",
		Permissions: "read",
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		fmt.Sprintf("/api/ledgers/%s/users", ledgerID),
		invalidShareReq,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
