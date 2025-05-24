package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatSession struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid" json:"user_id"`
	TeamID    *uuid.UUID `gorm:"type:uuid" json:"team_id,omitempty"`
	Title     string     `json:"title"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (c *ChatSession) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New()
	return nil
}

type ChatMessage struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	SessionID   uuid.UUID `gorm:"type:uuid" json:"session_id"`
	SenderID    uuid.UUID `gorm:"type:uuid" json:"sender_id"`
	Content     string    `json:"content"`
	FileURL     *string   `json:"file_url,omitempty"`
	IsAIMessage bool      `json:"is_ai_message"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New()
	return nil
}

// ChatParticipant represents a user who has access to a chat session
type ChatParticipant struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	SessionID uuid.UUID `gorm:"type:uuid" json:"session_id"`
	UserID    uuid.UUID `gorm:"type:uuid" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *ChatParticipant) BeforeCreate(tx *gorm.DB) error {
	c.ID = uuid.New()
	return nil
}
