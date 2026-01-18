package webhook

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the webhook parent command
var Cmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage global webhooks",
	Long: `Create, list, update, and delete global webhooks.

Global webhooks receive notifications for all inboxes associated with your API key.
For inbox-specific webhooks, use 'vsb inbox webhook'.`,
	RunE: runWebhook,
}

func runWebhook(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return cmd.Help()
}
