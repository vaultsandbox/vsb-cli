package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
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

// maskAPIKey masks an API key for display, showing first 7 and last 4 characters.
func maskAPIKey(key string) string {
	if len(key) >= 11 {
		return key[:7] + "..." + key[len(key)-4:]
	}
	return "****"
}

func runConfigInteractive(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Load existing config if present
	existing, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

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
		prompt = fmt.Sprintf("API Key [%s]: ", maskAPIKey(existing.APIKey))
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

	// Save config
	cfg := &config.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	configPath, _ := config.Path()
	fmt.Printf("\nConfig saved to %s\n", configPath)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	configPath, err := config.Path()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Mask API key for display
	maskedKey := ""
	if cfg.APIKey != "" {
		maskedKey = maskAPIKey(cfg.APIKey)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.vaultsandbox.com"
	}

	// JSON output
	if getOutput(cmd) == "json" {
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

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update the appropriate key
	switch key {
	case "api-key":
		cfg.APIKey = value
	case "base-url":
		cfg.BaseURL = value
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: api-key, base-url)", key)
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s successfully\n", key)
	return nil
}

