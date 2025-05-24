package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Team struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `gorm:"type:uuid" json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (t *Team) BeforeCreate(tx *gorm.DB) error {
	t.ID = uuid.New()
	return nil
}
