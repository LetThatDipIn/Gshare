package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GoalStatus string

const (
	GoalStatusPending    GoalStatus = "pending"
	GoalStatusInProgress GoalStatus = "in_progress"
	GoalStatusCompleted  GoalStatus = "completed"
)

type RoleType string

const (
	RoleTypeStudent      RoleType = "student"
	RoleTypeEntrepreneur RoleType = "entrepreneur"
)

type Goal struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid" json:"user_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	RoleType       RoleType   `json:"role_type"`
	Status         GoalStatus `json:"status"`
	CompletionPerc float32    `json:"completion_perc"`
	Deadline       *time.Time `json:"deadline,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (g *Goal) BeforeCreate(tx *gorm.DB) error {
	g.ID = uuid.New()
	return nil
}
