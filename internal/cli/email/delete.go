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
  vsb email delete abc123 --inbox foo@vaultsandbox.com`,
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

var (
	deleteInbox string
)

func init() {
	Cmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringVar(&deleteInbox, "inbox", "",
		"Use specific inbox (default: active)")
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	emailID := args[0]

	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, deleteInbox)
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
