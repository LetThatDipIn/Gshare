package controllers

import (
	"assistdeck/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type GeminiController struct {
	geminiService *services.GeminiService
}

func NewGeminiController(geminiService *services.GeminiService) *GeminiController {
	return &GeminiController{geminiService: geminiService}
}

// CreateSession creates a new Gemini chat session
func (c *GeminiController) CreateSession(ctx *gin.Context) {
	var req struct {
		Title string `json:"title" binding:"required"`
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

	session, err := c.geminiService.CreateSession(userID.(uuid.UUID), req.Title)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, session)
}

// GetSessions gets all Gemini chat sessions for a user
func (c *GeminiController) GetSessions(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessions, err := c.geminiService.GetSessions(userID.(uuid.UUID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, sessions)
}

// GetSession gets a specific Gemini chat session
func (c *GeminiController) GetSession(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	session, err := c.geminiService.GetSession(sessionID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	ctx.JSON(http.StatusOK, session)
}

// GetMessages gets all messages for a Gemini chat session
func (c *GeminiController) GetMessages(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	messages, err := c.geminiService.GetMessages(sessionID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, messages)
}

// SendMessage sends a message to Gemini API and gets a response
func (c *GeminiController) SendMessage(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
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

	response, err := c.geminiService.GenerateResponse(sessionID, userID.(uuid.UUID), req.Content)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"response": response,
	})
}
