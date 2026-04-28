package logger

import (
	"flag"
	"fmt"

	"go.uber.org/zap"
)

var logger *zap.Logger

var logLevel = flag.String("logLevel", "info", "Set the log level (debug, info, warn, error)")

// Init constructs the package-level zap logger from the parsed -logLevel flag.
// Subsequent calls are no-ops; safe to call multiple times. Returns an error
// for unknown level values.
func Init() error {
	if logger != nil {
		return nil
	}

	cfg := zap.NewProductionConfig()
	switch *logLevel {
	case "debug":
		cfg.Level.SetLevel(zap.DebugLevel)
	case "info":
		cfg.Level.SetLevel(zap.InfoLevel)
	case "warn":
		cfg.Level.SetLevel(zap.WarnLevel)
	case "error":
		cfg.Level.SetLevel(zap.ErrorLevel)
	default:
		return fmt.Errorf("unknown log level %q (expected debug, info, warn, or error)", *logLevel)
	}

	l, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("build zap logger: %w", err)
	}
	logger = l
	return nil
}

func Error(msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.WithOptions(zap.AddCallerSkip(1)).Error(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.WithOptions(zap.AddCallerSkip(1)).Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	logger.WithOptions(zap.AddCallerSkip(1)).Warn(msg, fields...)
}

func With(fields ...zap.Field) *zap.Logger {
	if logger == nil {
		return zap.NewNop()
	}
	return logger.WithOptions(zap.AddCallerSkip(1)).With(fields...)
}

func Stop() {
	if logger != nil {
		_ = logger.Sync()
	}
}
