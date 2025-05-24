package controllers

import (
	"assistdeck/services"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

func (c *AuthController) GoogleLogin(ctx *gin.Context) {
	// Verify config is initialized
	if googleOauthConfig == nil {
		fmt.Println("ERROR: Google OAuth config is nil!")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth configuration error"})
		return
	}

	// Debug info
	fmt.Println("GoogleLogin called, redirecting to:", googleOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline))

	url := googleOauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	ctx.Redirect(http.StatusTemporaryRedirect, url)
}

func (c *AuthController) GoogleCallback(ctx *gin.Context) {
	// Verify config is initialized
	if googleOauthConfig == nil {
		fmt.Println("ERROR: Google OAuth config is nil in callback!")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth configuration error"})
		return
	}

	// Debug info
	fmt.Println("Callback received with code:", ctx.Query("code"))
	fmt.Println("Using client ID:", googleOauthConfig.ClientID)
	fmt.Println("Using redirect URL:", googleOauthConfig.RedirectURL)

	code := ctx.Query("code")
	if code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code is missing"})
		return
	}

	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		fmt.Println("Token exchange error:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token", "details": err.Error()})
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Decode error", "details": err.Error()})
		return
	}

	// Find or create user using the service
	user, err := c.authService.FindOrCreateUser(userInfo.Email, userInfo.Name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User creation error", "details": err.Error()})
		return
	}

	// Generate token using the service
	tokenStr, err := c.authService.GenerateToken(user.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token", "details": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
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
