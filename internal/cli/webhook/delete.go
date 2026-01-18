package webhook

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var deleteCmd = &cobra.Command{
	Use:     "delete <webhook-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a webhook",
	Long:    `Delete a webhook. This action cannot be undone.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runDelete,
}

var deleteForce bool

func init() {
	Cmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false,
		"Skip confirmation prompt")
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	webhookID := args[0]

	// Confirmation
	if !deleteForce {
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

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Delete webhook
	if err := client.Admin().DeleteWebhook(ctx, webhookID); err != nil {
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
