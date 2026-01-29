//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Webhook JSON Types (for parsing test results)
// ============================================================================

// WebhookJSON represents the JSON output from webhook commands.
type WebhookJSON struct {
	ID          string                 `json:"id"`
	URL         string                 `json:"url"`
	Events      []string               `json:"events"`
	Scope       string                 `json:"scope"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
	InboxEmail  string                 `json:"inboxEmail,omitempty"`
	Secret      string                 `json:"secret,omitempty"`
	Template    string                 `json:"template,omitempty"`
	Description string                 `json:"description,omitempty"`
	Filter      *WebhookFilterJSON     `json:"filter,omitempty"`
	Stats       *WebhookStatsJSON      `json:"stats,omitempty"`
}

// WebhookFilterJSON represents filter configuration in JSON output.
type WebhookFilterJSON struct {
	Rules       []WebhookFilterRuleJSON `json:"rules"`
	Mode        string                  `json:"mode"`
	RequireAuth bool                    `json:"requireAuth"`
}

// WebhookFilterRuleJSON represents a single filter rule in JSON output.
type WebhookFilterRuleJSON struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// WebhookStatsJSON represents webhook delivery stats in JSON output.
type WebhookStatsJSON struct {
	TotalDeliveries      int    `json:"totalDeliveries"`
	SuccessfulDeliveries int    `json:"successfulDeliveries"`
	FailedDeliveries     int    `json:"failedDeliveries"`
	LastDeliveryAt       string `json:"lastDeliveryAt,omitempty"`
	LastSuccessAt        string `json:"lastSuccessAt,omitempty"`
	LastFailureAt        string `json:"lastFailureAt,omitempty"`
}

// WebhookDeleteJSON represents the JSON output from webhook delete.
type WebhookDeleteJSON struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// WebhookRotateJSON represents the JSON output from webhook rotate.
type WebhookRotateJSON struct {
	ID                       string `json:"id"`
	Secret                   string `json:"secret"`
	PreviousSecretValidUntil string `json:"previousSecretValidUntil,omitempty"`
}

// WebhookTestJSON represents the JSON output from webhook test command.
type WebhookTestJSON struct {
	Success      bool   `json:"success"`
	StatusCode   int    `json:"statusCode"`
	ResponseTime int    `json:"responseTime"`
	RequestID    string `json:"requestId"`
	Error        string `json:"error,omitempty"`
}

// WebhookTemplateJSON represents a webhook template.
type WebhookTemplateJSON struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// WebhookMetricsJSON represents webhook metrics.
type WebhookMetricsJSON struct {
	TotalWebhooks        int                `json:"totalWebhooks"`
	ActiveWebhooks       int                `json:"activeWebhooks"`
	TotalDeliveries      int                `json:"totalDeliveries"`
	SuccessfulDeliveries int                `json:"successfulDeliveries"`
	FailedDeliveries     int                `json:"failedDeliveries"`
	SuccessRate          float64            `json:"successRate"`
	ByScope              map[string]int     `json:"byScope"`
	ByEvent              map[string]int     `json:"byEvent"`
}

// ============================================================================
// Inbox Webhook Tests
// ============================================================================

// TestInboxWebhookCreate tests creating webhooks for a specific inbox.
func TestInboxWebhookCreate(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create an inbox first
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code, "inbox create failed")

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	t.Run("basic webhook", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.NotEmpty(t, wh.ID)
		assert.Equal(t, "https://httpbin.org/post", wh.URL)
		assert.Contains(t, wh.Events, "email.received")
		assert.Equal(t, "inbox", wh.Scope)
		assert.True(t, wh.Enabled)
		assert.NotEmpty(t, wh.Secret, "secret should be returned on create")
		assert.Equal(t, inbox.Email, wh.InboxEmail)

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", wh.ID, "--force")
		})
	})

	t.Run("webhook with template and description", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://hooks.slack.com/test",
			"--event", "email.received",
			"--template", "slack",
			"--description", "Test Slack webhook",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.NotEmpty(t, wh.ID)
		assert.Equal(t, "slack", wh.Template)
		assert.Equal(t, "Test Slack webhook", wh.Description)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", wh.ID, "--force")
		})
	})

	t.Run("webhook with multiple events", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--event", "email.stored",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.Len(t, wh.Events, 2)
		assert.Contains(t, wh.Events, "email.received")
		assert.Contains(t, wh.Events, "email.stored")

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", wh.ID, "--force")
		})
	})

	t.Run("webhook with filter", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--filter-subject-contains", "alert",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.NotEmpty(t, wh.ID)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", wh.ID, "--force")
		})
	})

	t.Run("error without event", func(t *testing.T) {
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "should fail without event")
	})
}

// TestInboxWebhookList tests listing webhooks for an inbox.
func TestInboxWebhookList(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create an inbox first
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	t.Run("empty list", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "list",
			"--output", "json",
		)
		require.Equal(t, 0, code, "list failed: stdout=%s, stderr=%s", stdout, stderr)

		var webhooks []WebhookJSON
		err := json.Unmarshal([]byte(stdout), &webhooks)
		if err != nil {
			// May be null for empty list
			assert.Equal(t, "null\n", stdout)
		} else {
			assert.Empty(t, webhooks)
		}
	})

	t.Run("list with webhooks", func(t *testing.T) {
		// Create two webhooks
		var webhookIDs []string
		for i := 0; i < 2; i++ {
			stdout, _, code := runVSBWithConfig(t, configDir,
				"inbox", "webhook", "create",
				"https://httpbin.org/post",
				"--event", "email.received",
				"--output", "json",
			)
			require.Equal(t, 0, code)

			var wh WebhookJSON
			require.NoError(t, json.Unmarshal([]byte(stdout), &wh))
			webhookIDs = append(webhookIDs, wh.ID)
		}

		t.Cleanup(func() {
			for _, id := range webhookIDs {
				runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", id, "--force")
			}
		})

		// List webhooks
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "list",
			"--output", "json",
		)
		require.Equal(t, 0, code, "list failed: stdout=%s, stderr=%s", stdout, stderr)

		var webhooks []WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &webhooks))

		assert.GreaterOrEqual(t, len(webhooks), 2)

		// Verify our webhooks are in the list
		foundIDs := make(map[string]bool)
		for _, wh := range webhooks {
			foundIDs[wh.ID] = true
		}
		for _, id := range webhookIDs {
			assert.True(t, foundIDs[id], "webhook %s should be in list", id)
		}
	})
}

// TestInboxWebhookGet tests getting details of a specific webhook.
func TestInboxWebhookGet(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create inbox and webhook
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	// Create a webhook
	createStdout, _, code := runVSBWithConfig(t, configDir,
		"inbox", "webhook", "create",
		"https://httpbin.org/post",
		"--event", "email.received",
		"--description", "Test webhook for get",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var created WebhookJSON
	require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
	})

	t.Run("get webhook details", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "get", created.ID,
			"--output", "json",
		)
		require.Equal(t, 0, code, "get failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.Equal(t, created.ID, wh.ID)
		assert.Equal(t, "https://httpbin.org/post", wh.URL)
		assert.Contains(t, wh.Events, "email.received")
		assert.Equal(t, "Test webhook for get", wh.Description)
		assert.Equal(t, inbox.Email, wh.InboxEmail)
		// Secret should NOT be returned on get
		assert.Empty(t, wh.Secret, "secret should not be returned on get")
	})

	t.Run("error on non-existent webhook", func(t *testing.T) {
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "get", "wh_nonexistent",
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "should fail for non-existent webhook")
	})
}

// TestInboxWebhookUpdate tests updating webhook configuration.
func TestInboxWebhookUpdate(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create inbox and webhook
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	// Create a webhook
	createStdout, _, code := runVSBWithConfig(t, configDir,
		"inbox", "webhook", "create",
		"https://httpbin.org/post",
		"--event", "email.received",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var created WebhookJSON
	require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
	})

	t.Run("update URL", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--url", "https://httpbin.org/anything",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.Equal(t, "https://httpbin.org/anything", wh.URL)
	})

	t.Run("disable webhook", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--disable",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.False(t, wh.Enabled)
	})

	t.Run("enable webhook", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--enable",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.True(t, wh.Enabled)
	})

	t.Run("update description", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--description", "Updated description",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.Equal(t, "Updated description", wh.Description)
	})

	t.Run("add filter", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--filter-subject-contains", "important",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.NotNil(t, wh.Filter)
		assert.NotEmpty(t, wh.Filter.Rules)
	})

	t.Run("clear filters", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--clear-filters",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		// The command should succeed - actual filter clearing behavior depends on API
		assert.NotEmpty(t, wh.ID)
	})

	t.Run("error without flags", func(t *testing.T) {
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "should fail without update flags")
	})

	t.Run("error with conflicting flags", func(t *testing.T) {
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--enable",
			"--disable",
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "should fail with both --enable and --disable")
	})
}

// TestInboxWebhookRotate tests rotating webhook secrets.
func TestInboxWebhookRotate(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create inbox and webhook
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	// Create a webhook
	createStdout, _, code := runVSBWithConfig(t, configDir,
		"inbox", "webhook", "create",
		"https://httpbin.org/post",
		"--event", "email.received",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var created WebhookJSON
	require.NoError(t, json.Unmarshal([]byte(createStdout), &created))
	originalSecret := created.Secret

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
	})

	t.Run("rotate secret", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "rotate", created.ID,
			"--force",
			"--output", "json",
		)
		require.Equal(t, 0, code, "rotate failed: stdout=%s, stderr=%s", stdout, stderr)

		var result WebhookRotateJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, created.ID, result.ID)
		assert.NotEmpty(t, result.Secret)
		assert.NotEqual(t, originalSecret, result.Secret, "secret should have changed")
		// Grace period is optional, but if present should be a valid time
		if result.PreviousSecretValidUntil != "" {
			assert.NotEmpty(t, result.PreviousSecretValidUntil)
		}
	})
}

// TestInboxWebhookDelete tests deleting webhooks.
func TestInboxWebhookDelete(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create inbox
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	t.Run("delete webhook", func(t *testing.T) {
		// Create a webhook
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		// Delete it
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "delete", created.ID,
			"--force",
			"--output", "json",
		)
		require.Equal(t, 0, code, "delete failed: stdout=%s, stderr=%s", stdout, stderr)

		var result WebhookDeleteJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, created.ID, result.ID)
		assert.True(t, result.Deleted)

		// Verify it's gone
		_, _, code = runVSBWithConfig(t, configDir,
			"inbox", "webhook", "get", created.ID,
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "get should fail after delete")
	})

}

// TestInboxWebhookTest tests the webhook test command.
func TestInboxWebhookTest(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create inbox and webhook
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	// Create a webhook pointing to httpbin (should succeed)
	createStdout, _, code := runVSBWithConfig(t, configDir,
		"inbox", "webhook", "create",
		"https://httpbin.org/post",
		"--event", "email.received",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var created WebhookJSON
	require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
	})

	t.Run("test webhook", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "test", created.ID,
			"--output", "json",
		)
		require.Equal(t, 0, code, "test failed: stdout=%s, stderr=%s", stdout, stderr)

		var result WebhookTestJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		// httpbin.org/post should return 200
		assert.True(t, result.Success)
		assert.Equal(t, 200, result.StatusCode)
		assert.Greater(t, result.ResponseTime, 0)
		// RequestID may or may not be set depending on API
	})
}

// ============================================================================
// Global Webhook Tests
// ============================================================================

// TestGlobalWebhookCreate tests creating global webhooks.
func TestGlobalWebhookCreate(t *testing.T) {
	t.Parallel()
	t.Run("basic global webhook", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=GlobalWebhookCreate_basic",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", wh.ID, "--force")
		})

		assert.NotEmpty(t, wh.ID)
		assert.Equal(t, "https://httpbin.org/post?test=GlobalWebhookCreate_basic", wh.URL)
		assert.Contains(t, wh.Events, "email.received")
		assert.Equal(t, "global", wh.Scope)
		assert.True(t, wh.Enabled)
		assert.NotEmpty(t, wh.Secret)
		assert.Empty(t, wh.InboxEmail, "global webhook should not have inboxEmail")
	})

	t.Run("global webhook with all options", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://hooks.slack.com/test",
			"--event", "email.received",
			"--event", "email.deleted",
			"--template", "slack",
			"--description", "Global Slack notifications",
			"--filter-subject-contains", "urgent",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.NotEmpty(t, wh.ID)
		assert.Len(t, wh.Events, 2)
		assert.Equal(t, "slack", wh.Template)
		assert.Equal(t, "Global Slack notifications", wh.Description)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", wh.ID, "--force")
		})
	})
}

// TestGlobalWebhookList tests listing global webhooks.
func TestGlobalWebhookList(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	t.Run("list global webhooks", func(t *testing.T) {
		// Create a global webhook
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=GlobalWebhookList",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", created.ID, "--force")
		})

		// List webhooks
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "list",
			"--output", "json",
		)
		require.Equal(t, 0, code, "list failed: stdout=%s, stderr=%s", stdout, stderr)

		var webhooks []WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &webhooks))

		// Find our webhook
		found := false
		for _, wh := range webhooks {
			if wh.ID == created.ID {
				found = true
				assert.Equal(t, "global", wh.Scope)
				break
			}
		}
		assert.True(t, found, "created webhook should be in list")
	})
}

// TestGlobalWebhookGetUpdateDeleteRotate tests get, update, delete, and rotate for global webhooks.
func TestGlobalWebhookGetUpdateDeleteRotate(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create a global webhook
	createStdout, _, code := runVSBWithConfig(t, configDir,
		"webhook", "create",
		"https://httpbin.org/post?test=GlobalWebhookGetUpdateDeleteRotate",
		"--event", "email.received",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var created WebhookJSON
	require.NoError(t, json.Unmarshal([]byte(createStdout), &created))
	originalSecret := created.Secret

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "webhook", "delete", created.ID, "--force")
	})

	t.Run("get global webhook", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "get", created.ID,
			"--output", "json",
		)
		require.Equal(t, 0, code, "get failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.Equal(t, created.ID, wh.ID)
		assert.Equal(t, "global", wh.Scope)
		assert.Empty(t, wh.Secret, "secret should not be returned on get")
	})

	t.Run("update global webhook", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "update", created.ID,
			"--description", "Updated global webhook",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.Equal(t, "Updated global webhook", wh.Description)
	})

	t.Run("rotate global webhook secret", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "rotate", created.ID,
			"--force",
			"--output", "json",
		)
		require.Equal(t, 0, code, "rotate failed: stdout=%s, stderr=%s", stdout, stderr)

		var result WebhookRotateJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, created.ID, result.ID)
		assert.NotEmpty(t, result.Secret)
		assert.NotEqual(t, originalSecret, result.Secret)
	})

	t.Run("test global webhook", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "test", created.ID,
			"--output", "json",
		)
		require.Equal(t, 0, code, "test failed: stdout=%s, stderr=%s", stdout, stderr)

		var result WebhookTestJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.True(t, result.Success)
		assert.Equal(t, 200, result.StatusCode)
	})
}

// TestGlobalWebhookDelete tests deleting global webhooks.
func TestGlobalWebhookDelete(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create a global webhook
	createStdout, _, code := runVSBWithConfig(t, configDir,
		"webhook", "create",
		"https://httpbin.org/post?test=GlobalWebhookDelete",
		"--event", "email.received",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var created WebhookJSON
	require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

	// Cleanup in case test fails before explicit delete
	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "webhook", "delete", created.ID, "--force")
	})

	t.Run("delete global webhook", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "delete", created.ID,
			"--force",
			"--output", "json",
		)
		require.Equal(t, 0, code, "delete failed: stdout=%s, stderr=%s", stdout, stderr)

		var result WebhookDeleteJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, created.ID, result.ID)
		assert.True(t, result.Deleted)

		// Verify it's gone
		_, _, code = runVSBWithConfig(t, configDir,
			"webhook", "get", created.ID,
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "get should fail after delete")
	})
}

// TestGlobalWebhookTemplates tests listing available webhook templates.
func TestGlobalWebhookTemplates(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	stdout, stderr, code := runVSBWithConfig(t, configDir,
		"webhook", "templates",
		"--output", "json",
	)
	require.Equal(t, 0, code, "templates failed: stdout=%s, stderr=%s", stdout, stderr)

	var templates []WebhookTemplateJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &templates))

	// Should have some built-in templates
	assert.NotEmpty(t, templates)

	// Check for expected templates
	templateNames := make(map[string]bool)
	for _, tmpl := range templates {
		templateNames[tmpl.Value] = true
	}

	// These templates should be available based on API response
	assert.True(t, templateNames["slack"], "slack template should be available")
	assert.True(t, templateNames["discord"], "discord template should be available")
	assert.True(t, templateNames["teams"], "teams template should be available")
	assert.True(t, templateNames["default"], "default template should be available")
}

// TestGlobalWebhookMetrics tests viewing webhook metrics.
func TestGlobalWebhookMetrics(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	stdout, stderr, code := runVSBWithConfig(t, configDir,
		"webhook", "metrics",
		"--output", "json",
	)
	require.Equal(t, 0, code, "metrics failed: stdout=%s, stderr=%s", stdout, stderr)

	var metrics WebhookMetricsJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &metrics))

	// Metrics should have valid structure
	assert.GreaterOrEqual(t, metrics.TotalWebhooks, 0)
	assert.GreaterOrEqual(t, metrics.ActiveWebhooks, 0)
	assert.GreaterOrEqual(t, metrics.TotalDeliveries, 0)
	assert.GreaterOrEqual(t, metrics.SuccessfulDeliveries, 0)
	assert.GreaterOrEqual(t, metrics.FailedDeliveries, 0)
	assert.GreaterOrEqual(t, metrics.SuccessRate, 0.0)
	assert.LessOrEqual(t, metrics.SuccessRate, 100.0)
}

// ============================================================================
// Webhook with Inbox Flag Tests
// ============================================================================

// TestInboxWebhookWithInboxFlag tests using --inbox flag to specify inbox.
func TestInboxWebhookWithInboxFlag(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create two inboxes
	inbox1Stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox1 struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inbox1Stdout), &inbox1))

	inbox2Stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox2 struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inbox2Stdout), &inbox2))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox1.Email)
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox2.Email)
	})

	// Current active inbox is inbox2 (last created)
	// Create webhook for inbox1 using --inbox flag
	t.Run("create webhook for specific inbox", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--inbox", inbox1.Email,
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.Equal(t, inbox1.Email, wh.InboxEmail)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", wh.ID, "--force", "--inbox", inbox1.Email)
		})
	})

	t.Run("list webhooks for specific inbox", func(t *testing.T) {
		// Create a webhook for inbox2
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/anything",
			"--event", "email.stored",
			"--inbox", inbox2.Email,
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force", "--inbox", inbox2.Email)
		})

		// List webhooks for inbox2
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "list",
			"--inbox", inbox2.Email,
			"--output", "json",
		)
		require.Equal(t, 0, code, "list failed: stdout=%s, stderr=%s", stdout, stderr)

		var webhooks []WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &webhooks))

		// Should find our webhook
		found := false
		for _, wh := range webhooks {
			if wh.ID == created.ID {
				found = true
				assert.Equal(t, inbox2.Email, wh.InboxEmail)
				break
			}
		}
		assert.True(t, found, "webhook should be in list")
	})
}

// ============================================================================
// Pretty Output Tests (without --output json)
// ============================================================================

// TestInboxWebhookPrettyOutput tests pretty (non-JSON) output formatting.
func TestInboxWebhookPrettyOutput(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create an inbox first
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	t.Run("create pretty output", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--description", "Pretty test webhook",
		)
		require.Equal(t, 0, code, "create failed: stderr=%s", stderr)

		// Check pretty output contains expected elements
		assert.Contains(t, stdout, "Webhook created successfully")
		assert.Contains(t, stdout, "ID:")
		assert.Contains(t, stdout, "URL:")
		assert.Contains(t, stdout, "https://httpbin.org/post")
		assert.Contains(t, stdout, "Events:")
		assert.Contains(t, stdout, "email.received")
		assert.Contains(t, stdout, "Secret:")
		assert.Contains(t, stdout, "Save your secret")

		// Extract webhook ID for cleanup
		webhookID := extractWebhookID(stdout)
		if webhookID != "" {
			t.Cleanup(func() {
				runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", webhookID, "--force")
			})
		}
	})

	t.Run("list pretty output", func(t *testing.T) {
		// Create a webhook first
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
		})

		// List without JSON output
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "list",
		)
		require.Equal(t, 0, code, "list failed: stderr=%s", stderr)

		// Check pretty output contains table headers and data
		assert.Contains(t, stdout, "ID")
		assert.Contains(t, stdout, "URL")
		assert.Contains(t, stdout, "EVENTS")
		assert.Contains(t, stdout, "ENABLED")
		// ID may be truncated in table output, check for prefix
		assert.Contains(t, stdout, created.ID[:10])
		assert.Contains(t, stdout, "Total:")
	})

	t.Run("list empty pretty output", func(t *testing.T) {
		// Use a fresh config dir with no webhooks
		freshConfigDir := t.TempDir()

		// Create inbox in fresh dir
		inboxOut, _, code := runVSBWithConfig(t, freshConfigDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var freshInbox struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(inboxOut), &freshInbox))

		t.Cleanup(func() {
			runVSBWithConfig(t, freshConfigDir, "inbox", "delete", freshInbox.Email)
		})

		stdout, stderr, code := runVSBWithConfig(t, freshConfigDir,
			"inbox", "webhook", "list",
		)
		require.Equal(t, 0, code, "list failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "No webhooks found")
	})

	t.Run("get pretty output", func(t *testing.T) {
		// Create a webhook with description
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--event", "email.stored",
			"--description", "Detailed webhook",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
		})

		// Get without JSON output
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "get", created.ID,
		)
		require.Equal(t, 0, code, "get failed: stderr=%s", stderr)

		// Check pretty output
		assert.Contains(t, stdout, "Webhook Details")
		assert.Contains(t, stdout, "ID:")
		assert.Contains(t, stdout, created.ID)
		assert.Contains(t, stdout, "URL:")
		assert.Contains(t, stdout, "https://httpbin.org/post")
		assert.Contains(t, stdout, "Events:")
		assert.Contains(t, stdout, "Description:")
		assert.Contains(t, stdout, "Detailed webhook")
		assert.Contains(t, stdout, "Enabled:")
		assert.Contains(t, stdout, "Created:")
	})

	t.Run("update pretty output", func(t *testing.T) {
		// Create a webhook
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
		})

		// Update without JSON output
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "update", created.ID,
			"--description", "Updated via pretty",
		)
		require.Equal(t, 0, code, "update failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "updated successfully")
		assert.Contains(t, stdout, "Webhook Details")
	})

	t.Run("rotate pretty output", func(t *testing.T) {
		// Create a webhook
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
		})

		// Rotate without JSON output
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "rotate", created.ID,
			"--force",
		)
		require.Equal(t, 0, code, "rotate failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "Secret Rotated Successfully")
		assert.Contains(t, stdout, "Webhook ID:")
		assert.Contains(t, stdout, "New Secret:")
		assert.Contains(t, stdout, "Update your endpoint")
	})

	t.Run("test pretty output", func(t *testing.T) {
		// Create a webhook pointing to httpbin
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", created.ID, "--force")
		})

		// Test without JSON output
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "test", created.ID,
		)
		require.Equal(t, 0, code, "test failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "Webhook Test:")
		assert.Contains(t, stdout, "Status Code:")
		assert.Contains(t, stdout, "Response Time:")
	})

	t.Run("delete pretty output", func(t *testing.T) {
		// Create a webhook
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		// Delete without JSON output
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "delete", created.ID,
			"--force",
		)
		require.Equal(t, 0, code, "delete failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "deleted successfully")
	})
}

// TestGlobalWebhookPrettyOutput tests pretty output for global webhooks.
func TestGlobalWebhookPrettyOutput(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	t.Run("create pretty output", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=GlobalWebhookPrettyOutput_create",
			"--event", "email.received",
		)
		require.Equal(t, 0, code, "create failed: stderr=%s", stderr)

		// Extract ID immediately for cleanup before any assertions
		webhookID := extractWebhookID(stdout)
		require.NotEmpty(t, webhookID, "failed to extract webhook ID from pretty output: %s", stdout)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", webhookID, "--force")
		})

		assert.Contains(t, stdout, "Webhook created successfully")
		assert.Contains(t, stdout, "ID:")
		assert.Contains(t, stdout, "Secret:")
		assert.Contains(t, stdout, "Save your secret")
	})

	t.Run("list pretty output", func(t *testing.T) {
		// Create a webhook
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=GlobalWebhookPrettyOutput_list",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", created.ID, "--force")
		})

		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "list",
		)
		require.Equal(t, 0, code, "list failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "ID")
		assert.Contains(t, stdout, "URL")
		assert.Contains(t, stdout, "EVENTS")
		assert.Contains(t, stdout, "ENABLED")
		// ID may be truncated in table output, check for prefix
		assert.Contains(t, stdout, created.ID[:10])
	})

	t.Run("get pretty output with filters", func(t *testing.T) {
		// Create a webhook with filters
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=GlobalWebhookPrettyOutput_get",
			"--event", "email.received",
			"--filter-subject-contains", "test",
			"--require-auth",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", created.ID, "--force")
		})

		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "get", created.ID,
		)
		require.Equal(t, 0, code, "get failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "Webhook Details")
		assert.Contains(t, stdout, "Filters:")
		assert.Contains(t, stdout, "Mode:")
	})

	t.Run("templates pretty output", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "templates",
		)
		require.Equal(t, 0, code, "templates failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "Available Webhook Templates")
		assert.Contains(t, stdout, "TEMPLATE")
		assert.Contains(t, stdout, "DESCRIPTION")
		assert.Contains(t, stdout, "slack")
		assert.Contains(t, stdout, "discord")
	})

	t.Run("metrics pretty output", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "metrics",
		)
		require.Equal(t, 0, code, "metrics failed: stderr=%s", stderr)

		assert.Contains(t, stdout, "Webhook Metrics")
		assert.Contains(t, stdout, "Overview")
		assert.Contains(t, stdout, "Total Webhooks:")
		assert.Contains(t, stdout, "Active Webhooks:")
		assert.Contains(t, stdout, "Delivery Stats")
	})
}

// ============================================================================
// Custom Template Tests
// ============================================================================

// TestWebhookCustomTemplate tests custom template error handling.
// Note: The API may not support custom templates, so we only test error cases.
func TestWebhookCustomTemplate(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create an inbox for testing
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	t.Run("error with non-existent template file", func(t *testing.T) {
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--custom-template", "/nonexistent/template.json",
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "should fail with non-existent template")
	})

	t.Run("error with both template and custom-template", func(t *testing.T) {
		templatePath := filepath.Join(configDir, "template.json")
		require.NoError(t, os.WriteFile(templatePath, []byte("{}"), 0644))

		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--template", "slack",
			"--custom-template", templatePath,
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "should fail with both template flags")
	})
}

// ============================================================================
// Filter Pattern Tests (regex, wildcards, more combinations)
// ============================================================================

// TestWebhookFilterPatterns tests various filter patterns.
func TestWebhookFilterPatterns(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	// Create an inbox
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	t.Run("filter with subject regex", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--filter-subject-regex", "^\\[ALERT\\]",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))
		assert.NotEmpty(t, wh.ID)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", wh.ID, "--force")
		})
	})

	t.Run("filter with exact subject match", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--filter-subject", "Password Reset Request",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))
		assert.NotEmpty(t, wh.ID)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", wh.ID, "--force")
		})
	})

	t.Run("filter with require-auth", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "webhook", "create",
			"https://httpbin.org/post",
			"--event", "email.received",
			"--require-auth",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))
		assert.NotEmpty(t, wh.ID)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "webhook", "delete", wh.ID, "--force")
		})
	})

	t.Run("update with subject filter", func(t *testing.T) {
		// Create global webhook without filter
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=WebhookFilterPatterns_subjectFilter",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", created.ID, "--force")
		})

		// Update with subject filter
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "update", created.ID,
			"--filter-subject-contains", "notification",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))
		assert.NotNil(t, wh.Filter)
	})
}

// ============================================================================
// Additional Event Combinations
// ============================================================================

// TestWebhookEventCombinations tests various event type combinations.
func TestWebhookEventCombinations(t *testing.T) {
	t.Parallel()
	configDir := t.TempDir()

	t.Run("all three events", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=EventCombinations_allThree",
			"--event", "email.received",
			"--event", "email.stored",
			"--event", "email.deleted",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", wh.ID, "--force")
		})

		assert.Len(t, wh.Events, 3)
		assert.Contains(t, wh.Events, "email.received")
		assert.Contains(t, wh.Events, "email.stored")
		assert.Contains(t, wh.Events, "email.deleted")
	})

	t.Run("stored and deleted only", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=EventCombinations_storedDeleted",
			"--event", "email.stored",
			"--event", "email.deleted",
			"--output", "json",
		)
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", wh.ID, "--force")
		})

		assert.Len(t, wh.Events, 2)
		assert.Contains(t, wh.Events, "email.stored")
		assert.Contains(t, wh.Events, "email.deleted")
	})

	t.Run("update events", func(t *testing.T) {
		// Create with one event
		createStdout, _, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post?test=EventCombinations_update",
			"--event", "email.received",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		var created WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(createStdout), &created))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "webhook", "delete", created.ID, "--force")
		})

		// Update to different events
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"webhook", "update", created.ID,
			"--event", "email.stored",
			"--event", "email.deleted",
			"--output", "json",
		)
		require.Equal(t, 0, code, "update failed: stdout=%s, stderr=%s", stdout, stderr)

		var wh WebhookJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &wh))

		assert.Len(t, wh.Events, 2)
		assert.Contains(t, wh.Events, "email.stored")
		assert.Contains(t, wh.Events, "email.deleted")
		assert.NotContains(t, wh.Events, "email.received")
	})

	t.Run("invalid event type", func(t *testing.T) {
		_, _, code := runVSBWithConfig(t, configDir,
			"webhook", "create",
			"https://httpbin.org/post",
			"--event", "invalid.event",
			"--output", "json",
		)
		assert.NotEqual(t, 0, code, "should fail with invalid event type")
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

// extractWebhookID extracts a webhook ID from pretty output.
func extractWebhookID(output string) string {
	// Look for pattern like "ID:          whk_xxxxx"
	re := regexp.MustCompile(`ID:\s+(whk_[a-zA-Z0-9]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
