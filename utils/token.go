package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateJWT(userID interface{}) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// GetEnv retrieves an environment variable or returns a default value if not found
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// GetUserIDFromContext extracts the user ID from the Gin context
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		return uuid.UUID{}, fmt.Errorf("user ID not found in context")
	}

	// Handle different types of userID (could be string or uuid.UUID)
	switch userID := userIDVal.(type) {
	case uuid.UUID:
		return userID, nil
	case string:
		return uuid.Parse(userID)
	default:
		// Try to convert to string and then parse
		userIDStr, ok := interface{}(userID).(string)
		if !ok {
			return uuid.UUID{}, fmt.Errorf("user ID is of unknown type: %T", userID)
		}
		return uuid.Parse(userIDStr)
	}
}
