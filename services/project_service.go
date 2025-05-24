package services

import (
	"assistdeck/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectService struct {
	DB *gorm.DB
}

func NewProjectService(db *gorm.DB) *ProjectService {
	return &ProjectService{DB: db}
}

func (s *ProjectService) CreateProject(project *models.Project) error {
	return s.DB.Create(project).Error
}

func (s *ProjectService) GetUserProjects(userID uuid.UUID) ([]models.Project, error) {
	var projects []models.Project
	if err := s.DB.Where("user_id = ?", userID).Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *ProjectService) GetTeamProjects(teamID uuid.UUID) ([]models.Project, error) {
	var projects []models.Project
	if err := s.DB.Where("team_id = ?", teamID).Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *ProjectService) UpdateProject(project *models.Project) error {
	return s.DB.Save(project).Error
}

func (s *ProjectService) DeleteProject(projectID, userID uuid.UUID) error {
	return s.DB.Where("id = ? AND user_id = ?", projectID, userID).Delete(&models.Project{}).Error
}
