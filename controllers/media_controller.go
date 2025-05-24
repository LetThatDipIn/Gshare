package controllers

import (
	"assistdeck/models"
	"assistdeck/services"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MediaController handles file upload and retrieval operations
type MediaController struct {
	mediaService *services.MediaService
}

// NewMediaController creates a new MediaController
func NewMediaController(mediaService *services.MediaService) *MediaController {
	return &MediaController{mediaService: mediaService}
}

// UploadFile handles file uploads
func (c *MediaController) UploadFile(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Temporarily disable subscription check for testing
	/*
		// Check if user has an active subscription or trial
		hasSubscription, err := c.mediaService.CheckUserSubscription(userID.(uuid.UUID))
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check subscription status"})
			return
		}

		if !hasSubscription {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "subscription required for file uploads"})
			return
		}
	*/

	// Get file from form data
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	// Validate file extension
	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file must have an extension"})
		return
	}

	// Get optional fields
	description := ctx.PostForm("description")

	var projectID *uuid.UUID
	projectIDStr := ctx.PostForm("project_id")
	if projectIDStr != "" {
		id, err := uuid.Parse(projectIDStr)
		if err == nil {
			projectID = &id
		}
	}

	var teamID *uuid.UUID
	teamIDStr := ctx.PostForm("team_id")
	if teamIDStr != "" {
		id, err := uuid.Parse(teamIDStr)
		if err == nil {
			teamID = &id
		}
	}

	// Upload file
	media, err := c.mediaService.UploadFile(ctx, file, userID.(uuid.UUID), description, projectID, teamID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, media)
}

// GetFile retrieves a file by ID
func (c *MediaController) GetFile(ctx *gin.Context) {
	fileID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid file ID"})
		return
	}

	media, err := c.mediaService.GetFile(fileID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Serve the file
	ctx.File(media.StoragePath)
}

// GetFileMetadata retrieves file metadata
func (c *MediaController) GetFileMetadata(ctx *gin.Context) {
	fileID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid file ID"})
		return
	}

	media, err := c.mediaService.GetFile(fileID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// If it's an audio file, get audio metadata
	if media.MediaType == models.MediaTypeAudio {
		audioFile, err := c.mediaService.GetAudioFile(fileID)
		if err == nil {
			ctx.JSON(http.StatusOK, gin.H{
				"file":  media,
				"audio": audioFile,
			})
			return
		}
	}

	ctx.JSON(http.StatusOK, media)
}

// GetUserFiles retrieves all files for a user
func (c *MediaController) GetUserFiles(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Filter by media type
	mediaType := ctx.Query("type")

	files, err := c.mediaService.GetUserFiles(userID.(uuid.UUID), mediaType)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, files)
}

// DeleteFile deletes a file
func (c *MediaController) DeleteFile(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	fileID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid file ID"})
		return
	}

	err = c.mediaService.DeleteFile(fileID, userID.(uuid.UUID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}

// UploadAudio handles audio file uploads with additional metadata
func (c *MediaController) UploadAudio(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Temporarily disable subscription check for testing
	/*
		// Check if user has an active subscription or trial
		hasSubscription, err := c.mediaService.CheckUserSubscription(userID.(uuid.UUID))
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check subscription status"})
			return
		}

		if !hasSubscription {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "subscription required for audio uploads"})
			return
		}
	*/

	// Get file from form data
	file, err := ctx.FormFile("audio")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "audio file is required"})
		return
	}

	// Validate file extension
	ext := filepath.Ext(file.Filename)
	allowedExtensions := map[string]bool{
		".mp3": true,
		".wav": true,
		".m4a": true,
		".aac": true,
		".ogg": true,
	}

	if !allowedExtensions[ext] {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "unsupported audio format"})
		return
	}

	// Get optional fields
	description := ctx.PostForm("description")

	var projectID *uuid.UUID
	projectIDStr := ctx.PostForm("project_id")
	if projectIDStr != "" {
		id, err := uuid.Parse(projectIDStr)
		if err == nil {
			projectID = &id
		}
	}

	var teamID *uuid.UUID
	teamIDStr := ctx.PostForm("team_id")
	if teamIDStr != "" {
		id, err := uuid.Parse(teamIDStr)
		if err == nil {
			teamID = &id
		}
	}

	// Upload file
	media, err := c.mediaService.UploadFile(ctx, file, userID.(uuid.UUID), description, projectID, teamID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"file":                 media,
		"transcription_status": "pending", // Transcription would be done asynchronously
	})
}
