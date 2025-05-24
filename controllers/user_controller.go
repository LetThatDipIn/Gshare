package controllers

import (
	"assistdeck/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUserProfile gets the current user's profile
func GetUserProfile(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, _ := c.Get("user_id")

	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateUserRole updates the user's selected role
func UpdateUserRole(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, _ := c.Get("user_id")

	// Parse request body
	var requestBody struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role
	if requestBody.Role != "student" && requestBody.Role != "entrepreneur" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role must be either 'student' or 'entrepreneur'"})
		return
	}

	// Update the user
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Role = requestBody.Role
	db.Save(&user)

	c.JSON(http.StatusOK, gin.H{"user": user})
}
