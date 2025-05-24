package services

import (
	"assistdeck/models"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MediaService handles file operations
type MediaService struct {
	DB           *gorm.DB
	StoragePath  string
	MaxFileSize  int64
	AllowedTypes map[string]bool
}

// NewMediaService creates a new MediaService
func NewMediaService(db *gorm.DB, storagePath string) *MediaService {
	// Create storage directory if it doesn't exist
	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		os.MkdirAll(storagePath, 0755)
	}

	// Define allowed MIME types
	allowedTypes := map[string]bool{
		// Images
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
		// Audio
		"audio/mpeg": true,
		"audio/mp4":  true,
		"audio/ogg":  true,
		"audio/wav":  true,
		"audio/webm": true,
		// Documents
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.ms-excel": true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
		// Text files
		"text/plain": true,
		"text/csv":   true,
	}

	return &MediaService{
		DB:           db,
		StoragePath:  storagePath,
		MaxFileSize:  100 * 1024 * 1024, // 100MB
		AllowedTypes: allowedTypes,
	}
}

// UploadFile uploads a file and creates a media record
func (s *MediaService) UploadFile(c *gin.Context, file *multipart.FileHeader, userID uuid.UUID, description string, projectID, teamID *uuid.UUID) (*models.MediaFile, error) {
	// Validate file size
	if file.Size > s.MaxFileSize {
		return nil, errors.New("file size exceeds maximum allowed size")
	}

	// Get file content type
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	contentType := strings.ToLower(strings.TrimSpace(http.DetectContentType(buffer)))

	// Reset the file reader
	src.Seek(0, 0)

	// Check if content type is allowed
	if !s.AllowedTypes[contentType] {
		return nil, errors.New("file type not allowed")
	}

	// Determine media type
	mediaType := determineMediaType(contentType)

	// Generate a unique filename
	fileName := fmt.Sprintf("%s-%s", uuid.New().String(), filepath.Base(file.Filename))
	fileID := uuid.New()
	year, month, day := time.Now().Date()
	storagePath := filepath.Join(s.StoragePath, fmt.Sprintf("%d/%d/%d/%s", year, month, day, fileID.String()))

	// Create directory if it doesn't exist
	os.MkdirAll(filepath.Dir(storagePath), 0755)

	// Save the file
	dst, err := os.Create(storagePath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return nil, err
	}

	// Create media record
	media := &models.MediaFile{
		ID:          fileID,
		UserID:      userID,
		FileName:    fileName,
		FileSize:    file.Size,
		ContentType: contentType,
		MediaType:   mediaType,
		StoragePath: storagePath,
		URL:         fmt.Sprintf("/api/media/%s", fileID.String()),
		Description: description,
		ProjectID:   projectID,
		TeamID:      teamID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.DB.Create(media).Error; err != nil {
		// Delete the file if database insertion fails
		os.Remove(storagePath)
		return nil, err
	}

	// If it's an audio file, create an AudioFile record
	if mediaType == models.MediaTypeAudio {
		audioFile := &models.AudioFile{
			MediaFileID: fileID,
			IsProcessed: false,
		}
		if err := s.DB.Create(audioFile).Error; err != nil {
			return media, err // Return the media file even if audio metadata creation fails
		}
	}

	return media, nil
}

// GetFile retrieves a file by ID
func (s *MediaService) GetFile(id uuid.UUID) (*models.MediaFile, error) {
	var media models.MediaFile
	if err := s.DB.First(&media, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &media, nil
}

// GetUserFiles gets all files uploaded by a user
func (s *MediaService) GetUserFiles(userID uuid.UUID, mediaType string) ([]models.MediaFile, error) {
	var mediaFiles []models.MediaFile
	query := s.DB.Where("user_id = ?", userID)

	if mediaType != "" {
		query = query.Where("media_type = ?", mediaType)
	}

	if err := query.Order("created_at DESC").Find(&mediaFiles).Error; err != nil {
		return nil, err
	}

	return mediaFiles, nil
}

// DeleteFile deletes a file
func (s *MediaService) DeleteFile(id uuid.UUID, userID uuid.UUID) error {
	var media models.MediaFile
	if err := s.DB.First(&media, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return err
	}

	// Delete AudioFile record if it exists
	if media.MediaType == models.MediaTypeAudio {
		s.DB.Where("media_file_id = ?", id).Delete(&models.AudioFile{})
	}

	// Delete the actual file
	if err := os.Remove(media.StoragePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Delete from database
	return s.DB.Delete(&media).Error
}

// GetAudioFile gets audio file metadata
func (s *MediaService) GetAudioFile(mediaFileID uuid.UUID) (*models.AudioFile, error) {
	var audioFile models.AudioFile
	if err := s.DB.Where("media_file_id = ?", mediaFileID).First(&audioFile).Error; err != nil {
		return nil, err
	}
	return &audioFile, nil
}

// UpdateAudioMetadata updates audio metadata
func (s *MediaService) UpdateAudioMetadata(mediaFileID uuid.UUID, duration int, transcription string) error {
	return s.DB.Model(&models.AudioFile{}).
		Where("media_file_id = ?", mediaFileID).
		Updates(map[string]interface{}{
			"duration":      duration,
			"transcription": transcription,
			"is_processed":  true,
		}).Error
}

// determineMediaType determines the media type based on content type
func determineMediaType(contentType string) models.MediaType {
	if strings.HasPrefix(contentType, "image/") {
		return models.MediaTypeImage
	} else if strings.HasPrefix(contentType, "audio/") {
		return models.MediaTypeAudio
	} else if strings.HasPrefix(contentType, "video/") {
		return models.MediaTypeVideo
	} else if strings.HasPrefix(contentType, "application/") || strings.HasPrefix(contentType, "text/") {
		return models.MediaTypeFile
	}
	return models.MediaTypeOther
}

// CheckUserSubscription checks if a user has an active subscription or trial
func (s *MediaService) CheckUserSubscription(userID uuid.UUID) (bool, error) {
	var subscription models.Subscription
	result := s.DB.Where("user_id = ?", userID).First(&subscription)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil // No subscription found
		}
		return false, result.Error
	}

	// Check if trial is active and not expired
	if subscription.TrialStatus && time.Now().Before(subscription.TrialEndsAt) {
		return true, nil
	}

	// Check if paid subscription is active and not expired
	if subscription.Active && (subscription.EndsAt == nil || time.Now().Before(*subscription.EndsAt)) {
		return true, nil
	}

	return false, nil
}
