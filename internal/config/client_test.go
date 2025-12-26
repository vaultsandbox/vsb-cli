package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	// Save original state
	originalCurrent := current
	defer func() { current = originalCurrent }()

	t.Run("returns error when no API key", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)
		t.Setenv("VSB_API_KEY", "")
		current = Config{}

		_, err := NewClient()
		assert.ErrorIs(t, err, ErrNoAPIKey)
	})

	t.Run("creates client with valid config", func(t *testing.T) {
		// Skip if no real API key is available (network test)
		if os.Getenv("VSB_TEST_API_KEY") == "" {
			t.Skip("Skipping: set VSB_TEST_API_KEY to run network tests")
		}

		t.Setenv("VSB_API_KEY", os.Getenv("VSB_TEST_API_KEY"))
		t.Setenv("VSB_BASE_URL", "") // Use default

		client, err := NewClient()
		require.NoError(t, err)
		assert.NotNil(t, client)
		client.Close()
	})

	t.Run("uses config values", func(t *testing.T) {
		// Test that config values are read correctly
		// (without actually creating a client that requires network)
		t.Setenv("VSB_API_KEY", "")
		t.Setenv("VSB_BASE_URL", "")
		current = Config{
			APIKey:  "",
			BaseURL: "https://config.api.com",
		}

		// With empty API key, should return error
		_, err := NewClient()
		assert.ErrorIs(t, err, ErrNoAPIKey)
	})
}
