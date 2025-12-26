package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
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

	ks, err := LoadKeystoreOrError()
	if err != nil {
		return err
	}

	inbox, err := GetInbox(ks, partial)
	if err != nil {
		return err
	}

	if err := ks.SetActiveInbox(inbox.Email); err != nil {
		return err
	}

	fmt.Println(styles.PassStyle.Render(fmt.Sprintf("✓ Active inbox set to %s", inbox.Email)))
	return nil
}
