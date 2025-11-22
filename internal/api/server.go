package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"

	"taskqueue/internal/api/handlers"
	"taskqueue/internal/api/middleware"
	"taskqueue/internal/broker"
	"taskqueue/internal/storage"
	"taskqueue/pkg/config"
	"taskqueue/pkg/instrumentation"
)

// Server wires HTTP handlers, storage, and broker.
type Server struct {
	engine *gin.Engine
	store  storage.Storage
	broker broker.Broker
	logger *zap.Logger
	cfg    config.APIConfig
}

// NewServer creates an HTTP server.
// metrics is optional - if provided, uses the new instrumentation metrics system.
func NewServer(cfg config.APIConfig, store storage.Storage, broker broker.Broker, logger *zap.Logger, metrics *instrumentation.Metrics) *Server {
	if logger == nil {
		logger, _ = zap.NewDevelopment()
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.RequestID())
	
	// Use new Prometheus middleware if metrics provided, otherwise use existing logger
	if metrics != nil {
		router.Use(instrumentation.PrometheusMiddleware(metrics))
	} else {
		router.Use(middleware.Logger(logger))
	}
	
	router.Use(otelgin.Middleware(cfg.ServiceName()))
	
	// CORS configuration
	corsOrigins := []string{"*"} // Default to allow all
	if cfg.CORSOrigins != nil && len(cfg.CORSOrigins) > 0 {
		corsOrigins = cfg.CORSOrigins
	}
	router.Use(middleware.CORS(corsOrigins))
	
	if cfg.EnableRateLimiter {
		router.Use(middleware.RateLimit(cfg.RateLimitPerMinute))
	}

	s := &Server{
		engine: router,
		store:  store,
		broker: broker,
		logger: logger,
		cfg:    cfg,
	}

	jobHandler := handlers.NewJobHandler(store, broker, logger, cfg.QueueStream, nil)
	api := router.Group("/v1")
	if cfg.JWTSecret != "" {
		api.Use(middleware.JWTAuth(cfg.JWTSecret))
	}
	api.POST("/jobs", jobHandler.CreateJob)
	api.GET("/jobs", jobHandler.ListJobs)
	api.GET("/jobs/:id", jobHandler.GetJob)
	api.DELETE("/jobs/:id", jobHandler.CancelJob)
	api.POST("/jobs/:id/retry", jobHandler.RetryJob)
	api.GET("/jobs/:id/logs", jobHandler.GetJobLogs)
	api.GET("/queues", jobHandler.QueueStats)
	api.GET("/workers", jobHandler.ListWorkers)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().UTC()})
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	return s
}

// Engine returns the underlying Gin engine (for use in main.go)
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Run starts serving on the provided address.
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

