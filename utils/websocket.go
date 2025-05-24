// utils/websocket.go
package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// In production, check origin against allowed domains
		return true
	},
	// Add error handling for handshake failures
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		log.Printf("WebSocket upgrade error: %v, status: %d", reason, status)
	},
}

// Client represents a websocket client
type Client struct {
	ID           uuid.UUID
	Conn         *websocket.Conn
	SessionID    uuid.UUID
	Send         chan []byte
	LastActivity time.Time
}

// Manager manages websocket clients
type Manager struct {
	clients           map[*Client]bool
	register          chan *Client
	unregister        chan *Client
	broadcast         chan []byte
	sessions          map[uuid.UUID]map[*Client]bool
	mutex             sync.RWMutex
	healthCheckTicker *time.Ticker
	maxInactivity     time.Duration
}

// NewManager creates a new websocket manager
func NewManager() *Manager {
	manager := &Manager{
		clients:       make(map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		broadcast:     make(chan []byte),
		sessions:      make(map[uuid.UUID]map[*Client]bool),
		maxInactivity: 5 * time.Minute, // Set maximum inactivity time to 5 minutes
	}

	// Start health check ticker
	manager.healthCheckTicker = time.NewTicker(1 * time.Minute)
	go manager.runHealthCheck()

	return manager
}

// Start starts the websocket manager
func (m *Manager) Start() {
	for {
		select {
		case client := <-m.register:
			m.mutex.Lock()
			m.clients[client] = true

			// Add client to session
			if _, ok := m.sessions[client.SessionID]; !ok {
				m.sessions[client.SessionID] = make(map[*Client]bool)
			}
			m.sessions[client.SessionID][client] = true
			m.mutex.Unlock()

		case client := <-m.unregister:
			m.mutex.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				close(client.Send)

				// Remove from session
				if _, ok := m.sessions[client.SessionID]; ok {
					delete(m.sessions[client.SessionID], client)
					if len(m.sessions[client.SessionID]) == 0 {
						delete(m.sessions, client.SessionID)
					}
				}
			}
			m.mutex.Unlock()

		case message := <-m.broadcast:
			m.mutex.RLock()
			for client := range m.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(m.clients, client)
				}
			}
			m.mutex.RUnlock()
		}
	}
}

// cleanupInactiveConnections removes connections that have been inactive for too long
func (m *Manager) cleanupInactiveConnections() {
	now := time.Now()
	inactiveClients := []*Client{}

	// First, identify inactive clients
	m.mutex.RLock()
	for client := range m.clients {
		if now.Sub(client.LastActivity) > m.maxInactivity {
			inactiveClients = append(inactiveClients, client)
		}
	}
	m.mutex.RUnlock()

	// Then clean them up
	if len(inactiveClients) > 0 {
		log.Printf("Cleaning up %d inactive WebSocket connections", len(inactiveClients))
		for _, client := range inactiveClients {
			// Close connection with appropriate code
			closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Connection inactive")
			client.Conn.WriteMessage(websocket.CloseMessage, closeMsg)
			m.unregister <- client
		}
	}
}

// runHealthCheck periodically checks for inactive connections
func (m *Manager) runHealthCheck() {
	for range m.healthCheckTicker.C {
		m.cleanupInactiveConnections()
	}
}

// BroadcastToSession sends a message to all clients in a session
func (m *Manager) BroadcastToSession(sessionID uuid.UUID, message []byte) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if clients, ok := m.sessions[sessionID]; ok {
		log.Printf("Broadcasting message to %d clients in session %s", len(clients), sessionID.String())
		for client := range clients {
			select {
			case client.Send <- message:
				// Message sent successfully to channel
			default:
				// Client send buffer is full or disconnected
				log.Printf("Failed to broadcast to client %s (buffer full or disconnected)", client.ID.String())
				close(client.Send)
				delete(m.clients, client)
				delete(clients, client)
			}
		}
	} else {
		log.Printf("Session %s not found for broadcasting", sessionID.String())
	}
}

// ServeWs handles websocket requests from clients
func (m *Manager) ServeWs(c *gin.Context) {
	// The JWT middleware should have already authenticated the user
	// and set the userID in the context, regardless of whether the token
	// was provided in the Authorization header or the query parameter

	userIDVal, exists := c.Get("userID")
	if !exists {
		// Check both userID (new format) and user_id (legacy format)
		userIDVal, exists = c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized - authentication required"})
			return
		}
	}

	// Convert to UUID
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		// Try to convert from string if it's not a UUID already
		if userIDStr, isStr := userIDVal.(string); isStr {
			var err error
			userID, err = uuid.Parse(userIDStr)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID format"})
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID type"})
			return
		}
	}

	// Log the user ID for debugging
	log.Printf("WebSocket connection attempt with userID: %v (type: %T)", userID, userID)

	// Handle the WebSocket connection
	handleWebSocketConnection(m, c, userID)
}

