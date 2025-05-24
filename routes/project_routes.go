package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

func SetupProjectRoutes(router *gin.Engine, projectController *controllers.ProjectController, authMiddleware gin.HandlerFunc) {
	projectGroup := router.Group("/api/projects")
	projectGroup.Use(authMiddleware)
	{
		projectGroup.POST("", projectController.CreateProject)
		projectGroup.GET("", projectController.GetProjects)
		projectGroup.GET("/:id", projectController.GetProject)
		projectGroup.PUT("/:id", projectController.UpdateProject)
		projectGroup.DELETE("/:id", projectController.DeleteProject)
	}
}
