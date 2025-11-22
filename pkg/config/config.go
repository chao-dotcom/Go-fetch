package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// APIConfig holds settings for the REST API.
type APIConfig struct {
	Addr           string
	GRPCAddr       string
	DBURL          string
	RedisAddr      string
	RedisUsername  string
	RedisPassword  string
	RedisDB        int
	QueueStream    string
	ConsumerGroup  string
	JWTSecret      string
	StorageDriver  string
	BrokerDriver   string
	RateLimitPerMinute int
	EnableRateLimiter bool
	CORSOrigins    []string
}

// ServiceName returns the identifier used for telemetry/metrics.
func (c APIConfig) ServiceName() string {
	if name := os.Getenv("SERVICE_NAME"); name != "" {
		return name
	}
	return "taskqueue-api"
}

// WorkerConfig holds worker specific settings.
type WorkerConfig struct {
	WorkerID        string
	Hostname        string
	PoolSize        int
	MaxRetries      int
	QueueStream     string
	ConsumerGroup   string
	ShutdownTimeout time.Duration
	DBURL           string
	RedisAddr       string
	RedisUsername   string
	RedisPassword   string
	RedisDB         int
	StorageDriver   string
	BrokerDriver    string
}

// ServiceName returns the telemetry identifier for workers.
func (c WorkerConfig) ServiceName() string {
	if name := os.Getenv("SERVICE_NAME"); name != "" {
		return name
	}
	return "taskqueue-worker"
}

// LoadAPIConfig constructs an APIConfig from environment variables (with defaults for local dev).
func LoadAPIConfig() APIConfig {
	return APIConfig{
		Addr:             getEnv("API_ADDR", ":8080"),
		GRPCAddr:         getEnv("GRPC_ADDR", ":9090"),
		DBURL:            os.Getenv("DATABASE_URL"),
		RedisAddr:        getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisUsername:    getEnv("REDIS_USERNAME", ""),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:          getEnvInt("REDIS_DB", 0),
		QueueStream:      getEnv("QUEUE_STREAM", "job_stream"),
		ConsumerGroup:    getEnv("CONSUMER_GROUP", "workers"),
		JWTSecret:        getEnv("JWT_SECRET", ""), // 默认禁用 JWT（开发环境）
		StorageDriver:    getEnv("STORAGE_DRIVER", "memory"),
		BrokerDriver:     getEnv("BROKER_DRIVER", "memory"),
		RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 120),
		EnableRateLimiter: getEnvBool("ENABLE_RATE_LIMITER", false),
		CORSOrigins:       parseCORSOrigins(getEnv("CORS_ORIGINS", "*")),
	}
}

// LoadWorkerConfig builds a WorkerConfig from environment variables.
func LoadWorkerConfig() WorkerConfig {
	return WorkerConfig{
		WorkerID:        getEnv("WORKER_ID", ""),
		Hostname:        getEnv("WORKER_HOSTNAME", ""),
		PoolSize:        getEnvInt("WORKER_POOL_SIZE", 8),
		MaxRetries:      getEnvInt("WORKER_MAX_RETRIES", 5),
		QueueStream:     getEnv("QUEUE_STREAM", "job_stream"),
		ConsumerGroup:   getEnv("CONSUMER_GROUP", "workers"),
		ShutdownTimeout: time.Duration(getEnvInt("WORKER_SHUTDOWN_TIMEOUT_SECONDS", 30)) * time.Second,
		DBURL:           os.Getenv("DATABASE_URL"),
		RedisAddr:       getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisUsername:   getEnv("REDIS_USERNAME", ""),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         getEnvInt("REDIS_DB", 0),
		StorageDriver:   getEnv("STORAGE_DRIVER", "memory"),
		BrokerDriver:    getEnv("BROKER_DRIVER", "memory"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		switch val {
		case "1", "true", "TRUE", "True":
			return true
		case "0", "false", "FALSE", "False":
			return false
		}
	}
	return fallback
}

func parseCORSOrigins(val string) []string {
	if val == "" {
		return []string{"*"}
	}
	// Split by comma and trim spaces
	origins := []string{}
	for _, origin := range strings.Split(val, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

