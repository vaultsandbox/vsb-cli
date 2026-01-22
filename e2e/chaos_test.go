//go:build e2e

package e2e

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Chaos JSON Types (for parsing test results)
// ============================================================================

// ChaosConfigJSON represents the JSON output from chaos commands.
type ChaosConfigJSON struct {
	Enabled        bool                       `json:"enabled"`
	ExpiresAt      string                     `json:"expiresAt,omitempty"`
	Latency        *LatencyConfigJSON         `json:"latency,omitempty"`
	ConnectionDrop *ConnectionDropConfigJSON  `json:"connectionDrop,omitempty"`
	RandomError    *RandomErrorConfigJSON     `json:"randomError,omitempty"`
	Greylist       *GreylistConfigJSON        `json:"greylist,omitempty"`
	Blackhole      *BlackholeConfigJSON       `json:"blackhole,omitempty"`
}

// LatencyConfigJSON represents latency configuration in JSON output.
type LatencyConfigJSON struct {
	Enabled     bool    `json:"enabled"`
	MinDelayMs  int     `json:"minDelayMs"`
	MaxDelayMs  int     `json:"maxDelayMs"`
	Jitter      bool    `json:"jitter"`
	Probability float64 `json:"probability"`
}

// ConnectionDropConfigJSON represents connection drop configuration in JSON output.
type ConnectionDropConfigJSON struct {
	Enabled     bool    `json:"enabled"`
	Probability float64 `json:"probability"`
	Graceful    bool    `json:"graceful"`
}

// RandomErrorConfigJSON represents random error configuration in JSON output.
type RandomErrorConfigJSON struct {
	Enabled    bool     `json:"enabled"`
	ErrorRate  float64  `json:"errorRate"`
	ErrorTypes []string `json:"errorTypes"`
}

// GreylistConfigJSON represents greylist configuration in JSON output.
type GreylistConfigJSON struct {
	Enabled       bool   `json:"enabled"`
	RetryWindowMs int    `json:"retryWindowMs"`
	MaxAttempts   int    `json:"maxAttempts"`
	TrackBy       string `json:"trackBy"`
}

// BlackholeConfigJSON represents blackhole configuration in JSON output.
type BlackholeConfigJSON struct {
	Enabled         bool `json:"enabled"`
	TriggerWebhooks bool `json:"triggerWebhooks"`
}

// ChaosDisableJSON represents the JSON output from chaos disable command.
type ChaosDisableJSON struct {
	Email    string `json:"email"`
	Disabled bool   `json:"disabled"`
}

// ============================================================================
// Chaos Set Tests
// ============================================================================

