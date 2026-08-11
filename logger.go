package magento2

import (
	"os"
	"sync/atomic"

	"github.com/rs/zerolog"
)

// The package-level logger defaults to a no-op logger: importing this
// package never mutates global state (such as the global zerolog logger) and
// never writes to stderr on its own. Access goes through an atomic pointer
// so the logger can be swapped (SetZeroLogger, EnableDebugLogging,
// DisableDebugLogging) while requests are in flight without a data race.
var (
	nopLogger     = zerolog.Nop()
	currentLogger atomic.Pointer[zerolog.Logger]
	// customLogger remembers the logger installed via SetZeroLogger so that
	// EnableDebugLogging/DisableDebugLogging adjust its level instead of
	// discarding it.
	customLogger atomic.Pointer[zerolog.Logger]
)

func init() {
	currentLogger.Store(&nopLogger)
}

// logger returns the current package-level logger.
func logger() *zerolog.Logger {
	return currentLogger.Load()
}

// SetZeroLogger allows users to set a custom zerolog logger for this library.
// It only changes the package-level logger; the global zerolog logger is left
// untouched.
func SetZeroLogger(l zerolog.Logger) {
	customLogger.Store(&l)
	currentLogger.Store(&l)
}

// EnableDebugLogging raises the package-level logger to debug level. A
// custom logger installed via SetZeroLogger is kept — only its level is
// lowered to debug; when no custom logger was set, a debug-level console
// logger writing to stderr is installed.
func EnableDebugLogging() {
	if custom := customLogger.Load(); custom != nil {
		debug := custom.Level(zerolog.DebugLevel)
		currentLogger.Store(&debug)
		return
	}
	output := zerolog.ConsoleWriter{Out: os.Stderr}
	l := zerolog.New(output).With().Timestamp().Caller().Logger().Level(zerolog.DebugLevel)
	currentLogger.Store(&l)
}

// DisableDebugLogging reverts the package-level logger to the custom logger
// installed via SetZeroLogger (at its original level), or to the default
// no-op logger when none was set.
func DisableDebugLogging() {
	if custom := customLogger.Load(); custom != nil {
		currentLogger.Store(custom)
		return
	}
	currentLogger.Store(&nopLogger)
}
