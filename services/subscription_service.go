package services

import (
	"assistdeck/models"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
) // SubscriptionService handles subscription-related operations
type SubscriptionService struct {
	DB *gorm.DB
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(db *gorm.DB) *SubscriptionService {
	return &SubscriptionService{
		DB: db,
	}
} // GetUserSubscription gets a user's subscription
func (s *SubscriptionService) GetUserSubscription(userID uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := s.DB.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No subscription found
		}
		return nil, err
	}
	return &subscription, nil
} // CreateTrialSubscription creates a trial subscription for a user
func (s *SubscriptionService) CreateTrialSubscription(userID uuid.UUID, planType string, trialDays int) (*models.Subscription, error) {
	// Check if user already has a subscription
	existingSub, err := s.GetUserSubscription(userID)
	if err != nil {
		return nil, err
	}

	if existingSub != nil && existingSub.Active {
		return nil, errors.New("user already has an active subscription")
	}

	// Create new subscription with trial period
	currentTime := time.Now()
	trialEndTime := currentTime.AddDate(0, 0, trialDays)

	subscription := &models.Subscription{
		ID:          uuid.New(),
		UserID:      userID,
		PlanType:    planType,
		Active:      false,
		TrialStatus: true,
		TrialEndsAt: trialEndTime,
		CreatedAt:   currentTime,
		UpdatedAt:   currentTime,
	}

	if err := s.DB.Create(subscription).Error; err != nil {
		return nil, err
	}

	// Update user's plan
	if err := s.DB.Model(&models.User{}).Where("id = ?", userID).Update("plan", planType).Error; err != nil {
		return nil, err
	}

	return subscription, nil
} // CancelSubscription cancels a subscription
func (s *SubscriptionService) CancelSubscription(subscriptionID uuid.UUID, cancelAtPeriodEnd bool) error {
	var subscription models.Subscription
	if err := s.DB.First(&subscription, "id = ?", subscriptionID).Error; err != nil {
		return err
	}

	currentTime := time.Now()
	updates := map[string]interface{}{
		"updated_at":  currentTime,
		"canceled_at": &currentTime,
	}

	if !cancelAtPeriodEnd {
		updates["active"] = false
		updates["ends_at"] = &currentTime
	}

	if err := s.DB.Model(&subscription).Updates(updates).Error; err != nil {
		return err
	}

	// If cancelling immediately, update user plan to free
	if !cancelAtPeriodEnd {
		if err := s.DB.Model(&models.User{}).Where("id = ?", subscription.UserID).Update("plan", "free").Error; err != nil {
			return err
		}
	}

	return nil
} // GetSubscriptionStats gets subscription statistics
func (s *SubscriptionService) GetSubscriptionStats() (map[string]interface{}, error) {
	// Get total active subscriptions
	var totalActive int64
	if err := s.DB.Model(&models.Subscription{}).Where("active = ?", true).Count(&totalActive).Error; err != nil {
		return nil, err
	}

	// Get total trial subscriptions
	var totalTrial int64
	if err := s.DB.Model(&models.Subscription{}).Where("trial_status = ?", true).Count(&totalTrial).Error; err != nil {
		return nil, err
	}

	// Get subscriptions by plan type
	type PlanCount struct {
		PlanType string `json:"plan_type"`
		Count    int64  `json:"count"`
	}
	var planCounts []PlanCount
	if err := s.DB.Model(&models.Subscription{}).Select("plan_type, count(*) as count").Group("plan_type").Find(&planCounts).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_active":      totalActive,
		"total_trial":       totalTrial,
		"plan_distribution": planCounts,
	}, nil
} // CheckExpiringSubscriptions checks for and updates expired subscriptions
func (s *SubscriptionService) CheckExpiringSubscriptions() error {
	currentTime := time.Now()

	// Check for expired trials
	if err := s.DB.Model(&models.Subscription{}).
		Where("trial_status = ? AND trial_ends_at < ?", true, currentTime).
		Updates(map[string]interface{}{
			"trial_status": false,
			"updated_at":   currentTime,
		}).Error; err != nil {
		return err
	}

	// Check for expired subscriptions
	var expiredSubs []models.Subscription
	if err := s.DB.Where("active = ? AND ends_at IS NOT NULL AND ends_at < ?", true, currentTime).Find(&expiredSubs).Error; err != nil {
		return err
	}

	for _, sub := range expiredSubs {
		// Update subscription
		if err := s.DB.Model(&sub).Updates(map[string]interface{}{
			"active":     false,
			"updated_at": currentTime,
		}).Error; err != nil {
			return err
		}

		// Update user plan to free
		if err := s.DB.Model(&models.User{}).Where("id = ?", sub.UserID).Update("plan", "free").Error; err != nil {
			return err
		}
	}

	return nil
}