// TestChaosSet tests the chaos set command with various configurations.
func TestChaosSet(t *testing.T) {
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

	t.Run("set latency chaos", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency",
			"--min-delay", "1000",
			"--max-delay", "5000",
			"--probability", "0.5",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.Latency)
		assert.True(t, cfg.Latency.Enabled)
		assert.Equal(t, 1000, cfg.Latency.MinDelayMs)
		assert.Equal(t, 5000, cfg.Latency.MaxDelayMs)
		assert.Equal(t, 0.5, cfg.Latency.Probability)
		assert.True(t, cfg.Latency.Jitter, "jitter should be enabled by default")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set latency with no-jitter", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency",
			"--min-delay", "500",
			"--max-delay", "2000",
			"--no-jitter",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		require.NotNil(t, cfg.Latency)
		assert.False(t, cfg.Latency.Jitter, "jitter should be disabled with --no-jitter")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set connection drop chaos", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--connection-drop",
			"--drop-probability", "0.3",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.ConnectionDrop)
		assert.True(t, cfg.ConnectionDrop.Enabled)
		assert.Equal(t, 0.3, cfg.ConnectionDrop.Probability)
		assert.True(t, cfg.ConnectionDrop.Graceful, "graceful should be true by default (not abrupt)")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set connection drop with abrupt", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--connection-drop",
			"--abrupt",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		require.NotNil(t, cfg.ConnectionDrop)
		assert.False(t, cfg.ConnectionDrop.Graceful, "graceful should be false with --abrupt")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set random error chaos", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--random-error",
			"--error-rate", "0.2",
			"--error-types", "temporary",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.RandomError)
		assert.True(t, cfg.RandomError.Enabled)
		assert.Equal(t, 0.2, cfg.RandomError.ErrorRate)
		assert.Contains(t, cfg.RandomError.ErrorTypes, "temporary")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set random error with multiple error types", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--random-error",
			"--error-rate", "0.15",
			"--error-types", "temporary,permanent",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		require.NotNil(t, cfg.RandomError)
		assert.Len(t, cfg.RandomError.ErrorTypes, 2)
		assert.Contains(t, cfg.RandomError.ErrorTypes, "temporary")
		assert.Contains(t, cfg.RandomError.ErrorTypes, "permanent")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set greylist chaos", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--greylist",
			"--max-attempts", "3",
			"--retry-window", "600000",
			"--track-by", "ip_sender",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.Greylist)
		assert.True(t, cfg.Greylist.Enabled)
		assert.Equal(t, 3, cfg.Greylist.MaxAttempts)
		assert.Equal(t, 600000, cfg.Greylist.RetryWindowMs)
		assert.Equal(t, "ip_sender", cfg.Greylist.TrackBy)

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set greylist with different track-by options", func(t *testing.T) {
		trackByOptions := []string{"ip", "sender", "ip_sender"}

		for _, trackBy := range trackByOptions {
			stdout, stderr, code := runVSBWithConfig(t, configDir,
				"inbox", "chaos", "set",
				"--greylist",
				"--track-by", trackBy,
				"--output", "json",
			)
			require.Equal(t, 0, code, "set failed for track-by=%s: stdout=%s, stderr=%s", trackBy, stdout, stderr)

			var cfg ChaosConfigJSON
			require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

			require.NotNil(t, cfg.Greylist)
			assert.Equal(t, trackBy, cfg.Greylist.TrackBy)

			// Disable before next iteration
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		}
	})

	t.Run("set blackhole chaos", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--blackhole",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.Blackhole)
		assert.True(t, cfg.Blackhole.Enabled)
		assert.False(t, cfg.Blackhole.TriggerWebhooks, "trigger-webhooks should be false by default")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set blackhole with trigger-webhooks", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--blackhole",
			"--trigger-webhooks",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		require.NotNil(t, cfg.Blackhole)
		assert.True(t, cfg.Blackhole.TriggerWebhooks)

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set multiple chaos types", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency", "--min-delay", "500", "--max-delay", "2000", "--probability", "0.3",
			"--connection-drop", "--drop-probability", "0.1",
			"--random-error", "--error-rate", "0.05",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.Latency)
		require.NotNil(t, cfg.ConnectionDrop)
		require.NotNil(t, cfg.RandomError)

		assert.Equal(t, 500, cfg.Latency.MinDelayMs)
		assert.Equal(t, 0.1, cfg.ConnectionDrop.Probability)
		assert.Equal(t, 0.05, cfg.RandomError.ErrorRate)

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("set chaos with expiration duration", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency",
			"--expires", "1h",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		assert.NotEmpty(t, cfg.ExpiresAt, "expiresAt should be set")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})
}

// ============================================================================
// Chaos Get Tests
// ============================================================================

// TestChaosGet tests the chaos get command.
func TestChaosGet(t *testing.T) {
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

	t.Run("get chaos when disabled", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "get",
			"--output", "json",
		)
		require.Equal(t, 0, code, "get failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.False(t, cfg.Enabled)
	})

	t.Run("get chaos after setting latency", func(t *testing.T) {
		// Set latency
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency",
			"--min-delay", "1500",
			"--max-delay", "3000",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		// Get and verify
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "get",
			"--output", "json",
		)
		require.Equal(t, 0, code, "get failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.Latency)
		assert.Equal(t, 1500, cfg.Latency.MinDelayMs)
		assert.Equal(t, 3000, cfg.Latency.MaxDelayMs)

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})

	t.Run("get chaos pretty output", func(t *testing.T) {
		// Set some chaos
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency",
			"--min-delay", "1000",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		// Get in pretty mode
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "get",
		)
		require.Equal(t, 0, code, "get failed: stdout=%s, stderr=%s", stdout, stderr)

		// Verify pretty output contains expected strings
		assert.Contains(t, stdout, "Chaos Configuration")
		assert.Contains(t, stdout, "Enabled")
		assert.Contains(t, stdout, "Latency")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--force")
		})
	})
}

