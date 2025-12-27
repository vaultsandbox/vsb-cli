package inbox

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var useCmd = &cobra.Command{
	Use:   "use <email>",
	Short: "Switch active inbox",
	Long: `Set the active inbox for commands.

Supports partial matching - if only one inbox contains the given string,
it will be selected automatically.

Examples:
  vsb inbox use abc123@vaultsandbox.com
  vsb inbox use abc     # Partial match`,
	Args: cobra.ExactArgs(1),
	RunE: runUse,
}

func init() {
	Cmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	partial := args[0]

	ks, err := cliutil.LoadKeystoreOrError()
	if err != nil {
		return err
	}

	inbox, err := cliutil.GetInbox(ks, partial)
	if err != nil {
		return err
	}

	if err := ks.SetActiveInbox(inbox.Email); err != nil {
		return err
	}

	fmt.Println(styles.PassStyle.Render(fmt.Sprintf("✓ Active inbox set to %s", inbox.Email)))
	return nil
}
