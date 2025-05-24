package controllers

import (
	"assistdeck/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StartFreeTrial starts the 7-day free trial for a user
func StartFreeTrial(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, _ := c.Get("user_id")

	// Parse request body
	var requestBody struct {
		PlanType string `json:"plan_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate plan type
	if requestBody.PlanType != "student" && requestBody.PlanType != "entrepreneur" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plan type must be either 'student' or 'entrepreneur'"})
		return
	}

	// Get the user
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if user already has an active subscription
	var existingSub models.Subscription
	result := db.Where("user_id = ? AND (trial_status = ? OR active = ?)", userID, true, true).First(&existingSub)
	if result.RowsAffected > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already has an active subscription or trial"})
		return
	}

	// Create trial subscription
	now := time.Now()
	trialEndsAt := now.AddDate(0, 0, 7) // 7 days trial

	subscription := models.Subscription{
		ID:          uuid.New(),
		UserID:      user.ID,
		PlanType:    requestBody.PlanType,
		TrialStatus: true,
		TrialEndsAt: trialEndsAt,
		Active:      false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := db.Create(&subscription).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	// Update user plan
	user.Plan = "trial"
	trialEndsAtPtr := subscription.TrialEndsAt
	user.TrialEndsAt = &trialEndsAtPtr

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Free trial started successfully",
		"subscription": subscription,
		"user":         user,
	})
}

// GetSubscriptionStatus gets the current subscription status for a user
func GetSubscriptionStatus(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID, _ := c.Get("user_id")

	var subscription models.Subscription
	result := db.Where("user_id = ? AND (trial_status = ? OR active = ?)",
		userID, true, true).First(&subscription)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "inactive"})
		return
	}

	// Check if trial has expired
	if subscription.TrialStatus && time.Now().After(subscription.TrialEndsAt) {
		c.JSON(http.StatusOK, gin.H{
			"status":       "trial_expired",
			"subscription": subscription,
		})
		return
	}

	// Check if subscription has expired
	if subscription.Active && subscription.EndsAt != nil && time.Now().After(*subscription.EndsAt) {
		c.JSON(http.StatusOK, gin.H{
			"status":       "subscription_expired",
			"subscription": subscription,
		})
		return
	}

	// Active subscription
	status := "trial"
	if subscription.Active {
		status = "active"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       status,
		"subscription": subscription,
	})
}
