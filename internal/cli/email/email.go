package email

import (
	"github.com/spf13/cobra"
)

// Cmd is the email parent command
var Cmd = &cobra.Command{
	Use:   "email",
	Short: "View and manage emails",
	Long:  `List, view, audit, and delete emails in your inboxes.`,
}
