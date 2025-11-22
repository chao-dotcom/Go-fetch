package instrumentation

import (
	"time"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware creates a Gin middleware for collecting HTTP metrics
func PrometheusMiddleware(metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method

		// Use request path if FullPath is empty (happens for 404s)
		if path == "" {
			path = c.Request.URL.Path
		}

		// Track in-flight requests
		metrics.IncrementHTTPRequestsInFlight(method)
		defer metrics.DecrementHTTPRequestsInFlight(method)

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		metrics.RecordHTTPRequest(method, path, statusCode, duration)
	}
}

