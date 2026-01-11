package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey        string `yaml:"api_key"`
	BaseURL       string `yaml:"base_url"`
	DefaultOutput string `yaml:"default_output"`
	Strategy      string `yaml:"strategy"`
}

// DefaultBaseURL
const DefaultBaseURL = "https://"

// DefaultStrategy is the default delivery strategy
const DefaultStrategy = "sse"

// Package-level state
var current Config

// Dir returns the vsb config directory path.
// Respects VSB_CONFIG_DIR environment variable if set.
func Dir() (string, error) {
	// Check for explicit config directory override
	if dir := os.Getenv("VSB_CONFIG_DIR"); dir != "" {
		return dir, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "vsb"), nil
}

// Path returns the config file path (~/.config/vsb/config.yaml)
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the config file and returns the Config struct.
// Returns an empty Config if the file doesn't exist.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// EnsureDir creates the config directory if it doesn't exist
func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

// LoadFromFile reads configuration from a YAML file
func LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config file is fine
		}
		return err
	}
	return yaml.Unmarshal(data, &current)
}

// getConfigValue returns config with priority: env (VSB_<key>) > config file > default
func getConfigValue(envKey, fileValue, defaultValue string) string {
	if env := os.Getenv("VSB_" + envKey); env != "" {
		return env
	}
	if fileValue != "" {
		return fileValue
	}
	return defaultValue
}

// GetAPIKey returns API key with priority: env > config file
func GetAPIKey() string {
	return getConfigValue("API_KEY", current.APIKey, "")
}

// GetBaseURL returns base URL with priority: env > config file > default
func GetBaseURL() string {
	return getConfigValue("BASE_URL", current.BaseURL, DefaultBaseURL)
}

// GetDefaultOutput returns the output format with priority: env > config file > default
func GetDefaultOutput() string {
	return getConfigValue("OUTPUT", current.DefaultOutput, "pretty")
}

// GetStrategy returns the delivery strategy with priority: env > config file > default
func GetStrategy() string {
	return getConfigValue("STRATEGY", current.Strategy, DefaultStrategy)
}

// Save writes the config to disk as YAML
func Save(cfg *Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}

	dir, _ := Dir()
	configPath := filepath.Join(dir, "config.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}
