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
	Long: `Set the active inbox for commands like 'watch', 'wait-for', etc.

Examples:
  vsb inbox use abc123@vaultsandbox.com`,
	Args: cobra.ExactArgs(1),
	RunE: runInboxUse,
}

func init() {
	inboxCmd.AddCommand(inboxUseCmd)
}

func runInboxUse(cmd *cobra.Command, args []string) error {
	email := args[0]

	keystore, err := config.LoadKeystore()
	if err != nil {
		return fmt.Errorf("failed to load keystore: %w", err)
	}

	if err := keystore.SetActiveInbox(email); err != nil {
		if errors.Is(err, config.ErrInboxNotFound) {
			return fmt.Errorf("inbox not found: %s", email)
		}
		return err
	}

	fmt.Println(output.PrintSuccess(fmt.Sprintf("Active inbox set to %s", email)))
	return nil
}
