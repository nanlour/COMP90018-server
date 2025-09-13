package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rongwang/COMP90018-server/internal/api"
	"github.com/rongwang/COMP90018-server/internal/config"
	"github.com/rongwang/COMP90018-server/internal/repository"
	"github.com/rongwang/COMP90018-server/internal/service"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Set up database connection
	db, err := config.SetupDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to set up database: %v", err)
	}
	defer db.Close()

	// Create repository
	repo := repository.NewPostgresRepository(db)

	// Create service
	svc := service.NewDefaultService(repo, cfg.Auth.JWTSecret)

	// Create API handler
	handler := api.NewHandler(svc)

	// Set up Gin router
	router := gin.Default()

	// Add middleware for JWT secret
	router.Use(func(c *gin.Context) {
		c.Set("jwtSecret", []byte(cfg.Auth.JWTSecret))
		c.Next()
	})

	// Set up routes
	handler.SetupRoutes(router)

	// Start server
	serverAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting server on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
