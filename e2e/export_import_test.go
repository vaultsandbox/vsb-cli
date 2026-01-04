//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ExportedInboxFile mirrors the config package's export format for testing
type ExportedInboxFile struct {
	Version      int       `json:"version"`
	EmailAddress string    `json:"emailAddress"`
	InboxHash    string    `json:"inboxHash"`
	ExpiresAt    time.Time `json:"expiresAt"`
	ExportedAt   time.Time `json:"exportedAt"`
	Keys         struct {
		KEMPrivate  string `json:"kemPrivate"`
		KEMPublic   string `json:"kemPublic"`
		ServerSigPK string `json:"serverSigPk"`
	} `json:"keys"`
}

// TestExport tests exporting inboxes.
func TestExport(t *testing.T) {
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

	t.Run("export active inbox", func(t *testing.T) {
		exportDir := t.TempDir()
		exportPath := filepath.Join(exportDir, "inbox-export.json")

		_, stderr, code := runVSBWithConfig(t, configDir, "export", "--out", exportPath)
		require.Equal(t, 0, code, "export failed: stderr=%s", stderr)

		// Verify file exists
		_, err := os.Stat(exportPath)
		require.NoError(t, err, "export file should exist")

		// Read and verify contents
		data, err := os.ReadFile(exportPath)
		require.NoError(t, err)

		var exported ExportedInboxFile
		require.NoError(t, json.Unmarshal(data, &exported))

		assert.Equal(t, 1, exported.Version)
		assert.Equal(t, inboxEmail, exported.EmailAddress)
		assert.NotEmpty(t, exported.InboxHash)
		assert.NotEmpty(t, exported.Keys.KEMPrivate)
		// KEMPublic is intentionally empty - derived from secret key per spec Section 4.2
		assert.Empty(t, exported.Keys.KEMPublic)
		assert.NotEmpty(t, exported.Keys.ServerSigPK)
		assert.True(t, exported.ExpiresAt.After(time.Now()))
	})

	t.Run("export specific inbox", func(t *testing.T) {
		exportDir := t.TempDir()
		exportPath := filepath.Join(exportDir, "specific-export.json")

		// Use full email address
		_, stderr, code := runVSBWithConfig(t, configDir, "export", inboxEmail, "--out", exportPath)
		require.Equal(t, 0, code, "export specific failed: stderr=%s", stderr)

		// Verify file contents
		data, err := os.ReadFile(exportPath)
		require.NoError(t, err)

		var exported ExportedInboxFile
		require.NoError(t, json.Unmarshal(data, &exported))
		assert.Equal(t, inboxEmail, exported.EmailAddress)
	})

	t.Run("export creates default filename", func(t *testing.T) {
		// Create a new config dir to avoid filename collision
		exportConfigDir := t.TempDir()

		// Create inbox in the new config dir
		stdout, _, code := runVSBWithConfig(t, exportConfigDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		t.Cleanup(func() {
			runVSBWithConfig(t, exportConfigDir, "inbox", "delete", result.Email)
		})

		// Export without --out flag (will create email.json in current dir)
		_, stderr, code := runVSBWithConfig(t, exportConfigDir, "export")
		require.Equal(t, 0, code, "export default failed: stderr=%s", stderr)

		// The default filename is based on the sanitized email
		// Check that some json file was created
		files, err := os.ReadDir(exportConfigDir)
		require.NoError(t, err)

		found := false
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".json") && !strings.Contains(f.Name(), "keystore") {
				found = true
				break
			}
		}
		assert.True(t, found, "default export file should be created")
	})

	t.Run("verify file permissions", func(t *testing.T) {
		exportDir := t.TempDir()
		exportPath := filepath.Join(exportDir, "secure-export.json")

		_, stderr, code := runVSBWithConfig(t, configDir, "export", "--out", exportPath)
		require.Equal(t, 0, code, "export failed: stderr=%s", stderr)

		// Check file permissions (should be 0600)
		info, err := os.Stat(exportPath)
		require.NoError(t, err)

		mode := info.Mode().Perm()
		assert.Equal(t, os.FileMode(0600), mode, "export file should have 0600 permissions")
	})
}

