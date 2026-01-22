package inbox

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var chaosDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable all chaos",
	Long:  `Disable all chaos engineering scenarios for an inbox.`,
	Args:  cobra.NoArgs,
	RunE:  runChaosDisable,
}

var chaosDisableForce bool

func init() {
	chaosCmd.AddCommand(chaosDisableCmd)

	chaosDisableCmd.Flags().BoolVarP(&chaosDisableForce, "force", "f", false,
		"Skip confirmation prompt")
}

func runChaosDisable(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check if chaos is enabled on server
	if err := checkChaosEnabled(); err != nil {
		return err
	}

	// Load inbox first to get email for confirmation
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, ChaosInboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// Confirmation
	if !chaosDisableForce {
		fmt.Printf("Disable all chaos for %s? [y/N]: ", inbox.EmailAddress())
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// Disable chaos
	if err := inbox.DisableChaos(ctx); err != nil {
		return fmt.Errorf("failed to disable chaos: %w", err)
	}

	// Output
	if cliutil.GetOutput(cmd) == "json" {
		return cliutil.OutputJSON(map[string]interface{}{
			"email":    inbox.EmailAddress(),
			"disabled": true,
		})
	}

	fmt.Println(styles.PassStyle.Render(fmt.Sprintf("Chaos disabled for %s", inbox.EmailAddress())))
	return nil
}
