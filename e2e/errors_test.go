//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Error Scenario Tests
// ============================================================================
// These tests verify proper error handling across the CLI, ensuring
// appropriate exit codes, error messages, and graceful failures.

// TestInboxErrors tests error handling for inbox commands.
func TestInboxErrors(t *testing.T) {
	t.Run("info without inbox", func(t *testing.T) {
		configDir := t.TempDir()

		// No inbox exists - should fail gracefully
		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "info")
		assert.NotEqual(t, 0, code, "should fail when no inbox exists")
		assert.True(t,
			strings.Contains(stderr, "no inbox") ||
				strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no active"),
			"error should indicate no inbox found, got: %s", stderr)
	})

	t.Run("use non-existent inbox", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "use", "nonexistent@example.com")
		assert.NotEqual(t, 0, code, "should fail for non-existent inbox")
		assert.True(t,
			strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no match") ||
				strings.Contains(stderr, "does not exist"),
			"error should indicate inbox not found, got: %s", stderr)
	})

	t.Run("delete non-existent inbox", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "delete", "fake@example.com")
		assert.NotEqual(t, 0, code, "should fail for non-existent inbox")
		assert.True(t,
			strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no match") ||
				strings.Contains(stderr, "does not exist"),
			"error should indicate inbox not found, got: %s", stderr)
	})

	t.Run("info with ambiguous partial match", func(t *testing.T) {
		configDir := t.TempDir()

		// Create two inboxes that might share a prefix
		var emails []string
		for i := 0; i < 2; i++ {
			stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
			require.Equal(t, 0, code)

			var result struct {
				Email string `json:"email"`
			}
			require.NoError(t, json.Unmarshal([]byte(stdout), &result))
			emails = append(emails, result.Email)
		}

		t.Cleanup(func() {
			for _, email := range emails {
				runVSBWithConfig(t, configDir, "inbox", "delete", email)
			}
		})

		// Try with a very short prefix that might match multiple
		// This depends on email format, so we test with just one character
		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "info", "@")

		// Either should fail with ambiguity or successfully match
		// We're testing that it handles the edge case gracefully
		if code != 0 {
			// Should have meaningful error
			assert.NotEmpty(t, stderr)
		}
	})

	t.Run("create with invalid TTL format", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "create", "--ttl", "invalid")
		assert.NotEqual(t, 0, code, "should fail for invalid TTL format")
		assert.True(t,
			strings.Contains(strings.ToLower(stderr), "invalid") ||
				strings.Contains(strings.ToLower(stderr), "duration") ||
				strings.Contains(strings.ToLower(stderr), "parse") ||
				strings.Contains(strings.ToLower(stderr), "ttl"),
			"error should indicate invalid TTL, got: %s", stderr)
	})
}

