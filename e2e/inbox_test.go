//go:build e2e

package e2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInboxCreate tests inbox creation with various TTL values.
func TestInboxCreate(t *testing.T) {
	t.Run("default TTL", func(t *testing.T) {
		configDir := t.TempDir()

		// Create inbox
		stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		// Parse JSON output
		var result struct {
			Email     string `json:"email"`
			ExpiresAt string `json:"expiresAt"`
			CreatedAt string `json:"createdAt"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		// Verify email format
		assert.Contains(t, result.Email, "@", "email should contain @")
		assert.NotEmpty(t, result.ExpiresAt)
		assert.NotEmpty(t, result.CreatedAt)

		// Verify expiry is ~24h from now (default TTL)
		expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
		require.NoError(t, err)
		expectedExpiry := time.Now().Add(24 * time.Hour)
		assert.WithinDuration(t, expectedExpiry, expiresAt, 5*time.Minute)

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", result.Email)
		})
	})

	t.Run("custom TTL 1h", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "create", "--ttl", "1h", "--output", "json")
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			Email     string `json:"email"`
			ExpiresAt string `json:"expiresAt"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		// Verify expiry is ~1h from now
		expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
		require.NoError(t, err)
		expectedExpiry := time.Now().Add(1 * time.Hour)
		assert.WithinDuration(t, expectedExpiry, expiresAt, 5*time.Minute)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", result.Email)
		})
	})

	t.Run("custom TTL 7d", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "create", "--ttl", "7d", "--output", "json")
		require.Equal(t, 0, code, "create failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			Email     string `json:"email"`
			ExpiresAt string `json:"expiresAt"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		// Verify expiry is ~7d from now
		expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
		require.NoError(t, err)
		expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
		assert.WithinDuration(t, expectedExpiry, expiresAt, 5*time.Minute)

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", result.Email)
		})
	})
}

// TestInboxList tests listing inboxes.
func TestInboxList(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		configDir := t.TempDir()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code, "list failed: stdout=%s, stderr=%s", stdout, stderr)

		// Should return empty array or null
		var result []interface{}
		err := json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			// Might be null
			assert.Equal(t, "null\n", stdout)
		} else {
			assert.Empty(t, result)
		}
	})

	t.Run("with multiple inboxes", func(t *testing.T) {
		configDir := t.TempDir()
		var emails []string

		// Create 2 inboxes
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

		// List inboxes
		stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code, "list failed: stdout=%s, stderr=%s", stdout, stderr)

		var result []struct {
			Email     string `json:"email"`
			ExpiresAt string `json:"expiresAt"`
			IsActive  bool   `json:"isActive"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Len(t, result, 2)

		// Verify both emails are in the list
		foundEmails := make(map[string]bool)
		for _, inbox := range result {
			foundEmails[inbox.Email] = true
		}
		for _, email := range emails {
			assert.True(t, foundEmails[email], "email %s should be in list", email)
		}

		// One should be active (the last created one)
		activeCount := 0
		for _, inbox := range result {
			if inbox.IsActive {
				activeCount++
			}
		}
		assert.Equal(t, 1, activeCount, "exactly one inbox should be active")
	})
}

// TestInboxInfo tests getting inbox information.
func TestInboxInfo(t *testing.T) {
	configDir := t.TempDir()

	// Create inbox first
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	email := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", email)
	})

	t.Run("active inbox info", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "info", "--output", "json")
		require.Equal(t, 0, code, "info failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			Email      string `json:"email"`
			ID         string `json:"id"`
			CreatedAt  string `json:"createdAt"`
			ExpiresAt  string `json:"expiresAt"`
			IsExpired  bool   `json:"isExpired"`
			IsActive   bool   `json:"isActive"`
			EmailCount int    `json:"emailCount"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, email, result.Email)
		assert.NotEmpty(t, result.ID)
		assert.False(t, result.IsExpired)
		assert.True(t, result.IsActive)
	})

	t.Run("info by full email", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "info", email, "--output", "json")
		require.Equal(t, 0, code, "info failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, email, result.Email)
	})

	t.Run("info by partial email", func(t *testing.T) {
		// Use first 6 chars of the local part as partial match
		parts := strings.Split(email, "@")
		partial := parts[0][:6]

		stdout, stderr, code := runVSBWithConfig(t, configDir, "inbox", "info", partial, "--output", "json")
		require.Equal(t, 0, code, "info failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, email, result.Email)
	})
}

// TestInboxUse tests switching the active inbox.
func TestInboxUse(t *testing.T) {
	configDir := t.TempDir()

	// Create two inboxes
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

	t.Run("switch active inbox", func(t *testing.T) {
		// Second inbox should be active (last created)
		// Switch to first inbox
		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "use", emails[0])
		require.Equal(t, 0, code, "use failed: stderr=%s", stderr)

		// Verify first inbox is now active
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "info", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email    string `json:"email"`
			IsActive bool   `json:"isActive"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, emails[0], result.Email)
		assert.True(t, result.IsActive)
	})

	t.Run("switch with partial match", func(t *testing.T) {
		// Switch to second inbox using partial match
		parts := strings.Split(emails[1], "@")
		partial := parts[0][:6]

		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "use", partial)
		require.Equal(t, 0, code, "use failed: stderr=%s", stderr)

		// Verify second inbox is now active
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "info", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, emails[1], result.Email)
	})

	t.Run("error on non-existent inbox", func(t *testing.T) {
		_, _, code := runVSBWithConfig(t, configDir, "inbox", "use", "nonexistent@example.com")
		assert.NotEqual(t, 0, code, "should fail for non-existent inbox")
	})
}

// TestInboxDelete tests deleting inboxes.
func TestInboxDelete(t *testing.T) {
	t.Run("delete from server and local", func(t *testing.T) {
		configDir := t.TempDir()

		// Create inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		email := result.Email

		// Delete inbox
		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "delete", email)
		require.Equal(t, 0, code, "delete failed: stderr=%s", stderr)

		// Verify inbox no longer in list
		stdout, _, code = runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code)

		var listResult []interface{}
		err := json.Unmarshal([]byte(stdout), &listResult)
		if err == nil {
			assert.Empty(t, listResult)
		}
	})

	t.Run("delete local only", func(t *testing.T) {
		configDir := t.TempDir()

		// Create inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		email := result.Email

		// Delete local only
		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "delete", "--local", email)
		require.Equal(t, 0, code, "delete --local failed: stderr=%s", stderr)

		// Verify inbox no longer in local list
		stdout, _, code = runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code)

		var listResult []interface{}
		err := json.Unmarshal([]byte(stdout), &listResult)
		if err == nil {
			assert.Empty(t, listResult)
		}

		// Note: The inbox still exists on the server, but we can't verify that
		// without re-importing it. The --local flag is primarily for cleaning
		// up local state when the server inbox was already deleted.
	})

	t.Run("delete with partial match", func(t *testing.T) {
		configDir := t.TempDir()

		// Create inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		email := result.Email

		// Delete with partial match
		parts := strings.Split(email, "@")
		partial := parts[0][:6]

		_, stderr, code := runVSBWithConfig(t, configDir, "inbox", "delete", partial)
		require.Equal(t, 0, code, "delete failed: stderr=%s", stderr)

		// Verify inbox no longer in list
		stdout, _, code = runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code)

		var listResult []interface{}
		err := json.Unmarshal([]byte(stdout), &listResult)
		if err == nil {
			assert.Empty(t, listResult)
		}
	})
}
