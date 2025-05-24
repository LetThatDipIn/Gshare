package services

import (
	"assistdeck/models"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

type CalendarService struct {
	DB                 *gorm.DB
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

func NewCalendarService(db *gorm.DB, googleClientID, googleClientSecret, googleRedirectURL string) *CalendarService {
	return &CalendarService{
		DB:                 db,
		GoogleClientID:     googleClientID,
		GoogleClientSecret: googleClientSecret,
		GoogleRedirectURL:  googleRedirectURL,
	}
}

// CreateEvent creates a new calendar event
func (s *CalendarService) CreateEvent(event *models.CalendarEvent) error {
	return s.DB.Create(event).Error
}

// UpdateEvent updates an existing calendar event
func (s *CalendarService) UpdateEvent(event *models.CalendarEvent) error {
	return s.DB.Save(event).Error
}

// DeleteEvent deletes a calendar event
func (s *CalendarService) DeleteEvent(eventID, userID uuid.UUID) error {
	result := s.DB.Where("id = ? AND user_id = ?", eventID, userID).Delete(&models.CalendarEvent{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// GetEvent retrieves a specific calendar event
func (s *CalendarService) GetEvent(eventID, userID uuid.UUID) (*models.CalendarEvent, error) {
	var event models.CalendarEvent
	if err := s.DB.Where("id = ? AND user_id = ?", eventID, userID).First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// GetEvents retrieves events for a user with optional filters
func (s *CalendarService) GetEvents(userID uuid.UUID, startDate, endDate time.Time) ([]models.CalendarEvent, error) {
	var events []models.CalendarEvent
	query := s.DB.Where("user_id = ?", userID)

	if !startDate.IsZero() {
		query = query.Where("start_time >= ?", startDate)
	}

	if !endDate.IsZero() {
		query = query.Where("end_time <= ?", endDate)
	}

	if err := query.Order("start_time asc").Find(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}

// GetUserCalendarConnection gets the Google Calendar connection for a user
func (s *CalendarService) GetUserCalendarConnection(userID uuid.UUID) (*models.UserGoogleCalendar, error) {
	var connection models.UserGoogleCalendar
	err := s.DB.Where("user_id = ?", userID).First(&connection).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No connection found, but not an error
		}
		return nil, err
	}
	return &connection, nil
}

// SaveUserCalendarConnection saves or updates the Google Calendar connection
func (s *CalendarService) SaveUserCalendarConnection(connection *models.UserGoogleCalendar) error {
	// Check if a connection already exists
	var existingConn models.UserGoogleCalendar
	result := s.DB.Where("user_id = ?", connection.UserID).First(&existingConn)

	if result.Error == nil {
		// Update existing connection
		connection.ID = existingConn.ID
		connection.UpdatedAt = time.Now()
		return s.DB.Save(connection).Error
	} else if result.Error == gorm.ErrRecordNotFound {
		// Create new connection
		connection.CreatedAt = time.Now()
		connection.UpdatedAt = time.Now()
		return s.DB.Create(connection).Error
	} else {
		return result.Error
	}
}

// DeleteUserCalendarConnection removes a user's Google Calendar connection
func (s *CalendarService) DeleteUserCalendarConnection(userID uuid.UUID) error {
	return s.DB.Where("user_id = ?", userID).Delete(&models.UserGoogleCalendar{}).Error
}

// GetOAuthConfig returns the OAuth2 config for Google Calendar
func (s *CalendarService) GetOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.GoogleClientID,
		ClientSecret: s.GoogleClientSecret,
		RedirectURL:  s.GoogleRedirectURL,
		Scopes: []string{
			calendar.CalendarScope,
		},
		Endpoint: google.Endpoint,
	}
}

// GetOAuthURL returns the URL for the user to authorize with Google Calendar
func (s *CalendarService) GetOAuthURL(state string) string {
	config := s.GetOAuthConfig()
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCodeForToken exchanges an authorization code for access and refresh tokens
func (s *CalendarService) ExchangeCodeForToken(code string) (*oauth2.Token, error) {
	config := s.GetOAuthConfig()
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// GetTokenFromDB retrieves and constructs an OAuth token from stored credentials
func (s *CalendarService) GetTokenFromDB(userID uuid.UUID) (*oauth2.Token, error) {
	var connection models.UserGoogleCalendar
	if err := s.DB.Where("user_id = ?", userID).First(&connection).Error; err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  connection.AccessToken,
		RefreshToken: connection.RefreshToken,
		Expiry:       connection.TokenExpiry,
		TokenType:    "Bearer",
	}, nil
}

// GetCalendarClient returns a Google Calendar client for a user
func (s *CalendarService) GetCalendarClient(userID uuid.UUID) (*calendar.Service, error) {
	token, err := s.GetTokenFromDB(userID)
	if err != nil {
		return nil, err
	}

	config := s.GetOAuthConfig()
	client := config.Client(context.Background(), token)

	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return srv, nil
}

// SyncUserEvents syncs events from Google Calendar to the local database
func (s *CalendarService) SyncUserEvents(userID uuid.UUID) error {
	// Get the calendar connection
	connection, err := s.GetUserCalendarConnection(userID)
	if err != nil {
		return err
	}
	if connection == nil || !connection.IsConnected {
		return fmt.Errorf("User does not have a connected Google Calendar")
	}

	// Get Google Calendar client
	calendarClient, err := s.GetCalendarClient(userID)
	if err != nil {
		return err
	}

	// Set time range for sync (1 month back to 6 months forward)
	timeMin := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
	timeMax := time.Now().AddDate(0, 6, 0).Format(time.RFC3339)

	// Get events from Google Calendar
	events, err := calendarClient.Events.List(connection.PrimaryCalendarID).
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return err
	}

	// Begin a transaction
	tx := s.DB.Begin()

	// Process each event
	for _, item := range events.Items {
		// Skip events without an ID
		if item.Id == "" {
			continue
		}

		// Check if event already exists
		var existingEvent models.CalendarEvent
		result := tx.Where("google_event_id = ? AND user_id = ?", item.Id, userID).First(&existingEvent)

		// Parse start and end times
		var startTime, endTime time.Time
		var isAllDay bool

		if item.Start.DateTime != "" {
			startTime, _ = time.Parse(time.RFC3339, item.Start.DateTime)
			endTime, _ = time.Parse(time.RFC3339, item.End.DateTime)
			isAllDay = false
		} else {
			// All-day event
			startTime, _ = time.Parse("2006-01-02", item.Start.Date)
			endTime, _ = time.Parse("2006-01-02", item.End.Date)
			isAllDay = true
		}

		if result.Error == nil {
			// Update existing event
			existingEvent.Title = item.Summary
			existingEvent.Description = item.Description
			existingEvent.StartTime = startTime
			existingEvent.EndTime = endTime
			existingEvent.Location = item.Location
			existingEvent.IsAllDay = isAllDay
			existingEvent.UpdatedAt = time.Now()

			if err := tx.Save(&existingEvent).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else if result.Error == gorm.ErrRecordNotFound {
			// Create new event
			newEvent := models.CalendarEvent{
				UserID:        userID,
				GoogleEventID: item.Id,
				Title:         item.Summary,
				Description:   item.Description,
				StartTime:     startTime,
				EndTime:       endTime,
				Location:      item.Location,
				IsAllDay:      isAllDay,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			if err := tx.Create(&newEvent).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else {
			// Other DB error
			tx.Rollback()
			return result.Error
		}
	}

	// Update last sync time
	if err := tx.Model(&models.UserGoogleCalendar{}).
		Where("user_id = ?", userID).
		Update("last_sync_time", time.Now()).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	return tx.Commit().Error
}

// CreateGoogleCalendarEvent creates an event in Google Calendar
func (s *CalendarService) CreateGoogleCalendarEvent(userID uuid.UUID, event *models.CalendarEvent) error {
	// Get the calendar connection
	connection, err := s.GetUserCalendarConnection(userID)
	if err != nil {
		return err
	}
	if connection == nil || !connection.IsConnected {
		return fmt.Errorf("User does not have a connected Google Calendar")
	}

	// Get Google Calendar client
	calendarClient, err := s.GetCalendarClient(userID)
	if err != nil {
		return err
	}

	// Create Google Calendar event
	googleEvent := &calendar.Event{
		Summary:     event.Title,
		Description: event.Description,
		Location:    event.Location,
	}

	if event.IsAllDay {
		date := event.StartTime.Format("2006-01-02")
		endDate := event.EndTime.Format("2006-01-02")
		googleEvent.Start = &calendar.EventDateTime{
			Date: date,
		}
		googleEvent.End = &calendar.EventDateTime{
			Date: endDate,
		}
	} else {
		googleEvent.Start = &calendar.EventDateTime{
			DateTime: event.StartTime.Format(time.RFC3339),
		}
		googleEvent.End = &calendar.EventDateTime{
			DateTime: event.EndTime.Format(time.RFC3339),
		}
	}

	// Add recurrence rule if specified
	if event.RecurrenceRule != "" {
		googleEvent.Recurrence = []string{event.RecurrenceRule}
	}

	// Add reminder if specified
	if event.ReminderTime != nil {
		minutes := int64(time.Until(*event.ReminderTime).Minutes())
		googleEvent.Reminders = &calendar.EventReminders{
			UseDefault: false,
			Overrides: []*calendar.EventReminder{
				{
					Method:  "popup",
					Minutes: minutes,
				},
			},
		}
	}

	// Insert event into Google Calendar
	createdEvent, err := calendarClient.Events.Insert(connection.PrimaryCalendarID, googleEvent).Do()
	if err != nil {
		return err
	}

	// Update local event with Google event ID
	event.GoogleEventID = createdEvent.Id

	return s.DB.Save(event).Error
}
