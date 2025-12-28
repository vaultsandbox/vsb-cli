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

// InboxFlag is shared across all email subcommands
var InboxFlag string

func init() {
	Cmd.PersistentFlags().StringVar(&InboxFlag, "inbox", "",
		"Use specific inbox (default: active)")
}
