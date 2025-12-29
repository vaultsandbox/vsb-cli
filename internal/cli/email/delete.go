package email

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <email-id>",
	Short: "Delete an email",
	Long: `Delete an email from an inbox.

The email is permanently removed from the server.

Examples:
  vsb email delete abc123
  vsb email delete abc123 --inbox foo@abc123.vsx.email`,
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

func init() {
	Cmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	emailID := args[0]

	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, InboxFlag)
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
