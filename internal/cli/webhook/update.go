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

var updateCmd = &cobra.Command{
	Use:   "update <webhook-id>",
	Short: "Update an existing webhook",
	Long: `Update an existing webhook's configuration.

Examples:
  # Disable webhook
  vsb webhook update wh_abc123 --disable

  # Change URL and add filter
  vsb webhook update wh_abc123 --url https://new-endpoint.com --filter-domain example.com

  # Clear all filters
  vsb webhook update wh_abc123 --clear-filters`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdate,
}

var (
	updateURL            string
	updateEvents         []string
	updateTemplate       string
	updateCustomTemplate string
	updateContentType    string
	updateDescription    string
	updateEnable         bool
	updateDisable        bool
	updateClearFilters   bool
	updateFilters        filterFlags
)

func init() {
	Cmd.AddCommand(updateCmd)

	updateCmd.Flags().StringVar(&updateURL, "url", "",
		"New endpoint URL")
	updateCmd.Flags().StringArrayVar(&updateEvents, "event", nil,
		"Replace events (repeatable)")
	updateCmd.Flags().StringVar(&updateTemplate, "template", "",
		"Change to built-in template")
	updateCmd.Flags().StringVar(&updateCustomTemplate, "custom-template", "",
		"Change to custom template file")
	updateCmd.Flags().StringVar(&updateContentType, "content-type", "application/json",
		"Content-Type for custom template")
	updateCmd.Flags().StringVar(&updateDescription, "description", "",
		"Update description")
	updateCmd.Flags().BoolVar(&updateEnable, "enable", false,
		"Enable webhook")
	updateCmd.Flags().BoolVar(&updateDisable, "disable", false,
		"Disable webhook")
	updateCmd.Flags().BoolVar(&updateClearFilters, "clear-filters", false,
		"Remove all filters")

	addFilterFlags(updateCmd, &updateFilters)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	jsonMode := cliutil.GetOutput(cmd) == "json"
	webhookID := args[0]

	// Validate flags
	if updateEnable && updateDisable {
		return fmt.Errorf("cannot use both --enable and --disable")
	}

	if updateTemplate != "" && updateCustomTemplate != "" {
		return fmt.Errorf("cannot use both --template and --custom-template")
	}

	// Build options
	var opts []vaultsandbox.WebhookUpdateOption

	if updateURL != "" {
		opts = append(opts, vaultsandbox.WithUpdateURL(updateURL))
	}

	if len(updateEvents) > 0 {
		events, err := parseEvents(updateEvents)
		if err != nil {
			return err
		}
		opts = append(opts, vaultsandbox.WithUpdateEvents(events...))
	}

	if updateTemplate != "" {
		opts = append(opts, vaultsandbox.WithUpdateTemplate(updateTemplate))
	}

	if updateCustomTemplate != "" {
		body, err := loadCustomTemplate(updateCustomTemplate)
		if err != nil {
			return err
		}
		opts = append(opts, vaultsandbox.WithUpdateCustomTemplate(body, updateContentType))
	}

	if updateDescription != "" {
		opts = append(opts, vaultsandbox.WithUpdateDescription(updateDescription))
	}

	if updateEnable {
		opts = append(opts, vaultsandbox.WithUpdateEnabled(true))
	}

	if updateDisable {
		opts = append(opts, vaultsandbox.WithUpdateEnabled(false))
	}

	if updateClearFilters {
		opts = append(opts, vaultsandbox.WithClearFilter())
	} else {
		// Check if any filter flags were provided
		filter, err := buildFilterFromFlags(&updateFilters)
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

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Update webhook
	webhook, err := client.Admin().UpdateWebhook(ctx, webhookID, opts...)
	if err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	// Output
	if jsonMode {
		return cliutil.OutputJSON(cliutil.WebhookFullJSON(webhook))
	}

	fmt.Println(styles.PassStyle.Render(fmt.Sprintf("Webhook %s updated successfully", webhookID)))
	fmt.Println()
	printWebhookDetails(webhook)
	return nil
}
