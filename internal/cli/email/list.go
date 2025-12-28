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

func init() {
	Cmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, InboxFlag)
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
		cliutil.Column{Header: "ID", Width: styles.ColWidthID}.WithStyle(styles.IDStyle),
		cliutil.Column{Header: "SUBJECT", Width: styles.ColWidthSubject}.WithStyle(styles.SubjectStyle),
		cliutil.Column{Header: "FROM", Width: styles.ColWidthFrom}.WithStyle(styles.FromStyle),
		cliutil.Column{Header: "RECEIVED"}.WithStyle(styles.TimeStyle),
	)
	table.PrintHeader()

	for _, email := range emails {
		table.PrintRow(email.ID, email.Subject, email.From, cliutil.FormatRelativeTime(email.ReceivedAt))
	}

	fmt.Println()
	fmt.Printf("  %d email(s)\n\n", len(emails))

	return nil
}
