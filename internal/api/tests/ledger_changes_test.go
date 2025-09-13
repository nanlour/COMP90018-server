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

func TestLedgerChanges(t *testing.T) {
	testCtx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(testCtx)

	// First, create a test ledger
	createLedgerReq := models.CreateLedgerRequest{
		Name:        "Test Changes Ledger",
		Description: "A test ledger for testing changes",
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

	// Test Get Latest Sequence Number - Initially should be 0
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodGet,
		fmt.Sprintf("/api/ledgers/%s/sequence", ledgerID),
		nil,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var seqResponse models.SequenceNumberResponse
	err = json.Unmarshal(w.Body.Bytes(), &seqResponse)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), seqResponse.LatestSequenceNumber)

	// Test Submit Ledger Change
	changeReq := models.LedgerChangeRequest{
		SQLStatement: "INSERT INTO entries (id, amount, description) VALUES ('entry1', 100.50, 'Test Entry')",
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		fmt.Sprintf("/api/ledgers/%s/changes", ledgerID),
		changeReq,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var changeResponse models.LedgerChangeResponse
	err = json.Unmarshal(w.Body.Bytes(), &changeResponse)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), changeResponse.AssignedSequenceNumber)

	// Check that sequence number is updated
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodGet,
		fmt.Sprintf("/api/ledgers/%s/sequence", ledgerID),
		nil,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	err = json.Unmarshal(w.Body.Bytes(), &seqResponse)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), seqResponse.LatestSequenceNumber)

	// Test Get Ledger Changes
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodGet,
		fmt.Sprintf("/api/ledgers/%s/changes?fromSequence=0", ledgerID),
		nil,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	var changesResponse models.GetLedgerChangesResponse
	err = json.Unmarshal(w.Body.Bytes(), &changesResponse)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(changesResponse.Changes))
	assert.Equal(t, "INSERT INTO entries (id, amount, description) VALUES ('entry1', 100.50, 'Test Entry')",
		changesResponse.Changes[0].SQLStatement)

	// Submit multiple changes
	for i := 0; i < 3; i++ {
		changeReq := models.LedgerChangeRequest{
			SQLStatement: fmt.Sprintf("INSERT INTO entries (id, amount, description) VALUES ('entry%d', %f, 'Test Entry %d')",
				i+2, 200.50+float64(i)*10, i+2),
		}

		w = testutils.PerformRequest(
			testCtx.Router,
			http.MethodPost,
			fmt.Sprintf("/api/ledgers/%s/changes", ledgerID),
			changeReq,
			testutils.AuthHeaders(testCtx.TestUserJWT),
		)

		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Check final sequence number
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodGet,
		fmt.Sprintf("/api/ledgers/%s/sequence", ledgerID),
		nil,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	err = json.Unmarshal(w.Body.Bytes(), &seqResponse)
	assert.NoError(t, err)
	assert.Equal(t, int64(4), seqResponse.LatestSequenceNumber)

	// Get changes with range
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodGet,
		fmt.Sprintf("/api/ledgers/%s/changes?fromSequence=2&toSequence=3", ledgerID),
		nil,
		testutils.AuthHeaders(testCtx.TestUserJWT),
	)

	assert.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &changesResponse)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(changesResponse.Changes))
	assert.Equal(t, int64(2), changesResponse.Changes[0].SequenceNumber)
	assert.Equal(t, int64(3), changesResponse.Changes[1].SequenceNumber)
}
