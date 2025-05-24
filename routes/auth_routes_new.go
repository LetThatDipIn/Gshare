package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

func SetupAuthRoutes(router *gin.Engine, authController *controllers.AuthController) {
	// Define the Google login route
	router.GET("/auth/google/login", authController.GoogleLogin)
	// Define the Google callback route (where Google redirects to after login)
	router.GET("/auth/google/callback", authController.GoogleCallback)
}
