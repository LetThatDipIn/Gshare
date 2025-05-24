package models

import (
	"time"

	"github.com/google/uuid"
)

// MediaType represents the type of media
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
	MediaTypeFile  MediaType = "file"
	MediaTypeOther MediaType = "other"
)

// MediaFile represents a media file uploaded by a user
type MediaFile struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid" json:"user_id"`
	FileName    string     `json:"file_name"`
	FileSize    int64      `json:"file_size"`
	ContentType string     `json:"content_type"`
	MediaType   MediaType  `json:"media_type"`
	StoragePath string     `json:"storage_path"`
	URL         string     `json:"url"`
	Description string     `json:"description,omitempty"`
	ProjectID   *uuid.UUID `gorm:"type:uuid" json:"project_id,omitempty"`
	TeamID      *uuid.UUID `gorm:"type:uuid" json:"team_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AudioFile extends MediaFile with audio-specific metadata
type AudioFile struct {
	MediaFileID   uuid.UUID `gorm:"type:uuid;primary_key" json:"media_file_id"`
	MediaFile     MediaFile `gorm:"foreignKey:MediaFileID" json:"-"`
	Duration      int       `json:"duration"` // Duration in seconds
	Transcription string    `json:"transcription,omitempty"`
	IsProcessed   bool      `gorm:"default:false" json:"is_processed"`
}
