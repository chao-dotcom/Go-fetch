package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "request_id"

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists in header
		requestID := c.GetHeader("X-Request-ID")

		// Generate new ID if not present
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in context and response header
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// RequestIDValue returns the request identifier stored in the Gin context.
func RequestIDValue(c *gin.Context) string {
	if val, ok := c.Get(requestIDKey); ok {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

