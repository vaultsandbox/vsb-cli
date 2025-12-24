package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure API key and server URL",
	Long:  `Interactively configure your VaultSandbox API credentials.`,
	RunE:  runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Get config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	vsbConfigDir := filepath.Join(configDir, "vsb")
	configPath := filepath.Join(vsbConfigDir, "config.yaml")

	// Load existing config if present
	existing := loadExistingConfig(configPath)

	// Prompt for base URL
	defaultURL := "https://api.vaultsandbox.com"
	if existing.BaseURL != "" {
		defaultURL = existing.BaseURL
	}
	fmt.Printf("Server URL [%s]: ", defaultURL)
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultURL
	}

	// Prompt for API key
	prompt := "API Key: "
	if existing.APIKey != "" {
		masked := existing.APIKey[:7] + "..." + existing.APIKey[len(existing.APIKey)-4:]
		prompt = fmt.Sprintf("API Key [%s]: ", masked)
	}
	fmt.Print(prompt)
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" && existing.APIKey != "" {
		apiKey = existing.APIKey
	}

	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Create config directory
	if err := os.MkdirAll(vsbConfigDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save config
	cfg := Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("\nConfig saved to %s\n", configPath)
	return nil
}

func loadExistingConfig(path string) Config {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	yaml.Unmarshal(data, &cfg)
	return cfg
}
