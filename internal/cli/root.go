package cli

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "vsb",
	Short: "VaultSandbox CLI - Test email flows with quantum-safe encryption",
	Long: `vsb is a developer companion for testing email flows.

It provides temporary inboxes with end-to-end encryption using
quantum-safe algorithms (ML-KEM-768, ML-DSA-65).

The server never sees your email content - all decryption
happens locally on your machine.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.config/vsb/config.yaml)")

	// Global flags - store pointers for priority resolution
	apiKey := rootCmd.PersistentFlags().String("api-key", "", "API key (overrides config)")
	baseURL := rootCmd.PersistentFlags().String("base-url", "", "API base URL")
	output := rootCmd.PersistentFlags().StringP("output", "o", "", "Output format: pretty, json, minimal")

	config.SetFlagPointers(apiKey, baseURL, output)
}

func initConfig() {
	var configPath string
	if cfgFile != "" {
		configPath = cfgFile
	} else {
		dir, err := config.Dir()
		if err != nil {
			return
		}
		configPath = filepath.Join(dir, "config.yaml")
	}
	config.LoadFromFile(configPath)
}
