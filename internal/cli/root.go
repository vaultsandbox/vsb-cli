package cli

import (
	"context"
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/tui/watch"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "vsb",
	Short: "VaultSandbox CLI - Test email flows with quantum-safe encryption",
	Long: `vsb is a developer companion for testing email flows.
https://vaultsandbox.com

It provides temporary inboxes with quantum-safe encryption
(ML-KEM-768, ML-DSA-65). Emails are encrypted on receipt and
can only be decrypted locally with your private keys.

Running 'vsb' opens the real-time email dashboard for all inboxes.`,
	RunE: runRoot,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.config/vsb/config.yaml)")

	// Global output format flag
	output := rootCmd.PersistentFlags().StringP("output", "o", "", "Output format: pretty, json")

	config.SetFlagPointers(output)
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

func runRoot(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load keystore
	keystore, err := LoadKeystoreOrError()
	if err != nil {
		return err
	}

	// Load all inboxes
	storedInboxes := keystore.ListInboxes()
	if len(storedInboxes) == 0 {
		return fmt.Errorf("no inboxes found. Create one with 'vsb inbox create'")
	}

	// Find active inbox index
	activeIdx := 0
	if activeInbox, err := keystore.GetActiveInbox(); err == nil {
		for i, stored := range storedInboxes {
			if stored.Email == activeInbox.Email {
				activeIdx = i
				break
			}
		}
	}

	// Create SDK client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Import inboxes into client
	var inboxes []*vaultsandbox.Inbox
	for _, stored := range storedInboxes {
		exported := stored.ToExportedInbox()
		inbox, err := client.ImportInbox(ctx, exported)
		if err != nil {
			return fmt.Errorf("failed to import inbox %s: %w", stored.Email, err)
		}
		inboxes = append(inboxes, inbox)
	}

	// Create TUI model starting on active inbox
	model := watch.NewModel(client, inboxes, activeIdx)

	// Create and run TUI program
	p := tea.NewProgram(&model, tea.WithAltScreen())

	// Start watching for emails (must happen after program is created)
	model.WatchEmails(p)
	model.LoadExistingEmails(p)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
