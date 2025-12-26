package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure API key and server URL",
	Long: `Manage VaultSandbox CLI configuration.

Running 'vsb config' without subcommands starts interactive configuration.

Examples:
  vsb config                    # Interactive configuration
  vsb config show               # Show current configuration
  vsb config set api-key <key>  # Set API key
  vsb config set base-url <url> # Set base URL`,
	RunE: runConfigInteractive,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  api-key   - Your VaultSandbox API key
  base-url  - API server URL (default: https://api.vaultsandbox.com)

Examples:
  vsb config set api-key vsb_abc123
  vsb config set base-url https://api.vaultsandbox.com`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigInteractive(cmd *cobra.Command, args []string) error {
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
		if len(existing.APIKey) >= 11 {
			masked := existing.APIKey[:7] + "..." + existing.APIKey[len(existing.APIKey)-4:]
			prompt = fmt.Sprintf("API Key [%s]: ", masked)
		} else {
			prompt = "API Key [****]: "
		}
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
	cfg := config.Config{
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

func runConfigShow(cmd *cobra.Command, args []string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	configPath := filepath.Join(configDir, "vsb", "config.yaml")

	cfg := loadExistingConfig(configPath)

	// Mask API key for display
	maskedKey := ""
	if cfg.APIKey != "" {
		if len(cfg.APIKey) >= 11 {
			maskedKey = cfg.APIKey[:7] + "..." + cfg.APIKey[len(cfg.APIKey)-4:]
		} else {
			maskedKey = "****"
		}
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.vaultsandbox.com"
	}

	// JSON output
	if config.GetOutput() == "json" {
		data := map[string]interface{}{
			"configFile": configPath,
			"apiKey":     maskedKey,
			"baseUrl":    baseURL,
		}
		out, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	// Pretty output
	if maskedKey == "" {
		maskedKey = "(not set)"
	}

	fmt.Printf("Config file: %s\n\n", configPath)
	fmt.Printf("api-key:  %s\n", maskedKey)
	fmt.Printf("base-url: %s\n", baseURL)

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	vsbConfigDir := filepath.Join(configDir, "vsb")
	configPath := filepath.Join(vsbConfigDir, "config.yaml")

	// Load existing config
	cfg := loadExistingConfig(configPath)

	// Update the appropriate key
	switch key {
	case "api-key":
		cfg.APIKey = value
	case "base-url":
		cfg.BaseURL = value
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: api-key, base-url)", key)
	}

	// Create config directory
	if err := os.MkdirAll(vsbConfigDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save config
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Set %s successfully\n", key)
	return nil
}

func loadExistingConfig(path string) config.Config {
	var cfg config.Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	yaml.Unmarshal(data, &cfg)
	return cfg
}
