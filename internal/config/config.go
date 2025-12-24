package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey        string `mapstructure:"api_key"`
	BaseURL       string `mapstructure:"base_url"`
	DefaultOutput string `mapstructure:"default_output"`
}

// DefaultBaseURL is the production API endpoint
const DefaultBaseURL = "https://api.vaultsandbox.com"

// Dir returns the vsb config directory path
func Dir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "vsb"), nil
}

// EnsureDir creates the config directory if it doesn't exist
func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

// Load reads configuration from viper (already initialized in root.go)
func Load() *Config {
	return &Config{
		APIKey:        viper.GetString("api_key"),
		BaseURL:       viper.GetString("base_url"),
		DefaultOutput: viper.GetString("default_output"),
	}
}

// GetAPIKey returns the API key, checking env vars and config
func GetAPIKey() string {
	// Priority: flag > env > config file
	if key := viper.GetString("api_key"); key != "" {
		return key
	}
	return os.Getenv("VSB_API_KEY")
}

// GetBaseURL returns the base URL with default fallback
func GetBaseURL() string {
	if url := viper.GetString("base_url"); url != "" {
		return url
	}
	return DefaultBaseURL
}

// Save writes the current config to disk
func Save(cfg *Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}

	dir, _ := Dir()
	configPath := filepath.Join(dir, "config.yaml")

	viper.Set("api_key", cfg.APIKey)
	viper.Set("base_url", cfg.BaseURL)
	viper.Set("default_output", cfg.DefaultOutput)

	return viper.WriteConfigAs(configPath)
}