// TestImport tests importing inboxes.
func TestImport(t *testing.T) {
	t.Run("import valid export file", func(t *testing.T) {
		configDir := t.TempDir()

		// Create and export inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var createResult struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
		originalEmail := createResult.Email

		exportPath := filepath.Join(t.TempDir(), "import-test.json")
		_, _, code = runVSBWithConfig(t, configDir, "export", "--out", exportPath)
		require.Equal(t, 0, code)

		// Delete the inbox locally
		_, _, code = runVSBWithConfig(t, configDir, "inbox", "delete", "--local", originalEmail)
		require.Equal(t, 0, code)

		// Verify inbox is gone
		stdout, _, code = runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code)

		var listResult []struct {
			Email string `json:"email"`
		}
		json.Unmarshal([]byte(stdout), &listResult)

		found := false
		for _, inbox := range listResult {
			if inbox.Email == originalEmail {
				found = true
				break
			}
		}
		assert.False(t, found, "inbox should be deleted before import")

		// Import the export file
		_, stderr, code := runVSBWithConfig(t, configDir, "import", exportPath)
		require.Equal(t, 0, code, "import failed: stderr=%s", stderr)

		// Verify inbox is back
		stdout, _, code = runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code)

		json.Unmarshal([]byte(stdout), &listResult)

		found = false
		for _, inbox := range listResult {
			if inbox.Email == originalEmail {
				found = true
				break
			}
		}
		assert.True(t, found, "inbox should be restored after import")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", originalEmail)
		})
	})

	t.Run("import with local flag", func(t *testing.T) {
		configDir := t.TempDir()

		// Create and export inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var createResult struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
		originalEmail := createResult.Email

		exportPath := filepath.Join(t.TempDir(), "local-import.json")
		_, _, code = runVSBWithConfig(t, configDir, "export", "--out", exportPath)
		require.Equal(t, 0, code)

		// Delete locally
		_, _, code = runVSBWithConfig(t, configDir, "inbox", "delete", "--local", originalEmail)
		require.Equal(t, 0, code)

		// Import with --local flag (skip server verification)
		_, stderr, code := runVSBWithConfig(t, configDir, "import", "--local", exportPath)
		require.Equal(t, 0, code, "import --local failed: stderr=%s", stderr)

		// Verify inbox is back
		stdout, _, code = runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
		require.Equal(t, 0, code)

		var listResult []struct {
			Email string `json:"email"`
		}
		json.Unmarshal([]byte(stdout), &listResult)

		found := false
		for _, inbox := range listResult {
			if inbox.Email == originalEmail {
				found = true
				break
			}
		}
		assert.True(t, found, "inbox should be restored with --local flag")

		// Cleanup
		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", originalEmail)
		})
	})

	t.Run("reject duplicate without force", func(t *testing.T) {
		configDir := t.TempDir()

		// Create and export inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var createResult struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
		originalEmail := createResult.Email

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", originalEmail)
		})

		exportPath := filepath.Join(t.TempDir(), "duplicate-test.json")
		_, _, code = runVSBWithConfig(t, configDir, "export", "--out", exportPath)
		require.Equal(t, 0, code)

		// Try to import while inbox still exists (should fail)
		_, stderr, code := runVSBWithConfig(t, configDir, "import", exportPath)
		assert.NotEqual(t, 0, code, "import should fail for duplicate")
		assert.Contains(t, stderr, "already exists", "error should mention inbox already exists")
	})

	t.Run("override with force flag", func(t *testing.T) {
		configDir := t.TempDir()

		// Create and export inbox
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var createResult struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
		originalEmail := createResult.Email

		t.Cleanup(func() {
			runVSBWithConfig(t, configDir, "inbox", "delete", originalEmail)
		})

		exportPath := filepath.Join(t.TempDir(), "force-test.json")
		_, _, code = runVSBWithConfig(t, configDir, "export", "--out", exportPath)
		require.Equal(t, 0, code)

		// Import with --force while inbox exists (should succeed)
		_, stderr, code := runVSBWithConfig(t, configDir, "import", "--force", exportPath)
		require.Equal(t, 0, code, "import --force failed: stderr=%s", stderr)
	})

	t.Run("reject expired inbox", func(t *testing.T) {
		configDir := t.TempDir()

		// Create an export file with an expired timestamp
		expiredExport := ExportedInboxFile{
			Version:      1,
			EmailAddress: "expired@example.com",
			InboxHash:    "abc123",
			ExpiresAt:    time.Now().Add(-24 * time.Hour), // Expired yesterday
			ExportedAt:   time.Now(),
			Keys: struct {
				KEMPrivate  string `json:"kemPrivate"`
				KEMPublic   string `json:"kemPublic"`
				ServerSigPK string `json:"serverSigPk"`
			}{
				KEMPrivate:  "fakeprivatekey",
				KEMPublic:   "fakepublickey",
				ServerSigPK: "fakeserverkey",
			},
		}

		exportPath := filepath.Join(t.TempDir(), "expired-export.json")
		data, err := json.Marshal(expiredExport)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(exportPath, data, 0600))

		// Try to import expired inbox (should fail)
		_, stderr, code := runVSBWithConfig(t, configDir, "import", exportPath)
		assert.NotEqual(t, 0, code, "import should fail for expired inbox")
		assert.Contains(t, stderr, "expired", "error should mention expiration")
	})

	t.Run("reject invalid version", func(t *testing.T) {
		configDir := t.TempDir()

		// Create export file with invalid version
		invalidExport := map[string]interface{}{
			"version":      999,
			"emailAddress": "test@example.com",
			"inboxHash":    "abc123",
			"expiresAt":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"exportedAt":   time.Now().Format(time.RFC3339),
			"keys": map[string]string{
				"kemPrivate":  "key1",
				"kemPublic":   "key2",
				"serverSigPk": "key3",
			},
		}

		exportPath := filepath.Join(t.TempDir(), "invalid-version.json")
		data, err := json.Marshal(invalidExport)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(exportPath, data, 0600))

		// Try to import (should fail)
		_, stderr, code := runVSBWithConfig(t, configDir, "import", exportPath)
		assert.NotEqual(t, 0, code, "import should fail for invalid version")
		assert.Contains(t, stderr, "version", "error should mention version")
	})
}

