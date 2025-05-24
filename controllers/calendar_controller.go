package controllers

import (
	"assistdeck/models"
	"assistdeck/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CalendarController struct {
	calendarService *services.CalendarService
}

func NewCalendarController(calendarService *services.CalendarService) *CalendarController {
	return &CalendarController{calendarService: calendarService}
}

// CreateEvent creates a new calendar event
func (c *CalendarController) CreateEvent(ctx *gin.Context) {
	var req struct {
		Title          string     `json:"title" binding:"required"`
		Description    string     `json:"description"`
		StartTime      time.Time  `json:"start_time" binding:"required"`
		EndTime        time.Time  `json:"end_time" binding:"required"`
		Location       string     `json:"location"`
		IsAllDay       bool       `json:"is_all_day"`
		RecurrenceRule string     `json:"recurrence_rule"`
		TeamID         *uuid.UUID `json:"team_id"`
		ProjectID      *uuid.UUID `json:"project_id"`
		GoalID         *uuid.UUID `json:"goal_id"`
		ReminderTime   *time.Time `json:"reminder_time"`
		SyncToGoogle   bool       `json:"sync_to_google"`
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

	// Create the event
	event := &models.CalendarEvent{
		UserID:         userID.(uuid.UUID),
		Title:          req.Title,
		Description:    req.Description,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
		Location:       req.Location,
		IsAllDay:       req.IsAllDay,
		RecurrenceRule: req.RecurrenceRule,
		TeamID:         req.TeamID,
		ProjectID:      req.ProjectID,
		GoalID:         req.GoalID,
		ReminderTime:   req.ReminderTime,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := c.calendarService.CreateEvent(event)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// If requested, sync to Google Calendar
	if req.SyncToGoogle {
		// Check if the user has connected their Google Calendar
		connection, err := c.calendarService.GetUserCalendarConnection(userID.(uuid.UUID))
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"event": event,
				"google_sync": gin.H{
					"success": false,
					"error":   "Error checking Google Calendar connection",
				},
			})
			return
		}

		if connection == nil || !connection.IsConnected {
			ctx.JSON(http.StatusOK, gin.H{
				"event": event,
				"google_sync": gin.H{
					"success": false,
					"error":   "Google Calendar not connected",
				},
			})
			return
		}

		// Create the event in Google Calendar
		err = c.calendarService.CreateGoogleCalendarEvent(userID.(uuid.UUID), event)
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"event": event,
				"google_sync": gin.H{
					"success": false,
					"error":   err.Error(),
				},
			})
			return
		}

		ctx.JSON(http.StatusCreated, gin.H{
			"event": event,
			"google_sync": gin.H{
				"success": true,
			},
		})
	} else {
		ctx.JSON(http.StatusCreated, event)
	}
}

// UpdateEvent updates an existing calendar event
func (c *CalendarController) UpdateEvent(ctx *gin.Context) {
	eventID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	var req struct {
		Title          string     `json:"title"`
		Description    string     `json:"description"`
		StartTime      time.Time  `json:"start_time"`
		EndTime        time.Time  `json:"end_time"`
		Location       string     `json:"location"`
		IsAllDay       bool       `json:"is_all_day"`
		RecurrenceRule string     `json:"recurrence_rule"`
		TeamID         *uuid.UUID `json:"team_id"`
		ProjectID      *uuid.UUID `json:"project_id"`
		GoalID         *uuid.UUID `json:"goal_id"`
		ReminderTime   *time.Time `json:"reminder_time"`
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

	// Get the existing event
	event, err := c.calendarService.GetEvent(eventID, userID.(uuid.UUID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	// Update the event
	event.Title = req.Title
	event.Description = req.Description
	event.StartTime = req.StartTime
	event.EndTime = req.EndTime
	event.Location = req.Location
	event.IsAllDay = req.IsAllDay
	event.RecurrenceRule = req.RecurrenceRule
	event.TeamID = req.TeamID
	event.ProjectID = req.ProjectID
	event.GoalID = req.GoalID
	event.ReminderTime = req.ReminderTime
	event.UpdatedAt = time.Now()

	if err := c.calendarService.UpdateEvent(event); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, event)
}

// DeleteEvent deletes a calendar event
func (c *CalendarController) DeleteEvent(ctx *gin.Context) {
	eventID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := c.calendarService.DeleteEvent(eventID, userID.(uuid.UUID)); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "event deleted"})
}

