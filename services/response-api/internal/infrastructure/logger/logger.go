package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"jan-server/services/response-api/internal/config"
)

// New creates a zerolog.Logger configured for the response service.
func New(cfg *config.Config) zerolog.Logger {
	level := parseLevel(cfg.LogLevel)
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	base := log.Output(output).
		With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Str("environment", cfg.Environment).
		Logger().
		Level(level)
	return base
}

func parseLevel(raw string) zerolog.Level {
	if raw == "" {
		return zerolog.InfoLevel
	}
	level, err := zerolog.ParseLevel(strings.ToLower(raw))
	if err != nil {
		return zerolog.InfoLevel
	}
	return level
}
