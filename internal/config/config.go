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
}

// DefaultBaseURL is the production API endpoint
const DefaultBaseURL = "https://api.vaultsandbox.com"

// Package-level state (replaces viper's global state)
var (
	current                            Config
	flagAPIKey, flagBaseURL, flagOutput *string
)

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

// getEnv returns env var with VSB_ prefix, or fallback
func getEnv(key, fallback string) string {
	if v := os.Getenv("VSB_" + key); v != "" {
		return v
	}
	return fallback
}

// Load returns the current config (for backwards compatibility)
func Load() *Config {
	return &Config{
		APIKey:        GetAPIKey(),
		BaseURL:       GetBaseURL(),
		DefaultOutput: GetOutput(),
	}
}

// GetAPIKey returns API key with priority: flag > env > config file
func GetAPIKey() string {
	if flagAPIKey != nil && *flagAPIKey != "" {
		return *flagAPIKey
	}
	return getEnv("API_KEY", current.APIKey)
}

// GetBaseURL returns base URL with priority: flag > env > config file > default
func GetBaseURL() string {
	if flagBaseURL != nil && *flagBaseURL != "" {
		return *flagBaseURL
	}
	if url := getEnv("BASE_URL", current.BaseURL); url != "" {
		return url
	}
	return DefaultBaseURL
}

// GetOutput returns output format with priority: flag > env > config file
func GetOutput() string {
	if flagOutput != nil && *flagOutput != "" {
		return *flagOutput
	}
	if out := getEnv("OUTPUT", current.DefaultOutput); out != "" {
		return out
	}
	return "pretty"
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

// SetFlagPointers allows root.go to pass flag pointers for priority resolution
func SetFlagPointers(apiKey, baseURL, output *string) {
	flagAPIKey, flagBaseURL, flagOutput = apiKey, baseURL, output
}