// ============================================================================
// Chaos Disable Tests
// ============================================================================

// TestChaosDisable tests the chaos disable command.
func TestChaosDisable(t *testing.T) {
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

	t.Run("disable chaos with force flag", func(t *testing.T) {
		// Set chaos first
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		// Disable with force
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "disable",
			"--force",
			"--output", "json",
		)
		require.Equal(t, 0, code, "disable failed: stdout=%s, stderr=%s", stdout, stderr)

		var result ChaosDisableJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, inbox.Email, result.Email)
		assert.True(t, result.Disabled)

		// Verify chaos is disabled
		getStdout, _, _ := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "get",
			"--output", "json",
		)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(getStdout), &cfg))
		assert.False(t, cfg.Enabled)
	})

	t.Run("disable chaos pretty output", func(t *testing.T) {
		// Set chaos first
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--blackhole",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		// Disable in pretty mode
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "disable",
			"--force",
		)
		require.Equal(t, 0, code, "disable failed: stdout=%s, stderr=%s", stdout, stderr)

		assert.Contains(t, stdout, "disabled")
	})
}

// ============================================================================
// Chaos Error Handling Tests
// ============================================================================

// TestChaosSetErrors tests error scenarios for chaos set command.
func TestChaosSetErrors(t *testing.T) {
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

	t.Run("error when no chaos type specified", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
		)
		assert.NotEqual(t, 0, code, "should fail when no chaos type specified")
		assert.Contains(t, stderr, "at least one chaos type must be enabled")
	})

	t.Run("error for invalid track-by value", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--greylist",
			"--track-by", "invalid_track",
		)
		assert.NotEqual(t, 0, code, "should fail for invalid track-by")
		assert.Contains(t, stderr, "track-by")
	})

	t.Run("error for invalid probability", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency",
			"--probability", "1.5",
		)
		assert.NotEqual(t, 0, code, "should fail for probability > 1.0")
		assert.Contains(t, stderr, "probability")
	})

	t.Run("error for min-delay greater than max-delay", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--latency",
			"--min-delay", "5000",
			"--max-delay", "1000",
		)
		assert.NotEqual(t, 0, code, "should fail when min-delay > max-delay")
		assert.Contains(t, stderr, "min-delay")
	})

	t.Run("error for invalid error-types", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--random-error",
			"--error-types", "invalid_type",
		)
		assert.NotEqual(t, 0, code, "should fail for invalid error type")
		assert.Contains(t, stderr, "error-types")
	})
}

// ============================================================================
// Chaos with Specific Inbox Tests
// ============================================================================

