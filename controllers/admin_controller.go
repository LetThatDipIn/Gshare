package controllers

import (
	"assistdeck/models"
	"assistdeck/services"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminController struct {
	adminService *services.AdminService
	userService  *services.UserService
}

func NewAdminController(adminService *services.AdminService, userService *services.UserService) *AdminController {
	return &AdminController{
		adminService: adminService,
		userService:  userService,
	}
}

// GetSettings gets all settings or settings by category
func (c *AdminController) GetSettings(ctx *gin.Context) {
	category := ctx.Query("category")

	settings, err := c.adminService.GetSettings(category)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, settings)
}

// GetSetting gets a specific setting by key
func (c *AdminController) GetSetting(ctx *gin.Context) {
	key := ctx.Param("key")

	setting, err := c.adminService.GetSetting(key)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "setting not found"})
		return
	}

	ctx.JSON(http.StatusOK, setting)
}

// SaveSetting creates or updates a setting
func (c *AdminController) SaveSetting(ctx *gin.Context) {
	var req struct {
		Key         string `json:"key" binding:"required"`
		Value       string `json:"value" binding:"required"`
		Category    string `json:"category" binding:"required"`
		Description string `json:"description"`
		IsEncrypted bool   `json:"is_encrypted"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setting, err := c.adminService.SaveSetting(
		req.Key,
		req.Value,
		req.Category,
		req.Description,
		req.IsEncrypted,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, setting)
}

// DeleteSetting deletes a setting
func (c *AdminController) DeleteSetting(ctx *gin.Context) {
	key := ctx.Param("key")

	if err := c.adminService.DeleteSetting(key); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "setting deleted"})
}

// GetUserStats gets user statistics
func (c *AdminController) GetUserStats(ctx *gin.Context) {
	stats, err := c.adminService.GetUserStats()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

// GetAllUsers gets all users (with pagination)
func (c *AdminController) GetAllUsers(ctx *gin.Context) {
	page := ctx.DefaultQuery("page", "1")
	pageSize := ctx.DefaultQuery("page_size", "50")

	// Since there's no GetUsers method, let's query the users directly
	var users []models.User
	var totalCount int64

	// Count total records for pagination
	if err := c.userService.DB.Model(&models.User{}).Count(&totalCount).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse pagination parameters
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt < 1 {
		pageInt = 1
	}

	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeInt < 1 {
		pageSizeInt = 50
	}

	// Calculate offset
	offset := (pageInt - 1) * pageSizeInt

	// Get users with pagination
	if err := c.userService.DB.Offset(offset).Limit(pageSizeInt).Order("created_at desc").Find(&users).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create pagination info
	pagination := map[string]interface{}{
		"current_page": pageInt,
		"page_size":    pageSizeInt,
		"total":        totalCount,
		"total_pages":  int(math.Ceil(float64(totalCount) / float64(pageSizeInt))),
	}

	ctx.JSON(http.StatusOK, gin.H{
		"users":      users,
		"pagination": pagination,
	})
}

// GetUserDetails gets detailed information about a user
func (c *AdminController) GetUserDetails(ctx *gin.Context) {
	userIDStr := ctx.Param("id")

	// Parse the user ID from string to UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	user, err := c.userService.GetUserByID(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get subscription info directly from the database
	var subscription models.Subscription
	c.userService.DB.Where("user_id = ?", user.ID).First(&subscription)

	ctx.JSON(http.StatusOK, gin.H{
		"user":         user,
		"subscription": subscription,
	})
}

// UpdateUserRole updates a user's role (admin only)
func (c *AdminController) UpdateUserRole(ctx *gin.Context) {
	userIDStr := ctx.Param("id")

	// Parse the user ID from string to UUID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role
	if req.Role != "user" && req.Role != "admin" && req.Role != "manager" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}

	// Get the user first
	user, err := c.userService.GetUserByID(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Update the user's role
	user.Role = req.Role
	if err := c.userService.UpdateUser(user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "user role updated"})
}
