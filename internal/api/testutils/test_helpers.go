package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rongwang/COMP90018-server/internal/api"
	"github.com/rongwang/COMP90018-server/internal/config"
	"github.com/rongwang/COMP90018-server/internal/models"
	"github.com/rongwang/COMP90018-server/internal/repository"
	"github.com/rongwang/COMP90018-server/internal/service"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// TestContext holds all dependencies for tests
type TestContext struct {
	Router      *gin.Engine
	Repository  repository.Repository
	Service     service.Service
	JWTSecret   []byte
	DB          *sqlx.DB
	TestUserID  string
	TestUserJWT string
}

// SetupTestContext creates a new test context with initialized dependencies
func SetupTestContext(t *testing.T) *TestContext {
	// Load configuration from environment
	cfg := config.LoadConfig()

	// Override with test-specific config
	if cfg.Database.DBName == "billapp" && cfg.Database.TestDBName != "" {
		cfg.Database.DBName = cfg.Database.TestDBName
	} else if cfg.Database.TestDBName == "" {
		// Fallback to hardcoded test DB if not in environment
		cfg.Database.DBName = "billapp_test"
	}

	// Use a test JWT secret
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "test-secret-key"
	}

	// Set up database
	db, err := config.SetupDatabase(cfg)
	assert.NoError(t, err, "Failed to set up test database")

	// Create repository
	repo := repository.NewPostgresRepository(db)

	// Create service
	svc := service.NewDefaultService(repo, cfg.Auth.JWTSecret)

	// Create API handler
	handler := api.NewHandler(svc)

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Add middleware for JWT secret
	router.Use(func(c *gin.Context) {
		c.Set("jwtSecret", []byte(cfg.Auth.JWTSecret))
		c.Next()
	})

	// Set up routes
	handler.SetupRoutes(router)

	// Create test user if needed
	testUserID, token := createTestUser(t, repo, cfg.Auth.JWTSecret)

	return &TestContext{
		Router:      router,
		Repository:  repo,
		Service:     svc,
		JWTSecret:   []byte(cfg.Auth.JWTSecret),
		DB:          db,
		TestUserID:  testUserID,
		TestUserJWT: token,
	}
}

// CleanupTestContext cleans up test resources
func CleanupTestContext(t *TestContext) {
	// Clean up database
	if t.DB != nil {
		cleanupTestDatabase(nil, t.Repository)
		t.DB.Close()
	}
}

// cleanupTestDatabase removes any existing test users and data
func cleanupTestDatabase(t *testing.T, repo repository.Repository) {
	// Execute cleanup SQL directly through the DB connection
	if pgRepo, ok := repo.(*repository.PostgresRepository); ok {
		db := pgRepo.GetDB()

		// Delete all ledger_changes
		_, err := db.Exec("DELETE FROM ledger_changes")
		if t != nil && err != nil {
			t.Logf("Warning: Failed to clean ledger_changes: %v", err)
		}

		// Delete all ledger_users
		_, err = db.Exec("DELETE FROM ledger_users")
		if t != nil && err != nil {
			t.Logf("Warning: Failed to clean ledger_users: %v", err)
		}

		// Delete all ledgers
		_, err = db.Exec("DELETE FROM ledgers")
		if t != nil && err != nil {
			t.Logf("Warning: Failed to clean ledgers: %v", err)
		}

		// Delete all users
		_, err = db.Exec("DELETE FROM users")
		if t != nil && err != nil {
			t.Logf("Warning: Failed to clean users: %v", err)
		}
	}
}

// Helper functions
func createTestUser(t *testing.T, repo repository.Repository, jwtSecret string) (string, string) {
	// Clean up any existing test users first
	cleanupTestDatabase(t, repo)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)

	user := &models.User{
		ID:        uuid.New().String(),
		Email:     "testuser@example.com",
		Name:      "Test User",
		Password:  string(hashedPassword),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	err := repo.CreateUser(context.Background(), user)
	assert.NoError(t, err, "Failed to create test user")

	// Generate JWT token with the provided secret key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(jwtSecret))
	assert.NoError(t, err, "Failed to generate JWT token")

	return user.ID, tokenString
}

// PerformRequest executes an HTTP request against the router
func PerformRequest(r http.Handler, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody *bytes.Buffer

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// AuthHeaders returns headers with Authorization token
func AuthHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
}