// Helper function to handle WebSocket connection after authentication
func handleWebSocketConnection(m *Manager, c *gin.Context, userID uuid.UUID) {
	sessionIDStr := c.Query("session_id")
	if sessionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	log.Printf("WebSocket connection request with session_id: %s for user: %s", sessionIDStr, userID.String())

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session_id"})
		return
	}

	// Check if session exists and user has access
	if DB != nil {
		log.Printf("DEBUG: Checking access for session: %s, user: %s", sessionID.String(), userID.String())

		// First, check if user is the session owner
		var ownerCount int64
		if err := DB.Table("chat_sessions").
			Where("id = ? AND user_id = ?", sessionID, userID).
			Count(&ownerCount).Error; err != nil {
			log.Printf("DB error checking session ownership: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify session ownership"})
			return
		}

		if ownerCount > 0 {
			log.Printf("Access granted: User %s is the owner of session %s", userID.String(), sessionID.String())
			log.Printf("Session access granted for user %s to session %s", userID.String(), sessionID.String())
		} else {
			// If not the owner, check if user is a participant
			var participantCount int64
			if err := DB.Table("chat_participants").
				Where("session_id = ? AND user_id = ?", sessionID, userID).
				Count(&participantCount).Error; err != nil {
				log.Printf("DB error checking participant access: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify participant access"})
				return
			}

			if participantCount > 0 {
				log.Printf("Access granted: User %s is a participant in session %s", userID.String(), sessionID.String())
				log.Printf("Session access granted for user %s to session %s", userID.String(), sessionID.String())
			} else {
				log.Printf("Access denied: User %s is neither the owner nor a participant of session %s", userID.String(), sessionID.String())
				c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this chat session"})
				return
			}
		}
	} else {
		log.Printf("Warning: DB not set, skipping session access verification")
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Error upgrading connection to WebSocket: %v", err)
		return
	}

	// Enable TCP keepalive on the underlying connection
	if tcpConn, ok := conn.UnderlyingConn().(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			log.Printf("Warning: Failed to enable TCP keepalive: %v", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			log.Printf("Warning: Failed to set TCP keepalive period: %v", err)
		}
	}

	log.Printf("WebSocket connection established for user %s, session %s", userID.String(), sessionID.String())

	client := &Client{
		ID:           userID,
		Conn:         conn,
		SessionID:    sessionID,
		Send:         make(chan []byte, 256),
		LastActivity: time.Now(),
	}

	// Register client asynchronously to avoid blocking the HTTP handler
	go func() {
		m.register <- client
		log.Printf("Client %s registered for session %s", userID.String(), sessionID.String())
	}()

	// Start goroutines for reading and writing
	go client.writePump(m)
	go client.readPump(m)
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump(manager *Manager) {
	// Send ping messages every 15 seconds for better keep-alive behavior
	pingTicker := time.NewTicker(15 * time.Second)
	// Log activity periodically
	logTicker := time.NewTicker(30 * time.Second)

	defer func() {
		pingTicker.Stop()
		logTicker.Stop()
		log.Printf("writePump ending for user %s in session %s", c.ID.String(), c.SessionID.String())
		c.Conn.Close()
	}()

	// Send an initial ping to establish the connection properly
	if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
		log.Printf("Initial ping failed: %v", err)
		return
	}

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if !ok {
				// Channel was closed
				log.Printf("Send channel closed for user %s", c.ID.String())
				c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error getting writer: %v", err)
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				log.Printf("Error closing writer: %v", err)
				return
			}

			// Update activity time
			c.LastActivity = time.Now()

		case <-pingTicker.C:
			// Send websocket ping message - crucial for keeping the connection alive
			c.Conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("Error sending ping: %v", err)
				return
			}

		case <-logTicker.C:
			// Log connection status for debugging
			log.Printf("WebSocket connection active for user %s in session %s (last activity: %v)",
				c.ID.String(), c.SessionID.String(), time.Since(c.LastActivity))
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump(manager *Manager) {
	defer func() {
		log.Printf("readPump ending for user %s in session %s", c.ID.String(), c.SessionID.String())
		manager.unregister <- c
		c.Conn.Close()
	}()

	// Increase read buffer limit to accommodate larger messages
	c.Conn.SetReadLimit(8192)
	// Set initial read deadline
	c.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))

	// Set pong handler to extend read deadline when pong is received
	c.Conn.SetPongHandler(func(string) error {
		// Log pong received for debugging
		log.Printf("Pong received from user %s", c.ID.String())
		// Extend the read deadline when a pong is received
		c.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		c.LastActivity = time.Now()
		return nil
	})

	// Set close handler to log connection closures
	c.Conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket connection closed for user %s: code %d, reason: %s",
			c.ID.String(), code, text)
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error for user %s in session %s: %v",
					c.ID.String(), c.SessionID.String(), err)
			}
			break
		}

		c.LastActivity = time.Now()

		// Process the message and save to database
		processedMessage, err := manager.ProcessMessage(c, message)
		if err != nil {
			log.Printf("Error processing message: %v", err)
			// Send error back to client
			errorMsg := fmt.Sprintf(`{"error": "%s"}`, err.Error())
			c.Send <- []byte(errorMsg)
			continue
		}

		// Broadcast the processed message to all clients in this session
		manager.BroadcastToSession(c.SessionID, processedMessage)
	}
}

