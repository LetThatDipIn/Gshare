package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

func SetupCalendarRoutes(router *gin.Engine, calendarController *controllers.CalendarController, authMiddleware gin.HandlerFunc) {
	calendarGroup := router.Group("/api/calendar")
	calendarGroup.Use(authMiddleware)
	{
		// Event management
		calendarGroup.POST("/events", calendarController.CreateEvent)
		calendarGroup.GET("/events", calendarController.GetEvents)
		calendarGroup.GET("/events/:id", calendarController.GetEvent)
		calendarGroup.PUT("/events/:id", calendarController.UpdateEvent)
		calendarGroup.DELETE("/events/:id", calendarController.DeleteEvent)

		// Google Calendar integration
		calendarGroup.GET("/google/auth-url", calendarController.GetGoogleAuthURL)
		calendarGroup.GET("/google/status", calendarController.GetGoogleConnectionStatus)
		calendarGroup.POST("/google/sync", calendarController.SyncWithGoogle)
		calendarGroup.DELETE("/google/disconnect", calendarController.DisconnectGoogle)
	}

	// Google OAuth callback (no auth required)
	router.GET("/api/calendar/auth/callback", calendarController.HandleGoogleAuthCallback)
}
