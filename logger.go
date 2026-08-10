package magento2

import (
	"os"

	"github.com/rs/zerolog"
)

// logger is the package-level logger used by this library. It defaults to a
// no-op logger: importing this package never mutates global state (such as
// the global zerolog logger) and never writes to stderr on its own.
var logger = zerolog.Nop()

// SetZeroLogger allows users to set a custom zerolog logger for this library.
// It only changes the package-level logger; the global zerolog logger is left
// untouched.
func SetZeroLogger(customLogger zerolog.Logger) {
	logger = customLogger
}

// EnableDebugLogging explicitly installs a debug-level console logger
// (writing to stderr) as the package-level logger.
func EnableDebugLogging() {
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	logger = zerolog.New(output).With().Timestamp().Caller().Logger().Level(zerolog.DebugLevel)
}

// DisableDebugLogging reverts the package-level logger to the default no-op
// logger.
func DisableDebugLogging() {
	logger = zerolog.Nop()
}
