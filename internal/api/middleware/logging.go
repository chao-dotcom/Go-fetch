package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"taskqueue/pkg/metrics"
)

// Logger creates a request logging middleware
func Logger(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		requestID := RequestIDValue(c)

		// Log request
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// Add error if exists
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		// Record metrics
		statusStr := strconv.Itoa(statusCode)
		metrics.ObserveAPIRequest(method, path, statusStr, latency)

		// Log level based on status code
		switch {
		case statusCode >= 500:
			logger.Error("Server error", fields...)
		case statusCode >= 400:
			logger.Warn("Client error", fields...)
		default:
			logger.Info("Request completed", fields...)
		}
	}
}

