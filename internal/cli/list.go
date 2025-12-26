package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List emails in the active inbox",
	Long: `List all emails in the active inbox.

Displays email ID, subject, sender, and received time.
Use the email ID with other commands like 'vsb view <id>'.

Examples:
  vsb email list              # List emails in active inbox
  vsb email list --inbox abc  # List emails in specific inbox
  vsb email list -o json      # JSON output`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

var (
	listInbox string
)

func init() {
	emailCmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listInbox, "inbox", "",
		"Use specific inbox (default: active)")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	inbox, cleanup, err := LoadAndImportInbox(ctx, listInbox)
	if err != nil {
		return err
	}
	defer cleanup()

	emails, err := inbox.GetEmails(ctx)
	if err != nil {
		return fmt.Errorf("failed to get emails: %w", err)
	}

	// JSON output
	if getOutput(cmd) == "json" {
		var result []map[string]interface{}
		for _, email := range emails {
			result = append(result, EmailSummaryJSON(email))
		}
		return outputJSON(result)
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
	fromStyle := lipgloss.NewStyle().Foreground(styles.Primary)
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
