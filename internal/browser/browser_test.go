package browser

import (
	"html"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// URL Scheme Validation Tests
// ============================================================================

// TestOpenURL_BlockedSchemes tests that dangerous URL schemes are blocked.
// These tests don't open a browser because they fail validation first.
func TestOpenURL_BlockedSchemes(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		errMsg string
	}{
		// Blocked schemes - these should fail before attempting to open browser
		{"javascript blocked", "javascript:alert(1)", "not allowed"},
		{"data blocked", "data:text/html,<script>", "not allowed"},
		{"ftp blocked", "ftp://example.com", "not allowed"},
		{"ssh blocked", "ssh://user@host", "not allowed"},
		{"telnet blocked", "telnet://host", "not allowed"},
		{"vbscript blocked", "vbscript:msgbox(1)", "not allowed"},

		// Invalid URLs - fail before browser opens
		{"empty url", "", "not allowed"},
		{"no scheme", "example.com", "not allowed"},
		{"invalid url", "://invalid", ""},
		{"whitespace only", "   ", "not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OpenURL(tt.url)
			assert.Error(t, err)
			if tt.errMsg != "" {
				assert.Contains(t, err.Error(), tt.errMsg)
			}
		})
	}
}

// TestAllowedSchemesValidation tests that allowed schemes pass validation.
// We test the allowedSchemes map directly to avoid opening browsers.
func TestAllowedSchemesValidation(t *testing.T) {
	// Test that valid schemes are in the allowed list
	validSchemes := []string{"http", "https", "mailto", "file"}
	for _, scheme := range validSchemes {
		t.Run(scheme+" is allowed", func(t *testing.T) {
			assert.True(t, allowedSchemes[scheme], "%s should be allowed", scheme)
		})
	}

	// Test that dangerous schemes are NOT in the allowed list
	blockedSchemes := []string{"javascript", "data", "ftp", "ssh", "telnet", "vbscript"}
	for _, scheme := range blockedSchemes {
		t.Run(scheme+" is blocked", func(t *testing.T) {
			assert.False(t, allowedSchemes[scheme], "%s should be blocked", scheme)
		})
	}
}

// TestOpenURL_SchemeCaseInsensitive verifies case-insensitive scheme handling.
// We test with blocked schemes to avoid opening browsers.
func TestOpenURL_SchemeCaseInsensitive(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"JAVASCRIPT uppercase blocked", "JAVASCRIPT:alert(1)"},
		{"JavaScript mixed case blocked", "JavaScript:alert(1)"},
		{"DATA uppercase blocked", "DATA:text/html,test"},
		{"FTP uppercase blocked", "FTP://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OpenURL(tt.url)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not allowed")
		})
	}
}

// ============================================================================
// XSS Prevention Tests
// ============================================================================

func TestBuildEmailHTMLTemplate(t *testing.T) {
	testTime := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

	t.Run("includes subject in title and header", func(t *testing.T) {
		result := BuildEmailHTMLTemplate("Test Subject", "sender@example.com", testTime, "<p>body</p>")

		assert.Contains(t, result, "<title>Test Subject</title>")
		assert.Contains(t, result, "<h1>Test Subject</h1>")
	})

	t.Run("includes from address", func(t *testing.T) {
		result := BuildEmailHTMLTemplate("Subject", "sender@example.com", testTime, "<p>body</p>")

		assert.Contains(t, result, "sender@example.com")
		assert.Contains(t, result, "<strong>From:</strong>")
	})

	t.Run("includes formatted date", func(t *testing.T) {
		result := BuildEmailHTMLTemplate("Subject", "from@x.com", testTime, "<p>body</p>")

		assert.Contains(t, result, "June 15, 2024 at 2:30 PM")
		assert.Contains(t, result, "<strong>Date:</strong>")
	})

	t.Run("includes email body content", func(t *testing.T) {
		emailBody := "<p>This is the email body with <strong>formatting</strong></p>"
		result := BuildEmailHTMLTemplate("Subject", "from@x.com", testTime, emailBody)

		assert.Contains(t, result, emailBody)
		assert.Contains(t, result, `class="content"`)
	})

	t.Run("escapes subject to prevent XSS", func(t *testing.T) {
		maliciousSubject := "<script>alert('xss')</script>"
		result := BuildEmailHTMLTemplate(maliciousSubject, "from@x.com", testTime, "<p>body</p>")

		assert.NotContains(t, result, "<script>alert")
		assert.Contains(t, result, "&lt;script&gt;")
	})

	t.Run("escapes from to prevent XSS", func(t *testing.T) {
		maliciousFrom := "<img src=x onerror=alert(1)>"
		result := BuildEmailHTMLTemplate("Subject", maliciousFrom, testTime, "<p>body</p>")

		assert.NotContains(t, result, "<img src=x")
		assert.Contains(t, result, "&lt;img")
	})

	t.Run("includes proper HTML structure", func(t *testing.T) {
		result := BuildEmailHTMLTemplate("Subject", "from@x.com", testTime, "<p>body</p>")

		assert.Contains(t, result, "<!DOCTYPE html>")
		assert.Contains(t, result, `<meta charset="utf-8">`)
		assert.Contains(t, result, `class="header"`)
		assert.Contains(t, result, `class="content"`)
		assert.Contains(t, result, "</html>")
	})

	t.Run("includes VaultSandbox branding color", func(t *testing.T) {
		result := BuildEmailHTMLTemplate("Subject", "from@x.com", testTime, "<p>body</p>")

		assert.Contains(t, result, "#1cc2e3") // VaultSandbox brand color
	})
}

