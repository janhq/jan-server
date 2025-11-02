package logger

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

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
	return writer.Level(lvl), nil
}
