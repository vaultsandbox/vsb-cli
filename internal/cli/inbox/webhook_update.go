package inbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var webhookUpdateCmd = &cobra.Command{
	Use:   "update <webhook-id>",
	Short: "Update an existing webhook",
	Long: `Update an existing webhook's configuration.

Examples:
  # Disable webhook
  vsb inbox webhook update wh_abc123 --disable

  # Change URL and add filter
  vsb inbox webhook update wh_abc123 --url https://new-endpoint.com --filter-domain example.com`,
	Args: cobra.ExactArgs(1),
	RunE: runWebhookUpdate,
}

var (
	whUpdateURL            string
	whUpdateEvents         []string
	whUpdateTemplate       string
	whUpdateCustomTemplate string
	whUpdateContentType    string
	whUpdateDescription    string
	whUpdateEnable         bool
	whUpdateDisable        bool
	whUpdateClearFilters   bool
	whUpdateFilters        webhookFilterFlags
)

func init() {
	webhookCmd.AddCommand(webhookUpdateCmd)

	webhookUpdateCmd.Flags().StringVar(&whUpdateURL, "url", "",
		"New endpoint URL")
	webhookUpdateCmd.Flags().StringArrayVar(&whUpdateEvents, "event", nil,
		"Replace events (repeatable)")
	webhookUpdateCmd.Flags().StringVar(&whUpdateTemplate, "template", "",
		"Change to built-in template")
	webhookUpdateCmd.Flags().StringVar(&whUpdateCustomTemplate, "custom-template", "",
		"Change to custom template file")
	webhookUpdateCmd.Flags().StringVar(&whUpdateContentType, "content-type", "application/json",
		"Content-Type for custom template")
	webhookUpdateCmd.Flags().StringVar(&whUpdateDescription, "description", "",
		"Update description")
	webhookUpdateCmd.Flags().BoolVar(&whUpdateEnable, "enable", false,
		"Enable webhook")
	webhookUpdateCmd.Flags().BoolVar(&whUpdateDisable, "disable", false,
		"Disable webhook")
	webhookUpdateCmd.Flags().BoolVar(&whUpdateClearFilters, "clear-filters", false,
		"Remove all filters")

	addWebhookFilterFlags(webhookUpdateCmd, &whUpdateFilters)
}

func runWebhookUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	jsonMode := cliutil.GetOutput(cmd) == "json"
	webhookID := args[0]

	// Validate flags
	if whUpdateEnable && whUpdateDisable {
		return fmt.Errorf("cannot use both --enable and --disable")
	}

	if whUpdateTemplate != "" && whUpdateCustomTemplate != "" {
		return fmt.Errorf("cannot use both --template and --custom-template")
	}

	// Build options
	var opts []vaultsandbox.WebhookUpdateOption

	if whUpdateURL != "" {
		opts = append(opts, vaultsandbox.WithUpdateURL(whUpdateURL))
	}

	if len(whUpdateEvents) > 0 {
		events, err := parseWebhookEvents(whUpdateEvents)
		if err != nil {
			return err
		}
		opts = append(opts, vaultsandbox.WithUpdateEvents(events...))
	}

	if whUpdateTemplate != "" {
		opts = append(opts, vaultsandbox.WithUpdateTemplate(whUpdateTemplate))
	}

	if whUpdateCustomTemplate != "" {
		body, err := loadWebhookCustomTemplate(whUpdateCustomTemplate)
		if err != nil {
			return err
		}
		opts = append(opts, vaultsandbox.WithUpdateCustomTemplate(body, whUpdateContentType))
	}

	if whUpdateDescription != "" {
		opts = append(opts, vaultsandbox.WithUpdateDescription(whUpdateDescription))
	}

	if whUpdateEnable {
		opts = append(opts, vaultsandbox.WithUpdateEnabled(true))
	}

	if whUpdateDisable {
		opts = append(opts, vaultsandbox.WithUpdateEnabled(false))
	}

	if whUpdateClearFilters {
		opts = append(opts, vaultsandbox.WithClearFilter())
	} else {
		// Check if any filter flags were provided
		filter, err := buildWebhookFilterFromFlags(&whUpdateFilters)
		if err != nil {
			return err
		}
		if filter != nil {
			opts = append(opts, vaultsandbox.WithUpdateFilter(filter))
		}
	}

	if len(opts) == 0 {
		return fmt.Errorf("no update flags provided")
	}

	// Show progress
	if !jsonMode {
		fmt.Println(styles.MutedStyle.Render("• Updating webhook..."))
	}

	// Load inbox
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, InboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// Update webhook
	webhook, err := inbox.UpdateWebhook(ctx, webhookID, opts...)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	// Output
	if jsonMode {
		return cliutil.OutputJSON(cliutil.WebhookFullJSON(webhook))
	}

	fmt.Println(styles.PassStyle.Render(fmt.Sprintf("Webhook %s updated successfully", webhookID)))
	fmt.Println()
	printInboxWebhookDetails(webhook)
	return nil
}