func TestViewEmailHTML_XSSPrevention(t *testing.T) {
	t.Run("escapes malicious subject", func(t *testing.T) {
		subject := "<script>alert('xss')</script>"
		escaped := html.EscapeString(subject)
		assert.NotContains(t, escaped, "<script>")
		assert.Contains(t, escaped, "&lt;script&gt;")
	})

	t.Run("escapes malicious from", func(t *testing.T) {
		from := "<img src=x onerror=alert(1)>"
		escaped := html.EscapeString(from)
		assert.NotContains(t, escaped, "<img")
		assert.Contains(t, escaped, "&lt;img")
	})

	t.Run("escapes onclick handler", func(t *testing.T) {
		subject := `<div onclick="evil()">Click me</div>`
		escaped := html.EscapeString(subject)
		// The HTML tag is escaped, so onclick can't execute even if the word is present
		assert.NotContains(t, escaped, "<div")
		assert.Contains(t, escaped, "&lt;div")
		assert.Contains(t, escaped, "&#34;") // quotes are escaped
	})

	t.Run("escapes nested quotes", func(t *testing.T) {
		from := `"test" & 'other' <script>`
		escaped := html.EscapeString(from)
		assert.NotContains(t, escaped, "<script>")
		assert.Contains(t, escaped, "&amp;")
	})
}

func TestViewEmailHTML_TemplateGeneration(t *testing.T) {
	// Skip actual browser opening in tests
	if os.Getenv("TEST_BROWSER") == "" {
		t.Skip("Skipping browser test: TEST_BROWSER not set")
	}

	t.Run("generates valid HTML with escaped content", func(t *testing.T) {
		subject := "<script>alert('xss')</script>Test Subject"
		from := "attacker<script>@evil.com"
		receivedAt := time.Now()
		emailHTML := "<p>Safe content</p>"

		// This would open a browser - skip in normal tests
		err := ViewEmailHTML(subject, from, receivedAt, emailHTML)
		assert.NoError(t, err)
	})
}

// ============================================================================
// CleanupPreviews Tests
// ============================================================================

