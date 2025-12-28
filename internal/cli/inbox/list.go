package inbox

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all stored inboxes",
	Long:    `Display all inboxes stored in the local keystore.`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

var (
	listShowExpired bool
)

func init() {
	Cmd.AddCommand(listCmd)

	listCmd.Flags().BoolVarP(&listShowExpired, "all", "a", false,
		"Show expired inboxes too")
}

// filterInboxes returns inboxes, optionally filtering out expired ones.
func filterInboxes(inboxes []config.StoredInbox, showExpired bool, now time.Time) []config.StoredInbox {
	var filtered []config.StoredInbox
	for _, inbox := range inboxes {
		isExpired := inbox.ExpiresAt.Before(now)
		if isExpired && !showExpired {
			continue
		}
		filtered = append(filtered, inbox)
	}
	return filtered
}

func runList(cmd *cobra.Command, args []string) error {
	keystore, err := cliutil.LoadKeystoreOrError()
	if err != nil {
		return err
	}

	inboxes := keystore.ListInboxes()
	now := time.Now()

	filtered := filterInboxes(inboxes, listShowExpired, now)

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		var result []map[string]interface{}
		for _, inbox := range filtered {
			isActive := inbox.Email == keystore.ActiveInbox
			result = append(result, cliutil.InboxSummaryJSON(&inbox, isActive, now))
		}
		return cliutil.OutputJSON(result)
	}

	// Pretty output
	if len(filtered) == 0 {
		fmt.Println("No inboxes found. Create one with 'vsb inbox create'")
		return nil
	}

	// Header
	table := cliutil.NewTable(
		cliutil.Column{Header: "EMAIL", Width: styles.ColWidthEmail},
		cliutil.Column{Header: "EXPIRES"},
	).WithIndent("   ")
	table.PrintHeader()

	for _, inbox := range filtered {
		isActive := inbox.Email == keystore.ActiveInbox
		isExpired := inbox.ExpiresAt.Before(now)

		// Active marker
		marker := "  "
		if isActive {
			marker = styles.ActiveStyle.Render("> ")
		}

		// Email (pad before styling to preserve alignment)
		emailPadded := fmt.Sprintf("%-*s", styles.ColWidthEmail, inbox.Email)
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
			expiry = cliutil.FormatDuration(remaining)
		}

		fmt.Printf("%s%s  %s\n", marker, emailPadded, expiry)
	}

	fmt.Println()
	return nil
}

