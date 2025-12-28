package email

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
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
	Cmd.AddCommand(listCmd)

	listCmd.Flags().StringVar(&listInbox, "inbox", "",
		"Use specific inbox (default: active)")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, listInbox)
	if err != nil {
		return err
	}
	defer cleanup()

	emails, err := inbox.GetEmails(ctx)
	if err != nil {
		return fmt.Errorf("failed to get emails: %w", err)
	}

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		var result []map[string]interface{}
		for _, email := range emails {
			result = append(result, cliutil.EmailSummaryJSON(email))
		}
		return cliutil.OutputJSON(result)
	}

	// Pretty output
	if len(emails) == 0 {
		fmt.Println("No emails in inbox")
		return nil
	}

	// Header
	table := cliutil.NewTable(
		cliutil.Column{Header: "ID", Width: 8},
		cliutil.Column{Header: "SUBJECT", Width: 30},
		cliutil.Column{Header: "FROM", Width: 25},
		cliutil.Column{Header: "RECEIVED"},
	)
	table.PrintHeader()

	for _, email := range emails {
		// Truncate and pad fields for display
		id := fmt.Sprintf("%-8s", cliutil.Truncate(email.ID, 8))
		subject := fmt.Sprintf("%-30s", cliutil.Truncate(email.Subject, 30))
		from := fmt.Sprintf("%-25s", cliutil.Truncate(email.From, 25))
		received := cliutil.FormatRelativeTime(email.ReceivedAt)

		fmt.Printf("  %s  %s  %s  %s\n",
			styles.IDStyle.Render(id),
			styles.SubjectStyle.Render(subject),
			styles.FromStyle.Render(from),
			styles.TimeStyle.Render(received))
	}

	fmt.Println()
	fmt.Printf("  %d email(s)\n\n", len(emails))

	return nil
}
