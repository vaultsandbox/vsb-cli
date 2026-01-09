package inbox

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the inbox parent command
var Cmd = &cobra.Command{
	Use:   "inbox",
	Short: "Manage temporary email inboxes",
	Long:  `Create, list, switch, and delete temporary email inboxes.`,
	RunE:  runInbox,
}

func runInbox(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return cmd.Help()
}
