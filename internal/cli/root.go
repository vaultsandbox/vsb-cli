package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

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

	// Global flags
	rootCmd.PersistentFlags().String("api-key", "", "API key (overrides config)")
	rootCmd.PersistentFlags().String("base-url", "", "API base URL")
	rootCmd.PersistentFlags().StringP("output", "o", "pretty",
		"Output format: pretty, json, minimal")

	// Bind flags to viper
	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("base_url", rootCmd.PersistentFlags().Lookup("base-url"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return
		}

		vsbConfigDir := filepath.Join(configDir, "vsb")
		viper.AddConfigPath(vsbConfigDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("VSB")
	viper.AutomaticEnv()

	// Read config (ignore error if not found)
	viper.ReadInConfig()
}
