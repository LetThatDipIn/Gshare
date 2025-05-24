package services

import (
	"assistdeck/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GoalService struct {
	DB *gorm.DB
}

func NewGoalService(db *gorm.DB) *GoalService {
	return &GoalService{DB: db}
}

func (s *GoalService) CreateGoal(goal *models.Goal) error {
	return s.DB.Create(goal).Error
}

func (s *GoalService) GetUserGoals(userID uuid.UUID) ([]models.Goal, error) {
	var goals []models.Goal
	if err := s.DB.Where("user_id = ?", userID).Find(&goals).Error; err != nil {
		return nil, err
	}
	return goals, nil
}

func (s *GoalService) UpdateGoal(goal *models.Goal) error {
	return s.DB.Save(goal).Error
}

func (s *GoalService) DeleteGoal(goalID, userID uuid.UUID) error {
	return s.DB.Where("id = ? AND user_id = ?", goalID, userID).Delete(&models.Goal{}).Error
}
