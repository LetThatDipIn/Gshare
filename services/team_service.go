package services

import (
	"assistdeck/models"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TeamService handles team-related operations
type TeamService struct {
	DB *gorm.DB
}

// NewTeamService creates a new team service
func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{
		DB: db,
	}
}

// CreateTeam creates a new team
func (s *TeamService) CreateTeam(name string, ownerID uuid.UUID) (*models.Team, error) {
	team := &models.Team{
		ID:      uuid.New(),
		Name:    name,
		OwnerID: ownerID,
	}

	if err := s.DB.Create(team).Error; err != nil {
		return nil, err
	}

	// Add team owner as a member with admin role
	member := &models.TeamMember{
		ID:     uuid.New(),
		TeamID: team.ID,
		UserID: ownerID,
		Role:   models.TeamRoleAdmin,
	}

	if err := s.DB.Create(member).Error; err != nil {
		return nil, err
	}

	return team, nil
}

// GetTeamByID gets a team by ID
func (s *TeamService) GetTeamByID(id uuid.UUID) (*models.Team, error) {
	var team models.Team
	if err := s.DB.First(&team, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &team, nil
}

// GetUserTeams gets all teams a user is a member of
func (s *TeamService) GetUserTeams(userID uuid.UUID) ([]models.Team, error) {
	var teamMembers []models.TeamMember
	if err := s.DB.Where("user_id = ?", userID).Find(&teamMembers).Error; err != nil {
		return nil, err
	}

	var teamIDs []uuid.UUID
	for _, member := range teamMembers {
		teamIDs = append(teamIDs, member.TeamID)
	}

	var teams []models.Team
	if len(teamIDs) > 0 {
		if err := s.DB.Where("id IN ?", teamIDs).Find(&teams).Error; err != nil {
			return nil, err
		}
	}

	return teams, nil
}

// AddTeamMember adds a member to a team
func (s *TeamService) AddTeamMember(teamID, userID uuid.UUID, role string) (*models.TeamMember, error) {
	// Check if user is already a member
	var existingMember models.TeamMember
	if err := s.DB.Where("team_id = ? AND user_id = ?", teamID, userID).First(&existingMember).Error; err == nil {
		return nil, errors.New("user is already a member of this team")
	}

	// Add member
	member := &models.TeamMember{
		ID:     uuid.New(),
		TeamID: teamID,
		UserID: userID,
		Role:   models.TeamRole(role),
	}

	if err := s.DB.Create(member).Error; err != nil {
		return nil, err
	}

	return member, nil
}

// RemoveTeamMember removes a member from a team
func (s *TeamService) RemoveTeamMember(teamID, userID uuid.UUID) error {
	// Check if user is the owner
	var team models.Team
	if err := s.DB.First(&team, "id = ?", teamID).Error; err != nil {
		return err
	}

	if team.OwnerID == userID {
		return errors.New("cannot remove team owner")
	}

	// Remove member
	if err := s.DB.Where("team_id = ? AND user_id = ?", teamID, userID).Delete(&models.TeamMember{}).Error; err != nil {
		return err
	}

	return nil
}

// UpdateTeam updates a team
func (s *TeamService) UpdateTeam(teamID uuid.UUID, name string) (*models.Team, error) {
	var team models.Team
	if err := s.DB.First(&team, "id = ?", teamID).Error; err != nil {
		return nil, err
	}

	team.Name = name

	if err := s.DB.Save(&team).Error; err != nil {
		return nil, err
	}

	return &team, nil
}

// DeleteTeam deletes a team
func (s *TeamService) DeleteTeam(teamID uuid.UUID) error {
	// Delete team members first
	if err := s.DB.Where("team_id = ?", teamID).Delete(&models.TeamMember{}).Error; err != nil {
		return err
	}

	// Delete team
	if err := s.DB.Delete(&models.Team{}, "id = ?", teamID).Error; err != nil {
		return err
	}

	return nil
}

// GetTeamMembers gets all members of a team
func (s *TeamService) GetTeamMembers(teamID uuid.UUID) ([]models.TeamMember, error) {
	var members []models.TeamMember
	if err := s.DB.Where("team_id = ?", teamID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}
