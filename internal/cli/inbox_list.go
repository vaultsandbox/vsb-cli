package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
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

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED"))

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#10B981"))

	expiredStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Strikethrough(true)

	now := time.Now()

	// Header
	fmt.Println()
	fmt.Printf("%s  %-35s  %-12s  %s\n",
		headerStyle.Render(" "),
		headerStyle.Render("EMAIL"),
		headerStyle.Render("LABEL"),
		headerStyle.Render("EXPIRES"))
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
			marker = activeStyle.Render("> ")
		}

		// Email
		email := inbox.Email
		if isExpired {
			email = expiredStyle.Render(email)
		} else if isActive {
			email = activeStyle.Render(email)
		}

		// Label
		label := inbox.Label
		if label == "" {
			label = "-"
		}

		// Expiry
		var expiry string
		if isExpired {
			expiry = expiredStyle.Render("expired")
		} else {
			remaining := inbox.ExpiresAt.Sub(now).Round(time.Minute)
			expiry = formatDuration(remaining)
		}

		fmt.Printf("%s%-35s  %-12s  %s\n", marker, email, label, expiry)
	}

	fmt.Println()
	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
