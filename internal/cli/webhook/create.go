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

var createCmd = &cobra.Command{
	Use:   "create <url>",
	Short: "Create a new global webhook",
	Long: `Create a new global webhook that receives notifications for all inboxes.

The webhook will be triggered when the specified events occur. You can use
built-in templates (slack, discord, teams) or provide a custom template.

Examples:
  # Basic webhook for received emails
  vsb webhook create https://example.com/webhook --event email.received

  # Slack notification with filter
  vsb webhook create https://hooks.slack.com/xxx \
    --event email.received \
    --template slack \
    --filter-subject-contains "alert" \
    --description "Alert notifications"

  # Custom template
  vsb webhook create https://api.example.com/notify \
    --event email.received \
    --custom-template ./template.json`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

var (
	createEvents         []string
	createTemplate       string
	createCustomTemplate string
	createContentType    string
	createDescription    string
	createFilters        filterFlags
)

func init() {
	Cmd.AddCommand(createCmd)

	createCmd.Flags().StringArrayVar(&createEvents, "event", nil,
		"Events to trigger on: email.received, email.stored, email.deleted (repeatable)")
	createCmd.Flags().StringVar(&createTemplate, "template", "",
		"Built-in template: slack, discord, teams, generic")
	createCmd.Flags().StringVar(&createCustomTemplate, "custom-template", "",
		"Path to custom template file (Go template syntax)")
	createCmd.Flags().StringVar(&createContentType, "content-type", "application/json",
		"Content-Type for custom template")
	createCmd.Flags().StringVar(&createDescription, "description", "",
		"Optional description")

	addFilterFlags(createCmd, &createFilters)
}

func runCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	jsonMode := cliutil.GetOutput(cmd) == "json"
	url := args[0]

	// Parse events
	events, err := parseEvents(createEvents)
	if err != nil {
		return err
	}

	// Build options
	opts := []vaultsandbox.WebhookCreateOption{
		vaultsandbox.WithWebhookEvents(events...),
	}

	// Template
	if createTemplate != "" && createCustomTemplate != "" {
		return fmt.Errorf("cannot use both --template and --custom-template")
	}

	if createTemplate != "" {
		opts = append(opts, vaultsandbox.WithWebhookTemplate(createTemplate))
	}

	if createCustomTemplate != "" {
		body, err := loadCustomTemplate(createCustomTemplate)
		if err != nil {
			return err
		}
		opts = append(opts, vaultsandbox.WithWebhookCustomTemplate(body, createContentType))
	}

	// Description
	if createDescription != "" {
		opts = append(opts, vaultsandbox.WithWebhookDescription(createDescription))
	}

	// Filters
	filter, err := buildFilterFromFlags(&createFilters)
	if err != nil {
		return err
	}
	if filter != nil {
		opts = append(opts, vaultsandbox.WithWebhookFilter(filter))
	}

	// Show progress
	if !jsonMode {
		fmt.Println(styles.MutedStyle.Render("• Creating webhook..."))
	}

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Create webhook via Admin interface
	webhook, err := client.Admin().CreateWebhook(ctx, url, opts...)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	// Output
	if jsonMode {
		return cliutil.OutputJSON(cliutil.WebhookCreatedJSON(webhook))
	}

	printWebhookCreated(webhook)
	return nil
}

func printWebhookCreated(wh *vaultsandbox.Webhook) {
	title := styles.SuccessTitleStyle.Render("Webhook created successfully")

	details := fmt.Sprintf(`

  ID:          %s
  URL:         %s
  Events:      %s
  Scope:       %s
  Secret:      %s

`,
		wh.ID,
		wh.URL,
		cliutil.FormatEventsString(wh.Events),
		wh.Scope,
		wh.Secret,
	)

	warning := styles.WarnStyle.Render("  Save your secret! It won't be shown again.")

	box := styles.SuccessBoxStyle.Render(title + details + warning)
	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}
