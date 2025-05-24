package routes

import (
	"assistdeck/controllers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthRoutes(r *gin.Engine, db *gorm.DB) {
	// Attach the DB to the context
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	// Define the Google login route
	r.GET("/auth/google/login", controllers.GoogleLogin)
	// Define the Google callback route (where Google redirects to after login)
	r.GET("/auth/google/callback", controllers.GoogleCallback)
}
