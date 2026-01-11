//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigShow tests showing configuration.
func TestConfigShow(t *testing.T) {
	t.Run("show config with masked API key", func(t *testing.T) {
		configDir := t.TempDir()

		// First set a config value to ensure config file exists
		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "api-key", "vsb_test1234567890abcdef")
		require.Equal(t, 0, code, "config set failed: stderr=%s", stderr)

		// Show config
		stdout, stderr, code := runVSBWithConfig(t, configDir, "config", "show")
		require.Equal(t, 0, code, "config show failed: stderr=%s", stderr)

		// API key should be masked (showing first 7 and last 4 chars)
		assert.Contains(t, stdout, "api-key")
		assert.Contains(t, stdout, "vsb_tes") // First 7 chars
		assert.Contains(t, stdout, "...")     // Masked middle
		assert.Contains(t, stdout, "cdef")    // Last 4 chars

		// Should NOT contain the full API key
		assert.NotContains(t, stdout, "vsb_test1234567890abcdef")
	})

	t.Run("show config JSON output", func(t *testing.T) {
		configDir := t.TempDir()

		// Set config
		runVSBWithConfig(t, configDir, "config", "set", "api-key", "vsb_jsontest12345abcdef")
		runVSBWithConfig(t, configDir, "config", "set", "base-url", "https://custom.example.com")

		// Show config as JSON
		stdout, stderr, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
		require.Equal(t, 0, code, "config show --output json failed: stderr=%s", stderr)

		var result struct {
			ConfigFile string `json:"configFile"`
			APIKey     string `json:"apiKey"`
			BaseURL    string `json:"baseUrl"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.NotEmpty(t, result.ConfigFile)
		assert.Contains(t, result.APIKey, "vsb_jso") // Masked
		assert.Contains(t, result.APIKey, "...")
		assert.Equal(t, "https://custom.example.com", result.BaseURL)
	})

	t.Run("show config with empty values", func(t *testing.T) {
		configDir := t.TempDir()

		// Don't set any config - should show defaults or "(not set)"
		stdout, stderr, code := runVSBWithConfig(t, configDir, "config", "show")
		require.Equal(t, 0, code, "config show failed: stderr=%s", stderr)

		// Should show some default base URL
		assert.Contains(t, stdout, "base-url")
	})

	t.Run("show config file path", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "config", "show")
		require.Equal(t, 0, code, "config show failed: stderr=%s", stderr)

		// Should show the config file path
		assert.Contains(t, stdout, "Config file:")
	})
}

// TestConfigSet tests setting configuration values.
func TestConfigSet(t *testing.T) {
	t.Run("set api-key", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "api-key", "vsb_newkey123456789")
		require.Equal(t, 0, code, "config set api-key failed: stderr=%s", stderr)

		// Verify by showing config
		stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			APIKey string `json:"apiKey"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Contains(t, result.APIKey, "vsb_new") // First 7 chars
		assert.Contains(t, result.APIKey, "789")     // Last 4 chars (from "123456789")
	})

	t.Run("set base-url", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "base-url", "https://api.custom.example.com")
		require.Equal(t, 0, code, "config set base-url failed: stderr=%s", stderr)

		// Verify by showing config
		stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			BaseURL string `json:"baseUrl"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, "https://api.custom.example.com", result.BaseURL)
	})

	t.Run("error on invalid key", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "invalid-key", "somevalue")
		assert.NotEqual(t, 0, code, "config set should fail for invalid key")
		assert.Contains(t, stderr, "unknown config key")
	})

	t.Run("update existing value", func(t *testing.T) {
		configDir := t.TempDir()

		// Set initial value
		_, _, code := runVSBWithConfig(t, configDir, "config", "set", "base-url", "https://first.example.com")
		require.Equal(t, 0, code)

		// Update value
		_, _, code = runVSBWithConfig(t, configDir, "config", "set", "base-url", "https://second.example.com")
		require.Equal(t, 0, code)

		// Verify updated value
		stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			BaseURL string `json:"baseUrl"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, "https://second.example.com", result.BaseURL)
	})

	t.Run("config file created in correct location", func(t *testing.T) {
		configDir := t.TempDir()

		// Set a value (should create config file)
		_, _, code := runVSBWithConfig(t, configDir, "config", "set", "api-key", "vsb_locationtest123")
		require.Equal(t, 0, code)

		// Config file should exist in the config directory
		configPath := filepath.Join(configDir, "config.yaml")
		_, err := os.Stat(configPath)
		assert.NoError(t, err, "config.yaml should exist in config directory")
	})
}

// TestConfigPersistence tests that config values persist across invocations.
func TestConfigPersistence(t *testing.T) {
	configDir := t.TempDir()

	// Set values in first invocation
	_, _, code := runVSBWithConfig(t, configDir, "config", "set", "api-key", "vsb_persist123456789abc")
	require.Equal(t, 0, code)

	_, _, code = runVSBWithConfig(t, configDir, "config", "set", "base-url", "https://persistent.example.com")
	require.Equal(t, 0, code)

	// Verify in second invocation (simulating new CLI run)
	stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
	require.Equal(t, 0, code)

	var result struct {
		APIKey  string `json:"apiKey"`
		BaseURL string `json:"baseUrl"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	assert.Contains(t, result.APIKey, "vsb_per")
	assert.Equal(t, "https://persistent.example.com", result.BaseURL)
}

// TestConfigEnvironmentVariables tests that environment variables override config.
func TestConfigEnvironmentVariables(t *testing.T) {
	t.Run("env vars take precedence", func(t *testing.T) {
		configDir := t.TempDir()

		// Set config file values
		_, _, code := runVSBWithConfig(t, configDir, "config", "set", "base-url", "https://file.example.com")
		require.Equal(t, 0, code)

		// Note: VSB_BASE_URL is already set by our test harness to the actual API URL
		// So the config file value won't be used for API calls
		// But we can verify the config show still shows file values when displaying

		stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			BaseURL string `json:"baseUrl"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		// The config show should display what's in the config file
		// (not the env var override, since that's a runtime thing)
		assert.Equal(t, "https://file.example.com", result.BaseURL)
	})
}

