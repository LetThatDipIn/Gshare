package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectStatus string

const (
	ProjectStatusPlanning  ProjectStatus = "planning"
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusPaused    ProjectStatus = "paused"
	ProjectStatusCompleted ProjectStatus = "completed"
)

type Project struct {
	ID          uuid.UUID     `gorm:"type:uuid;primary_key" json:"id"`
	UserID      uuid.UUID     `gorm:"type:uuid" json:"user_id"`
	TeamID      *uuid.UUID    `gorm:"type:uuid" json:"team_id,omitempty"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      ProjectStatus `json:"status"`
	StartDate   time.Time     `json:"start_date"`
	EndDate     *time.Time    `json:"end_date,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) error {
	p.ID = uuid.New()
	return nil
}
