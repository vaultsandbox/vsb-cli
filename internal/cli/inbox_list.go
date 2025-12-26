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

	inboxListCmd.Flags().BoolVarP(&listShowExpired, "all", "a", false,
		"Show expired inboxes too")
}

func runInboxList(cmd *cobra.Command, args []string) error {
	keystore, err := LoadKeystoreOrError()
	if err != nil {
		return err
	}

	inboxes := keystore.ListInboxes()
	now := time.Now()

	// Filter expired if needed
	var filtered []config.StoredInbox
	for _, inbox := range inboxes {
		isExpired := inbox.ExpiresAt.Before(now)
		if isExpired && !listShowExpired {
			continue
		}
		filtered = append(filtered, inbox)
	}

	// JSON output
	if getOutput(cmd) == "json" {
		var result []map[string]interface{}
		for _, inbox := range filtered {
			isActive := inbox.Email == keystore.ActiveInbox
			result = append(result, InboxSummaryJSON(&inbox, isActive, now))
		}
		return outputJSON(result)
	}

	// Pretty output
	if len(filtered) == 0 {
		fmt.Println("No inboxes found. Create one with 'vsb inbox create'")
		return nil
	}

	// Header
	headerStyle := styles.HeaderStyle.MarginBottom(0)
	fmt.Println()
	fmt.Printf("   %s  %s\n",
		headerStyle.Render(fmt.Sprintf("%-38s", "EMAIL")),
		headerStyle.Render("EXPIRES"))
	fmt.Println(strings.Repeat("-", 55))

	for _, inbox := range filtered {
		isActive := inbox.Email == keystore.ActiveInbox
		isExpired := inbox.ExpiresAt.Before(now)

		// Active marker
		marker := "  "
		if isActive {
			marker = styles.ActiveStyle.Render("> ")
		}

		// Email (pad before styling to preserve alignment)
		emailPadded := fmt.Sprintf("%-38s", inbox.Email)
		if isExpired {
			emailPadded = styles.ExpiredStyle.Render(emailPadded)
		} else if isActive {
			emailPadded = styles.ActiveStyle.Render(emailPadded)
		}

		// Expiry
		var expiry string
		if isExpired {
			expiry = styles.ExpiredStyle.Render("expired")
		} else {
			remaining := inbox.ExpiresAt.Sub(now).Round(time.Minute)
			expiry = formatDuration(remaining)
		}

		fmt.Printf("%s%s  %s\n", marker, emailPadded, expiry)
	}

	fmt.Println()
	return nil
}

