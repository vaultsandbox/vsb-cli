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

var (
	rootAll   bool
	rootEmail string
)

var rootCmd = &cobra.Command{
	Use:   "vsb",
	Short: "VaultSandbox CLI - Test email flows with quantum-safe encryption",
	Long: `vsb is a developer companion for testing email flows.

It provides temporary inboxes with end-to-end encryption using
quantum-safe algorithms (ML-KEM-768, ML-DSA-65).

The server never sees your email content - all decryption
happens locally on your machine.

Running 'vsb' with no subcommand opens the real-time email dashboard.

Examples:
  vsb                 # Watch active inbox
  vsb -a              # Watch all stored inboxes
  vsb --email abc@vaultsandbox.com`,
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

	// Root command flags (TUI)
	rootCmd.Flags().BoolVarP(&rootAll, "all", "a", false,
		"Watch all stored inboxes")
	rootCmd.Flags().StringVar(&rootEmail, "email", "",
		"Watch specific inbox by email address")
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

	// Determine which inboxes to watch
	var storedInboxes []config.StoredInbox

	if rootAll {
		storedInboxes = keystore.ListInboxes()
		if len(storedInboxes) == 0 {
			return fmt.Errorf("no inboxes found. Create one with 'vsb inbox create'")
		}
	} else {
		inbox, err := GetInbox(keystore, rootEmail)
		if err != nil {
			return err
		}
		storedInboxes = []config.StoredInbox{*inbox}
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

	// Create TUI model
	model := watch.NewModel(client, inboxes, rootAll)

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
