package magento2

import (
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	magento2 "github.com/florinel-chis/go-m2rest"
)

// TestConfig holds configuration for functional tests
type TestConfig struct {
	Host        string
	BearerToken string
	StoreCode   string
	APIVersion  string
	RestPrefix  string
	Timeout     time.Duration
	Debug       bool
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		StoreCode:  "default",
		APIVersion: "V1",
		RestPrefix: "/rest",
		Timeout:    60 * time.Second,
		Debug:      false,
	}
}

// LoadTestConfigFromEnv loads test configuration from environment variables
func LoadTestConfigFromEnv() (*TestConfig, error) {
	config := DefaultTestConfig()

	// Load from environment variables
	if host := os.Getenv("MAGENTO_HOST"); host != "" {
		config.Host = host
	}

	if token := os.Getenv("MAGENTO_BEARER_TOKEN"); token != "" {
		config.BearerToken = token
	}

	if storeCode := os.Getenv("MAGENTO_STORE_CODE"); storeCode != "" {
		config.StoreCode = storeCode
	}

	if apiVersion := os.Getenv("MAGENTO_API_VERSION"); apiVersion != "" {
		config.APIVersion = apiVersion
	}

	if restPrefix := os.Getenv("MAGENTO_REST_PREFIX"); restPrefix != "" {
		config.RestPrefix = restPrefix
	}

	if timeoutStr := os.Getenv("TEST_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = timeout
		}
	}

	if debugStr := os.Getenv("TEST_DEBUG"); debugStr != "" {
		if debug, err := strconv.ParseBool(debugStr); err == nil {
			config.Debug = debug
		}
	}

	// Validate required fields. There is deliberately no default host: live
	// tests only ever run against a store the caller names explicitly.
	if config.Host == "" {
		return nil, fmt.Errorf("MAGENTO_HOST is required")
	}
	if config.BearerToken == "" {
		return nil, fmt.Errorf("MAGENTO_BEARER_TOKEN is required")
	}

	// Validate and parse host URL
	if _, err := url.Parse(config.Host); err != nil {
		return nil, fmt.Errorf("invalid MAGENTO_HOST URL: %w", err)
	}

	return config, nil
}

// CreateStoreConfig creates a StoreConfig from TestConfig
func (tc *TestConfig) CreateStoreConfig() (*magento2.StoreConfig, error) {
	parsedURL, err := url.Parse(tc.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse host URL: %w", err)
	}

	return &magento2.StoreConfig{
		Scheme:    parsedURL.Scheme,
		HostName:  parsedURL.Host,
		StoreCode: tc.StoreCode,
	}, nil
}

// allowWritesEnv names the variable that enables tests which create, modify
// or delete data on the live store.
const allowWritesEnv = "MAGENTO_TEST_ALLOW_WRITES"

// skipWithoutWrites skips a test that writes to the live store unless
// MAGENTO_TEST_ALLOW_WRITES=1 is set. Run write tests only against a
// disposable store.
func skipWithoutWrites(t *testing.T) {
	t.Helper()
	loadDotEnv()
	if os.Getenv(allowWritesEnv) != "1" {
		t.Skip(allowWritesEnv + "=1 not set; skipping test that writes to the store")
	}
}

// SetupTestClient creates a configured API client for testing
func SetupTestClient() (*magento2.Client, *TestConfig, error) {
	// Try to load .env file if it exists
	loadDotEnv()

	config, err := LoadTestConfigFromEnv()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load test config: %w", err)
	}

	// Create store config
	storeConfig, err := config.CreateStoreConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create store config: %w", err)
	}

	// Create API client; TEST_DEBUG=true logs one line per request
	// (method, path, status) to stderr.
	var opts []magento2.ClientOption
	if config.Debug {
		opts = append(opts, magento2.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	}
	client, err := magento2.NewAPIClientFromIntegration(storeConfig, config.BearerToken, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return client, config, nil
}

// loadDotEnv loads environment variables from .env file if it exists
func loadDotEnv() {
	// Try current directory first, then parent directory
	envPaths := []string{".env", "../.env"}

	var file *os.File
	var err error

	for _, path := range envPaths {
		file, err = os.Open(path)
		if err == nil {
			break
		}
	}

	if err != nil {
		return // .env file doesn't exist, which is fine
	}
	defer file.Close()

	// Simple .env parser
	data, err := io.ReadAll(file)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Only set if not already set by actual environment
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
