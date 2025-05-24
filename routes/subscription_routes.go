package routes

import (
	"assistdeck/controllers"
	"assistdeck/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SubscriptionRoutes(r *gin.Engine, db *gorm.DB) {
	// Subscription routes - all protected by auth middleware
	subGroup := r.Group("/subscription")
	subGroup.Use(utils.AuthMiddleware())
	{
		subGroup.POST("/trial", controllers.StartFreeTrial)
		subGroup.GET("/status", controllers.GetSubscriptionStatus)
	}
}
