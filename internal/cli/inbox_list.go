package cli

import (
	"encoding/json"
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
	keystore, err := config.LoadKeystore()
	if err != nil {
		return fmt.Errorf("failed to load keystore: %w", err)
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
	if config.GetOutput() == "json" {
		type inboxJSON struct {
			Email     string `json:"email"`
			ExpiresAt string `json:"expiresAt"`
			IsActive  bool   `json:"isActive"`
			IsExpired bool   `json:"isExpired"`
		}
		var result []inboxJSON
		for _, inbox := range filtered {
			result = append(result, inboxJSON{
				Email:     inbox.Email,
				ExpiresAt: inbox.ExpiresAt.Format(time.RFC3339),
				IsActive:  inbox.Email == keystore.ActiveInbox,
				IsExpired: inbox.ExpiresAt.Before(now),
			})
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Pretty output
	if len(filtered) == 0 {
		fmt.Println("No inboxes found. Create one with 'vsb inbox create'")
		return nil
	}

	// Header
	fmt.Println()
	fmt.Printf("%s  %-38s  %s\n",
		styles.HeaderStyle.Render(" "),
		styles.HeaderStyle.Render("EMAIL"),
		styles.HeaderStyle.Render("EXPIRES"))
	fmt.Println(strings.Repeat("-", 55))

	for _, inbox := range filtered {
		isActive := inbox.Email == keystore.ActiveInbox
		isExpired := inbox.ExpiresAt.Before(now)

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

		// Expiry
		var expiry string
		if isExpired {
			expiry = styles.ExpiredStyle.Render("expired")
		} else {
			remaining := inbox.ExpiresAt.Sub(now).Round(time.Minute)
			expiry = formatDuration(remaining)
		}

		fmt.Printf("%s%-38s  %s\n", marker, email, expiry)
	}

	fmt.Println()
	return nil
}

