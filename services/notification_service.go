package services

import (
	"assistdeck/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationService struct {
	DB *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{DB: db}
}

func (s *NotificationService) CreateNotification(userID uuid.UUID, title, content, notificationType string, redirectURL string) (*models.Notification, error) {
	notification := &models.Notification{
		UserID:      userID,
		Title:       title,
		Content:     content,
		Type:        notificationType,
		RedirectURL: redirectURL,
		IsRead:      false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.DB.Create(notification).Error; err != nil {
		return nil, err
	}

	return notification, nil
}

func (s *NotificationService) GetNotifications(userID uuid.UUID, limit, offset int, includeRead bool) ([]models.Notification, error) {
	var notifications []models.Notification
	query := s.DB.Where("user_id = ?", userID)

	if !includeRead {
		query = query.Where("is_read = ?", false)
	}

	if err := query.Order("created_at desc").Limit(limit).Offset(offset).Find(&notifications).Error; err != nil {
		return nil, err
	}

	return notifications, nil
}

func (s *NotificationService) GetNotificationCount(userID uuid.UUID, unreadOnly bool) (int64, error) {
	var count int64
	query := s.DB.Model(&models.Notification{}).Where("user_id = ?", userID)

	if unreadOnly {
		query = query.Where("is_read = ?", false)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (s *NotificationService) MarkAsRead(notificationID, userID uuid.UUID) error {
	result := s.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("is_read", true)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *NotificationService) MarkAllAsRead(userID uuid.UUID) error {
	return s.DB.Model(&models.Notification{}).
		Where("user_id = ?", userID).
		Update("is_read", true).Error
}

func (s *NotificationService) DeleteNotification(notificationID, userID uuid.UUID) error {
	result := s.DB.Where("id = ? AND user_id = ?", notificationID, userID).
		Delete(&models.Notification{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *NotificationService) DeleteAllNotifications(userID uuid.UUID) error {
	return s.DB.Where("user_id = ?", userID).Delete(&models.Notification{}).Error
}
