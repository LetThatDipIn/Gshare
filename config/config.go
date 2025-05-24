// config/config.go
package config

import (
	"os"
)

type Config struct {
	Port                       string
	DatabaseURL                string
	JWTSecret                  string
	GoogleClientID             string
	GoogleClientSecret         string
	GeminiAPIKey               string
	GoogleCalendarClientID     string
	GoogleCalendarClientSecret string
	GoogleCalendarRedirectURL  string
}

func LoadConfig() *Config {
	return &Config{
		Port:                       getEnv("PORT", "8080"),
		DatabaseURL:                getEnv("DATABASE_URL", "postgres://assistuser:ASSISTDECK@localhost:5432/assistdeck?sslmode=disable"),
		JWTSecret:                  getEnv("JWT_SECRET", "your-secret-key"),
		GoogleClientID:             getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:         getEnv("GOOGLE_CLIENT_SECRET", ""),
		GeminiAPIKey:               getEnv("GEMINI_API_KEY", ""),
		GoogleCalendarClientID:     getEnv("GOOGLE_CALENDAR_CLIENT_ID", ""),
		GoogleCalendarClientSecret: getEnv("GOOGLE_CALENDAR_CLIENT_SECRET", ""),
		GoogleCalendarRedirectURL:  getEnv("GOOGLE_CALENDAR_REDIRECT_URL", "http://localhost:8080/api/calendar/auth/callback"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
