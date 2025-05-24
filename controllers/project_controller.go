package controllers

import (
	"assistdeck/models"
	"assistdeck/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectController struct {
	projectService *services.ProjectService
}

func NewProjectController(projectService *services.ProjectService) *ProjectController {
	return &ProjectController{projectService: projectService}
}

func (c *ProjectController) CreateProject(ctx *gin.Context) {
	var project models.Project
	if err := ctx.ShouldBindJSON(&project); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	project.UserID = userID.(uuid.UUID)

	if err := c.projectService.CreateProject(&project); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, project)
}

func (c *ProjectController) GetProjects(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	teamID := ctx.Query("team_id")
	var projects []models.Project
	var err error

	if teamID != "" {
		teamUUID, err := uuid.Parse(teamID)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid team ID"})
			return
		}
		projects, err = c.projectService.GetTeamProjects(teamUUID)
	} else {
		projects, err = c.projectService.GetUserProjects(userID.(uuid.UUID))
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, projects)
}

func (c *ProjectController) GetProject(ctx *gin.Context) {
	projectID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	var project models.Project
	if err := c.projectService.DB.First(&project, "id = ?", projectID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	ctx.JSON(http.StatusOK, project)
}

func (c *ProjectController) UpdateProject(ctx *gin.Context) {
	projectID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var project models.Project
	if err := c.projectService.DB.First(&project, "id = ? AND user_id = ?", projectID, userID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	var updatedProject models.Project
	if err := ctx.ShouldBindJSON(&updatedProject); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedProject.ID = project.ID
	updatedProject.UserID = project.UserID

	if err := c.projectService.UpdateProject(&updatedProject); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updatedProject)
}

func (c *ProjectController) DeleteProject(ctx *gin.Context) {
	projectID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid project ID"})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := c.projectService.DeleteProject(projectID, userID.(uuid.UUID)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}