// TestChaosWithInboxFlag tests using the --inbox flag with chaos commands.
func TestChaosWithInboxFlag(t *testing.T) {
	configDir := t.TempDir()

	// Create two inboxes
	inbox1Stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code, "inbox1 create failed")

	var inbox1 struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inbox1Stdout), &inbox1))

	inbox2Stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code, "inbox2 create failed")

	var inbox2 struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inbox2Stdout), &inbox2))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox1.Email)
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox2.Email)
	})

	t.Run("set chaos on specific inbox", func(t *testing.T) {
		// Set chaos on inbox2 (not the active one)
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--inbox", inbox2.Email,
			"--latency",
			"--min-delay", "2000",
			"--output", "json",
		)
		require.Equal(t, 0, code, "set failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.Latency)
		assert.Equal(t, 2000, cfg.Latency.MinDelayMs)

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--inbox", inbox2.Email, "--force")
		})
	})

	t.Run("get chaos from specific inbox", func(t *testing.T) {
		// Set chaos on inbox1
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--inbox", inbox1.Email,
			"--blackhole",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		// Get chaos from inbox1 using --inbox flag
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "get",
			"--inbox", inbox1.Email,
			"--output", "json",
		)
		require.Equal(t, 0, code, "get failed: stdout=%s, stderr=%s", stdout, stderr)

		var cfg ChaosConfigJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &cfg))

		assert.True(t, cfg.Enabled)
		require.NotNil(t, cfg.Blackhole)

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "chaos", "disable", "--inbox", inbox1.Email, "--force")
		})
	})

	t.Run("disable chaos on specific inbox", func(t *testing.T) {
		// Set chaos on inbox2
		_, _, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "set",
			"--inbox", inbox2.Email,
			"--greylist",
			"--output", "json",
		)
		require.Equal(t, 0, code)

		// Disable chaos on inbox2
		stdout, stderr, code := runVSBWithConfig(t, configDir,
			"inbox", "chaos", "disable",
			"--inbox", inbox2.Email,
			"--force",
			"--output", "json",
		)
		require.Equal(t, 0, code, "disable failed: stdout=%s, stderr=%s", stdout, stderr)

		var result ChaosDisableJSON
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, inbox2.Email, result.Email)
		assert.True(t, result.Disabled)
	})
}

// ============================================================================
// Chaos Workflow Tests
// ============================================================================

// TestChaosWorkflow tests a complete chaos workflow: set -> get -> disable.
func TestChaosWorkflow(t *testing.T) {
	configDir := t.TempDir()

	// Create an inbox
	inboxStdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code, "inbox create failed")

	var inbox struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(inboxStdout), &inbox))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inbox.Email)
	})

	// Step 1: Initial state - chaos should be disabled
	t.Log("Step 1: Verify initial state (chaos disabled)")
	stdout, _, code := runVSBWithConfig(t, configDir,
		"inbox", "chaos", "get",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var initialCfg ChaosConfigJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &initialCfg))
	assert.False(t, initialCfg.Enabled, "chaos should be disabled initially")

	// Step 2: Enable latency chaos
	t.Log("Step 2: Enable latency chaos")
	stdout, _, code = runVSBWithConfig(t, configDir,
		"inbox", "chaos", "set",
		"--latency",
		"--min-delay", "1000",
		"--max-delay", "3000",
		"--probability", "0.8",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var latencyCfg ChaosConfigJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &latencyCfg))
	assert.True(t, latencyCfg.Enabled)
	require.NotNil(t, latencyCfg.Latency)

	// Step 3: Add connection drop chaos (should merge with existing)
	t.Log("Step 3: Add connection drop chaos")
	stdout, _, code = runVSBWithConfig(t, configDir,
		"inbox", "chaos", "set",
		"--connection-drop",
		"--drop-probability", "0.2",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var combinedCfg ChaosConfigJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &combinedCfg))
	assert.True(t, combinedCfg.Enabled)
	require.NotNil(t, combinedCfg.ConnectionDrop)
	assert.Equal(t, 0.2, combinedCfg.ConnectionDrop.Probability)

	// Step 4: Get current configuration
	t.Log("Step 4: Verify current configuration via get")
	stdout, _, code = runVSBWithConfig(t, configDir,
		"inbox", "chaos", "get",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var getCfg ChaosConfigJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &getCfg))
	assert.True(t, getCfg.Enabled)

	// Step 5: Disable all chaos
	t.Log("Step 5: Disable all chaos")
	stdout, _, code = runVSBWithConfig(t, configDir,
		"inbox", "chaos", "disable",
		"--force",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var disableResult ChaosDisableJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &disableResult))
	assert.True(t, disableResult.Disabled)

	// Step 6: Verify chaos is disabled
	t.Log("Step 6: Verify chaos is disabled")
	stdout, _, code = runVSBWithConfig(t, configDir,
		"inbox", "chaos", "get",
		"--output", "json",
	)
	require.Equal(t, 0, code)

	var finalCfg ChaosConfigJSON
	require.NoError(t, json.Unmarshal([]byte(stdout), &finalCfg))
	assert.False(t, finalCfg.Enabled, "chaos should be disabled after disable command")
}
