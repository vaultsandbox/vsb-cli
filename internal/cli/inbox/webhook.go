package inbox

import (
	"fmt"

	"github.com/spf13/cobra"
)

// webhookCmd is the inbox webhook parent command
var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage inbox-scoped webhooks",
	Long: `Create, list, update, and delete webhooks for a specific inbox.

Inbox-scoped webhooks receive notifications only for emails to the specified inbox.
For global webhooks, use 'vsb webhook'.`,
	RunE: runWebhook,
}

// InboxFlag is used by webhook subcommands to specify which inbox to operate on
var InboxFlag string

func init() {
	Cmd.AddCommand(webhookCmd)

	// Persistent flag for all webhook subcommands
	webhookCmd.PersistentFlags().StringVar(&InboxFlag, "inbox", "",
		"Use specific inbox (default: active)")
}

func runWebhook(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return cmd.Help()
}
