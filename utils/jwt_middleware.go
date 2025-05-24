package utils

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTMiddleware creates a middleware that verifies the JWT token in the Authorization header
func JWTMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// First check query param for token (for WebSocket connections)
		tokenParam := c.Query("token")
		if tokenParam != "" {
			tokenString = tokenParam
		} else {
			// If not in query, check Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
				c.Abort()
				return
			}

			// Extract the token from the header
			// Format should be "Bearer <token>"
			bearerToken := strings.Split(authHeader, " ")
			if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format. Use 'Bearer <token>'"})
				c.Abort()
				return
			}

			tokenString = bearerToken[1]
		}

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// Return the secret key
			return []byte(jwtSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Check expiration (jwt.Parse already checks this, but just to be safe)
			if claims["exp"] != nil {
				// Get user ID from either "user_id" (old format) or "sub" (new format)
				var userID interface{}
				if claims["user_id"] != nil {
					userID = claims["user_id"]
				} else if claims["sub"] != nil {
					// Convert string UUID to actual UUID
					userIDStr, ok := claims["sub"].(string)
					if !ok {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID type in token"})
						c.Abort()
						return
					}

					parsedID, err := uuid.Parse(userIDStr)
					if err != nil {
						c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format in token: " + err.Error()})
						c.Abort()
						return
					}
					userID = parsedID
				} else {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Token missing user identifier"})
					c.Abort()
					return
				}

				// Extract user role if present
				if claims["role"] != nil {
					c.Set("userRole", claims["role"])
				} else {
					// Default to "user" role if not specified
					c.Set("userRole", "user")
				}

				// Log the user ID for debugging
				fmt.Printf("Setting userID in context: %v (type: %T)\n", userID, userID)

				// Set user ID in context for later use
				c.Set("userID", userID)  // New controller format
				c.Set("user_id", userID) // Legacy controller format
				c.Next()
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token missing expiration claim"})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}
	}
}
