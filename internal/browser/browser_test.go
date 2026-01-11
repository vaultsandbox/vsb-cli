package browser

import (
	"errors"
	"html"
	"io/fs"
	"os"
	"os/exec"
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

// ============================================================================
// OpenURL Success Path Tests (with mocked exec.Command)
// ============================================================================

// mockExecCommand creates a mock command that doesn't actually run
func mockExecCommand(name string, args ...string) *exec.Cmd {
	// Use "true" command which always succeeds (cross-platform for testing)
	return exec.Command("true")
}

func TestOpenURL_SuccessPaths(t *testing.T) {
	// Save original and restore after test
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	// Mock exec.Command to prevent actual browser opening
	execCommand = mockExecCommand

	tests := []struct {
		name string
		url  string
	}{
		{"http URL", "http://example.com"},
		{"https URL", "https://example.com"},
		{"https with path", "https://example.com/path/to/page"},
		{"https with query", "https://example.com?query=value"},
		{"https with fragment", "https://example.com#section"},
		{"https with port", "https://example.com:8080"},
		{"mailto URL", "mailto:test@example.com"},
		{"mailto with subject", "mailto:test@example.com?subject=Hello"},
		{"file URL", "file:///tmp/test.html"},
		{"file URL windows style", "file:///C:/temp/test.html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := openURLInternal(tt.url)
			assert.NoError(t, err)
		})
	}
}

func TestOpenURL_CommandConstruction(t *testing.T) {
	// This test verifies the command construction logic
	// by checking that the correct command name is used per platform

	// Save original and restore after test
	originalExecCommand := execCommand
	originalGoos := goos
	defer func() {
		execCommand = originalExecCommand
		goos = originalGoos
	}()

	var capturedName string
	var capturedArgs []string

	execCommand = func(name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = args
		return exec.Command("true")
	}

	testURL := "https://example.com"

	t.Run("darwin platform", func(t *testing.T) {
		goos = "darwin"
		err := openURLInternal(testURL)
		assert.NoError(t, err)
		assert.Equal(t, "open", capturedName)
		assert.Equal(t, []string{testURL}, capturedArgs)
	})

	t.Run("linux platform", func(t *testing.T) {
		goos = "linux"
		err := openURLInternal(testURL)
		assert.NoError(t, err)
		assert.Equal(t, "xdg-open", capturedName)
		assert.Equal(t, []string{testURL}, capturedArgs)
	})

	t.Run("windows platform", func(t *testing.T) {
		goos = "windows"
		err := openURLInternal(testURL)
		assert.NoError(t, err)
		assert.Equal(t, "rundll32", capturedName)
		assert.Equal(t, []string{"url.dll,FileProtocolHandler", testURL}, capturedArgs)
	})

	t.Run("unsupported platform", func(t *testing.T) {
		goos = "freebsd"
		err := openURLInternal(testURL)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported platform")
		assert.Contains(t, err.Error(), "freebsd")
	})
}

func TestOpenURL_Wrapper(t *testing.T) {
	// Save original and restore after test
	originalOpenURLFunc := openURLFunc
	defer func() { openURLFunc = originalOpenURLFunc }()

	// Test that OpenURL correctly calls openURLFunc
	called := false
	capturedURL := ""
	openURLFunc = func(rawURL string) error {
		called = true
		capturedURL = rawURL
		return nil
	}

	testURL := "https://test.example.com"
	err := OpenURL(testURL)
	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, testURL, capturedURL)
}

// ============================================================================
// ViewHTML Tests (with mocked OpenURL)
// ============================================================================

func TestViewHTML_Success(t *testing.T) {
	// Save original and restore after test
	originalOpenURLFunc := openURLFunc
	defer func() { openURLFunc = originalOpenURLFunc }()

	var capturedURL string
	openURLFunc = func(rawURL string) error {
		capturedURL = rawURL
		return nil
	}

	t.Run("creates temp file with correct content", func(t *testing.T) {
		testHTML := "<html><body><h1>Test Content</h1></body></html>"
		err := ViewHTML(testHTML)
		assert.NoError(t, err)

		// Verify URL was captured
		assert.Contains(t, capturedURL, "file://")
		assert.Contains(t, capturedURL, previewFilePrefix)
		assert.Contains(t, capturedURL, ".html")

		// Extract file path and verify content
		filePath := capturedURL[len("file://"):]
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, testHTML, string(content))

		// Verify permissions (should be 0600)
		info, err := os.Stat(filePath)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		// Cleanup
		os.Remove(filePath)
	})

	t.Run("handles empty HTML", func(t *testing.T) {
		err := ViewHTML("")
		assert.NoError(t, err)

		// Cleanup
		filePath := capturedURL[len("file://"):]
		os.Remove(filePath)
	})

	t.Run("handles large HTML", func(t *testing.T) {
		largeHTML := "<html><body>"
		for i := 0; i < 1000; i++ {
			largeHTML += "<p>Large content paragraph</p>"
		}
		largeHTML += "</body></html>"

		err := ViewHTML(largeHTML)
		assert.NoError(t, err)

		// Cleanup
		filePath := capturedURL[len("file://"):]
		os.Remove(filePath)
	})

	t.Run("handles special characters in HTML", func(t *testing.T) {
		specialHTML := "<html><body><p>Special chars: äöü αβγ 日本語 &amp; &lt;</p></body></html>"
		err := ViewHTML(specialHTML)
		assert.NoError(t, err)

		filePath := capturedURL[len("file://"):]
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, specialHTML, string(content))

		os.Remove(filePath)
	})
}

