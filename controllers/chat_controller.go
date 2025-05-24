package controllers

import (
	"assistdeck/models"
	"assistdeck/services"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatController struct {
	chatService *services.ChatService
}

func NewChatController(chatService *services.ChatService) *ChatController {
	return &ChatController{chatService: chatService}
}

func (c *ChatController) CreateSession(ctx *gin.Context) {
	var req struct {
		Title  string     `json:"title" binding:"required"`
		TeamID *uuid.UUID `json:"team_id,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	session := &models.ChatSession{
		UserID: userID.(uuid.UUID),
		TeamID: req.TeamID,
		Title:  req.Title,
	}

	if err := c.chatService.DB.Create(session).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, session)
}

func (c *ChatController) GetSessions(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get all sessions where the user is the owner or a participant
	var sessions []models.ChatSession

	// This query gets both owned sessions and sessions where the user is a participant
	query := c.chatService.DB.Table("chat_sessions").
		Select("chat_sessions.*").
		Joins("LEFT JOIN chat_participants ON chat_participants.session_id = chat_sessions.id").
		Where("chat_sessions.user_id = ? OR chat_participants.user_id = ?", userID, userID).
		Group("chat_sessions.id")

	if err := query.Find(&sessions).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, sessions)
}

func (c *ChatController) GetSession(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var session models.ChatSession
	if err := c.chatService.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	ctx.JSON(http.StatusOK, session)
}

func (c *ChatController) SaveMessage(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req struct {
		Content     string  `json:"content" binding:"required"`
		FileURL     *string `json:"file_url,omitempty"`
		IsAIMessage bool    `json:"is_ai_message"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	message := &models.ChatMessage{
		SessionID:   sessionID,
		SenderID:    userID.(uuid.UUID),
		Content:     req.Content,
		FileURL:     req.FileURL,
		IsAIMessage: req.IsAIMessage,
	}

	if err := c.chatService.DB.Create(message).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, message)
}

func (c *ChatController) GetMessages(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var messages []models.ChatMessage
	if err := c.chatService.DB.Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, messages)
}

// AddParticipant adds a user as a participant to a chat session
func (c *ChatController) AddParticipant(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req struct {
		UserID uuid.UUID `json:"user_id" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the current user is the owner of the session
	currentUserID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var session models.ChatSession
	if err := c.chatService.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Only the session owner can add participants
	if session.UserID != currentUserID.(uuid.UUID) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "only the session owner can add participants"})
		return
	}

	// Check if the participant already exists
	var count int64
	if err := c.chatService.DB.Model(&models.ChatParticipant{}).
		Where("session_id = ? AND user_id = ?", sessionID, req.UserID).
		Count(&count).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if count > 0 {
		ctx.JSON(http.StatusConflict, gin.H{"error": "user is already a participant"})
		return
	}

	// Add the participant
	participant := &models.ChatParticipant{
		SessionID: sessionID,
		UserID:    req.UserID,
	}

	if err := c.chatService.DB.Create(participant).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, participant)
}

// GetParticipants returns all participants for a chat session
func (c *ChatController) GetParticipants(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	// Check if the user has access to the session
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Check if the user is the owner or a participant
	var count int64
	query := c.chatService.DB.Table("chat_sessions").
		Joins("LEFT JOIN chat_participants ON chat_participants.session_id = chat_sessions.id").
		Where("chat_sessions.id = ? AND (chat_sessions.user_id = ? OR chat_participants.user_id = ?)",
			sessionID, userID, userID)

	if err := query.Count(&count).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if count == 0 {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "you don't have access to this chat session"})
		return
	}

	// Get all participants including the owner
	var participants []struct {
		ID        uuid.UUID `json:"id"`
		UserID    uuid.UUID `json:"user_id"`
		Email     string    `json:"email"`
		Name      string    `json:"name"`
		IsOwner   bool      `json:"is_owner"`
		CreatedAt string    `json:"created_at"`
	}

	// Get owner information
	query = c.chatService.DB.Table("chat_sessions").
		Select("chat_sessions.id, users.id as user_id, users.email, users.name, TRUE as is_owner, chat_sessions.created_at").
		Joins("JOIN users ON users.id = chat_sessions.user_id").
		Where("chat_sessions.id = ?", sessionID)

	if err := query.Scan(&participants).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error getting owner info: %v", err.Error())})
		return
	}

	// Log owner info
	log.Printf("DEBUG: Owner participant data: %+v", participants)

	// Get participants
	var participantsList []struct {
		ID        uuid.UUID `json:"id"`
		UserID    uuid.UUID `json:"user_id"`
		Email     string    `json:"email"`
		Name      string    `json:"name"`
		IsOwner   bool      `json:"is_owner"`
		CreatedAt string    `json:"created_at"`
	}

	participantsQuery := c.chatService.DB.Table("chat_participants").
		Select("chat_participants.id, users.id as user_id, users.email, users.name, FALSE as is_owner, chat_participants.created_at").
		Joins("JOIN users ON users.id = chat_participants.user_id").
		Where("chat_participants.session_id = ?", sessionID)

	// Log generated SQL
	sql := participantsQuery.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(&participantsList)
	})
	log.Printf("DEBUG: Participants SQL: %s", sql)

	if err := participantsQuery.Scan(&participantsList).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error getting participants: %v", err.Error())})
		return
	}

	// Log participants data
	log.Printf("DEBUG: Participants data: %+v", participantsList)

	participants = append(participants, participantsList...)

	ctx.JSON(http.StatusOK, participants)
}

// RemoveParticipant removes a participant from a chat session
func (c *ChatController) RemoveParticipant(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	participantID, err := uuid.Parse(ctx.Param("userId"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant ID"})
		return
	}

	// Check if the current user is the owner of the session
	currentUserID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var session models.ChatSession
	if err := c.chatService.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	// Only the session owner can remove participants
	if session.UserID != currentUserID.(uuid.UUID) {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "only the session owner can remove participants"})
		return
	}

	// Remove the participant
	result := c.chatService.DB.Where("session_id = ? AND user_id = ?", sessionID, participantID).Delete(&models.ChatParticipant{})

	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "participant not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "participant removed"})
}
