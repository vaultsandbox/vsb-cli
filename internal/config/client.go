package config

import (
	"errors"

	vaultsandbox "github.com/vaultsandbox/client-go"
)

var ErrNoAPIKey = errors.New("API key not configured. Set VSB_API_KEY or run 'vsb config'")

// NewClient creates a VaultSandbox client using current configuration
func NewClient() (*vaultsandbox.Client, error) {
	apiKey := GetAPIKey()
	if apiKey == "" {
		return nil, ErrNoAPIKey
	}

	opts := []vaultsandbox.Option{}

	if baseURL := GetBaseURL(); baseURL != "" {
		opts = append(opts, vaultsandbox.WithBaseURL(baseURL))
	}

	return vaultsandbox.New(apiKey, opts...)
}
