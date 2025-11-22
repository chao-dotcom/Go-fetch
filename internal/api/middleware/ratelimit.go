package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter using token bucket algorithm
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	r := rate.Limit(float64(requestsPerMinute) / 60.0) // Per second
	burst := requestsPerMinute / 10
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst, // 10% burst capacity
	}
}

// getLimiter gets or creates limiter for IP
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// RateLimit middleware function using token bucket algorithm
func RateLimit(requestsPerMinute int) gin.HandlerFunc {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 120
	}
	limiter := NewRateLimiter(requestsPerMinute)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.getLimiter(ip).Allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
				"code":  "RATE_LIMITED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitWithDuration creates rate limit middleware with custom duration
func RateLimitWithDuration(limit int, within time.Duration) gin.HandlerFunc {
	if limit <= 0 {
		limit = 120
	}
	if within <= 0 {
		within = time.Minute
	}

	// Convert to requests per minute for consistency
	requestsPerMinute := int(float64(limit) * (time.Minute.Seconds() / within.Seconds()))
	return RateLimit(requestsPerMinute)
}

