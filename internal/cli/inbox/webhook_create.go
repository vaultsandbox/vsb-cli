package inbox

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var webhookCreateCmd = &cobra.Command{
	Use:   "create <url>",
	Short: "Create a new webhook for this inbox",
	Long: `Create a new webhook that receives notifications for this inbox only.

Examples:
  # Basic webhook for received emails
  vsb inbox webhook create https://example.com/webhook --event email.received

  # Slack notification with filter
  vsb inbox webhook create https://hooks.slack.com/xxx \
    --event email.received \
    --template slack \
    --filter-subject-contains "alert"`,
	Args: cobra.ExactArgs(1),
	RunE: runWebhookCreate,
}

var (
	whCreateEvents         []string
	whCreateTemplate       string
	whCreateCustomTemplate string
	whCreateContentType    string
	whCreateDescription    string
	whCreateFilters        webhookFilterFlags
)

// webhookFilterFlags holds filter-related flags for inbox webhook commands
type webhookFilterFlags struct {
	filterFrom            string
	filterTo              string
	filterSubject         string
	filterSubjectContains string
	filterSubjectRegex    string
	filterDomain          string
	filterMode            string
	requireAuth           bool
}

func init() {
	webhookCmd.AddCommand(webhookCreateCmd)

	webhookCreateCmd.Flags().StringArrayVar(&whCreateEvents, "event", nil,
		"Events to trigger on: email.received, email.stored, email.deleted (repeatable)")
	webhookCreateCmd.Flags().StringVar(&whCreateTemplate, "template", "",
		"Built-in template: slack, discord, teams, generic")
	webhookCreateCmd.Flags().StringVar(&whCreateCustomTemplate, "custom-template", "",
		"Path to custom template file (Go template syntax)")
	webhookCreateCmd.Flags().StringVar(&whCreateContentType, "content-type", "application/json",
		"Content-Type for custom template")
	webhookCreateCmd.Flags().StringVar(&whCreateDescription, "description", "",
		"Optional description")

	addWebhookFilterFlags(webhookCreateCmd, &whCreateFilters)
}

// addWebhookFilterFlags adds filter-related flags to a command.
func addWebhookFilterFlags(cmd *cobra.Command, f *webhookFilterFlags) {
	cmd.Flags().StringVar(&f.filterFrom, "filter-from", "",
		"Filter: sender email/pattern")
	cmd.Flags().StringVar(&f.filterTo, "filter-to", "",
		"Filter: recipient email/pattern")
	cmd.Flags().StringVar(&f.filterSubject, "filter-subject", "",
		"Filter: exact subject match")
	cmd.Flags().StringVar(&f.filterSubjectContains, "filter-subject-contains", "",
		"Filter: subject contains")
	cmd.Flags().StringVar(&f.filterSubjectRegex, "filter-subject-regex", "",
		"Filter: subject regex pattern")
	cmd.Flags().StringVar(&f.filterDomain, "filter-domain", "",
		"Filter: sender domain")
	cmd.Flags().StringVar(&f.filterMode, "filter-mode", "all",
		"Filter mode: all (AND) or any (OR)")
	cmd.Flags().BoolVar(&f.requireAuth, "require-auth", false,
		"Require email passes SPF/DKIM/DMARC")
}

func runWebhookCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	jsonMode := cliutil.GetOutput(cmd) == "json"
	url := args[0]

	// Parse events
	events, err := parseWebhookEvents(whCreateEvents)
	if err != nil {
		return err
	}

	// Build options
	opts := []vaultsandbox.WebhookCreateOption{
		vaultsandbox.WithWebhookEvents(events...),
	}

	// Template
	if whCreateTemplate != "" && whCreateCustomTemplate != "" {
		return fmt.Errorf("cannot use both --template and --custom-template")
	}

	if whCreateTemplate != "" {
		opts = append(opts, vaultsandbox.WithWebhookTemplate(whCreateTemplate))
	}

	if whCreateCustomTemplate != "" {
		body, err := loadWebhookCustomTemplate(whCreateCustomTemplate)
		if err != nil {
			return err
		}
		opts = append(opts, vaultsandbox.WithWebhookCustomTemplate(body, whCreateContentType))
	}

	// Description
	if whCreateDescription != "" {
		opts = append(opts, vaultsandbox.WithWebhookDescription(whCreateDescription))
	}

	// Filters
	filter, err := buildWebhookFilterFromFlags(&whCreateFilters)
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

	// Load inbox
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, InboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// Create webhook via inbox
	webhook, err := inbox.CreateWebhook(ctx, url, opts...)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}

	// Output
	if jsonMode {
		return cliutil.OutputJSON(cliutil.WebhookCreatedJSON(webhook))
	}

	printInboxWebhookCreated(webhook)
	return nil
}

