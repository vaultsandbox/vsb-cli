package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var inboxListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all stored inboxes",
	Long:    `Display all inboxes stored in the local keystore.`,
	Aliases: []string{"ls"},
	RunE:    runInboxList,
}

var (
	listShowExpired bool
)

func init() {
	inboxCmd.AddCommand(inboxListCmd)

	inboxListCmd.Flags().BoolVar(&listShowExpired, "all", false,
		"Show expired inboxes too")
}

func runInboxList(cmd *cobra.Command, args []string) error {
	keystore, err := config.LoadKeystore()
	if err != nil {
		return fmt.Errorf("failed to load keystore: %w", err)
	}

	inboxes := keystore.ListInboxes()
	if len(inboxes) == 0 {
		fmt.Println("No inboxes found. Create one with 'vsb inbox create'")
		return nil
	}

	now := time.Now()

	// Header
	fmt.Println()
	fmt.Printf("%s  %-35s  %-12s  %s\n",
		styles.HeaderStyle.Render(" "),
		styles.HeaderStyle.Render("EMAIL"),
		styles.HeaderStyle.Render("LABEL"),
		styles.HeaderStyle.Render("EXPIRES"))
	fmt.Println(strings.Repeat("-", 70))

	for _, inbox := range inboxes {
		isActive := inbox.Email == keystore.ActiveInbox
		isExpired := inbox.ExpiresAt.Before(now)

		if isExpired && !listShowExpired {
			continue
		}

		// Active marker
		marker := "  "
		if isActive {
			marker = styles.ActiveStyle.Render("> ")
		}

		// Email
		email := inbox.Email
		if isExpired {
			email = styles.ExpiredStyle.Render(email)
		} else if isActive {
			email = styles.ActiveStyle.Render(email)
		}

		// Label
		label := inbox.Label
		if label == "" {
			label = "-"
		}

		// Expiry
		var expiry string
		if isExpired {
			expiry = styles.ExpiredStyle.Render("expired")
		} else {
			remaining := inbox.ExpiresAt.Sub(now).Round(time.Minute)
			expiry = formatDuration(remaining)
		}

		fmt.Printf("%s%-35s  %-12s  %s\n", marker, email, label, expiry)
	}

	fmt.Println()
	return nil
}

