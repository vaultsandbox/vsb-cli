package webhook

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
)

// Common filter flags shared between create and update commands
type filterFlags struct {
	filterFrom            string
	filterTo              string
	filterSubject         string
	filterSubjectContains string
	filterSubjectRegex    string
	filterDomain          string
	filterMode            string
	requireAuth           bool
}

// addFilterFlags adds filter-related flags to a command.
func addFilterFlags(cmd *cobra.Command, f *filterFlags) {
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

// buildFilterFromFlags builds a FilterConfig from command flags.
// Returns nil if no filters are specified.
func buildFilterFromFlags(f *filterFlags) (*vaultsandbox.FilterConfig, error) {
	var rules []vaultsandbox.FilterRule

	if f.filterFrom != "" {
		rules = append(rules, vaultsandbox.FilterRule{
			Field:    "from",
			Operator: determineOperator(f.filterFrom),
			Value:    f.filterFrom,
		})
	}

	if f.filterTo != "" {
		rules = append(rules, vaultsandbox.FilterRule{
			Field:    "to",
			Operator: determineOperator(f.filterTo),
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

// determineOperator determines the filter operator based on the value pattern.
func determineOperator(value string) vaultsandbox.FilterOperator {
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

// parseEvents parses and validates event type strings.
func parseEvents(eventStrings []string) ([]vaultsandbox.WebhookEventType, error) {
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

// loadCustomTemplate reads a custom template from a file.
func loadCustomTemplate(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}
	return string(data), nil
}
