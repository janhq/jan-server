package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var globalLogger zerolog.Logger

func init() {
	// Default logger to stdout with pretty printing
	globalLogger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Logger()
}

// Init initializes the global logger with the given configuration
func Init(level string, format string, output io.Writer) {
	if output == nil {
		output = os.Stdout
	}

	var writer io.Writer = output
	if format == "console" || format == "pretty" {
		writer = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}
	}

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	globalLogger = zerolog.New(writer).
		Level(lvl).
		With().
		Timestamp().
		Logger()
}

// Get returns the global logger
func Get() *zerolog.Logger {
	return &globalLogger
}

// WithContext returns a logger with context values
func WithContext(ctx context.Context) *zerolog.Logger {
	logger := globalLogger.With().Logger()

	if requestID, ok := ctx.Value("request_id").(string); ok {
		logger = logger.With().Str("request_id", requestID).Logger()
	}

	if userID, ok := ctx.Value("user_id").(string); ok {
		logger = logger.With().Str("user_id", userID).Logger()
	}

	return &logger
}

// Debug logs a debug message
func Debug() *zerolog.Event {
	return globalLogger.Debug()
}

// Info logs an info message
func Info() *zerolog.Event {
	return globalLogger.Info()
}

// Warn logs a warning message
func Warn() *zerolog.Event {
	return globalLogger.Warn()
}

// Error logs an error message
func Error() *zerolog.Event {
	return globalLogger.Error()
}

// Fatal logs a fatal message and exits
func Fatal() *zerolog.Event {
	return globalLogger.Fatal()
}

// Panic logs a panic message and panics
func Panic() *zerolog.Event {
	return globalLogger.Panic()
}

// Trace logs a trace message
func Trace() *zerolog.Event {
	return globalLogger.Trace()
}
