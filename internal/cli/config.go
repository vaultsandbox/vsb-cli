package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
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
	Use:   "set <key> [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  api-key   - Your VaultSandbox API key
  base-url  - API server URL (default: https://api.vaultsandbox.com)
  strategy  - Delivery strategy: sse or polling (default: sse)

Examples:
  vsb config set api-key vsb_abc123
  vsb config set base-url https://api.vaultsandbox.com
  vsb config set strategy sse
  vsb config set strategy        # Interactive selection`,
	Args: cobra.RangeArgs(1, 2),
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

	// Prompt for strategy
	defaultStrategy := "sse"
	if existing.Strategy != "" {
		defaultStrategy = existing.Strategy
	}
	fmt.Printf("\nDelivery Strategy:\n")
	fmt.Printf("  [1] sse - Server-Sent Events (real-time)\n")
	fmt.Printf("  [2] polling - Periodic API calls\n")
	currentNum := "1"
	if defaultStrategy == "polling" {
		currentNum = "2"
	}
	fmt.Printf("Choice [%s]: ", currentNum)
	strategyInput, _ := reader.ReadString('\n')
	strategyInput = strings.TrimSpace(strategyInput)

	var strategy string
	switch strategyInput {
	case "", "1", "sse":
		strategy = "sse"
	case "2", "polling":
		strategy = "polling"
	default:
		return fmt.Errorf("invalid strategy selection: %s", strategyInput)
	}

	// Save config
	cfg := &config.Config{
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Strategy: strategy,
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

	strategy := cfg.Strategy
	if strategy == "" {
		strategy = config.DefaultStrategy
	}

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		data := map[string]interface{}{
			"configFile": configPath,
			"apiKey":     maskedKey,
			"baseUrl":    baseURL,
			"strategy":   strategy,
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
	fmt.Printf("strategy: %s\n", strategy)

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Handle interactive strategy selection
	if key == "strategy" && len(args) == 1 {
		return runStrategyInteractive()
	}

	if len(args) < 2 {
		return fmt.Errorf("value required for key: %s", key)
	}

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
	case "strategy":
		if value != "sse" && value != "polling" {
			return fmt.Errorf("invalid strategy: %s (valid: sse, polling)", value)
		}
		cfg.Strategy = value
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: api-key, base-url, strategy)", key)
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s successfully\n", key)
	return nil
}

func runStrategyInteractive() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	strategies := []string{"sse", "polling"}
	descriptions := map[string]string{
		"sse":     "Server-Sent Events (real-time push notifications)",
		"polling": "Periodic API calls (with exponential backoff)",
	}

	fmt.Println("Select delivery strategy:")
	fmt.Println()

	for i, s := range strategies {
		current := ""
		if cfg.Strategy == s || (cfg.Strategy == "" && s == "sse") {
			current = " (current)"
		}
		fmt.Printf("  [%d] %s - %s%s\n", i+1, s, descriptions[s], current)
	}

	fmt.Println()
	fmt.Print("Enter choice (1-2): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var selected string
	switch input {
	case "1", "sse":
		selected = "sse"
	case "2", "polling":
		selected = "polling"
	default:
		return fmt.Errorf("invalid selection: %s", input)
	}

	cfg.Strategy = selected
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\nStrategy set to: %s\n", selected)
	return nil
}

