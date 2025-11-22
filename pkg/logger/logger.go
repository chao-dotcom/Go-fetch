package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a zap logger with sane defaults for both development and production modes.
func New(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.StacktraceKey = ""
	if level == "" {
		level = os.Getenv("LOG_LEVEL")
	}
	if level != "" {
		var zapLevel zapcore.Level
		if err := zapLevel.Set(level); err == nil {
			cfg.Level = zap.NewAtomicLevelAt(zapLevel)
		}
	}

	// In local environments a more human friendly logger helps.
	if os.Getenv("ENV") == "development" {
		return zap.NewDevelopment()
	}

	return cfg.Build()
}

