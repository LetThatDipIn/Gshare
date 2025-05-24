package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

func SetupGoalRoutes(router *gin.Engine, goalController *controllers.GoalController, authMiddleware gin.HandlerFunc) {
	goalGroup := router.Group("/api/goals")
	goalGroup.Use(authMiddleware)
	{
		goalGroup.POST("", goalController.CreateGoal)
		goalGroup.GET("", goalController.GetGoals)
		goalGroup.GET("/:id", goalController.GetGoal)
		goalGroup.PUT("/:id", goalController.UpdateGoal)
		goalGroup.DELETE("/:id", goalController.DeleteGoal)
	}
}