// TestEmailErrors tests error handling for email commands.
func TestEmailErrors(t *testing.T) {
	configDir := t.TempDir()

	// Create an inbox for testing
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	inboxEmail := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inboxEmail)
	})

	t.Run("view non-existent email ID", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "view", "nonexistent-id-12345")
		assert.NotEqual(t, 0, code, "should fail for non-existent email ID")
		stderrLower := strings.ToLower(stderr)
		assert.True(t,
			strings.Contains(stderrLower, "not found") ||
				strings.Contains(stderrLower, "no email") ||
				strings.Contains(stderrLower, "does not exist"),
			"error should indicate email not found, got: %s", stderr)
	})

	t.Run("delete non-existent email ID", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "delete", "fake-email-id-999")
		assert.NotEqual(t, 0, code, "should fail for non-existent email ID")
		assert.True(t,
			strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no email") ||
				strings.Contains(stderr, "does not exist") ||
				strings.Contains(stderr, "error"),
			"error should indicate email not found, got: %s", stderr)
	})

	t.Run("view in empty inbox", func(t *testing.T) {
		// Fresh inbox with no emails
		freshConfigDir := t.TempDir()
		stdout, _, code := runVSBWithConfig(t, freshConfigDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		t.Cleanup(func() {
			runVSBWithConfig(t, freshConfigDir, "inbox", "delete", result.Email)
		})

		// Try to view when no emails exist
		_, stderr, code := runVSBWithConfig(t, freshConfigDir, "email", "view")
		assert.NotEqual(t, 0, code, "should fail when no emails exist")
		assert.True(t,
			strings.Contains(stderr, "no email") ||
				strings.Contains(stderr, "empty") ||
				strings.Contains(stderr, "not found"),
			"error should indicate no emails, got: %s", stderr)
	})

	t.Run("audit non-existent email", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "audit", "fake-id-audit")
		assert.NotEqual(t, 0, code, "should fail for non-existent email")
		assert.True(t,
			strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no email") ||
				strings.Contains(stderr, "error"),
			"error should indicate email not found, got: %s", stderr)
	})

	t.Run("url extraction on non-existent email", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "url", "fake-id-url")
		assert.NotEqual(t, 0, code, "should fail for non-existent email")
		assert.True(t,
			strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no email") ||
				strings.Contains(stderr, "error"),
			"error should indicate email not found, got: %s", stderr)
	})

	t.Run("attachment on non-existent email", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "attachment", "fake-id-attach")
		assert.NotEqual(t, 0, code, "should fail for non-existent email")
		assert.True(t,
			strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no email") ||
				strings.Contains(stderr, "error"),
			"error should indicate email not found, got: %s", stderr)
	})

	t.Run("list emails without active inbox", func(t *testing.T) {
		emptyConfigDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, emptyConfigDir, "email", "list")
		assert.NotEqual(t, 0, code, "should fail when no inbox exists")
		assert.True(t,
			strings.Contains(stderr, "no inbox") ||
				strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no active"),
			"error should indicate no inbox, got: %s", stderr)
	})

	t.Run("list emails with invalid inbox", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "list", "--inbox", "invalid@nowhere.com")
		assert.NotEqual(t, 0, code, "should fail for invalid inbox")
		assert.True(t,
			strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no match") ||
				strings.Contains(stderr, "does not exist"),
			"error should indicate inbox not found, got: %s", stderr)
	})
}

// TestWaitErrors tests error handling for wait command.
func TestWaitErrors(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	inboxEmail := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", inboxEmail)
	})

	t.Run("wait with short timeout", func(t *testing.T) {
		start := time.Now()

		_, stderr, code := runVSBWithConfig(t, configDir, "email", "wait",
			"--timeout", "2s",
			"--subject", "NonExistent Subject 12345")

		elapsed := time.Since(start)

		assert.NotEqual(t, 0, code, "should fail on timeout")
		assert.True(t,
			strings.Contains(stderr, "timeout") ||
				strings.Contains(stderr, "timed out") ||
				strings.Contains(stderr, "no email"),
			"error should mention timeout, got: %s", stderr)

		// Should have waited approximately the timeout
		assert.GreaterOrEqual(t, elapsed, 1*time.Second)
		assert.Less(t, elapsed, 10*time.Second)
	})

	t.Run("wait without inbox", func(t *testing.T) {
		noInboxConfigDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, noInboxConfigDir, "email", "wait", "--timeout", "1s")
		assert.NotEqual(t, 0, code, "should fail when no inbox exists")
		assert.True(t,
			strings.Contains(stderr, "no inbox") ||
				strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no active"),
			"error should indicate no inbox, got: %s", stderr)
	})

	t.Run("wait with invalid regex", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "wait",
			"--timeout", "1s",
			"--subject-regex", "[invalid(regex")

		assert.NotEqual(t, 0, code, "should fail for invalid regex")
		assert.True(t,
			strings.Contains(stderr, "regex") ||
				strings.Contains(stderr, "pattern") ||
				strings.Contains(stderr, "invalid") ||
				strings.Contains(stderr, "parse"),
			"error should mention invalid regex, got: %s", stderr)
	})

	t.Run("wait with invalid from regex", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "wait",
			"--timeout", "1s",
			"--from-regex", "[bad(pattern")

		assert.NotEqual(t, 0, code, "should fail for invalid regex")
		assert.True(t,
			strings.Contains(stderr, "regex") ||
				strings.Contains(stderr, "pattern") ||
				strings.Contains(stderr, "invalid") ||
				strings.Contains(stderr, "parse"),
			"error should mention invalid regex, got: %s", stderr)
	})

	t.Run("wait with invalid timeout format", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "wait", "--timeout", "notaduration")

		assert.NotEqual(t, 0, code, "should fail for invalid timeout")
		assert.True(t,
			strings.Contains(strings.ToLower(stderr), "invalid") ||
				strings.Contains(strings.ToLower(stderr), "duration") ||
				strings.Contains(strings.ToLower(stderr), "parse") ||
				strings.Contains(strings.ToLower(stderr), "timeout"),
			"error should indicate invalid timeout, got: %s", stderr)
	})

	t.Run("wait with invalid count", func(t *testing.T) {
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "wait",
			"--timeout", "1s",
			"--count", "-1")

		// Should either fail or treat as invalid
		if code != 0 {
			assert.True(t,
				strings.Contains(stderr, "count") ||
					strings.Contains(stderr, "invalid") ||
					strings.Contains(stderr, "positive"),
				"error should mention invalid count, got: %s", stderr)
		}
	})
}

