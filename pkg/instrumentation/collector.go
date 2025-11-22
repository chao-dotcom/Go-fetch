package instrumentation

import (
	"context"
	"database/sql"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// SystemMetricsCollector periodically collects system metrics
type SystemMetricsCollector struct {
	metrics     *Metrics
	db          *sql.DB
	redisClient *redis.Client
	logger      *zap.Logger
	interval    time.Duration
	stopChan    chan struct{}
}

// NewSystemMetricsCollector creates a new system metrics collector
func NewSystemMetricsCollector(
	metrics *Metrics,
	db *sql.DB,
	redisClient *redis.Client,
	logger *zap.Logger,
	interval time.Duration,
) *SystemMetricsCollector {
	if interval == 0 {
		interval = 10 * time.Second
	}

	return &SystemMetricsCollector{
		metrics:     metrics,
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		interval:    interval,
		stopChan:    make(chan struct{}),
	}
}

// Start begins collecting system metrics
func (c *SystemMetricsCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	if c.logger != nil {
		c.logger.Info("Starting system metrics collector",
			zap.Duration("interval", c.interval),
		)
	}

	// Collect immediately on start
	c.collectMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			if c.logger != nil {
				c.logger.Info("System metrics collector stopped")
			}
			return
		case <-c.stopChan:
			if c.logger != nil {
				c.logger.Info("System metrics collector stopped")
			}
			return
		case <-ticker.C:
			c.collectMetrics(ctx)
		}
	}
}

// Stop stops the metrics collector
func (c *SystemMetricsCollector) Stop() {
	close(c.stopChan)
}

// collectMetrics collects all system metrics
func (c *SystemMetricsCollector) collectMetrics(ctx context.Context) {
	// Collect database connection pool stats
	if c.db != nil {
		stats := c.db.Stats()
		c.metrics.SetDatabaseConnectionsActive(stats.InUse)
	}

	// Collect Redis connection pool stats
	if c.redisClient != nil {
		poolStats := c.redisClient.PoolStats()
		c.metrics.SetRedisConnectionsActive(int(poolStats.TotalConns))
	}
}

