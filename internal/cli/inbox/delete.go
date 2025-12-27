package inbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <email>",
	Short: "Delete an inbox",
	Long: `Delete an inbox from both the server and local keystore.

Supports partial matching - if only one inbox contains the given string,
it will be deleted automatically.

Examples:
  vsb inbox delete abc123@vaultsandbox.com
  vsb inbox delete abc       # Partial match
  vsb inbox delete abc -l    # Local only (don't delete on server)`,
	Aliases: []string{"rm"},
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

var (
	deleteLocal bool
)

func init() {
	Cmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVarP(&deleteLocal, "local", "l", false,
		"Only remove from local keystore, don't delete on server")
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	partial := args[0]

	ks, err := cliutil.LoadKeystoreOrError()
	if err != nil {
		return err
	}

	inbox, err := cliutil.GetInbox(ks, partial)
	if err != nil {
		return err
	}
	email := inbox.Email

	// Delete from server unless --local
	if !deleteLocal {
		client, err := config.NewClient()
		if err != nil {
			return err
		}
		defer client.Close()

		if err := client.DeleteInbox(ctx, email); err != nil {
			// Continue with local deletion even if server fails
			fmt.Println(styles.FailStyle.Render(fmt.Sprintf("✗ Warning: server deletion failed: %v", err)))
		} else {
			fmt.Println(styles.PassStyle.Render("✓ Deleted from server"))
		}
	}

	// Delete from keystore
	if err := ks.RemoveInbox(email); err != nil {
		if errors.Is(err, config.ErrInboxNotFound) {
			return fmt.Errorf("inbox not found in keystore: %s", email)
		}
		return err
	}

	fmt.Println(styles.PassStyle.Render("✓ Deleted from keystore"))
	return nil
}