// TestExportImportErrors tests error handling for export/import commands.
func TestExportImportErrors(t *testing.T) {
	t.Run("export without inbox", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "export")
		assert.NotEqual(t, 0, code, "should fail when no inbox exists")
		assert.True(t,
			strings.Contains(stderr, "no inbox") ||
				strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no active"),
			"error should indicate no inbox, got: %s", stderr)
	})

	t.Run("export to existing file", func(t *testing.T) {
		configDir := t.TempDir()

		// Create inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", result.Email)
		})

		// Create file at export path
		exportPath := filepath.Join(t.TempDir(), "existing.json")
		require.NoError(t, os.WriteFile(exportPath, []byte("existing"), 0600))

		// Try to export to existing file
		_, stderr, code := runVSBWithConfig(t, configDir, "export", "--out", exportPath)
		assert.NotEqual(t, 0, code, "should fail when file exists")
		assert.True(t,
			strings.Contains(stderr, "exists") ||
				strings.Contains(stderr, "already") ||
				strings.Contains(stderr, "overwrite"),
			"error should mention file exists, got: %s", stderr)
	})

	t.Run("export to invalid path", func(t *testing.T) {
		configDir := t.TempDir()

		// Create inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", result.Email)
		})

		// Try to export to non-existent directory
		invalidPath := "/nonexistent/directory/export.json"
		_, stderr, code := runVSBWithConfig(t, configDir, "export", "--out", invalidPath)
		assert.NotEqual(t, 0, code, "should fail for invalid path")
		assert.True(t,
			strings.Contains(stderr, "directory") ||
				strings.Contains(stderr, "path") ||
				strings.Contains(stderr, "no such") ||
				strings.Contains(stderr, "permission") ||
				strings.Contains(stderr, "error"),
			"error should indicate path issue, got: %s", stderr)
	})

	t.Run("import non-existent file", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "import", "/nonexistent/file.json")
		assert.NotEqual(t, 0, code, "should fail for non-existent file")
		assert.True(t,
			strings.Contains(stderr, "not found") ||
				strings.Contains(stderr, "no such") ||
				strings.Contains(stderr, "does not exist") ||
				strings.Contains(stderr, "error"),
			"error should indicate file not found, got: %s", stderr)
	})

	t.Run("import invalid JSON", func(t *testing.T) {
		configDir := t.TempDir()

		// Create invalid JSON file
		invalidPath := filepath.Join(t.TempDir(), "invalid.json")
		require.NoError(t, os.WriteFile(invalidPath, []byte("not valid json {{{"), 0600))

		_, stderr, code := runVSBWithConfig(t, configDir, "import", invalidPath)
		assert.NotEqual(t, 0, code, "should fail for invalid JSON")
		assert.True(t,
			strings.Contains(stderr, "json") ||
				strings.Contains(stderr, "parse") ||
				strings.Contains(stderr, "invalid") ||
				strings.Contains(stderr, "decode"),
			"error should indicate JSON parse error, got: %s", stderr)
	})

	t.Run("import missing required fields", func(t *testing.T) {
		configDir := t.TempDir()

		// Create JSON with missing fields
		incompletePath := filepath.Join(t.TempDir(), "incomplete.json")
		incompleteJSON := `{"version": 1, "emailAddress": "test@example.com"}`
		require.NoError(t, os.WriteFile(incompletePath, []byte(incompleteJSON), 0600))

		_, stderr, code := runVSBWithConfig(t, configDir, "import", incompletePath)
		assert.NotEqual(t, 0, code, "should fail for missing fields")
		stderrLower := strings.ToLower(stderr)
		assert.True(t,
			strings.Contains(stderrLower, "missing") ||
				strings.Contains(stderrLower, "required") ||
				strings.Contains(stderrLower, "invalid") ||
				strings.Contains(stderrLower, "key") ||
				strings.Contains(stderrLower, "expired") ||
				strings.Contains(stderrLower, "error"),
			"error should indicate missing fields, got: %s", stderr)
	})

	t.Run("import expired inbox", func(t *testing.T) {
		configDir := t.TempDir()

		// Create expired export file
		expiredPath := filepath.Join(t.TempDir(), "expired.json")
		expiredJSON := `{
			"version": 1,
			"emailAddress": "expired@example.com",
			"inboxHash": "abc123",
			"expiresAt": "2020-01-01T00:00:00Z",
			"exportedAt": "2020-01-01T00:00:00Z",
			"keys": {
				"kemPrivate": "key1",
				"kemPublic": "key2",
				"serverSigPk": "key3"
			}
		}`
		require.NoError(t, os.WriteFile(expiredPath, []byte(expiredJSON), 0600))

		_, stderr, code := runVSBWithConfig(t, configDir, "import", expiredPath)
		assert.NotEqual(t, 0, code, "should fail for expired inbox")
		assert.Contains(t, stderr, "expired", "error should mention expiration")
	})

	t.Run("import unsupported version", func(t *testing.T) {
		configDir := t.TempDir()

		// Create file with unsupported version
		versionPath := filepath.Join(t.TempDir(), "future-version.json")
		futureJSON := `{
			"version": 999,
			"emailAddress": "future@example.com",
			"inboxHash": "abc123",
			"expiresAt": "2099-01-01T00:00:00Z",
			"exportedAt": "2024-01-01T00:00:00Z",
			"keys": {
				"kemPrivate": "key1",
				"kemPublic": "key2",
				"serverSigPk": "key3"
			}
		}`
		require.NoError(t, os.WriteFile(versionPath, []byte(futureJSON), 0600))

		_, stderr, code := runVSBWithConfig(t, configDir, "import", versionPath)
		assert.NotEqual(t, 0, code, "should fail for unsupported version")
		assert.Contains(t, stderr, "version", "error should mention version")
	})
}

