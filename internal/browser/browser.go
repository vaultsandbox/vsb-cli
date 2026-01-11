package browser

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Allowed URL schemes for security
var allowedSchemes = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
	"file":   true,
}

// execCommand is a variable for exec.Command that can be overridden in tests
var execCommand = exec.Command

// openURLFunc is a variable for OpenURL that can be overridden in tests
var openURLFunc = openURLInternal

// goos is a variable for runtime.GOOS that can be overridden in tests
var goos = runtime.GOOS

// createTempFile is a variable for os.CreateTemp that can be overridden in tests
var createTempFile = os.CreateTemp

// readDir is a variable for os.ReadDir that can be overridden in tests
var readDir = os.ReadDir

// TempFile interface for testing file operations
type TempFile interface {
	Close() error
	Chmod(mode os.FileMode) error
	WriteString(s string) (int, error)
	Name() string
}

// osFile wraps *os.File to implement TempFile
type osFile struct {
	*os.File
}

// createTempFileWrapper wraps the result of createTempFile in our interface
var createTempFileWrapper = func(dir, pattern string) (TempFile, error) {
	f, err := createTempFile(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &osFile{f}, nil
}

// previewFilePrefix is used to identify temp files created by this package
const previewFilePrefix = "vsb-preview-"

// OpenURL opens a URL in the default browser.
// Only http, https, mailto, and file schemes are allowed.
func OpenURL(rawURL string) error {
	return openURLFunc(rawURL)
}

// openURLInternal is the actual implementation of OpenURL.
func openURLInternal(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if !allowedSchemes[scheme] {
		return fmt.Errorf("URL scheme %q not allowed", scheme)
	}

	var cmd *exec.Cmd

	switch goos {
	case "darwin":
		cmd = execCommand("open", rawURL)
	case "linux":
		cmd = execCommand("xdg-open", rawURL)
	case "windows":
		cmd = execCommand("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", goos)
	}

	return cmd.Start()
}

// ViewHTML writes HTML to a temp file and opens it in the browser.
// Uses secure temp file creation with restricted permissions.
func ViewHTML(html string) error {
	tmpFile, err := createTempFileWrapper("", previewFilePrefix+"*.html")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Set restrictive permissions (owner read/write only)
	if err := tmpFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	if _, err := tmpFile.WriteString(html); err != nil {
		return fmt.Errorf("failed to write HTML: %w", err)
	}

	return OpenURL("file://" + tmpFile.Name())
}

// BuildEmailHTMLTemplate generates the complete HTML for email preview.
// Exported for testing. Subject and from are escaped to prevent XSS.
func BuildEmailHTMLTemplate(subject, from string, receivedAt time.Time, emailHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .header {
            background: #1cc2e3;
            color: #0e0e0eff;
            padding: 20px;
            border-radius: 8px 8px 0 0;
        }
        .header h1 {
            margin: 0 0 10px 0;
            font-size: 1.2em;
        }
        .header .meta {
            font-size: 0.9em;
            opacity: 0.9;
        }
        .content {
            background: white;
            padding: 20px;
            border-radius: 0 0 8px 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>%s</h1>
        <div class="meta">
            <strong>From:</strong> %s<br>
            <strong>Date:</strong> %s
        </div>
    </div>
    <div class="content">
        %s
    </div>
</body>
</html>`,
		html.EscapeString(subject),
		html.EscapeString(subject),
		html.EscapeString(from),
		receivedAt.Format("January 2, 2006 at 3:04 PM"),
		emailHTML,
	)
}

// ViewEmailHTML wraps email HTML with styled template and opens in browser.
// Provides consistent styling with VaultSandbox branding across CLI and TUI.
func ViewEmailHTML(subject, from string, receivedAt time.Time, emailHTML string) error {
	return ViewHTML(BuildEmailHTMLTemplate(subject, from, receivedAt, emailHTML))
}

// CleanupPreviews removes old preview temp files older than the given duration.
func CleanupPreviews(olderThan time.Duration) error {
	tmpDir := os.TempDir()
	cutoff := time.Now().Add(-olderThan)

	entries, err := readDir(tmpDir)
	if err != nil {
		return fmt.Errorf("failed to read temp directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasPrefix(entry.Name(), previewFilePrefix) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			// Use filepath.Base to prevent path traversal
			os.Remove(filepath.Join(tmpDir, filepath.Base(entry.Name())))
		}
	}

	return nil
}