func printInboxWebhookCreated(wh *vaultsandbox.Webhook) {
	title := styles.SuccessTitleStyle.Render("Webhook created successfully")

	details := fmt.Sprintf(`

  ID:          %s
  URL:         %s
  Events:      %s
  Scope:       %s
  Inbox:       %s
  Secret:      %s

`,
		wh.ID,
		wh.URL,
		cliutil.FormatEventsString(wh.Events),
		wh.Scope,
		wh.InboxEmail,
		wh.Secret,
	)

	warning := styles.WarnStyle.Render("  Save your secret! It won't be shown again.")

	box := styles.SuccessBoxStyle.Render(title + details + warning)
	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}

// parseWebhookEvents parses and validates event type strings.
func parseWebhookEvents(eventStrings []string) ([]vaultsandbox.WebhookEventType, error) {
	if len(eventStrings) == 0 {
		return nil, fmt.Errorf("at least one event type is required (use --event)")
	}

	events := make([]vaultsandbox.WebhookEventType, len(eventStrings))
	validEvents := map[string]vaultsandbox.WebhookEventType{
		"email.received": vaultsandbox.WebhookEventEmailReceived,
		"email.stored":   vaultsandbox.WebhookEventEmailStored,
		"email.deleted":  vaultsandbox.WebhookEventEmailDeleted,
	}

	for i, e := range eventStrings {
		eventType, ok := validEvents[strings.ToLower(e)]
		if !ok {
			return nil, fmt.Errorf("invalid event type: %s (valid: email.received, email.stored, email.deleted)", e)
		}
		events[i] = eventType
	}

	return events, nil
}

// loadWebhookCustomTemplate reads a custom template from a file.
func loadWebhookCustomTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}
	return string(data), nil
}

// buildWebhookFilterFromFlags builds a FilterConfig from command flags.
func buildWebhookFilterFromFlags(f *webhookFilterFlags) (*vaultsandbox.FilterConfig, error) {
	var rules []vaultsandbox.FilterRule

	if f.filterFrom != "" {
		rules = append(rules, vaultsandbox.FilterRule{
			Field:    "from",
			Operator: determineWebhookOperator(f.filterFrom),
			Value:    f.filterFrom,
		})
	}

	if f.filterTo != "" {
		rules = append(rules, vaultsandbox.FilterRule{
			Field:    "to",
			Operator: determineWebhookOperator(f.filterTo),
			Value:    f.filterTo,
		})
	}

	if f.filterSubject != "" {
		rules = append(rules, vaultsandbox.FilterRule{
			Field:    "subject",
			Operator: vaultsandbox.FilterOperatorEquals,
			Value:    f.filterSubject,
		})
	}

	if f.filterSubjectContains != "" {
		rules = append(rules, vaultsandbox.FilterRule{
			Field:    "subject",
			Operator: vaultsandbox.FilterOperatorContains,
			Value:    f.filterSubjectContains,
		})
	}

	if f.filterSubjectRegex != "" {
		rules = append(rules, vaultsandbox.FilterRule{
			Field:    "subject",
			Operator: vaultsandbox.FilterOperatorRegex,
			Value:    f.filterSubjectRegex,
		})
	}

	if f.filterDomain != "" {
		rules = append(rules, vaultsandbox.FilterRule{
			Field:    "from",
			Operator: vaultsandbox.FilterOperatorDomain,
			Value:    f.filterDomain,
		})
	}

	// Only return a filter if we have rules or requireAuth is set
	if len(rules) == 0 && !f.requireAuth {
		return nil, nil
	}

	mode := vaultsandbox.FilterModeAll
	if strings.ToLower(f.filterMode) == "any" {
		mode = vaultsandbox.FilterModeAny
	}

	return &vaultsandbox.FilterConfig{
		Rules:       rules,
		Mode:        mode,
		RequireAuth: f.requireAuth,
	}, nil
}

// determineWebhookOperator determines the filter operator based on the value pattern.
func determineWebhookOperator(value string) vaultsandbox.FilterOperator {
	// Check for regex patterns (starts with ^ or ends with $)
	if strings.HasPrefix(value, "^") || strings.HasSuffix(value, "$") {
		return vaultsandbox.FilterOperatorRegex
	}
	// Check for wildcard patterns
	if strings.Contains(value, "*") {
		return vaultsandbox.FilterOperatorContains
	}
	// Default to contains for partial matching
	return vaultsandbox.FilterOperatorContains
}
