package cli

import (
	"github.com/spf13/cobra"
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "Manage temporary email inboxes",
	Long:  `Create, list, switch, and delete temporary email inboxes.`,
}

func init() {
	rootCmd.AddCommand(inboxCmd)
}
