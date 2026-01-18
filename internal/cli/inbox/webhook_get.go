package inbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var webhookGetCmd = &cobra.Command{
	Use:   "get <webhook-id>",
	Short: "Get details of a specific webhook",
	Long:  `Display detailed information about a specific webhook.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWebhookGet,
}

func init() {
	webhookCmd.AddCommand(webhookGetCmd)
}

func runWebhookGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	webhookID := args[0]

	// Load inbox
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, InboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// Get webhook
	webhook, err := inbox.GetWebhook(ctx, webhookID)
	if err != nil {
		return fmt.Errorf("failed to get webhook: %w", err)
	}

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		return cliutil.OutputJSON(cliutil.WebhookFullJSON(webhook))
	}

	// Pretty output
	printInboxWebhookDetails(webhook)
	return nil
}

func printInboxWebhookDetails(wh *vaultsandbox.Webhook) {
	labelStyle := styles.LabelStyle

	fmt.Println()
	fmt.Println(styles.TitleStyle.Render("Webhook Details"))
	fmt.Println()

	// Basic info
	enabled := "No"
	if wh.Enabled {
		enabled = "Yes"
	}

	fmt.Printf("  %s %s\n", labelStyle.Render("ID:"), wh.ID)
	fmt.Printf("  %s %s\n", labelStyle.Render("URL:"), wh.URL)
	fmt.Printf("  %s %s\n", labelStyle.Render("Events:"), cliutil.FormatEventsString(wh.Events))
	fmt.Printf("  %s %s\n", labelStyle.Render("Scope:"), wh.Scope)

	if wh.InboxEmail != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("Inbox:"), wh.InboxEmail)
	}

	if wh.Template != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("Template:"), wh.Template)
	}

	if wh.Description != "" {
		fmt.Printf("  %s %s\n", labelStyle.Render("Description:"), wh.Description)
	}

	fmt.Printf("  %s %s\n", labelStyle.Render("Enabled:"), enabled)
	fmt.Printf("  %s %s\n", labelStyle.Render("Created:"), wh.CreatedAt.Format(cliutil.TimeFormatWithZone))
	fmt.Printf("  %s %s\n", labelStyle.Render("Updated:"), wh.UpdatedAt.Format(cliutil.TimeFormatWithZone))

	// Filters
	if wh.Filter != nil && (len(wh.Filter.Rules) > 0 || wh.Filter.RequireAuth) {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("Filters:"))
		fmt.Printf("  %s %s\n", labelStyle.Render("Mode:"), fmt.Sprintf("%s (%s)", wh.Filter.Mode, inboxFilterModeDescription(wh.Filter.Mode)))

		if wh.Filter.RequireAuth {
			fmt.Printf("  %s %s\n", labelStyle.Render("Require Auth:"), "Yes")
		}

		if len(wh.Filter.Rules) > 0 {
			fmt.Printf("  %s\n", labelStyle.Render("Rules:"))
			for _, r := range wh.Filter.Rules {
				fmt.Printf("    %s %s %s %q\n", styles.MutedStyle.Render("•"), r.Field, r.Operator, r.Value)
			}
		}
	}

	// Delivery stats
	if wh.Stats != nil {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("Delivery Stats:"))

		rate := 0.0
		if wh.Stats.TotalDeliveries > 0 {
			rate = float64(wh.Stats.SuccessfulDeliveries) / float64(wh.Stats.TotalDeliveries) * 100
		}

		fmt.Printf("  %s %d\n", labelStyle.Render("Total:"), wh.Stats.TotalDeliveries)
		fmt.Printf("  %s %d (%.1f%%)\n", labelStyle.Render("Successful:"), wh.Stats.SuccessfulDeliveries, rate)
		fmt.Printf("  %s %d\n", labelStyle.Render("Failed:"), wh.Stats.FailedDeliveries)

		if wh.Stats.LastDeliveryAt != nil {
			fmt.Printf("  %s %s\n", labelStyle.Render("Last Delivery:"), wh.Stats.LastDeliveryAt.Format(cliutil.TimeFormatWithZone))
		}
		if wh.Stats.LastSuccessAt != nil {
			fmt.Printf("  %s %s\n", labelStyle.Render("Last Success:"), wh.Stats.LastSuccessAt.Format(cliutil.TimeFormatWithZone))
		}
	}

	fmt.Println()
}

func inboxFilterModeDescription(mode vaultsandbox.FilterMode) string {
	switch mode {
	case vaultsandbox.FilterModeAll:
		return "AND"
	case vaultsandbox.FilterModeAny:
		return "OR"
	default:
		return strings.ToUpper(string(mode))
	}
}
