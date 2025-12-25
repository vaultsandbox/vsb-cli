package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List emails in the active inbox",
	Long: `List all emails in the active inbox.

Displays email ID, subject, sender, and received time.
Use the email ID with other commands like 'vsb view <id>'.

Examples:
  vsb list              # List emails in active inbox
  vsb list --email abc  # List emails in specific inbox
  vsb list -o json      # JSON output`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

var (
	listEmail string
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listEmail, "email", "",
		"Use specific inbox (default: active)")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	inbox, cleanup, err := LoadAndImportInbox(ctx, listEmail)
	if err != nil {
		return err
	}
	defer cleanup()

	emails, err := inbox.GetEmails(ctx)
	if err != nil {
		return fmt.Errorf("failed to get emails: %w", err)
	}

	// JSON output
	if config.GetOutput() == "json" {
		type emailJSON struct {
			ID         string `json:"id"`
			Subject    string `json:"subject"`
			From       string `json:"from"`
			ReceivedAt string `json:"receivedAt"`
		}
		var result []emailJSON
		for _, email := range emails {
			result = append(result, emailJSON{
				ID:         email.ID,
				Subject:    email.Subject,
				From:       email.From,
				ReceivedAt: email.ReceivedAt.Format(time.RFC3339),
			})
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Pretty output
	if len(emails) == 0 {
		fmt.Println("No emails in inbox")
		return nil
	}

	// Styles
	headerStyle := styles.HeaderStyle.MarginBottom(0)
	idStyle := lipgloss.NewStyle().Foreground(styles.Gray)
	subjectStyle := lipgloss.NewStyle().Bold(true)
	fromStyle := lipgloss.NewStyle().Foreground(styles.Purple)
	timeStyle := lipgloss.NewStyle().Foreground(styles.Gray)

	// Header
	fmt.Println()
	fmt.Printf("  %s  %s  %s  %s\n",
		headerStyle.Render(fmt.Sprintf("%-8s", "ID")),
		headerStyle.Render(fmt.Sprintf("%-30s", "SUBJECT")),
		headerStyle.Render(fmt.Sprintf("%-25s", "FROM")),
		headerStyle.Render("RECEIVED"))
	fmt.Println(strings.Repeat("-", 80))

	for _, email := range emails {
		// Truncate fields for display
		id := truncate(email.ID, 8)
		subject := truncate(email.Subject, 30)
		from := truncate(email.From, 25)
		received := formatRelativeTime(email.ReceivedAt)

		fmt.Printf("  %s  %s  %s  %s\n",
			idStyle.Render(fmt.Sprintf("%-8s", id)),
			subjectStyle.Render(fmt.Sprintf("%-30s", subject)),
			fromStyle.Render(fmt.Sprintf("%-25s", from)),
			timeStyle.Render(received))
	}

	fmt.Println()
	fmt.Printf("  %d email(s)\n\n", len(emails))

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}