// mockTempFile implements TempFile for testing
type mockTempFile struct {
	name         string
	chmodErr     error
	writeErr     error
	closeErr     error
	closed       bool
	writtenData  string
}

func (m *mockTempFile) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockTempFile) Chmod(mode os.FileMode) error {
	return m.chmodErr
}

func (m *mockTempFile) WriteString(s string) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.writtenData = s
	return len(s), nil
}

func (m *mockTempFile) Name() string {
	return m.name
}

func TestCreateTempFileWrapper_ErrorPath(t *testing.T) {
	t.Run("default wrapper returns error when createTempFile fails", func(t *testing.T) {
		// Save original createTempFile and restore after test
		originalCreateTempFile := createTempFile
		defer func() { createTempFile = originalCreateTempFile }()

		// Mock createTempFile to return an error
		createTempFile = func(dir, pattern string) (*os.File, error) {
			return nil, errors.New("mock createTempFile error")
		}

		// Call the default wrapper directly - this exercises the error path
		// in the original createTempFileWrapper function
		_, err := createTempFileWrapper("", "test-*.html")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mock createTempFile error")
	})
}

func TestViewHTML_ErrorPaths(t *testing.T) {
	t.Run("returns error when CreateTemp fails", func(t *testing.T) {
		// Save original and restore after test
		originalWrapper := createTempFileWrapper
		defer func() { createTempFileWrapper = originalWrapper }()

		createTempFileWrapper = func(dir, pattern string) (TempFile, error) {
			return nil, errors.New("mock CreateTemp error")
		}

		err := ViewHTML("<html></html>")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create temp file")
	})

	t.Run("returns error when Chmod fails", func(t *testing.T) {
		// Save original and restore after test
		originalWrapper := createTempFileWrapper
		defer func() { createTempFileWrapper = originalWrapper }()

		createTempFileWrapper = func(dir, pattern string) (TempFile, error) {
			return &mockTempFile{
				name:     "/tmp/mock-file.html",
				chmodErr: errors.New("mock Chmod error"),
			}, nil
		}

		err := ViewHTML("<html></html>")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set file permissions")
	})

	t.Run("returns error when WriteString fails", func(t *testing.T) {
		// Save original and restore after test
		originalWrapper := createTempFileWrapper
		defer func() { createTempFileWrapper = originalWrapper }()

		createTempFileWrapper = func(dir, pattern string) (TempFile, error) {
			return &mockTempFile{
				name:     "/tmp/mock-file.html",
				writeErr: errors.New("mock WriteString error"),
			}, nil
		}

		err := ViewHTML("<html></html>")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write HTML")
	})
}

// ============================================================================
// ViewEmailHTML Tests (with mocked OpenURL)
// ============================================================================

func TestViewEmailHTML_Success(t *testing.T) {
	// Save original and restore after test
	originalOpenURLFunc := openURLFunc
	defer func() { openURLFunc = originalOpenURLFunc }()

	var capturedURL string
	openURLFunc = func(rawURL string) error {
		capturedURL = rawURL
		return nil
	}

	testTime := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

	t.Run("creates email preview with correct content", func(t *testing.T) {
		err := ViewEmailHTML("Test Subject", "sender@example.com", testTime, "<p>Email body</p>")
		assert.NoError(t, err)

		// Verify file content
		filePath := capturedURL[len("file://"):]
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)

		// Verify expected content in the generated HTML
		htmlContent := string(content)
		assert.Contains(t, htmlContent, "Test Subject")
		assert.Contains(t, htmlContent, "sender@example.com")
		assert.Contains(t, htmlContent, "June 15, 2024")
		assert.Contains(t, htmlContent, "<p>Email body</p>")
		assert.Contains(t, htmlContent, "#1cc2e3") // VaultSandbox brand color

		os.Remove(filePath)
	})

	t.Run("escapes XSS in subject and from", func(t *testing.T) {
		maliciousSubject := "<script>alert('xss')</script>"
		maliciousFrom := "<img onerror=alert(1)>"

		err := ViewEmailHTML(maliciousSubject, maliciousFrom, testTime, "<p>body</p>")
		assert.NoError(t, err)

		filePath := capturedURL[len("file://"):]
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err)

		htmlContent := string(content)
		assert.NotContains(t, htmlContent, "<script>alert")
		assert.NotContains(t, htmlContent, "<img onerror")
		assert.Contains(t, htmlContent, "&lt;script&gt;")

		os.Remove(filePath)
	})
}

