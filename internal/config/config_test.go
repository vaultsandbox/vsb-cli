package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDir(t *testing.T) {
	t.Run("default directory", func(t *testing.T) {
		t.Setenv("VSB_CONFIG_DIR", "")
		dir, err := Dir()
		require.NoError(t, err)
		assert.Contains(t, dir, "vsb")
	})

	t.Run("custom directory from env", func(t *testing.T) {
		customDir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", customDir)
		dir, err := Dir()
		require.NoError(t, err)
		assert.Equal(t, customDir, dir)
	})
}

func TestPath(t *testing.T) {
	t.Run("returns config.yaml path", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		path, err := Path()
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(path, "config.yaml"))
	})
}

func TestLoad(t *testing.T) {
	t.Run("missing file returns empty config", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		cfg, err := Load()
		require.NoError(t, err)
		assert.Empty(t, cfg.APIKey)
		assert.Empty(t, cfg.BaseURL)
	})

	t.Run("valid YAML", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		configContent := `api_key: test-key
base_url: https://api.example.com
default_output: json`
		err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "test-key", cfg.APIKey)
		assert.Equal(t, "https://api.example.com", cfg.BaseURL)
		assert.Equal(t, "json", cfg.DefaultOutput)
	})

	t.Run("invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("invalid: [yaml"), 0644)
		require.NoError(t, err)

		_, err = Load()
		assert.Error(t, err)
	})
}

func TestEnsureDir(t *testing.T) {
	t.Run("creates directory with correct permissions", func(t *testing.T) {
		base := t.TempDir()
		dir := filepath.Join(base, "new-config-dir")
		t.Setenv("VSB_CONFIG_DIR", dir)

		err := EnsureDir()
		require.NoError(t, err)

		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})

	t.Run("existing directory is ok", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		err := EnsureDir()
		assert.NoError(t, err)
	})
}

func TestGetAPIKey(t *testing.T) {
	// Save original state
	originalCurrent := current
	defer func() { current = originalCurrent }()

	t.Run("env var takes precedence", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)
		t.Setenv("VSB_API_KEY", "env-key")

		// Load config with different key
		current = Config{APIKey: "config-key"}

		key := GetAPIKey()
		assert.Equal(t, "env-key", key)
	})

	t.Run("falls back to config file", func(t *testing.T) {
		t.Setenv("VSB_API_KEY", "") // Clear env var
		current = Config{APIKey: "config-key"}

		key := GetAPIKey()
		assert.Equal(t, "config-key", key)
	})

	t.Run("returns empty if not set", func(t *testing.T) {
		t.Setenv("VSB_API_KEY", "")
		current = Config{}

		key := GetAPIKey()
		assert.Empty(t, key)
	})
}

func TestGetBaseURL(t *testing.T) {
	// Save original state
	originalCurrent := current
	defer func() { current = originalCurrent }()

	t.Run("env var takes precedence", func(t *testing.T) {
		t.Setenv("VSB_BASE_URL", "https://custom.api.com")
		current = Config{BaseURL: "https://config.api.com"}

		url := GetBaseURL()
		assert.Equal(t, "https://custom.api.com", url)
	})

	t.Run("returns default if not set", func(t *testing.T) {
		t.Setenv("VSB_BASE_URL", "")
		current = Config{}

		url := GetBaseURL()
		assert.Equal(t, DefaultBaseURL, url)
	})

	t.Run("uses config value when no env var", func(t *testing.T) {
		t.Setenv("VSB_BASE_URL", "")
		current = Config{BaseURL: "https://config.api.com"}

		url := GetBaseURL()
		assert.Equal(t, "https://config.api.com", url)
	})
}

func TestGetDefaultOutput(t *testing.T) {
	// Save original state
	originalCurrent := current
	defer func() { current = originalCurrent }()

	t.Run("defaults to pretty", func(t *testing.T) {
		t.Setenv("VSB_OUTPUT", "")
		current = Config{}

		output := GetDefaultOutput()
		assert.Equal(t, "pretty", output)
	})

	t.Run("env var override", func(t *testing.T) {
		t.Setenv("VSB_OUTPUT", "json")

		output := GetDefaultOutput()
		assert.Equal(t, "json", output)
	})

	t.Run("config file value", func(t *testing.T) {
		t.Setenv("VSB_OUTPUT", "")
		current = Config{DefaultOutput: "yaml"}

		output := GetDefaultOutput()
		assert.Equal(t, "yaml", output)
	})
}

func TestSave(t *testing.T) {
	t.Run("saves config to file", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		cfg := &Config{
			APIKey:        "test-key",
			BaseURL:       "https://api.example.com",
			DefaultOutput: "json",
		}

		err := Save(cfg)
		require.NoError(t, err)

		// Read back and verify
		loaded, err := Load()
		require.NoError(t, err)
		assert.Equal(t, cfg.APIKey, loaded.APIKey)
		assert.Equal(t, cfg.BaseURL, loaded.BaseURL)
		assert.Equal(t, cfg.DefaultOutput, loaded.DefaultOutput)
	})

	t.Run("creates directory if not exists", func(t *testing.T) {
		base := t.TempDir()
		dir := filepath.Join(base, "new-dir")
		t.Setenv("VSB_CONFIG_DIR", dir)

		cfg := &Config{APIKey: "test"}
		err := Save(cfg)
		require.NoError(t, err)

		// Verify file was created
		path := filepath.Join(dir, "config.yaml")
		_, err = os.Stat(path)
		assert.NoError(t, err)
	})
}

func TestLoadFromFile(t *testing.T) {
	// Save original state
	originalCurrent := current
	defer func() { current = originalCurrent }()

	t.Run("loads config into package state", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "config.yaml")
		content := `api_key: loaded-key
base_url: https://loaded.example.com`
		err := os.WriteFile(configPath, []byte(content), 0644)
		require.NoError(t, err)

		current = Config{} // Reset
		err = LoadFromFile(configPath)
		require.NoError(t, err)

		assert.Equal(t, "loaded-key", current.APIKey)
		assert.Equal(t, "https://loaded.example.com", current.BaseURL)
	})

	t.Run("missing file is ok", func(t *testing.T) {
		err := LoadFromFile("/nonexistent/path/config.yaml")
		assert.NoError(t, err)
	})
}
