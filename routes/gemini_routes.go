package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

func SetupGeminiRoutes(router *gin.Engine, geminiController *controllers.GeminiController, authMiddleware gin.HandlerFunc) {
	geminiGroup := router.Group("/api/gemini")
	geminiGroup.Use(authMiddleware)
	{
		geminiGroup.POST("/sessions", geminiController.CreateSession)
		geminiGroup.GET("/sessions", geminiController.GetSessions)
		geminiGroup.GET("/sessions/:id", geminiController.GetSession)
		geminiGroup.GET("/sessions/:id/messages", geminiController.GetMessages)
		geminiGroup.POST("/sessions/:id/messages", geminiController.SendMessage)
	}
}