// ============================================================================
// Additional CleanupPreviews Coverage Tests
// ============================================================================

func TestCleanupPreviews_EdgeCases(t *testing.T) {
	t.Run("handles files just at the threshold", func(t *testing.T) {
		tmpDir := os.TempDir()

		// Create a file exactly at the threshold
		thresholdFile := filepath.Join(tmpDir, previewFilePrefix+"test-threshold.html")
		require.NoError(t, os.WriteFile(thresholdFile, []byte("threshold"), 0600))

		// Set file modification time to exactly 1 hour ago
		exactTime := time.Now().Add(-1 * time.Hour)
		require.NoError(t, os.Chtimes(thresholdFile, exactTime, exactTime))

		// With 1 hour threshold, file at exactly 1 hour should NOT be deleted
		// (cutoff is files BEFORE the threshold time)
		err := CleanupPreviews(1 * time.Hour)
		assert.NoError(t, err)

		// Clean up manually
		os.Remove(thresholdFile)
	})

	t.Run("handles multiple preview files", func(t *testing.T) {
		tmpDir := os.TempDir()
		oldTime := time.Now().Add(-2 * time.Hour)

		// Create multiple old files
		var files []string
		for i := 0; i < 5; i++ {
			f := filepath.Join(tmpDir, previewFilePrefix+"multi-"+string(rune('a'+i))+".html")
			require.NoError(t, os.WriteFile(f, []byte("content"), 0600))
			require.NoError(t, os.Chtimes(f, oldTime, oldTime))
			files = append(files, f)
		}

		err := CleanupPreviews(1 * time.Hour)
		assert.NoError(t, err)

		// All should be removed
		for _, f := range files {
			_, err := os.Stat(f)
			assert.True(t, os.IsNotExist(err), "file %s should be removed", f)
		}
	})

	t.Run("preserves files with similar but different prefix", func(t *testing.T) {
		tmpDir := os.TempDir()

		// Create a file with a similar but different prefix
		similarFile := filepath.Join(tmpDir, "vsb-previewNOT-test.html")
		require.NoError(t, os.WriteFile(similarFile, []byte("similar"), 0600))

		oldTime := time.Now().Add(-2 * time.Hour)
		require.NoError(t, os.Chtimes(similarFile, oldTime, oldTime))

		err := CleanupPreviews(1 * time.Hour)
		assert.NoError(t, err)

		// File should NOT be removed (different prefix)
		_, err = os.Stat(similarFile)
		assert.NoError(t, err, "file with different prefix should not be removed")

		os.Remove(similarFile)
	})
}

func TestCleanupPreviews_ErrorPaths(t *testing.T) {
	t.Run("returns error when ReadDir fails", func(t *testing.T) {
		// Save original and restore after test
		originalReadDir := readDir
		defer func() { readDir = originalReadDir }()

		readDir = func(name string) ([]fs.DirEntry, error) {
			return nil, errors.New("mock ReadDir error")
		}

		err := CleanupPreviews(1 * time.Hour)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read temp directory")
	})

	t.Run("continues when entry.Info fails", func(t *testing.T) {
		// Save original and restore after test
		originalReadDir := readDir
		defer func() { readDir = originalReadDir }()

		// Create a mock DirEntry that returns error on Info()
		mockEntry := &mockDirEntry{
			name:    previewFilePrefix + "test.html",
			isDir:   false,
			infoErr: errors.New("mock Info error"),
		}

		readDir = func(name string) ([]fs.DirEntry, error) {
			return []fs.DirEntry{mockEntry}, nil
		}

		// Should not error, just continue past the problematic entry
		err := CleanupPreviews(1 * time.Hour)
		assert.NoError(t, err)
	})
}

// mockDirEntry implements fs.DirEntry for testing
type mockDirEntry struct {
	name    string
	isDir   bool
	infoErr error
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() fs.FileMode          { return 0 }
func (m *mockDirEntry) Info() (fs.FileInfo, error) { return nil, m.infoErr }
