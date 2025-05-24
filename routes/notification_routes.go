package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

func SetupNotificationRoutes(router *gin.Engine, notificationController *controllers.NotificationController, authMiddleware gin.HandlerFunc) {
	notificationGroup := router.Group("/api/notifications")
	notificationGroup.Use(authMiddleware)
	{
		notificationGroup.GET("", notificationController.GetNotifications)
		notificationGroup.GET("/count", notificationController.GetUnreadCount)
		notificationGroup.PUT("/:id/read", notificationController.MarkAsRead)
		notificationGroup.PUT("/read-all", notificationController.MarkAllAsRead)
		notificationGroup.DELETE("/:id", notificationController.DeleteNotification)
		notificationGroup.DELETE("", notificationController.DeleteAllNotifications)
	}
}