// ChatMessageData represents the structure of a chat message sent over WebSocket
type ChatMessageData struct {
	Content     string  `json:"content"`
	FileURL     *string `json:"file_url,omitempty"`
	IsAIMessage bool    `json:"is_ai_message"`
}

// DB reference for saving messages
var DB *gorm.DB

// SetDB sets the database connection for the websocket manager
func SetDB(db *gorm.DB) {
	DB = db
}

// ProcessMessage processes an incoming message, saves it to DB, and returns the processed message
func (m *Manager) ProcessMessage(client *Client, message []byte) ([]byte, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection not set")
	}

	// Parse the incoming message
	var messageData ChatMessageData
	if err := json.Unmarshal(message, &messageData); err != nil {
		return nil, err
	}

	// Create a new chat message
	chatMessage := struct {
		ID          uuid.UUID `json:"id"`
		SessionID   uuid.UUID `json:"session_id"`
		SenderID    uuid.UUID `json:"sender_id"`
		Content     string    `json:"content"`
		FileURL     *string   `json:"file_url,omitempty"`
		IsAIMessage bool      `json:"is_ai_message"`
		CreatedAt   time.Time `json:"created_at"`
	}{
		ID:          uuid.New(),
		SessionID:   client.SessionID,
		SenderID:    client.ID,
		Content:     messageData.Content,
		FileURL:     messageData.FileURL,
		IsAIMessage: messageData.IsAIMessage,
		CreatedAt:   time.Now(),
	}

	// Save to the database
	dbMessage := struct {
		ID          uuid.UUID `gorm:"type:uuid;primary_key"`
		SessionID   uuid.UUID `gorm:"type:uuid"`
		SenderID    uuid.UUID `gorm:"type:uuid"`
		Content     string
		FileURL     *string
		IsAIMessage bool
		CreatedAt   time.Time
	}{
		ID:          chatMessage.ID,
		SessionID:   chatMessage.SessionID,
		SenderID:    chatMessage.SenderID,
		Content:     chatMessage.Content,
		FileURL:     chatMessage.FileURL,
		IsAIMessage: chatMessage.IsAIMessage,
		CreatedAt:   chatMessage.CreatedAt,
	}

	if err := DB.Table("chat_messages").Create(&dbMessage).Error; err != nil {
		return nil, err
	}

	// Marshal the processed message back to JSON
	processedMessage, err := json.Marshal(chatMessage)
	if err != nil {
		return nil, err
	}

	return processedMessage, nil
}

// ValidateToken validates a JWT token and returns the user ID
func ValidateToken(tokenString, jwtSecret string) (uuid.UUID, error) {
	// Parse and validate the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return the secret key
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid token: %v", err)
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Get user ID from either "user_id" (old format) or "sub" (new format)
		var userIDStr string
		if claims["user_id"] != nil {
			// Convert to string if it's not already
			switch v := claims["user_id"].(type) {
			case string:
				userIDStr = v
			default:
				userIDStr = fmt.Sprintf("%v", v)
			}
		} else if claims["sub"] != nil {
			// Convert to string if it's not already
			switch v := claims["sub"].(type) {
			case string:
				userIDStr = v
			default:
				userIDStr = fmt.Sprintf("%v", v)
			}
		} else {
			return uuid.Nil, fmt.Errorf("token missing user identifier")
		}

		// Parse UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid user ID format: %v", err)
		}
		return userID, nil
	} else {
		return uuid.Nil, fmt.Errorf("invalid token claims")
	}
}
