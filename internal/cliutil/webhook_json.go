package cliutil

import (
	"strings"
	"time"

	vaultsandbox "github.com/vaultsandbox/client-go"
)

// WebhookJSONOptions controls which fields to include in webhook JSON output.
type WebhookJSONOptions struct {
	IncludeSecret  bool
	IncludeStats   bool
	IncludeFilter  bool
	IncludeDetails bool // template, description, etc.
}

// WebhookJSON returns a map for JSON output with configurable fields.
func WebhookJSON(wh *vaultsandbox.Webhook, opts WebhookJSONOptions) map[string]interface{} {
	m := map[string]interface{}{
		"id":        wh.ID,
		"url":       wh.URL,
		"events":    formatEvents(wh.Events),
		"scope":     string(wh.Scope),
		"enabled":   wh.Enabled,
		"createdAt": wh.CreatedAt.Format(time.RFC3339),
		"updatedAt": wh.UpdatedAt.Format(time.RFC3339),
	}

	if wh.InboxEmail != "" {
		m["inboxEmail"] = wh.InboxEmail
	}

	if opts.IncludeSecret && wh.Secret != "" {
		m["secret"] = wh.Secret
	}

	if opts.IncludeDetails {
		if wh.Template != "" {
			m["template"] = wh.Template
		}
		if wh.CustomTemplate != nil {
			m["customTemplate"] = map[string]string{
				"contentType": wh.CustomTemplate.ContentType,
			}
		}
		if wh.Description != "" {
			m["description"] = wh.Description
		}
	}

	if opts.IncludeFilter && wh.Filter != nil {
		m["filter"] = filterToJSON(wh.Filter)
	}

	if opts.IncludeStats && wh.Stats != nil {
		m["stats"] = statsToJSON(wh.Stats)
	}

	return m
}

// formatEvents converts event types to strings.
func formatEvents(events []vaultsandbox.WebhookEventType) []string {
	result := make([]string, len(events))
	for i, e := range events {
		result[i] = string(e)
	}
	return result
}

// filterToJSON converts a filter config to a JSON-friendly map.
func filterToJSON(f *vaultsandbox.FilterConfig) map[string]interface{} {
	rules := make([]map[string]interface{}, len(f.Rules))
	for i, r := range f.Rules {
		rules[i] = map[string]interface{}{
			"field":    r.Field,
			"operator": string(r.Operator),
			"value":    r.Value,
		}
		if r.CaseSensitive {
			rules[i]["caseSensitive"] = true
		}
	}
	return map[string]interface{}{
		"rules":       rules,
		"mode":        string(f.Mode),
		"requireAuth": f.RequireAuth,
	}
}

// statsToJSON converts webhook stats to a JSON-friendly map.
func statsToJSON(s *vaultsandbox.WebhookStats) map[string]interface{} {
	m := map[string]interface{}{
		"totalDeliveries":      s.TotalDeliveries,
		"successfulDeliveries": s.SuccessfulDeliveries,
		"failedDeliveries":     s.FailedDeliveries,
	}
	if s.LastDeliveryAt != nil {
		m["lastDeliveryAt"] = s.LastDeliveryAt.Format(time.RFC3339)
	}
	if s.LastSuccessAt != nil {
		m["lastSuccessAt"] = s.LastSuccessAt.Format(time.RFC3339)
	}
	if s.LastFailureAt != nil {
		m["lastFailureAt"] = s.LastFailureAt.Format(time.RFC3339)
	}
	return m
}

// WebhookSummaryJSON returns a map for JSON output of webhook list items.
func WebhookSummaryJSON(wh *vaultsandbox.Webhook) map[string]interface{} {
	return WebhookJSON(wh, WebhookJSONOptions{IncludeStats: true})
}

// WebhookFullJSON returns a map for JSON output of full webhook details.
func WebhookFullJSON(wh *vaultsandbox.Webhook) map[string]interface{} {
	return WebhookJSON(wh, WebhookJSONOptions{
		IncludeStats:   true,
		IncludeFilter:  true,
		IncludeDetails: true,
	})
}

// WebhookCreatedJSON returns a map for JSON output when a webhook is created.
func WebhookCreatedJSON(wh *vaultsandbox.Webhook) map[string]interface{} {
	return WebhookJSON(wh, WebhookJSONOptions{
		IncludeSecret:  true,
		IncludeDetails: true,
	})
}

// TestWebhookJSON returns a map for JSON output of webhook test results.
func TestWebhookJSON(resp *vaultsandbox.TestWebhookResponse) map[string]interface{} {
	m := map[string]interface{}{
		"success":      resp.Success,
		"statusCode":   resp.StatusCode,
		"responseTime": resp.ResponseTime,
		"requestId":    resp.RequestID,
	}
	if resp.Error != "" {
		m["error"] = resp.Error
	}
	return m
}

// RotateSecretJSON returns a map for JSON output of secret rotation.
func RotateSecretJSON(resp *vaultsandbox.RotateSecretResponse) map[string]interface{} {
	m := map[string]interface{}{
		"id":     resp.ID,
		"secret": resp.Secret,
	}
	if resp.PreviousSecretValidUntil != nil {
		m["previousSecretValidUntil"] = resp.PreviousSecretValidUntil.Format(time.RFC3339)
	}
	return m
}

// WebhookTemplateJSON returns a map for JSON output of a webhook template.
func WebhookTemplateJSON(t *vaultsandbox.WebhookTemplate) map[string]interface{} {
	return map[string]interface{}{
		"value": t.Value,
		"label": t.Label,
	}
}

// WebhookMetricsJSON returns a map for JSON output of webhook metrics.
func WebhookMetricsJSON(m *vaultsandbox.WebhookMetrics) map[string]interface{} {
	return map[string]interface{}{
		"totalWebhooks":        m.TotalWebhooks,
		"activeWebhooks":       m.ActiveWebhooks,
		"totalDeliveries":      m.TotalDeliveries,
		"successfulDeliveries": m.SuccessfulDeliveries,
		"failedDeliveries":     m.FailedDeliveries,
		"successRate":          m.SuccessRate,
		"byScope":              m.ByScope,
		"byEvent":              m.ByEvent,
	}
}

// FormatEventsString formats event types as a comma-separated string.
func FormatEventsString(events []vaultsandbox.WebhookEventType) string {
	strs := formatEvents(events)
	return strings.Join(strs, ", ")
}
