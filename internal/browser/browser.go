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

// previewFilePrefix is used to identify temp files created by this package
const previewFilePrefix = "vsb-preview-"

// OpenURL opens a URL in the default browser.
// Only http, https, mailto, and file schemes are allowed.
func OpenURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if !allowedSchemes[scheme] {
		return fmt.Errorf("URL scheme %q not allowed", scheme)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// ViewHTML writes HTML to a temp file and opens it in the browser.
// Uses secure temp file creation with restricted permissions.
func ViewHTML(html string) error {
	tmpFile, err := os.CreateTemp("", previewFilePrefix+"*.html")
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

	entries, err := os.ReadDir(tmpDir)
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
			os.Remove(filepath.Join(tmpDir, entry.Name()))
		}
	}

	return nil
}
