package cli

import (
	"github.com/spf13/cobra"
)

var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "View and manage emails",
	Long:  `List, view, audit, and delete emails in your inboxes.`,
}

func init() {
	rootCmd.AddCommand(emailCmd)
}
