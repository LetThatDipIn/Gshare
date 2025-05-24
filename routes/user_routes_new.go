package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

func SetupUserRoutes(router *gin.Engine, userController *controllers.UserController, authMiddleware gin.HandlerFunc) {
	// User routes - all protected by auth middleware
	userGroup := router.Group("/api/users")
	userGroup.Use(authMiddleware)
	{
		userGroup.GET("/profile", userController.GetProfile)
		userGroup.PUT("/role", userController.UpdateRole)
	}
}
