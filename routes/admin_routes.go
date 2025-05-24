package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

// SetupAdminRoutes configures all the admin routes for the application
func SetupAdminRoutes(router *gin.Engine, adminController *controllers.AdminController, authMiddleware gin.HandlerFunc) {
	adminGroup := router.Group("/admin", authMiddleware)
	{
		// Admin dashboard - use GetUserStats instead of Dashboard which doesn't exist
		adminGroup.GET("/dashboard", adminController.GetUserStats)

		// Admin settings routes
		adminGroup.GET("/settings", adminController.GetSettings)
		adminGroup.GET("/settings/:key", adminController.GetSetting)
		adminGroup.POST("/settings", adminController.SaveSetting)
		adminGroup.DELETE("/settings/:key", adminController.DeleteSetting)

		// User management routes
		adminGroup.GET("/users", adminController.GetAllUsers)
		adminGroup.GET("/users/:id", adminController.GetUserDetails)
		adminGroup.PUT("/users/:id/role", adminController.UpdateUserRole)

		// Stats routes
		adminGroup.GET("/stats/users", adminController.GetUserStats)
	}
}
