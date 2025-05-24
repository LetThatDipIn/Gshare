package main

import (
	"assistdeck/config"
	"assistdeck/controllers"
	"assistdeck/models"
	"assistdeck/routes"
	"assistdeck/services"
	"assistdeck/utils"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables from .env file
	config.LoadEnvVariables()

	// Load configuration
	cfg := config.LoadConfig()

	// Debug environment variables
	fmt.Println("Environment variables loaded:")
	fmt.Println("- DATABASE_URL:", maskString(cfg.DatabaseURL))
	fmt.Println("- GOOGLE_CLIENT_ID:", maskString(cfg.GoogleClientID))
	fmt.Println("- GOOGLE_CLIENT_SECRET:", maskString(cfg.GoogleClientSecret))
	fmt.Println("- JWT_SECRET:", maskString(cfg.JWTSecret))

	// Initialize Google OAuth config
	controllers.InitGoogleOAuth()

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	fmt.Println("✅ Connected to database successfully")

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Subscription{},
		&models.Team{},
		&models.TeamMember{},
		&models.Goal{},
		&models.Project{},
		&models.ChatSession{},
		&models.ChatMessage{},
		&models.ChatParticipant{},
		&models.Notification{},
		&models.CalendarEvent{},
		&models.AdminSettings{},
		&models.GeminiMessage{},
		&models.GeminiSession{},
		&models.UserGoogleCalendar{},
		// MediaFile and AudioFile models removed as per request
	)
	if err != nil {
		log.Fatalf("❌ Failed to migrate database: %v", err)
	}

	fmt.Println("✅ Database migrations completed")

	// Initialize services
	authService := services.NewAuthService(db, cfg)
	userService := services.NewUserService(db)
	teamService := services.NewTeamService(db)
	goalService := services.NewGoalService(db)
	projectService := services.NewProjectService(db)
	chatService := services.NewChatService(db)
	geminiService := services.NewGeminiService(db, cfg.GeminiAPIKey)
	notificationService := services.NewNotificationService(db)
	calendarService := services.NewCalendarService(
		db,
		cfg.GoogleCalendarClientID,
		cfg.GoogleCalendarClientSecret,
		cfg.GoogleCalendarRedirectURL,
	)
	adminService := services.NewAdminService(db, cfg.JWTSecret)

	// Initialize controllers
	authController := controllers.NewAuthController(authService)
	userController := controllers.NewUserController(userService)
	teamController := controllers.NewTeamController(teamService)
	goalController := controllers.NewGoalController(goalService)
	projectController := controllers.NewProjectController(projectService)
	chatController := controllers.NewChatController(chatService)
	geminiController := controllers.NewGeminiController(geminiService)
	notificationController := controllers.NewNotificationController(notificationService)
	calendarController := controllers.NewCalendarController(calendarService)
	adminController := controllers.NewAdminController(adminService, userService)
	// subscriptionController := controllers.NewSubscriptionController(services.NewSubscriptionService(db))

	// Initialize WebSocket manager
	wsManager := utils.NewManager()

	// Set database reference for WebSocket manager
	utils.SetDB(db)

	go wsManager.Start()
	// Setup router
	router := gin.Default()

	// Apply CORS middleware
	router.Use(utils.CORSMiddleware())

	// Create auth middleware
	authMiddleware := utils.JWTMiddleware(cfg.JWTSecret)

	// Setup routes - using existing route handlers temporarily to ensure it compiles
	// Phase 1: Add database context for legacy routes
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// Phase 1 routes - legacy routes
	// NOTE: AuthRoutes commented out to avoid route conflicts with new controller routes
	// routes.AuthRoutes(router, db)
	routes.UserRoutes(router, db)

	// Phase 2 routes - new controller-based architecture
	routes.SetupAuthRoutes(router, authController)
	routes.SetupUserRoutes(router, userController, authMiddleware)
	routes.SetupTeamRoutes(router, teamController, authMiddleware)
	routes.SetupGoalRoutes(router, goalController, authMiddleware)
	routes.SetupProjectRoutes(router, projectController, authMiddleware)
	routes.SetupChatRoutes(router, chatController, wsManager, authMiddleware)
	// New feature routes
	routes.SetupGeminiRoutes(router, geminiController, authMiddleware)
	routes.SetupNotificationRoutes(router, notificationController, authMiddleware)
	routes.SetupCalendarRoutes(router, calendarController, authMiddleware)
	routes.SetupAdminRoutes(router, adminController, authMiddleware)
	// Payment and subscription routes removed as per request

	// Media routes removed as per request

	// Load HTML templates
	router.LoadHTMLGlob("templates/*")

	// Debug routes - for development only
	routes.SetupDebugRoutes(router, authMiddleware)

	// Serve HTML test page (only in development)
	router.StaticFile("/chat-test", "./chat_test.html")

	// Define the frontend build directory
	frontendBuildDir := "./templates/public" // Adjust this path to your actual build directory

	// Create a file server handler for the static files
	fs := http.FileServer(http.Dir(frontendBuildDir))

	// Handle all static file requests
	router.GET("/static/*filepath", func(c *gin.Context) {
		c.Request.URL.Path = c.Param("filepath")
		fs.ServeHTTP(c.Writer, c.Request)
	})

	// Serve index.html for any other routes (SPA handling)
	router.NoRoute(func(c *gin.Context) {
		// Only serve the index file for non-API routes
		if c.Request.Method == "GET" && !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			indexFile := filepath.Join(frontendBuildDir, "index.html")
			if _, err := os.Stat(indexFile); err == nil {
				c.File(indexFile)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	})

	// Start server
	fmt.Println("🚀 Server running on port", cfg.Port)
	router.Run(":" + cfg.Port)
}

// Helper function to mask sensitive strings for logging
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	visible := len(s) / 4
	return s[:visible] + "****" + s[len(s)-visible:]
}