// TestExportImportRoundTrip tests the complete backup/restore workflow.
func TestExportImportRoundTrip(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Step 1: Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	originalEmail := createResult.Email

	// Step 2: Send some test emails
	sendTestEmail(t, originalEmail, "Round Trip Test 1", "First test email")
	sendTestEmail(t, originalEmail, "Round Trip Test 2", "Second test email")
	time.Sleep(2 * time.Second)

	// Step 3: Verify emails are received
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var emailsBefore []struct {
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emailsBefore))
	require.GreaterOrEqual(t, len(emailsBefore), 2)

	// Step 4: Export the inbox
	exportPath := filepath.Join(t.TempDir(), "roundtrip-export.json")
	_, _, code = runVSBWithConfig(t, configDir, "export", "--out", exportPath)
	require.Equal(t, 0, code)

	// Step 5: Delete inbox locally (but keep on server for now)
	_, _, code = runVSBWithConfig(t, configDir, "inbox", "delete", "--local", originalEmail)
	require.Equal(t, 0, code)

	// Step 6: Create a fresh config directory (simulating different machine)
	newConfigDir := t.TempDir()

	// Step 7: Import the export
	_, stderr, code := runVSBWithConfig(t, newConfigDir, "import", exportPath)
	require.Equal(t, 0, code, "import failed: stderr=%s", stderr)

	// Step 8: Verify inbox is available
	stdout, _, code = runVSBWithConfig(t, newConfigDir, "inbox", "info", "--output", "json")
	require.Equal(t, 0, code)

	var inboxInfo struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &inboxInfo))
	assert.Equal(t, originalEmail, inboxInfo.Email)

	// Step 9: Verify we can still read emails
	stdout, _, code = runVSBWithConfig(t, newConfigDir, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var emailsAfter []struct {
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emailsAfter))

	// Should have at least the same emails as before
	assert.GreaterOrEqual(t, len(emailsAfter), len(emailsBefore))

	// Verify our test emails are still accessible
	foundTest1 := false
	foundTest2 := false
	for _, email := range emailsAfter {
		if email.Subject == "Round Trip Test 1" {
			foundTest1 = true
		}
		if email.Subject == "Round Trip Test 2" {
			foundTest2 = true
		}
	}
	assert.True(t, foundTest1, "Round Trip Test 1 should be accessible after import")
	assert.True(t, foundTest2, "Round Trip Test 2 should be accessible after import")

	// Cleanup
	t.Cleanup(func() {
		runVSBWithConfig(t, newConfigDir, "inbox", "delete", originalEmail)
	})
}

// TestExportFileOverwrite tests that export doesn't overwrite existing files.
func TestExportFileOverwrite(t *testing.T) {
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

	// Create a file at the export path
	exportDir := t.TempDir()
	exportPath := filepath.Join(exportDir, "existing-file.json")
	require.NoError(t, os.WriteFile(exportPath, []byte("existing content"), 0600))

	// Try to export to the same path (should fail)
	_, stderr, code := runVSBWithConfig(t, configDir, "export", "--out", exportPath)
	assert.NotEqual(t, 0, code, "export should fail when file exists")
	assert.Contains(t, stderr, "exists", "error should mention file exists")

	// Verify original file content wasn't modified
	data, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	assert.Equal(t, "existing content", string(data))
}

// TestImportPartialMatch tests importing with partial email matching.
func TestImportPartialMatch(t *testing.T) {
	configDir := t.TempDir()

	// Create inbox
	stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
	require.Equal(t, 0, code)

	var createResult struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &createResult))
	originalEmail := createResult.Email

	t.Cleanup(func() {
		runVSBWithConfig(t, configDir, "inbox", "delete", originalEmail)
	})

	// Export using partial match
	parts := strings.Split(originalEmail, "@")
	partial := parts[0][:6]

	exportPath := filepath.Join(t.TempDir(), "partial-export.json")
	_, stderr, code := runVSBWithConfig(t, configDir, "export", partial, "--out", exportPath)
	require.Equal(t, 0, code, "export with partial match failed: stderr=%s", stderr)

	// Verify export contains full email
	data, err := os.ReadFile(exportPath)
	require.NoError(t, err)

	var exported ExportedInboxFile
	require.NoError(t, json.Unmarshal(data, &exported))
	assert.Equal(t, originalEmail, exported.EmailAddress)
}