// TestConfigShortAPIKey tests handling of short API keys.
func TestConfigShortAPIKey(t *testing.T) {
	configDir := t.TempDir()

	// Set a short API key (less than 11 chars)
	_, _, code := runVSBWithConfig(t, configDir, "config", "set", "api-key", "shortkey")
	require.Equal(t, 0, code)

	// Show config - should mask short key differently
	stdout, _, code := runVSBWithConfig(t, configDir, "config", "show")
	require.Equal(t, 0, code)

	// Short keys should show as "****"
	assert.Contains(t, stdout, "****")
	assert.NotContains(t, stdout, "shortkey")
}

// TestConfigEmptyAPIKey tests that empty API key shows "(not set)".
func TestConfigEmptyAPIKey(t *testing.T) {
	configDir := t.TempDir()

	// Don't set any API key
	stdout, _, code := runVSBWithConfig(t, configDir, "config", "show")
	require.Equal(t, 0, code)

	// Should show "(not set)" for API key
	output := strings.ToLower(stdout)
	assert.True(t, strings.Contains(output, "not set") || strings.Contains(output, "(not set)"),
		"should indicate API key is not set")
}

// TestConfigDefaultBaseURL tests that default base URL is used when not set.
func TestConfigDefaultBaseURL(t *testing.T) {
	configDir := t.TempDir()

	// Don't set base URL - should use default
	stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
	require.Equal(t, 0, code)

	var result struct {
		BaseURL string `json:"baseUrl"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	// Should have a default base URL
	assert.NotEmpty(t, result.BaseURL)
	assert.Contains(t, result.BaseURL, "https://")
}

// TestConfigMultipleValues tests setting multiple config values.
func TestConfigMultipleValues(t *testing.T) {
	configDir := t.TempDir()

	// Set multiple values
	_, _, code := runVSBWithConfig(t, configDir, "config", "set", "api-key", "vsb_multitest123456789")
	require.Equal(t, 0, code)

	_, _, code = runVSBWithConfig(t, configDir, "config", "set", "base-url", "https://multi.example.com")
	require.Equal(t, 0, code)

	// Verify both values
	stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
	require.Equal(t, 0, code)

	var result struct {
		APIKey  string `json:"apiKey"`
		BaseURL string `json:"baseUrl"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))

	assert.Contains(t, result.APIKey, "vsb_mul")
	assert.Equal(t, "https://multi.example.com", result.BaseURL)
}

// TestConfigValidKeys tests the valid config key names.
func TestConfigValidKeys(t *testing.T) {
	validKeys := []struct {
		key   string
		value string
	}{
		{"api-key", "vsb_validkey12345678"},
		{"base-url", "https://valid.example.com"},
		{"strategy", "sse"},
		{"strategy", "polling"},
	}

	for _, tc := range validKeys {
		t.Run("valid key: "+tc.key+"="+tc.value, func(t *testing.T) {
			configDir := t.TempDir()

			_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", tc.key, tc.value)
			assert.Equal(t, 0, code, "setting %s should succeed: stderr=%s", tc.key, stderr)
		})
	}

	invalidKeys := []string{
		"apikey",
		"api_key",
		"API-KEY",
		"baseurl",
		"base_url",
		"url",
		"key",
		"password",
		"secret",
	}

	for _, key := range invalidKeys {
		t.Run("invalid key: "+key, func(t *testing.T) {
			configDir := t.TempDir()

			_, _, code := runVSBWithConfig(t, configDir, "config", "set", key, "somevalue")
			assert.NotEqual(t, 0, code, "setting %s should fail", key)
		})
	}
}

// TestConfigStrategy tests setting and showing delivery strategy.
func TestConfigStrategy(t *testing.T) {
	t.Run("set strategy sse", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "strategy", "sse")
		require.Equal(t, 0, code, "config set strategy sse failed: stderr=%s", stderr)

		// Verify by showing config
		stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Strategy string `json:"strategy"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, "sse", result.Strategy)
	})

	t.Run("set strategy polling", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "strategy", "polling")
		require.Equal(t, 0, code, "config set strategy polling failed: stderr=%s", stderr)

		// Verify by showing config
		stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Strategy string `json:"strategy"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, "polling", result.Strategy)
	})

	t.Run("invalid strategy value", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "strategy", "invalid")
		assert.NotEqual(t, 0, code, "config set strategy invalid should fail")
		assert.Contains(t, stderr, "invalid strategy")
	})

	t.Run("default strategy is sse", func(t *testing.T) {
		configDir := t.TempDir()

		// Don't set strategy - should default to sse
		stdout, _, code := runVSBWithConfig(t, configDir, "config", "show", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Strategy string `json:"strategy"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, "sse", result.Strategy)
	})

	t.Run("strategy shown in pretty output", func(t *testing.T) {
		configDir := t.TempDir()

		_, _, code := runVSBWithConfig(t, configDir, "config", "set", "strategy", "polling")
		require.Equal(t, 0, code)

		stdout, _, code := runVSBWithConfig(t, configDir, "config", "show")
		require.Equal(t, 0, code)

		assert.Contains(t, stdout, "strategy:")
		assert.Contains(t, stdout, "polling")
	})
}
