package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rongwang/COMP90018-server/internal/models"
)

// AuthMiddleware returns a Gin middleware for authentication
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the JWT token from the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Status:  "error",
				Code:    "UNAUTHORIZED",
				Message: "Authentication required",
			})
			c.Abort()
			return
		}

		// Check if the Authorization header starts with "Bearer "
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Status:  "error",
				Code:    "UNAUTHORIZED",
				Message: "Invalid token format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse the JWT token
		jwtSecret := c.MustGet("jwtSecret").([]byte)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("invalid signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Status:  "error",
				Code:    "UNAUTHORIZED",
				Message: "Invalid token",
			})
			c.Abort()
			return
		}

		// Extract claims from the token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Status:  "error",
				Code:    "UNAUTHORIZED",
				Message: "Invalid token claims",
			})
			c.Abort()
			return
		}

		// Get user ID from the token claims
		userID, ok := claims["sub"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Status:  "error",
				Code:    "UNAUTHORIZED",
				Message: "Invalid user ID in token",
			})
			c.Abort()
			return
		}

		// Set user ID in the context
		c.Set("userId", userID)
		c.Next()
	}
}
