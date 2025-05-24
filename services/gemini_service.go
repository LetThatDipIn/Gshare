package services

import (
	"assistdeck/models"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GeminiService struct {
	DB      *gorm.DB
	APIKey  string
	BaseURL string
}

type GeminiRequest struct {
	Contents         []GeminiContent `json:"contents"`
	GenerationConfig GeminiConfig    `json:"generationConfig,omitempty"`
	SafetySettings   []GeminiSafety  `json:"safetySettings,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
}

type GeminiSafety struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GeminiResponse struct {
	Candidates     []GeminiCandidate `json:"candidates"`
	PromptFeedback GeminiFeedback    `json:"promptFeedback"`
}

type GeminiCandidate struct {
	Content       GeminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"`
}

type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type GeminiFeedback struct {
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"`
}

func NewGeminiService(db *gorm.DB, apiKey string) *GeminiService {
	return &GeminiService{
		DB:      db,
		APIKey:  apiKey,
		BaseURL: "https://generativelanguage.googleapis.com/v1/models/gemini-1.5-pro:generateContent",
	}
}

func (s *GeminiService) CreateSession(userID uuid.UUID, title string) (*models.GeminiSession, error) {
	session := &models.GeminiSession{
		UserID:    userID,
		Title:     title,
		ModelName: "gemini-pro",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.DB.Create(session).Error; err != nil {
		return nil, err
	}

	return session, nil
}

func (s *GeminiService) GetSessions(userID uuid.UUID) ([]models.GeminiSession, error) {
	var sessions []models.GeminiSession
	if err := s.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *GeminiService) GetSession(sessionID uuid.UUID) (*models.GeminiSession, error) {
	var session models.GeminiSession
	if err := s.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *GeminiService) GetMessages(sessionID uuid.UUID) ([]models.GeminiMessage, error) {
	var messages []models.GeminiMessage
	if err := s.DB.Where("session_id = ?", sessionID).Order("created_at asc").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *GeminiService) SaveMessage(sessionID, userID uuid.UUID, content, role string) (*models.GeminiMessage, error) {
	message := &models.GeminiMessage{
		SessionID: sessionID,
		UserID:    userID,
		Content:   content,
		Role:      role,
		CreatedAt: time.Now(),
	}

	if err := s.DB.Create(message).Error; err != nil {
		return nil, err
	}

	return message, nil
}

func (s *GeminiService) GenerateResponse(sessionID, userID uuid.UUID, prompt string) (string, error) {
	// First, save the user's message
	_, err := s.SaveMessage(sessionID, userID, prompt, "user")
	if err != nil {
		return "", err
	}

	// Get conversation history for context
	messages, err := s.GetMessages(sessionID)
	if err != nil {
		return "", err
	}

	// Prepare the request to Gemini API
	geminiContents := []GeminiContent{}

	// Add conversation history (up to last 10 messages to stay within context window)
	startIdx := 0
	if len(messages) > 10 {
		startIdx = len(messages) - 10
	}

	for _, msg := range messages[startIdx:] {
		geminiContents = append(geminiContents, GeminiContent{
			Role: msg.Role,
			Parts: []GeminiPart{
				{Text: msg.Content},
			},
		})
	}

	// Prepare the API request
	req := GeminiRequest{
		Contents: geminiContents,
		GenerationConfig: GeminiConfig{
			Temperature:     0.7,
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: 1024,
		},
	}

	// Convert request to JSON
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	// Create HTTP request
	url := fmt.Sprintf("%s?key=%s", s.BaseURL, s.APIKey)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonReq))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Check if request was successful
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API error: %s", string(body))
	}

	// Parse response
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", err
	}

	// Check if we have candidates
	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("No response generated")
	}

	// Extract the response text
	responseText := ""
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		responseText += part.Text
	}

	// Save the assistant's response
	_, err = s.SaveMessage(sessionID, userID, responseText, "assistant")
	if err != nil {
		return "", err
	}

	return responseText, nil
}
