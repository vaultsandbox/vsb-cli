package browser

import (
	"fmt"
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