// TestConfigErrors tests error handling for config commands.
func TestConfigErrors(t *testing.T) {
	t.Run("set invalid config key", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "invalid-key", "value")
		assert.NotEqual(t, 0, code, "should fail for invalid config key")
		assert.True(t,
			strings.Contains(stderr, "unknown") ||
				strings.Contains(stderr, "invalid") ||
				strings.Contains(stderr, "unrecognized") ||
				strings.Contains(stderr, "key"),
			"error should indicate unknown key, got: %s", stderr)
	})

	t.Run("set without value", func(t *testing.T) {
		configDir := t.TempDir()

		// This might be handled as missing argument
		_, stderr, code := runVSBWithConfig(t, configDir, "config", "set", "api-key")

		// Either should fail or prompt for value
		if code != 0 {
			assert.True(t,
				strings.Contains(stderr, "value") ||
					strings.Contains(stderr, "argument") ||
					strings.Contains(stderr, "required") ||
					strings.Contains(stderr, "missing"),
				"error should indicate missing value, got: %s", stderr)
		}
	})
}

// TestGlobalErrors tests global error scenarios.
func TestGlobalErrors(t *testing.T) {
	t.Run("unknown command", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "unknowncommand")
		assert.NotEqual(t, 0, code, "should fail for unknown command")
		assert.True(t,
			strings.Contains(stderr, "unknown") ||
				strings.Contains(stderr, "invalid") ||
				strings.Contains(stderr, "command"),
			"error should indicate unknown command, got: %s", stderr)
	})

	t.Run("unknown subcommand", func(t *testing.T) {
		configDir := t.TempDir()

		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "unknownsub")
		assert.NotEqual(t, 0, code, "should fail for unknown subcommand")
		assert.True(t,
			strings.Contains(stderr, "unknown") ||
				strings.Contains(stderr, "invalid") ||
				strings.Contains(stderr, "command"),
			"error should indicate unknown subcommand, got: %s", stderr)
	})

	t.Run("invalid output format", func(t *testing.T) {
		configDir := t.TempDir()

		// Create inbox first
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", result.Email)
		})

		// Try invalid output format
		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "list", "--output", "invalid")

		// May succeed with default or fail
		if code != 0 {
			assert.True(t,
				strings.Contains(stderr, "output") ||
					strings.Contains(stderr, "format") ||
					strings.Contains(stderr, "invalid"),
				"error should indicate invalid output format, got: %s", stderr)
		}
	})

	t.Run("conflicting flags", func(t *testing.T) {
		skipIfNoSMTP(t)
		configDir := t.TempDir()

		// Create inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", result.Email)
		})

		// Try conflicting flags (both --subject and --subject-regex)
		// Behavior depends on implementation - might use one or error
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "wait",
			"--timeout", "1s",
			"--subject", "exact",
			"--subject-regex", "regex.*")

		// Log the result - this test documents the behavior
		t.Logf("Conflicting flags result: code=%d, stderr=%s", code, stderr)
	})
}

