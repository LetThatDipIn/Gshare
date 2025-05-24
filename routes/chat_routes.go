package routes

import (
	"assistdeck/controllers"
	"assistdeck/utils"

	"github.com/gin-gonic/gin"
)

func SetupChatRoutes(router *gin.Engine, chatController *controllers.ChatController, wsManager *utils.Manager, authMiddleware gin.HandlerFunc) {
	chatGroup := router.Group("/api/chat")
	chatGroup.Use(authMiddleware)
	{
		chatGroup.POST("/sessions", chatController.CreateSession)
		chatGroup.GET("/sessions", chatController.GetSessions)
		chatGroup.GET("/sessions/:id", chatController.GetSession)
		chatGroup.POST("/sessions/:id/messages", chatController.SaveMessage)
		chatGroup.GET("/sessions/:id/messages", chatController.GetMessages)

		// Participant management
		chatGroup.POST("/sessions/:id/participants", chatController.AddParticipant)
		chatGroup.GET("/sessions/:id/participants", chatController.GetParticipants)
		chatGroup.DELETE("/sessions/:id/participants/:userId", chatController.RemoveParticipant)
	}

	// WebSocket endpoint
	router.GET("/ws", authMiddleware, wsManager.ServeWs)
}
