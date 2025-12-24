package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/output"
)

var inboxDeleteCmd = &cobra.Command{
	Use:   "delete <email>",
	Short: "Delete an inbox",
	Long: `Delete an inbox from both the server and local keystore.

Examples:
  vsb inbox delete abc123@vaultsandbox.com
  vsb inbox delete --local-only abc123@vaultsandbox.com`,
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE:    runInboxDelete,
}

var (
	deleteLocalOnly bool
)

func init() {
	inboxCmd.AddCommand(inboxDeleteCmd)

	inboxDeleteCmd.Flags().BoolVar(&deleteLocalOnly, "local-only", false,
		"Only remove from local keystore, don't delete on server")
}

func runInboxDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	email := args[0]

	keystore, err := config.LoadKeystore()
	if err != nil {
		return fmt.Errorf("failed to load keystore: %w", err)
	}

	// Delete from server unless --local-only
	if !deleteLocalOnly {
		client, err := config.NewClient()
		if err != nil {
			return err
		}
		defer client.Close()

		if err := client.DeleteInbox(ctx, email); err != nil {
			// Continue with local deletion even if server fails
			fmt.Println(output.PrintError(fmt.Sprintf("Warning: server deletion failed: %v", err)))
		} else {
			fmt.Println(output.PrintSuccess("Deleted from server"))
		}
	}

	// Delete from keystore
	if err := keystore.RemoveInbox(email); err != nil {
		if errors.Is(err, config.ErrInboxNotFound) {
			return fmt.Errorf("inbox not found in keystore: %s", email)
		}
		return err
	}

	fmt.Println(output.PrintSuccess("Deleted from keystore"))
	return nil
}
