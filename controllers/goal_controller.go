package controllers

import (
	"assistdeck/models"
	"assistdeck/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GoalController struct {
	goalService *services.GoalService
}

func NewGoalController(goalService *services.GoalService) *GoalController {
	return &GoalController{goalService: goalService}
}

func (c *GoalController) CreateGoal(ctx *gin.Context) {
	var goal models.Goal
	if err := ctx.ShouldBindJSON(&goal); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	goal.UserID = userID.(uuid.UUID)

	if err := c.goalService.CreateGoal(&goal); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, goal)
}

func (c *GoalController) GetGoals(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	goals, err := c.goalService.GetUserGoals(userID.(uuid.UUID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, goals)
}

func (c *GoalController) GetGoal(ctx *gin.Context) {
	goalID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal ID"})
		return
	}

	var goal models.Goal
	if err := c.goalService.DB.First(&goal, "id = ?", goalID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
		return
	}

	ctx.JSON(http.StatusOK, goal)
}

func (c *GoalController) UpdateGoal(ctx *gin.Context) {
	goalID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal ID"})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var goal models.Goal
	if err := c.goalService.DB.First(&goal, "id = ? AND user_id = ?", goalID, userID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "goal not found"})
		return
	}

	var updatedGoal models.Goal
	if err := ctx.ShouldBindJSON(&updatedGoal); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedGoal.ID = goal.ID
	updatedGoal.UserID = goal.UserID

	if err := c.goalService.UpdateGoal(&updatedGoal); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updatedGoal)
}

func (c *GoalController) DeleteGoal(ctx *gin.Context) {
	goalID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid goal ID"})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := c.goalService.DeleteGoal(goalID, userID.(uuid.UUID)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusOK)
}