// TestNetworkErrors tests behavior with network issues.
func TestNetworkErrors(t *testing.T) {
	t.Run("invalid base URL", func(t *testing.T) {
		configDir := t.TempDir()

		// Override with invalid URL
		stdout, stderr, code := runVSBWithConfigAndEnv(t, configDir,
			map[string]string{"VSB_BASE_URL": "http://invalid.local.domain:99999"},
			"inbox", "create")

		assert.NotEqual(t, 0, code, "should fail with invalid URL")
		assert.True(t,
			strings.Contains(stderr, "connection") ||
				strings.Contains(stderr, "connect") ||
				strings.Contains(stderr, "refused") ||
				strings.Contains(stderr, "error") ||
				strings.Contains(stderr, "dial") ||
				strings.Contains(stderr, "lookup") ||
				strings.Contains(stdout+stderr, "error"),
			"error should indicate connection issue, got stdout: %s, stderr: %s", stdout, stderr)
	})

	t.Run("empty API key", func(t *testing.T) {
		configDir := t.TempDir()

		// Override with empty API key
		stdout, stderr, code := runVSBWithConfigAndEnv(t, configDir,
			map[string]string{"VSB_API_KEY": ""},
			"inbox", "create")

		assert.NotEqual(t, 0, code, "should fail with empty API key")
		assert.True(t,
			strings.Contains(stderr, "api") ||
				strings.Contains(stderr, "key") ||
				strings.Contains(stderr, "auth") ||
				strings.Contains(stderr, "required") ||
				strings.Contains(stderr, "missing") ||
				strings.Contains(stdout+stderr, "error"),
			"error should indicate API key issue, got stdout: %s, stderr: %s", stdout, stderr)
	})

	t.Run("invalid API key", func(t *testing.T) {
		configDir := t.TempDir()

		// Override with invalid API key
		stdout, stderr, code := runVSBWithConfigAndEnv(t, configDir,
			map[string]string{"VSB_API_KEY": "invalid-key-format-123"},
			"inbox", "create")

		assert.NotEqual(t, 0, code, "should fail with invalid API key")
		// The error could be auth-related or connection-related depending on the server
		assert.True(t,
			strings.Contains(stderr, "auth") ||
				strings.Contains(stderr, "unauthorized") ||
				strings.Contains(stderr, "invalid") ||
				strings.Contains(stderr, "key") ||
				strings.Contains(stderr, "error") ||
				strings.Contains(stdout+stderr, "error") ||
				strings.Contains(stdout+stderr, "401"),
			"error should indicate auth issue, got stdout: %s, stderr: %s", stdout, stderr)
	})
}

