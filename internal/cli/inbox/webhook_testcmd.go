package inbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var webhookTestCmd = &cobra.Command{
	Use:   "test <webhook-id>",
	Short: "Send test request to webhook endpoint",
	Long:  `Send a test request to verify the webhook endpoint is reachable and responding correctly.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhookTest,
}

func init() {
	webhookCmd.AddCommand(webhookTestCmd)
}

func runWebhookTest(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	jsonMode := cliutil.GetOutput(cmd) == "json"
	webhookID := args[0]

	// Show progress
	if !jsonMode {
		fmt.Println(styles.MutedStyle.Render("• Sending test request..."))
	}

	// Load inbox
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, InboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// Test webhook
	resp, err := inbox.TestWebhook(ctx, webhookID)
	if err != nil {
		return fmt.Errorf("failed to test webhook: %w", err)
	}

	// JSON output
	if jsonMode {
		return cliutil.OutputJSON(cliutil.TestWebhookJSON(resp))
	}

	// Pretty output
	printInboxTestResult(resp)
	return nil
}

func printInboxTestResult(resp *vaultsandbox.TestWebhookResponse) {
	fmt.Println()

	if resp.Success {
		fmt.Println(styles.PassStyle.Render("Webhook Test: Success"))
	} else {
		fmt.Println(styles.FailStyle.Render("Webhook Test: Failed"))
	}

	fmt.Println()

	labelStyle := styles.LabelStyle
	fmt.Printf("  %s %d\n", labelStyle.Render("Status Code:"), resp.StatusCode)
	fmt.Printf("  %s %dms\n", labelStyle.Render("Response Time:"), resp.ResponseTime)

	if resp.Error != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("Error:"), resp.Error)
	}

	fmt.Printf("  %s %s\n", labelStyle.Render("Request ID:"), resp.RequestID)
	fmt.Println()
}
