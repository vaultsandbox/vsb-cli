package watch

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return nil
	}

	return cmd.Start()
}

// viewInBrowser writes HTML to a temp file and opens it
func viewInBrowser(html string) error {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "vsb-preview.html")

	if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
		return err
	}

	return openBrowser("file://" + tmpFile)
}
