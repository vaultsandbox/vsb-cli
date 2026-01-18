package webhook

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show global webhook delivery metrics",
	Long:  `Display overall webhook delivery statistics and metrics.`,
	RunE:  runMetrics,
}

func init() {
	Cmd.AddCommand(metricsCmd)
}

func runMetrics(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Get metrics
	metrics, err := client.GetWebhookMetrics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		return cliutil.OutputJSON(cliutil.WebhookMetricsJSON(metrics))
	}

	// Pretty output
	printMetrics(metrics)
	return nil
}

func printMetrics(m *vaultsandbox.WebhookMetrics) {
	labelStyle := styles.LabelStyle
	divider := styles.MutedStyle.Render("────────────────────────────")

	fmt.Println()
	fmt.Println(styles.TitleStyle.Render("Webhook Metrics"))
	fmt.Println()

	// Overview
	fmt.Println(styles.SectionStyle.Render("  Overview"))
	fmt.Println("  " + divider)
	fmt.Printf("  %s %d\n", labelStyle.Render("Total Webhooks:"), m.TotalWebhooks)
	fmt.Printf("  %s %d\n", labelStyle.Render("Active Webhooks:"), m.ActiveWebhooks)

	// Delivery Stats
	fmt.Println()
	fmt.Println(styles.SectionStyle.Render("  Delivery Stats"))
	fmt.Println("  " + divider)
	fmt.Printf("  %s %d\n", labelStyle.Render("Total Deliveries:"), m.TotalDeliveries)

	successStyle := styles.PassStyle
	if m.SuccessRate < 90 {
		successStyle = styles.WarnStyle
	}
	if m.SuccessRate < 70 {
		successStyle = styles.FailStyle
	}
	fmt.Printf("  %s %s\n", labelStyle.Render("Successful:"),
		successStyle.Render(fmt.Sprintf("%d (%.1f%%)", m.SuccessfulDeliveries, m.SuccessRate)))
	fmt.Printf("  %s %d (%.1f%%)\n", labelStyle.Render("Failed:"),
		m.FailedDeliveries, 100-m.SuccessRate)

	// By Scope
	if len(m.ByScope) > 0 {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("  By Scope"))
		fmt.Println("  " + divider)
		for scope, count := range m.ByScope {
			fmt.Printf("  %s %d\n", labelStyle.Render(scope+":"), count)
		}
	}

	// By Event
	if len(m.ByEvent) > 0 {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("  By Event"))
		fmt.Println("  " + divider)

		// Sort events for consistent output
		var events []string
		for event := range m.ByEvent {
			events = append(events, event)
		}
		sort.Strings(events)

		for _, event := range events {
			fmt.Printf("  %s %d\n", labelStyle.Render(event+":"), m.ByEvent[event])
		}
	}

	fmt.Println()
}
