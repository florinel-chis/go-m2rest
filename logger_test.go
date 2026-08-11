package magento2

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
)

// resetLogger restores the package logger state after a test.
func resetLogger(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		customLogger.Store(nil)
		currentLogger.Store(&nopLogger)
	})
}

// Enable/DisableDebugLogging must adjust the level of a logger installed via
// SetZeroLogger instead of discarding it.
func TestEnableDisableDebugLoggingPreservesCustomLogger(t *testing.T) {
	resetLogger(t)

	var buf bytes.Buffer
	SetZeroLogger(zerolog.New(&buf).Level(zerolog.InfoLevel))

	logger().Debug().Msg("dbg-before")
	EnableDebugLogging()
	logger().Debug().Msg("dbg-during")
	DisableDebugLogging()
	logger().Debug().Msg("dbg-after")
	logger().Info().Msg("info-after")

	out := buf.String()
	if strings.Contains(out, "dbg-before") {
		t.Error("debug message logged before EnableDebugLogging despite info level")
	}
	if !strings.Contains(out, "dbg-during") {
		t.Error("EnableDebugLogging did not raise the custom logger to debug level")
	}
	if strings.Contains(out, "dbg-after") {
		t.Error("DisableDebugLogging did not restore the custom logger's original level")
	}
	if !strings.Contains(out, "info-after") {
		t.Error("DisableDebugLogging discarded the custom logger instead of restoring it")
	}
}

func TestDisableDebugLoggingWithoutCustomLoggerIsNop(t *testing.T) {
	resetLogger(t)

	EnableDebugLogging()
	DisableDebugLogging()
	if got := logger().GetLevel(); got != zerolog.Disabled {
		t.Errorf("logger level after DisableDebugLogging = %v, want disabled (no-op)", got)
	}
}

// Toggling logging while other goroutines are logging must be race-free
// (run with -race).
func TestLoggerToggleConcurrentWithLogging(t *testing.T) {
	resetLogger(t)

	var buf bytes.Buffer
	var mu sync.Mutex
	safeWriter := writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return buf.Write(p)
	})
	SetZeroLogger(zerolog.New(safeWriter))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			EnableDebugLogging()
			DisableDebugLogging()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			logger().Debug().Msg("concurrent")
		}
	}()
	wg.Wait()
}

type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
