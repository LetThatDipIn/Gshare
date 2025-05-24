package routes

import (
	"assistdeck/controllers"
	"assistdeck/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func UserRoutes(r *gin.Engine, db *gorm.DB) {
	// User routes - all protected by auth middleware
	userGroup := r.Group("/user")
	userGroup.Use(utils.AuthMiddleware())
	{
		userGroup.GET("/profile", controllers.GetUserProfile)
		userGroup.PUT("/role", controllers.UpdateUserRole)
	}
}
