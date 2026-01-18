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

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete <webhook-id>",
	Short: "Delete a webhook",
	Long:  `Delete a webhook from this inbox. This action cannot be undone.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhookDelete,
}

var whDeleteForce bool

func init() {
	webhookCmd.AddCommand(webhookDeleteCmd)

	webhookDeleteCmd.Flags().BoolVarP(&whDeleteForce, "force", "f", false,
		"Skip confirmation prompt")
}

func runWebhookDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	webhookID := args[0]

	// Confirmation
	if !whDeleteForce {
		fmt.Printf("Delete webhook %s? This cannot be undone. [y/N]: ", webhookID)
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

	// Load inbox
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, InboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// Delete webhook
	if err := inbox.DeleteWebhook(ctx, webhookID); err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	// Output
	if cliutil.GetOutput(cmd) == "json" {
		return cliutil.OutputJSON(map[string]interface{}{
			"id":      webhookID,
			"deleted": true,
		})
	}

	fmt.Println(styles.PassStyle.Render(fmt.Sprintf("Webhook %s deleted successfully", webhookID)))
	return nil
}
