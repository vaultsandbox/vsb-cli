package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var emailDeleteCmd = &cobra.Command{
	Use:   "delete <email-id>",
	Short: "Delete an email",
	Long: `Delete an email from an inbox.

The email is permanently removed from the server.

Examples:
  vsb email delete abc123
  vsb email delete abc123 --inbox foo@vaultsandbox.com`,
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE:    runEmailDelete,
}

var (
	emailDeleteInbox string
)

func init() {
	emailCmd.AddCommand(emailDeleteCmd)

	emailDeleteCmd.Flags().StringVar(&emailDeleteInbox, "inbox", "",
		"Use specific inbox (default: active)")
}

func runEmailDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	emailID := args[0]

	inbox, cleanup, err := LoadAndImportInbox(ctx, emailDeleteInbox)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := inbox.DeleteEmail(ctx, emailID); err != nil {
		return fmt.Errorf("failed to delete email: %w", err)
	}

	fmt.Println(styles.PassStyle.Render(fmt.Sprintf("✓ Deleted email %s", emailID)))
	return nil
}
