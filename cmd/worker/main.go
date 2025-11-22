package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

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
	cfg := config.LoadWorkerConfig()

	zapLogger, err := logger.New(os.Getenv("LOG_LEVEL"))
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer zapLogger.Sync() //nolint:errcheck

	zapLogger.Info("Starting Task Queue Worker",
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

	store, storeCleanup, err := initWorkerStorage(ctx, cfg, zapLogger)
	if err != nil {
		zapLogger.Fatal("failed to init storage", zap.Error(err))
	}
	defer storeCleanup()

	queue, brokerCleanup, err := initWorkerBroker(ctx, cfg, zapLogger)
	if err != nil {
		zapLogger.Fatal("failed to init broker", zap.Error(err))
	}
	defer brokerCleanup()

	// Initialize metrics if enabled (for future use)
	if os.Getenv("ENABLE_METRICS") == "true" {
		_ = instrumentation.NewMetrics(cfg.ServiceName())
		zapLogger.Info("Prometheus metrics enabled")
	}

	// Initialize job registry with all job types
	registry := jobs.NewRegistry()

	zapLogger.Info("Job registry initialized",
		zap.Int("job_types_count", len(registry.ListTypes())),
	)

	// Initialize worker
	w, err := worker.New(worker.Config{
		WorkerID:   cfg.WorkerID,
		Hostname:   cfg.Hostname,
		PoolSize:   cfg.PoolSize,
		MaxRetries: cfg.MaxRetries,
		Logger:     zapLogger,
		Storage:    store,
		Broker:     queue,
		Registry:   registry,
		Stream:     cfg.QueueStream,
	})
	if err != nil {
		zapLogger.Fatal("failed to configure worker", zap.Error(err))
	}

	// Start worker in goroutine
	workerDone := make(chan error, 1)
	go func() {
		workerDone <- w.Start(ctx)
	}()

	// Wait for interrupt signal or worker error
	select {
	case <-ctx.Done():
		zapLogger.Info("Shutdown signal received")
		cancel() // Trigger graceful shutdown

		// Wait for worker to finish with timeout
		shutdownTimeout := 30 * time.Second
		if cfg.ShutdownTimeout > 0 {
			shutdownTimeout = cfg.ShutdownTimeout + 5*time.Second
		}
		select {
		case err := <-workerDone:
			if err != nil {
				zapLogger.Error("Worker shutdown error", zap.Error(err))
			} else {
				zapLogger.Info("Worker shutdown completed successfully")
			}
		case <-time.After(shutdownTimeout):
			zapLogger.Warn("Worker shutdown timeout exceeded")
		}

	case err := <-workerDone:
		zapLogger.Error("Worker exited unexpectedly", zap.Error(err))
		cancel()
	}

	zapLogger.Info("Worker exited")
}

func initWorkerStorage(ctx context.Context, cfg config.WorkerConfig, logger *zap.Logger) (storage.Storage, func(), error) {
	switch cfg.StorageDriver {
	case "postgres":
		if cfg.DBURL == "" {
			logger.Warn("DATABASE_URL is empty, fallback to memory storage")
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

func initWorkerBroker(ctx context.Context, cfg config.WorkerConfig, logger *zap.Logger) (broker.Broker, func(), error) {
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

