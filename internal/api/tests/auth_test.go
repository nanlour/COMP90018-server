package api_test

import (
	"net/http"
	"testing"

	"github.com/rongwang/COMP90018-server/internal/api/testutils"
	"github.com/rongwang/COMP90018-server/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestSignup(t *testing.T) {
	testCtx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(testCtx)

	// Test case 1: Successful signup
	signupReq := models.SignUpRequest{
		Email:    "newuser@example.com",
		Password: "Password123",
		Name:     "New User",
	}

	w := testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/auth/signup",
		signupReq,
		nil,
	)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Test case 2: Duplicate email
	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/auth/signup",
		signupReq,
		nil,
	)

	assert.Equal(t, http.StatusConflict, w.Code)

	// Test case 3: Invalid request (missing required fields)
	invalidReq := models.SignUpRequest{
		Email: "invalid@example.com",
		// Missing password and name
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/auth/signup",
		invalidReq,
		nil,
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin(t *testing.T) {
	testCtx := testutils.SetupTestContext(t)
	defer testutils.CleanupTestContext(testCtx)

	// Test case 1: Successful login
	loginReq := models.LoginRequest{
		Email:    "testuser@example.com",
		Password: "testpassword",
	}

	w := testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/auth/login",
		loginReq,
		nil,
	)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test case 2: Invalid credentials
	invalidLoginReq := models.LoginRequest{
		Email:    "testuser@example.com",
		Password: "wrongpassword",
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/auth/login",
		invalidLoginReq,
		nil,
	)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test case 3: User not found
	nonExistentUserReq := models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "testpassword",
	}

	w = testutils.PerformRequest(
		testCtx.Router,
		http.MethodPost,
		"/api/auth/login",
		nonExistentUserReq,
		nil,
	)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
