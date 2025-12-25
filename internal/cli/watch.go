package cli

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/tui/watch"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for incoming emails in real-time",
	Long: `Open a real-time dashboard showing emails as they arrive.

Uses Server-Sent Events (SSE) for instant notifications.
All emails are decrypted locally using your stored private keys.

Examples:
  vsb watch           # Watch active inbox
  vsb watch --all     # Watch all stored inboxes
  vsb watch --email abc@vaultsandbox.com`,
	RunE: runWatch,
}

var (
	watchAll   bool
	watchEmail string
)

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().BoolVar(&watchAll, "all", false,
		"Watch all stored inboxes")
	watchCmd.Flags().StringVar(&watchEmail, "email", "",
		"Watch specific inbox by email address")
}

func runWatch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load keystore
	keystore, err := LoadKeystoreOrError()
	if err != nil {
		return err
	}

	// Determine which inboxes to watch
	var storedInboxes []config.StoredInbox

	if watchAll {
		storedInboxes = keystore.ListInboxes()
		if len(storedInboxes) == 0 {
			return fmt.Errorf("no inboxes found. Create one with 'vsb inbox create'")
		}
	} else {
		inbox, err := GetInbox(keystore, watchEmail)
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
	model := watch.NewModel(client, inboxes, watchAll)

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
