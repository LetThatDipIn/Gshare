package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminMiddleware checks if the user has admin privileges
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user role from context (set by auth middleware)
		role, exists := c.Get("userRole")

		// Also check if user has admin flag (new admin system)
		isAdmin, adminExists := c.Get("isAdmin")

		// Allow access if user is admin in either system
		if (!exists || role != "admin") && (!adminExists || !isAdmin.(bool)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
