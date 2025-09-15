package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/rongwang/COMP90018-server/internal/api/testutils"
	"github.com/rongwang/COMP90018-server/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentLedgerChanges(t *testing.T) {
	testCtx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(testCtx)

	// Create a test ledger
	createLedgerReq := models.CreateLedgerRequest{
		Name:        "Concurrent Test Ledger",
		Description: "A test ledger for concurrent changes",
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

	// Test concurrent changes
	t.Run("TestConcurrentSequenceAssignment", func(t *testing.T) {
		const numGoroutines = 10
		const changesPerGoroutine = 5

		// Channel to collect responses
		responsesChan := make(chan models.LedgerChangeResponse, numGoroutines*changesPerGoroutine)
		var wg sync.WaitGroup

		// Start multiple goroutines to submit changes simultaneously
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()

				for j := 0; j < changesPerGoroutine; j++ {
					changeReq := models.LedgerChangeRequest{
						SQLStatement: fmt.Sprintf(
							"INSERT INTO entries (id, amount, description) VALUES ('entry_r%d_%d', %f, 'Concurrent Entry %d_%d')",
							routineID, j, 100.0+float64(routineID*changesPerGoroutine+j), routineID, j,
						),
					}

					w := testutils.PerformRequest(
						testCtx.Router,
						http.MethodPost,
						fmt.Sprintf("/api/ledgers/%s/changes", ledgerID),
						changeReq,
						testutils.AuthHeaders(testCtx.TestUserJWT),
					)

					assert.Equal(t, http.StatusOK, w.Code)

					var changeResponse models.LedgerChangeResponse
					err := json.Unmarshal(w.Body.Bytes(), &changeResponse)
					assert.NoError(t, err)

					responsesChan <- changeResponse
				}
			}(i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		close(responsesChan)

		// Collect all sequence numbers
		var sequences []int64
		for response := range responsesChan {
			sequences = append(sequences, response.AssignedSequenceNumber)
		}

		// Verify sequence number properties
		assert.Equal(t, numGoroutines*changesPerGoroutine, len(sequences),
			"Should have received all sequence numbers")

		// Sort sequences to check for gaps
		sorted := make([]int64, len(sequences))
		copy(sorted, sequences)
		for i := 0; i < len(sorted)-1; i++ {
			assert.Equal(t, int64(1), sorted[i+1]-sorted[i],
				"Sequence numbers should be continuous without gaps")
		}
	})

	// Test base sequence number handling
	t.Run("TestBaseSequenceHandling", func(t *testing.T) {
		const numGoroutines = 5
		var wg sync.WaitGroup
		var responses []models.LedgerChangeResponse

		// Start goroutines that wait briefly to ensure they read the same base sequence
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				// Small delay to increase chance of concurrent execution
				time.Sleep(time.Millisecond * time.Duration(i*10))

				changeReq := models.LedgerChangeRequest{
					SQLStatement: fmt.Sprintf(
						"INSERT INTO entries (id, amount) VALUES ('concurrent_entry_%d', %f)",
						i, 100.0+float64(i),
					),
				}

				w := testutils.PerformRequest(
					testCtx.Router,
					http.MethodPost,
					fmt.Sprintf("/api/ledgers/%s/changes", ledgerID),
					changeReq,
					testutils.AuthHeaders(testCtx.TestUserJWT),
				)

				assert.Equal(t, http.StatusOK, w.Code)

				var changeResponse models.LedgerChangeResponse
				err := json.Unmarshal(w.Body.Bytes(), &changeResponse)
				assert.NoError(t, err)

				responses = append(responses, changeResponse)
			}(i)
		}

		wg.Wait()

		// Get all changes to verify base sequence numbers
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

		// Verify that base sequence numbers are properly handled
		changes := changesResponse.Changes
		assert.GreaterOrEqual(t, len(changes), numGoroutines,
			"Should have received all concurrent changes")

		// Look at the last numGoroutines changes
		recentChanges := changes[len(changes)-numGoroutines:]
		for i, change := range recentChanges {
			// Each change's base sequence should be less than its own sequence number
			assert.Less(t, change.BaseSequenceNum, change.SequenceNumber,
				"Base sequence number should be less than the change's sequence number")

			// Base sequence number should be increasing
			if i > 0 {
				assert.GreaterOrEqual(t, change.BaseSequenceNum, recentChanges[i-1].BaseSequenceNum,
					"Base sequence numbers should be non-decreasing")
			}
		}
	})

	// Test sequence number retrieval during concurrent changes
	t.Run("TestSequenceNumberDuringConcurrency", func(t *testing.T) {
		// Start multiple goroutines to submit changes
		const numChanges = 5
		var wg sync.WaitGroup

		// Start a goroutine that continuously checks the sequence number
		sequenceCheckDone := make(chan bool)
		var lastSeqNum int64
		var seqNumGaps bool

		go func() {
			defer close(sequenceCheckDone)
			for {
				select {
				case <-sequenceCheckDone:
					return
				default:
					w := testutils.PerformRequest(
						testCtx.Router,
						http.MethodGet,
						fmt.Sprintf("/api/ledgers/%s/sequence", ledgerID),
						nil,
						testutils.AuthHeaders(testCtx.TestUserJWT),
					)

					if w.Code != http.StatusOK {
						continue
					}

					var seqResponse models.SequenceNumberResponse
					if err := json.Unmarshal(w.Body.Bytes(), &seqResponse); err != nil {
						continue
					}

					if lastSeqNum > 0 && seqResponse.LatestSequenceNumber-lastSeqNum > 1 {
						seqNumGaps = true
					}
					lastSeqNum = seqResponse.LatestSequenceNumber
					time.Sleep(time.Millisecond * 10) // Small delay between checks
				}
			}
		}()

		// Submit changes concurrently
		for i := 0; i < numChanges; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				changeReq := models.LedgerChangeRequest{
					SQLStatement: fmt.Sprintf("INSERT INTO entries (id) VALUES ('seq_check_%d')", i),
				}

				w := testutils.PerformRequest(
					testCtx.Router,
					http.MethodPost,
					fmt.Sprintf("/api/ledgers/%s/changes", ledgerID),
					changeReq,
					testutils.AuthHeaders(testCtx.TestUserJWT),
				)

				assert.Equal(t, http.StatusOK, w.Code)
			}(i)
		}

		wg.Wait()
		sequenceCheckDone <- true
		<-sequenceCheckDone

		assert.False(t, seqNumGaps, "Should not have gaps in sequence numbers during concurrent operations")
	})
}
