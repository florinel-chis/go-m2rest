package magento2

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	// Set up default logger with pretty console output
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	logger = zerolog.New(output).With().Timestamp().Caller().Logger()
	
	// Also set the global logger
	log.Logger = logger
}

// SetLogger allows users to set a custom zerolog logger
func SetZeroLogger(customLogger zerolog.Logger) {
	logger = customLogger
	log.Logger = customLogger
}

// EnableDebugLogging enables debug level logging
func EnableDebugLogging() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

// DisableDebugLogging sets logging back to info level
func DisableDebugLogging() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}