package webhook

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all global webhooks",
	Long:    `Display all global webhooks for your account.`,
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	Cmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// List webhooks
	resp, err := client.Admin().ListWebhooks(ctx)
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
		fmt.Println("No webhooks found. Create one with 'vsb webhook create'")
		return nil
	}

	printWebhookList(resp.Webhooks)

	fmt.Printf("\nTotal: %d webhook(s)\n\n", len(resp.Webhooks))
	return nil
}

func printWebhookList(webhooks []*vaultsandbox.Webhook) {
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
