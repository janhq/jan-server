package logger

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	globalLogger zerolog.Logger
	once         sync.Once
)

// GetLogger returns the global logger instance
func GetLogger() zerolog.Logger {
	once.Do(func() {
		// Default to console output with info level
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		globalLogger = zerolog.New(consoleWriter).With().Timestamp().Logger().Level(zerolog.InfoLevel)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	})
	return globalLogger
}

// New constructs a zerolog logger based on level and format configuration.
func New(level, format string) (zerolog.Logger, error) {
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		return zerolog.Logger{}, err
	}

	var writer zerolog.Logger
	switch strings.ToLower(format) {
	case "json":
		writer = zerolog.New(os.Stdout).With().Timestamp().Logger()
	case "console":
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		writer = zerolog.New(consoleWriter).With().Timestamp().Logger()
	default:
		return zerolog.Logger{}, errors.New("unsupported log format")
	}

	zerolog.SetGlobalLevel(lvl)

	// Update global logger
	globalLogger = writer.Level(lvl)

	return globalLogger, nil
}
