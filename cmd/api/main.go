package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"taskqueue/internal/api"
	"taskqueue/internal/broker"
	brokerMemory "taskqueue/internal/broker/memory"
	redisbroker "taskqueue/internal/broker/redis"
	"taskqueue/internal/jobs"
	"taskqueue/internal/storage"
	memory "taskqueue/internal/storage/memory"
	pgstorage "taskqueue/internal/storage/postgres"
	"taskqueue/internal/worker"
	"taskqueue/pkg/config"
	"taskqueue/pkg/instrumentation"
	"taskqueue/pkg/instrumentation/tracing"
	"taskqueue/pkg/logger"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cfg := config.LoadAPIConfig()

	zapLogger, err := logger.New(os.Getenv("LOG_LEVEL"))
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer zapLogger.Sync() //nolint:errcheck

	zapLogger.Info("Starting Task Queue API Server",
		zap.String("version", Version),
		zap.String("build_time", BuildTime),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	traceShutdown, err := tracing.Init(ctx, cfg.ServiceName())
	if err != nil {
		zapLogger.Warn("failed to init tracing", zap.Error(err))
	} else {
		defer traceShutdown(context.Background()) //nolint:errcheck
	}

	store, storeCleanup, err := initStorage(ctx, cfg, zapLogger)
	if err != nil {
		zapLogger.Fatal("failed to init storage", zap.Error(err))
	}
	defer storeCleanup()

	queue, queueCleanup, err := initBroker(ctx, cfg, zapLogger)
	if err != nil {
		zapLogger.Fatal("failed to init broker", zap.Error(err))
	}
	defer queueCleanup()

	// Initialize metrics if enabled
	var metrics *instrumentation.Metrics
	var collector *instrumentation.SystemMetricsCollector
	if os.Getenv("ENABLE_METRICS") == "true" {
		metrics = instrumentation.NewMetrics(cfg.ServiceName())
		zapLogger.Info("Prometheus metrics enabled")

		// Get database connection for metrics collector
		var sqlDB *sql.DB
		if gormStore, ok := store.(interface{ DB() (*sql.DB, error) }); ok {
			if db, err := gormStore.DB(); err == nil {
				sqlDB = db
			}
		}

		// Get Redis client for metrics collector
		var redisClient *redis.Client
		if redisBroker, ok := queue.(interface{ Client() *redis.Client }); ok {
			redisClient = redisBroker.Client()
		}

		// Start system metrics collector if we have connections
		if sqlDB != nil || redisClient != nil {
			collector = instrumentation.NewSystemMetricsCollector(
				metrics,
				sqlDB,
				redisClient,
				zapLogger,
				10*time.Second,
			)
			collectorCtx, collectorCancel := context.WithCancel(context.Background())
			defer collectorCancel()
			go collector.Start(collectorCtx)
		}
	}

	server := api.NewServer(cfg, store, queue, zapLogger, metrics)

	if os.Getenv("EMBEDDED_WORKER") == "true" {
		zapLogger.Info("starting embedded worker (memory broker + storage)")
		w, err := worker.New(worker.Config{
			Broker:   queue,
			Storage:  store,
			Registry: jobs.NewRegistry(),
			Logger:   zapLogger.Named("worker"),
			Stream:   cfg.QueueStream,
			PoolSize: 4,
		})
		if err != nil {
			zapLogger.Fatal("failed to init worker", zap.Error(err))
		}
		go func() {
			if err := w.Start(ctx); err != nil {
				zapLogger.Error("worker stopped", zap.Error(err))
				cancel()
			}
		}()
	}

	// Create HTTP server with optimized settings for high concurrency
	srv := &http.Server{
		Addr:           cfg.Addr,
		Handler:        server.Engine(),
		ReadTimeout:    30 * time.Second,  // 读取超时
		WriteTimeout:   30 * time.Second,  // 写入超时
		IdleTimeout:    120 * time.Second, // 空闲超时
		MaxHeaderBytes: 1 << 20,           // 1MB max header size
		// 启用 HTTP/2 和 Keep-Alive
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		zapLogger.Info("API server starting",
			zap.String("addr", cfg.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case <-ctx.Done():
		zapLogger.Info("Shutdown signal received")
	case err := <-serverErr:
		zapLogger.Fatal("Server error", zap.Error(err))
	}

	zapLogger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zapLogger.Error("Server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("Server exited")
}

func initStorage(ctx context.Context, cfg config.APIConfig, logger *zap.Logger) (storage.Storage, func(), error) {
	switch cfg.StorageDriver {
	case "postgres":
		if cfg.DBURL == "" {
			logger.Warn("DATABASE_URL is empty, falling back to memory storage")
			break
		}
		logger.Info("using postgres storage backend")
		store, err := pgstorage.New(ctx, cfg.DBURL, logger)
		if err != nil {
			return nil, nil, err
		}
		// Use type assertion to call Close() on concrete types
		return store, func() {
			if gormStore, ok := store.(*pgstorage.GormStore); ok {
				_ = gormStore.Close()
			} else if pgStore, ok := store.(*pgstorage.Store); ok {
				pgStore.Close()
			}
		}, nil
	}
	logger.Info("using in-memory storage backend")
	store := memory.New()
	return store, func() {}, nil
}

func initBroker(ctx context.Context, cfg config.APIConfig, logger *zap.Logger) (broker.Broker, func(), error) {
	switch cfg.BrokerDriver {
	case "redis":
		logger.Info("using redis broker backend", zap.String("addr", cfg.RedisAddr))
		b, err := redisbroker.New(ctx, &redis.Options{
			Addr:     cfg.RedisAddr,
			Username: cfg.RedisUsername,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}, cfg.QueueStream, cfg.ConsumerGroup, logger.Named("redis"))
		if err != nil {
			return nil, nil, err
		}
		return b, func() { b.Close() }, nil
	}
	logger.Info("using in-memory broker backend")
	queue := brokerMemory.New(cfg.QueueStream)
	return queue, func() {}, nil
}

