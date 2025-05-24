package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleMember TeamRole = "member"
	TeamRoleAdmin  TeamRole = "admin"
)

type TeamMember struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	TeamID    uuid.UUID `gorm:"type:uuid" json:"team_id"`
	UserID    uuid.UUID `gorm:"type:uuid" json:"user_id"`
	Role      TeamRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (tm *TeamMember) BeforeCreate(tx *gorm.DB) error {
	tm.ID = uuid.New()
	return nil
}
