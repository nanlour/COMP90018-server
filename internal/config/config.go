package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
}

// ServerConfig holds the server configuration
type ServerConfig struct {
	Port int
}

// DatabaseConfig holds the database configuration
type DatabaseConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	DBName     string
	SSLMode    string
	TestDBName string // Separate database for testing
}

// AuthConfig holds the authentication configuration
type AuthConfig struct {
	JWTSecret string
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.Username, c.Password, c.DBName, c.SSLMode,
	)
}

// LoadConfig loads the configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnvAsInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:       getEnv("DB_HOST", "localhost"),
			Port:       getEnvAsInt("DB_PORT", 5432),
			Username:   getEnv("DB_USERNAME", "postgres"),
			Password:   getEnv("DB_PASSWORD", "password"),
			DBName:     getEnv("DB_NAME", "billapp"),
			SSLMode:    getEnv("DB_SSLMODE", "disable"),
			TestDBName: getEnv("TEST_DB_NAME", "billapp_test"),
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", "your-secret-key-here"),
		},
	}
}

// Helper functions to read environment variables
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
