package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/rongwang/COMP90018-server/internal/api/testutils"
	"github.com/rongwang/COMP90018-server/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestCreateLedger(t *testing.T) {
	testCtx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(testCtx)

	// Test case 1: Successful ledger creation
	createLedgerReq := models.CreateLedgerRequest{
		Name:        "Test Ledger",
		Description: "A test ledger for unit testing",
		Currency:    "USD",
	}

	w := testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/ledgers",
		createLedgerReq,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Parse response to get ledger ID
	var response models.LedgerResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.LedgerID)
	ledgerID := response.LedgerID

	// Test case 2: Invalid request (missing required fields)
	invalidReq := models.CreateLedgerRequest{
		Name: "Invalid Ledger",
		// Missing currency
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/ledgers",
		invalidReq,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test case 3: Unauthorized request (no token)
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/ledgers",
		createLedgerReq,
		nil,
	)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Return the created ledger ID for use in other tests
	t.Logf("Created test ledger with ID: %s", ledgerID)
}

func TestDeleteLedger(t *testing.T) {
	testCtx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(testCtx)

	// First create a ledger to delete
	createLedgerReq := models.CreateLedgerRequest{
		Name:        "Ledger to Delete",
		Description: "This ledger will be deleted",
		Currency:    "USD",
	}

	w := testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/ledgers",
		createLedgerReq,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	var response models.LedgerResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.LedgerID)
	ledgerID := response.LedgerID

	// Test case 1: Successfully delete the ledger
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodDelete,
		"/api/ledgers/"+ledgerID,
		nil,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test case 2: Delete non-existent ledger
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodDelete,
		"/api/ledgers/non-existent-id",
		nil,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test case 3: Unauthorized request (no token)
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodDelete,
		"/api/ledgers/"+ledgerID,
		nil,
		nil,
	)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
