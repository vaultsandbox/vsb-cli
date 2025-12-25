package config

import (
	"context"
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

// WithClient creates a client, passes it to the callback, and ensures cleanup.
// This is the recommended way to use the client for simple operations.
func WithClient(ctx context.Context, fn func(context.Context, *vaultsandbox.Client) error) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	return fn(ctx, client)
}
