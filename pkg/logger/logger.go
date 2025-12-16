package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
)

var (
	// Global logger instance
	globalLogger *zap.Logger
	sugar        *zap.SugaredLogger
)

func init() {
	globalLogger = NewLogger()
	sugar = globalLogger.Sugar()
}

// NewLogger creates a new logger based on environment variables
// Environment variables:
//   - GOVERNANCE_LOG_ENABLED: "true" to enable logging, anything else disables it (default: false)
//   - GOVERNANCE_LOG_LEVEL: "debug", "info", "warn", "error" (default: "info")
//   - GOVERNANCE_LOG_FORMAT: "json" or "console" (default: "console")
func NewLogger() *zap.Logger {
	// Check if logging is enabled
	enabled := strings.ToLower(os.Getenv("GOVERNANCE_LOG_ENABLED")) == "true"
	if !enabled {
		// Return a no-op logger that does nothing
		return zap.NewNop()
	}

	// Determine log level
	levelStr := strings.ToLower(os.Getenv("GOVERNANCE_LOG_LEVEL"))
	var level zap.AtomicLevel
	switch levelStr {
	case "debug":
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Determine format
	format := strings.ToLower(os.Getenv("GOVERNANCE_LOG_FORMAT"))
	var config zap.Config
	if format == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}
	config.Level = level

	logger, err := config.Build()
	if err != nil {
		// Fallback to no-op logger if build fails
		return zap.NewNop()
	}

	return logger
}

// Get returns the global logger instance
func Get() *zap.Logger {
	return globalLogger
}

// Sugar returns the global sugared logger instance
func Sugar() *zap.SugaredLogger {
	return sugar
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	globalLogger.Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	globalLogger.Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	globalLogger.Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	globalLogger.Error(msg, fields...)
}

// Debugf logs a debug message with formatting
func Debugf(template string, args ...interface{}) {
	sugar.Debugf(template, args...)
}

// Infof logs an info message with formatting
func Infof(template string, args ...interface{}) {
	sugar.Infof(template, args...)
}

// Warnf logs a warning message with formatting
func Warnf(template string, args ...interface{}) {
	sugar.Warnf(template, args...)
}

// Errorf logs an error message with formatting
func Errorf(template string, args ...interface{}) {
	sugar.Errorf(template, args...)
}

// Sync flushes any buffered log entries
func Sync() error {
	return globalLogger.Sync()
}

// WithFields creates a new logger with additional fields
func WithFields(fields ...zap.Field) *zap.Logger {
	return globalLogger.With(fields...)
}
