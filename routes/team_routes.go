package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
)

func SetupTeamRoutes(router *gin.Engine, teamController *controllers.TeamController, authMiddleware gin.HandlerFunc) {
	teamGroup := router.Group("/api/teams")
	teamGroup.Use(authMiddleware)
	{
		teamGroup.POST("", teamController.CreateTeam)
		teamGroup.GET("", teamController.GetTeams)
		teamGroup.GET("/:id", teamController.GetTeam)
		teamGroup.POST("/:id/members", teamController.AddMember)
		teamGroup.DELETE("/:id/members/:userId", teamController.RemoveMember)
	}
}
