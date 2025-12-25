package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/output"
)

var inboxUseCmd = &cobra.Command{
	Use:   "use <email>",
	Short: "Switch active inbox",
	Long: `Set the active inbox for commands.

Supports partial matching - if only one inbox contains the given string,
it will be selected automatically.

Examples:
  vsb inbox use abc123@vaultsandbox.com
  vsb inbox use abc     # Partial match`,
	Args: cobra.ExactArgs(1),
	RunE: runInboxUse,
}

func init() {
	inboxCmd.AddCommand(inboxUseCmd)
}

func runInboxUse(cmd *cobra.Command, args []string) error {
	partial := args[0]

	keystore, err := config.LoadKeystore()
	if err != nil {
		return fmt.Errorf("failed to load keystore: %w", err)
	}

	// Find inbox with partial matching
	inbox, matches, err := keystore.FindInbox(partial)
	if err == config.ErrMultipleMatches {
		return fmt.Errorf("multiple inboxes match '%s': %v", partial, matches)
	}
	if errors.Is(err, config.ErrInboxNotFound) {
		return fmt.Errorf("inbox not found: %s", partial)
	}
	if err != nil {
		return err
	}

	if err := keystore.SetActiveInbox(inbox.Email); err != nil {
		return err
	}

	fmt.Println(output.PrintSuccess(fmt.Sprintf("Active inbox set to %s", inbox.Email)))
	return nil
}
