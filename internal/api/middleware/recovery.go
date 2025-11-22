package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery recovers from panics and logs them
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic with stack trace
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("request_id", RequestIDValue(c)),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("stack", string(debug.Stack())),
				)

				// Return 500 error
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
					"code":  "INTERNAL_ERROR",
				})

				c.Abort()
			}
		}()

		c.Next()
	}
}

