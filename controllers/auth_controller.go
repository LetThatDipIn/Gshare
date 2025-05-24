package controllers

import (
	"assistdeck/models"
	"assistdeck/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// Move this outside of init() to ensure it's properly initialized
var googleOauthConfig *oauth2.Config

// Initialize the config in a function that can be called from main
func InitGoogleOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	fmt.Println("=== Google OAuth Debug Info ===")
	fmt.Printf("Client ID: %s\n", clientID)

	// Only try to substring if clientSecret has enough characters
	if len(clientSecret) > 4 {
		fmt.Printf("Client Secret: %s\n", clientSecret[:4]+"...")
	} else {
		fmt.Printf("Client Secret: %s\n", clientSecret)
	}

	fmt.Printf("Redirect URL: %s\n", redirectURL)

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		fmt.Println("WARNING: Missing Google OAuth configuration!")
	}

	googleOauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func GoogleLogin(c *gin.Context) {
	// Verify config is initialized
	if googleOauthConfig == nil {
		fmt.Println("ERROR: Google OAuth config is nil!")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth configuration error"})
		return
	}

	// Debug info
	fmt.Println("GoogleLogin called, redirecting to:", googleOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline))

	url := googleOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GoogleCallback(c *gin.Context) {
	// Verify config is initialized
	if googleOauthConfig == nil {
		fmt.Println("ERROR: Google OAuth config is nil in callback!")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth configuration error"})
		return
	}

	// Debug info
	fmt.Println("Callback received with code:", c.Query("code"))
	fmt.Println("Using client ID:", googleOauthConfig.ClientID)
	fmt.Println("Using redirect URL:", googleOauthConfig.RedirectURL)

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code is missing"})
		return
	}

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Println("Token exchange error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token", "details": err.Error()})
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Decode error", "details": err.Error()})
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	var user models.User
	result := db.First(&user, "email = ?", userInfo.Email)

	if result.Error != nil {
		user = models.User{
			ID:    uuid.New(),
			Name:  userInfo.Name,
			Email: userInfo.Email,
			Plan:  "trial",
			Role:  "",
		}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "details": err.Error()})
			return
		}
	}

	tokenStr, err := utils.GenerateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenStr,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
			"plan":  user.Plan,
		},
	})
}