// GetEvent gets a specific calendar event
func (c *CalendarController) GetEvent(ctx *gin.Context) {
	eventID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid event ID"})
		return
	}

	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	event, err := c.calendarService.GetEvent(eventID, userID.(uuid.UUID))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}

	ctx.JSON(http.StatusOK, event)
}

// GetEvents gets all calendar events for a user within a date range
func (c *CalendarController) GetEvents(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse date range from query parameters
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date format, use YYYY-MM-DD"})
			return
		}
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date format, use YYYY-MM-DD"})
			return
		}
	}

	events, err := c.calendarService.GetEvents(userID.(uuid.UUID), startDate, endDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, events)
}

// GetGoogleAuthURL returns the URL for Google Calendar OAuth
func (c *CalendarController) GetGoogleAuthURL(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Generate a state parameter with the user ID to verify the callback
	state := userID.(uuid.UUID).String()

	authURL := c.calendarService.GetOAuthURL(state)

	ctx.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// HandleGoogleAuthCallback handles the callback from Google OAuth
func (c *CalendarController) HandleGoogleAuthCallback(ctx *gin.Context) {
	// Get the authorization code and state from the callback
	code := ctx.Query("code")
	state := ctx.Query("state")

	if code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "authorization code not provided"})
		return
	}

	// Parse the user ID from the state parameter
	userID, err := uuid.Parse(state)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid state parameter"})
		return
	}

	// Exchange the code for access and refresh tokens
	token, err := c.calendarService.ExchangeCodeForToken(code)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange authorization code: " + err.Error()})
		return
	}

	// Create a calendar client to get the primary calendar ID
	config := c.calendarService.GetOAuthConfig()
	// We create a client but don't need to store it in a variable since we use the service directly
	_ = config.Client(ctx, token)

	// Save the token first so GetCalendarClient can work
	connection := &models.UserGoogleCalendar{
		UserID:       userID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry,
		IsConnected:  true,
		LastSyncTime: time.Now(),
	}

	if err := c.calendarService.SaveUserCalendarConnection(connection); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save connection: " + err.Error()})
		return
	}

	// Now we can get a calendar service client
	srv, err := c.calendarService.GetCalendarClient(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create calendar client: " + err.Error()})
		return
	}

	// Get the primary calendar ID
	calendarList, err := srv.CalendarList.List().Do()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get calendar list: " + err.Error()})
		return
	}

	primaryCalendarID := ""
	for _, item := range calendarList.Items {
		if item.Primary {
			primaryCalendarID = item.Id
			break
		}
	}

	// Save the connection information with the primary calendar ID
	connection.PrimaryCalendarID = primaryCalendarID

	// Update the connection with the primary calendar ID
	if err := c.calendarService.SaveUserCalendarConnection(connection); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update connection with calendar ID: " + err.Error()})
		return
	}

	// Display a success message
	ctx.HTML(http.StatusOK, "oauth_success.html", gin.H{
		"message": "Google Calendar connected successfully! You can close this window.",
	})
}

// SyncWithGoogle syncs events with Google Calendar
func (c *CalendarController) SyncWithGoogle(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := c.calendarService.SyncUserEvents(userID.(uuid.UUID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "calendar synced successfully"})
}

// DisconnectGoogle disconnects Google Calendar
func (c *CalendarController) DisconnectGoogle(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := c.calendarService.DeleteUserCalendarConnection(userID.(uuid.UUID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Google Calendar disconnected"})
}

// GetGoogleConnectionStatus checks if Google Calendar is connected
func (c *CalendarController) GetGoogleConnectionStatus(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	connection, err := c.calendarService.GetUserCalendarConnection(userID.(uuid.UUID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if connection == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"is_connected": false,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"is_connected":   connection.IsConnected,
		"last_sync_time": connection.LastSyncTime,
	})
}
