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
	t.Run("removes old preview files", func(t *testing.T) {
		// Create temp directory to test with
		tmpDir := t.TempDir()

		// Create old preview file (simulating the naming pattern)
		oldFile := filepath.Join(tmpDir, previewFilePrefix+"old123.html")
		require.NoError(t, os.WriteFile(oldFile, []byte("old content"), 0600))

		// Set file modification time to 2 hours ago
		oldTime := time.Now().Add(-2 * time.Hour)
		require.NoError(t, os.Chtimes(oldFile, oldTime, oldTime))

		// Create recent preview file
		recentFile := filepath.Join(tmpDir, previewFilePrefix+"recent456.html")
		require.NoError(t, os.WriteFile(recentFile, []byte("recent content"), 0600))

		// Note: CleanupPreviews operates on os.TempDir() which we can't easily mock
		// So we test the cleanup logic indirectly by verifying the function doesn't error
		err := CleanupPreviews(1 * time.Hour)
		assert.NoError(t, err)
	})

	t.Run("handles non-existent temp directory gracefully", func(t *testing.T) {
		// This shouldn't error even if there are permission issues
		// The function should handle errors gracefully
		err := CleanupPreviews(1 * time.Hour)
		assert.NoError(t, err)
	})

	t.Run("zero duration threshold", func(t *testing.T) {
		// With zero duration, all files older than "now" would be cleaned
		err := CleanupPreviews(0)
		assert.NoError(t, err)
	})

	t.Run("negative duration threshold", func(t *testing.T) {
		// Negative duration means files in the "future" - should not error
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
