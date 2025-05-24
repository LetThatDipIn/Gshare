package routes

import (
	"assistdeck/models"
	"assistdeck/utils"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SetupDebugRoutes adds debugging routes that should be disabled in production
func SetupDebugRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	// Get the database connection for creating test users
	var db *gorm.DB
	router.Use(func(c *gin.Context) {
		dbInterface, exists := c.Get("db")
		if exists {
			db = dbInterface.(*gorm.DB)
		}
		c.Next()
	})

	debug := router.Group("/api/debug")
	{
		// Generate a test token - this endpoint doesn't need authentication
		// since it's used for testing
		debug.GET("/token", func(c *gin.Context) {
			// Check if a specific user ID is requested
			userIDParam := c.Query("user_id")

			var userID uuid.UUID
			var err error

			if userIDParam == "" || userIDParam == "owner" {
				// Default user ID for owner (first test user)
				userID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

				// Ensure the test owner user exists in the database
				if db != nil {
					var count int64
					db.Model(&models.User{}).Where("id = ?", userID).Count(&count)
					if count == 0 {
						testOwner := models.User{
							ID:    userID,
							Email: "owner@test.com",
							Name:  "Test Owner",
							Role:  "user",
						}
						db.Create(&testOwner)
					}
				}
			} else if userIDParam == "participant" || userIDParam == "participant1" {
				// Use a different ID for participant
				userID = uuid.MustParse("22222222-2222-2222-2222-222222222222")

				// Ensure the test participant user exists in the database
				if db != nil {
					var count int64
					db.Model(&models.User{}).Where("id = ?", userID).Count(&count)
					if count == 0 {
						testParticipant := models.User{
							ID:    userID,
							Email: "participant@test.com",
							Name:  "Test Participant",
							Role:  "user",
						}
						db.Create(&testParticipant)
					}
				}
			} else if userIDParam == "participant2" {
				// Use a different ID for participant 2
				userID = uuid.MustParse("33333333-3333-3333-3333-333333333333")

				// Ensure the test participant user exists in the database
				if db != nil {
					var count int64
					db.Model(&models.User{}).Where("id = ?", userID).Count(&count)
					if count == 0 {
						testParticipant := models.User{
							ID:    userID,
							Email: "participant2@test.com",
							Name:  "Test Participant 2",
							Role:  "user",
						}
						db.Create(&testParticipant)
					}
				}
			} else {
				// Try to parse the provided ID
				userID, err = uuid.Parse(userIDParam)
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid user_id parameter"})
					return
				}
			}

			// Get the JWT secret from environment or use a default for testing
			jwtSecret := utils.GetEnv("JWT_SECRET", "thisisasecretkeyforjwtauthenticationchangeitduringproduction")

			// Create a token with userID claim
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"sub": userID.String(),
				"iat": time.Now().Unix(),
				"exp": time.Now().Add(time.Hour * 24).Unix(),
			})

			// Sign the token
			tokenString, err := token.SignedString([]byte(jwtSecret))
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to generate token"})
				return
			}

			c.JSON(200, gin.H{
				"token":   tokenString,
				"user_id": userID.String(),
			})
		})

		// The following endpoints should be protected
		secureDebug := debug.Group("/")
		secureDebug.Use(authMiddleware)

		// Test endpoint to see what userID is in the context after JWT authentication
		secureDebug.GET("/user", func(c *gin.Context) {
			userID, exists := c.Get("userID")
			if !exists {
				c.JSON(200, gin.H{
					"error": "No userID found in context",
				})
				return
			}

			// Return the type and value of userID
			c.JSON(200, gin.H{
				"userID":     userID,
				"userIDType": fmt.Sprintf("%T", userID),
			})
		})

		// Verbose token inspection endpoint
		secureDebug.GET("/token/inspect", func(c *gin.Context) {
			// Get the Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(400, gin.H{"error": "Missing Authorization header"})
				return
			}

			// Check if the Authorization header is in the format "Bearer <token>"
			if !strings.HasPrefix(authHeader, "Bearer ") {
				c.JSON(400, gin.H{"error": "Invalid Authorization header format. Expected 'Bearer <token>'"})
				return
			}

			// Extract the token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse the token without verifying the signature
			token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Just return a dummy key since we're not validating the signature
				return []byte("dummy"), nil
			})

			// Extract the claims
			var claims map[string]interface{}
			if token != nil {
				if token.Claims != nil {
					claims = token.Claims.(jwt.MapClaims)
				}
			}

			// Get the userID from the context (this comes from the actual middleware)
			userID, exists := c.Get("userID")
			if !exists {
				userID = nil
			}

			// Response with detailed token information
			c.JSON(200, gin.H{
				"token_provided": tokenString != "",
				"token_header":   authHeader,
				"token_parsed":   token != nil,
				"claims":         claims,
				"context_userID": userID,
				"userIDType":     fmt.Sprintf("%T", userID),
			})
		})
	}
}