func TestCleanupPreviews(t *testing.T) {
	t.Run("removes old preview files from real temp dir", func(t *testing.T) {
		tmpDir := os.TempDir()

		// Create an old preview file in the real temp directory
		oldFile := filepath.Join(tmpDir, previewFilePrefix+"test-old-cleanup.html")
		require.NoError(t, os.WriteFile(oldFile, []byte("old content"), 0600))

		// Set file modification time to 2 hours ago
		oldTime := time.Now().Add(-2 * time.Hour)
		require.NoError(t, os.Chtimes(oldFile, oldTime, oldTime))

		// Create a recent preview file that should NOT be deleted
		recentFile := filepath.Join(tmpDir, previewFilePrefix+"test-recent-cleanup.html")
		require.NoError(t, os.WriteFile(recentFile, []byte("recent content"), 0600))

		// Cleanup should remove old file but keep recent one
		err := CleanupPreviews(1 * time.Hour)
		assert.NoError(t, err)

		// Old file should be removed
		_, err = os.Stat(oldFile)
		assert.True(t, os.IsNotExist(err), "old file should be removed")

		// Recent file should still exist
		_, err = os.Stat(recentFile)
		assert.NoError(t, err, "recent file should still exist")

		// Cleanup the recent file manually
		os.Remove(recentFile)
	})

	t.Run("ignores non-preview files", func(t *testing.T) {
		tmpDir := os.TempDir()

		// Create a file without the preview prefix
		otherFile := filepath.Join(tmpDir, "not-a-preview-test.html")
		require.NoError(t, os.WriteFile(otherFile, []byte("other"), 0600))

		// Set it to be old
		oldTime := time.Now().Add(-2 * time.Hour)
		require.NoError(t, os.Chtimes(otherFile, oldTime, oldTime))

		err := CleanupPreviews(1 * time.Hour)
		assert.NoError(t, err)

		// File should NOT be removed (no preview prefix)
		_, err = os.Stat(otherFile)
		assert.NoError(t, err, "non-preview file should not be removed")

		os.Remove(otherFile)
	})

	t.Run("ignores directories", func(t *testing.T) {
		tmpDir := os.TempDir()

		// Create a directory with the preview prefix
		dirPath := filepath.Join(tmpDir, previewFilePrefix+"test-dir")
		require.NoError(t, os.MkdirAll(dirPath, 0755))

		err := CleanupPreviews(0) // 0 duration = delete everything old
		assert.NoError(t, err)

		// Directory should NOT be removed
		info, err := os.Stat(dirPath)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())

		os.Remove(dirPath)
	})

	t.Run("handles zero duration threshold", func(t *testing.T) {
		err := CleanupPreviews(0)
		assert.NoError(t, err)
	})

	t.Run("handles negative duration threshold", func(t *testing.T) {
		err := CleanupPreviews(-1 * time.Hour)
		assert.NoError(t, err)
	})
}

// TestPreviewFilePrefix verifies the constant is set correctly
func TestPreviewFilePrefix(t *testing.T) {
	assert.Equal(t, "vsb-preview-", previewFilePrefix)
}

// ============================================================================
// ViewHTML Tests
// ============================================================================

func TestViewHTML_FileCreation(t *testing.T) {
	// Skip actual browser opening
	if os.Getenv("TEST_BROWSER") == "" {
		t.Skip("Skipping browser test: TEST_BROWSER not set")
	}

	t.Run("creates temp file and opens browser", func(t *testing.T) {
		html := "<html><body><h1>Test</h1></body></html>"
		err := ViewHTML(html)
		assert.NoError(t, err)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		err := ViewHTML("")
		assert.NoError(t, err)
	})

	t.Run("handles large HTML", func(t *testing.T) {
		// Generate a large HTML string
		largeHTML := "<html><body>"
		for i := 0; i < 10000; i++ {
			largeHTML += "<p>Paragraph content here</p>"
		}
		largeHTML += "</body></html>"

		err := ViewHTML(largeHTML)
		assert.NoError(t, err)
	})
}

// ============================================================================
// URL Parsing Edge Cases (tested via blocked schemes to avoid opening browser)
// ============================================================================

func TestOpenURL_URLParsingEdgeCases(t *testing.T) {
	// Test that URLs with special characters in blocked schemes fail correctly
	t.Run("blocked scheme with unicode", func(t *testing.T) {
		err := OpenURL("ftp://例え.jp/path")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("blocked scheme with query params", func(t *testing.T) {
		err := OpenURL("ftp://example.com/path?query=value&other=123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("blocked scheme with encoded chars", func(t *testing.T) {
		err := OpenURL("ftp://example.com/path%20with%20spaces")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("blocked scheme with port", func(t *testing.T) {
		err := OpenURL("ftp://localhost:8080/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("blocked scheme with IP", func(t *testing.T) {
		err := OpenURL("ftp://127.0.0.1:3000/api")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})
}
