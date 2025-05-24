package models

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents a notification to a user
type Notification struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid" json:"user_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Type        string    `gorm:"type:varchar(50)" json:"type"` // 'info', 'warning', 'error', etc.
	IsRead      bool      `gorm:"default:false" json:"is_read"`
	RedirectURL string    `json:"redirect_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CalendarEvent represents a calendar event
type CalendarEvent struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid" json:"user_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        time.Time  `json:"end_time"`
	Location       string     `json:"location,omitempty"`
	IsAllDay       bool       `gorm:"default:false" json:"is_all_day"`
	RecurrenceRule string     `json:"recurrence_rule,omitempty"` // iCalendar RRULE format
	GoogleEventID  string     `json:"google_event_id,omitempty"` // For syncing with Google Calendar
	TeamID         *uuid.UUID `gorm:"type:uuid" json:"team_id,omitempty"`
	ProjectID      *uuid.UUID `gorm:"type:uuid" json:"project_id,omitempty"`
	GoalID         *uuid.UUID `gorm:"type:uuid" json:"goal_id,omitempty"`
	ReminderTime   *time.Time `json:"reminder_time,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// AdminSettings represents global admin configuration
type AdminSettings struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SettingKey      string    `gorm:"uniqueIndex" json:"setting_key"`
	SettingValue    string    `json:"setting_value"`
	SettingCategory string    `json:"setting_category"`
	Description     string    `json:"description,omitempty"`
	IsEncrypted     bool      `gorm:"default:false" json:"is_encrypted"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// GeminiMessage represents a message exchanged with the Gemini API
type GeminiMessage struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid" json:"user_id"`
	SessionID uuid.UUID `gorm:"type:uuid" json:"session_id"`
	Content   string    `json:"content"`
	Role      string    `gorm:"type:varchar(10)" json:"role"` // 'user' or 'assistant'
	CreatedAt time.Time `json:"created_at"`
}

// GeminiSession represents a conversation session with Gemini
type GeminiSession struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid" json:"user_id"`
	Title     string    `json:"title"`
	ModelName string    `gorm:"default:'gemini-pro'" json:"model_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserGoogleCalendar stores Google Calendar integration details for a user
type UserGoogleCalendar struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"user_id"`
	AccessToken       string    `json:"-"` // Encrypted, not returned in JSON
	RefreshToken      string    `json:"-"` // Encrypted, not returned in JSON
	TokenExpiry       time.Time `json:"token_expiry"`
	IsConnected       bool      `gorm:"default:false" json:"is_connected"`
	LastSyncTime      time.Time `json:"last_sync_time,omitempty"`
	PrimaryCalendarID string    `json:"primary_calendar_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
