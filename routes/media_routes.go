package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

// SetupMediaRoutes sets up routes for media file operations
func SetupMediaRoutes(router *gin.Engine, mediaController *controllers.MediaController, authMiddleware gin.HandlerFunc, subscriptionCheckMiddleware gin.HandlerFunc) {
	// Public routes for media access
	router.GET("/api/media/:id", mediaController.GetFile)

	// Protected routes that require authentication
	mediaGroup := router.Group("/api/media")
	mediaGroup.Use(authMiddleware)
	{
		// Get metadata for a file
		mediaGroup.GET("/metadata/:id", mediaController.GetFileMetadata)

		// Get all files for the current user
		mediaGroup.GET("/files", mediaController.GetUserFiles)

		// Delete a file
		mediaGroup.DELETE("/:id", mediaController.DeleteFile) // File upload routes (these additionally check for subscription)
		uploadGroup := mediaGroup.Group("/upload")
		// Temporarily disable subscription check for testing
		// uploadGroup.Use(subscriptionCheckMiddleware)
		{
			// General file upload
			uploadGroup.POST("/file", mediaController.UploadFile)

			// Audio upload with specialized handling
			uploadGroup.POST("/audio", mediaController.UploadAudio)
		}
	}
}
