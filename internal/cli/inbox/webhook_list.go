package inbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var webhookListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List webhooks for this inbox",
	Long:    `Display all webhooks configured for this inbox.`,
	Aliases: []string{"ls"},
	RunE:    runWebhookList,
}

func init() {
	webhookCmd.AddCommand(webhookListCmd)
}

func runWebhookList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load inbox
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, InboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// List webhooks
	resp, err := inbox.ListWebhooks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", err)
	}

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		var result []map[string]interface{}
		for _, wh := range resp.Webhooks {
			result = append(result, cliutil.WebhookSummaryJSON(wh))
		}
		return cliutil.OutputJSON(result)
	}

	// Pretty output
	if len(resp.Webhooks) == 0 {
		fmt.Println("No webhooks found for this inbox. Create one with 'vsb inbox webhook create'")
		return nil
	}

	printInboxWebhookList(resp.Webhooks)

	fmt.Printf("\nTotal: %d webhook(s)\n\n", len(resp.Webhooks))
	return nil
}

func printInboxWebhookList(webhooks []*vaultsandbox.Webhook) {
	table := cliutil.NewTable(
		cliutil.Column{Header: "ID", Width: 15}.WithStyle(styles.IDStyle),
		cliutil.Column{Header: "URL", Width: 35},
		cliutil.Column{Header: "EVENTS", Width: 20},
		cliutil.Column{Header: "ENABLED", Width: 8},
		cliutil.Column{Header: "DELIVERIES", Width: 15},
	).WithIndent("   ")

	table.PrintHeader()

	for _, wh := range webhooks {
		// Enabled status
		enabled := styles.FailStyle.Render("No")
		if wh.Enabled {
			enabled = styles.PassStyle.Render("Yes")
		}

		// Delivery stats
		deliveries := "0 (0%)"
		if wh.Stats != nil && wh.Stats.TotalDeliveries > 0 {
			rate := float64(wh.Stats.SuccessfulDeliveries) / float64(wh.Stats.TotalDeliveries) * 100
			deliveries = fmt.Sprintf("%d (%.0f%%)", wh.Stats.TotalDeliveries, rate)
		}

		table.PrintRow(
			wh.ID,
			cliutil.Truncate(wh.URL, 35),
			cliutil.FormatEventsString(wh.Events),
			enabled,
			deliveries,
		)
	}
}