// runVSBWithConfigAndEnv runs vsb with custom environment overrides.
func runVSBWithConfigAndEnv(t *testing.T, configDir string, envOverrides map[string]string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(vsbBinPath, args...)
	cmd.Dir = configDir

	// Build base environment
	env := os.Environ()

	// Filter out keys we're overriding
	var filteredEnv []string
	for _, e := range env {
		override := false
		for key := range envOverrides {
			if strings.HasPrefix(e, key+"=") {
				override = true
				break
			}
		}
		if !override {
			filteredEnv = append(filteredEnv, e)
		}
	}

	// Add our overrides
	for key, value := range envOverrides {
		filteredEnv = append(filteredEnv, key+"="+value)
	}

	// Always set config dir and disable color
	filteredEnv = append(filteredEnv,
		"VSB_CONFIG_DIR="+configDir,
		"NO_COLOR=1",
	)

	cmd.Env = filteredEnv

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Logf("exec error: %v", err)
		exitCode = -1
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// ============================================================================
// Concurrent Operation Tests
// ============================================================================
// These tests verify the CLI handles concurrent operations correctly.

// TestConcurrentInboxOperations tests creating multiple inboxes concurrently.
func TestConcurrentInboxOperations(t *testing.T) {
	configDir := t.TempDir()

	t.Run("create multiple inboxes concurrently", func(t *testing.T) {
		const numInboxes = 5
		var wg sync.WaitGroup
		results := make(chan struct {
			email string
			err   error
		}, numInboxes)

		for i := 0; i < numInboxes; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				// Stagger requests to avoid rate limiting
				time.Sleep(time.Duration(idx) * 500 * time.Millisecond)

				stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
				if code != 0 {
					results <- struct {
						email string
						err   error
					}{err: fmt.Errorf("inbox create failed: code=%d, stderr=%s", code, stderr)}
					return
				}

				var result struct {
					Email string `json:"email"`
				}
				if err := json.Unmarshal([]byte(stdout), &result); err != nil {
					results <- struct {
						email string
						err   error
					}{err: fmt.Errorf("failed to parse JSON: %w", err)}
					return
				}

				results <- struct {
					email string
					err   error
				}{email: result.Email}
			}(i)
		}

		wg.Wait()
		close(results)

		var emails []string
		for res := range results {
			if res.err != nil {
				t.Error(res.err)
				continue
			}
			emails = append(emails, res.email)
		}

		// All inboxes should have been created
		assert.Len(t, emails, numInboxes, "expected %d inboxes to be created", numInboxes)

		// Verify all inboxes are unique
		emailSet := make(map[string]bool)
		for _, email := range emails {
			assert.False(t, emailSet[email], "duplicate email found: %s", email)
			emailSet[email] = true
		}

		// Verify list shows correct count
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code)

		var listResult []struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &listResult))
		assert.Equal(t, numInboxes, len(listResult), "list count should match created inboxes")

		// Cleanup
		t.Cleanup(func() {
			for _, email := range emails {
				runVSBWithConfig(t, configDir, "inbox", "delete", email)
				time.Sleep(200 * time.Millisecond) // Delay between deletes too
			}
		})
	})

	t.Run("concurrent list operations", func(t *testing.T) {
		// Create an inbox first
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", result.Email)
		})

		// Run concurrent list operations
		const numLists = 10
		var wg sync.WaitGroup
		errors := make(chan error, numLists)

		for i := 0; i < numLists; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "list")
				if code != 0 {
					errors <- fmt.Errorf("inbox list failed: stderr=%s", stderr)
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Error(err)
		}
	})
}

// TestConcurrentEmailOperations tests concurrent email listing.
func TestConcurrentEmailOperations(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", createResult.Email)
	})

	// Send test email
	sendTestEmail(t, createResult.Email, "Concurrent Test Email", "Test body")
	time.Sleep(2 * time.Second)

	t.Run("concurrent email list operations", func(t *testing.T) {
		const numReads = 5
		var wg sync.WaitGroup
		errors := make(chan error, numReads)

		for i := 0; i < numReads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, stderr, code := runVSBWithConfig(t, configDir, "email", "list")
				if code != 0 {
					errors <- fmt.Errorf("email list failed: stderr=%s", stderr)
				}
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Error(err)
		}
	})
}
