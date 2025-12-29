//go:build e2e

// Package e2e contains end-to-end tests for vsb-cli.
// These tests require a running VaultSandbox Gateway and SMTP server.
//
// Required environment variables:
//   - VAULTSANDBOX_API_KEY: API key for authentication
//   - VAULTSANDBOX_URL: Gateway URL (e.g., https://api.vaultsandbox.com)
//   - SMTP_HOST: SMTP server host for sending test emails
//   - SMTP_PORT: SMTP server port (default: 25)
//
// Run with:
//
//	go build -o vsb ./cmd/vsb && go test -tags=e2e -v -timeout 10m ./e2e/...

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

var (
	apiKey     string
	baseURL    string
	vsbBinPath string // Absolute path to the vsb binary
)

func TestMain(m *testing.M) {
	// Load .env file if it exists (won't error if missing)
	if err := godotenv.Load("../.env"); err != nil {
		fmt.Fprintln(os.Stderr, "Note: .env file not found at project root")
	}

	apiKey = os.Getenv("VAULTSANDBOX_API_KEY")
	baseURL = os.Getenv("VAULTSANDBOX_URL")

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Skipping e2e tests: VAULTSANDBOX_API_KEY not set")
		os.Exit(0)
	}

	if baseURL == "" {
		fmt.Fprintln(os.Stderr, "Skipping e2e tests: VAULTSANDBOX_URL not set")
		os.Exit(0)
	}

	// Check that the vsb binary exists and get absolute path
	vsbBinPath, _ = filepath.Abs("../vsb")
	if _, err := os.Stat(vsbBinPath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "Error: vsb binary not found. Run 'go build -o vsb ./cmd/vsb' first")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Running e2e tests...")
	fmt.Fprintln(os.Stderr, "API URL:", baseURL)

	os.Exit(m.Run())
}

// ============================================================================
// CLI Execution Helpers
// ============================================================================

// runVSB executes the vsb CLI with given arguments and returns output.
// Each test gets an isolated config directory.
func runVSB(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	return runVSBWithConfig(t, t.TempDir(), args...)
}

// runVSBWithConfig executes the vsb CLI with a specific config directory.
func runVSBWithConfig(t *testing.T, configDir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(vsbBinPath, args...)
	cmd.Dir = configDir // Run from the config directory for relative paths

	// Build environment, converting GOCOVERDIR to absolute path relative to project root
	env := os.Environ()
	for i, e := range env {
		if strings.HasPrefix(e, "GOCOVERDIR=") {
			coverDir := strings.TrimPrefix(e, "GOCOVERDIR=")
			if !filepath.IsAbs(coverDir) {
				// Resolve relative to project root (where vsb binary is)
				projectRoot := filepath.Dir(vsbBinPath)
				env[i] = "GOCOVERDIR=" + filepath.Join(projectRoot, coverDir)
			}
			break
		}
	}

	cmd.Env = append(env,
		"VSB_API_KEY="+apiKey,
		"VSB_BASE_URL="+baseURL,
		"VSB_CONFIG_DIR="+configDir,
		"NO_COLOR=1", // Disable color output for easier parsing
	)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		// Other error (e.g., binary not found)
		t.Logf("exec error: %v", err)
		exitCode = -1
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// runVSBJSON executes vsb with --output json and parses the result.
func runVSBJSON[T any](t *testing.T, args ...string) T {
	t.Helper()
	args = append(args, "--output", "json")
	stdout, stderr, code := runVSB(t, args...)
	require.Equal(t, 0, code, "vsb command failed: stdout=%s, stderr=%s", stdout, stderr)

	var result T
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), "failed to parse JSON output: %s", stdout)
	return result
}

// runVSBJSONWithConfig executes vsb with a specific config dir and --output json.
func runVSBJSONWithConfig[T any](t *testing.T, configDir string, args ...string) T {
	t.Helper()
	args = append(args, "--output", "json")
	stdout, stderr, code := runVSBWithConfig(t, configDir, args...)
	require.Equal(t, 0, code, "vsb command failed: stdout=%s, stderr=%s", stdout, stderr)

	var result T
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), "failed to parse JSON output: %s", stdout)
	return result
}

// ============================================================================
// SMTP Utilities (adapted from client-go/integration)
// ============================================================================

// getSMTPConfig returns SMTP host and port from environment.
func getSMTPConfig() (host, port string) {
	host = os.Getenv("SMTP_HOST")
	port = os.Getenv("SMTP_PORT")
	if port == "" {
		port = "25"
	}
	return host, port
}

// skipIfNoSMTP skips the test if SMTP is not configured.
func skipIfNoSMTP(t *testing.T) {
	t.Helper()
	host, _ := getSMTPConfig()
	if host == "" {
		t.Skip("skipping: SMTP_HOST not set")
	}
}

// sendTestEmail sends a plain text test email via SMTP.
func sendTestEmail(t *testing.T, to, subject, body string) {
	t.Helper()
	skipIfNoSMTP(t)

	smtpHost, smtpPort := getSMTPConfig()
	from := "test@example.com"
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		from, to, subject, body)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	if err := smtp.SendMail(addr, nil, from, []string{to}, []byte(msg)); err != nil {
		t.Fatalf("sendTestEmail() error = %v", err)
	}
	t.Logf("Sent email to %s with subject: %s", to, subject)
}

// sendTestHTMLEmail sends a test email with HTML content via SMTP.
func sendTestHTMLEmail(t *testing.T, to, subject, textBody, htmlBody string) {
	t.Helper()
	skipIfNoSMTP(t)

	smtpHost, smtpPort := getSMTPConfig()
	from := "test@example.com"
	boundary := "boundary-example-12345"

	msg := fmt.Sprintf(`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="%s"

--%s
Content-Type: text/plain; charset=utf-8

%s

--%s
Content-Type: text/html; charset=utf-8

%s

--%s--
`, from, to, subject, boundary, boundary, textBody, boundary, htmlBody, boundary)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	if err := smtp.SendMail(addr, nil, from, []string{to}, []byte(msg)); err != nil {
		t.Fatalf("sendTestHTMLEmail() error = %v", err)
	}
	t.Logf("Sent HTML email to %s with subject: %s", to, subject)
}

// sendTestEmailWithAttachment sends a test email with an attachment via SMTP.
func sendTestEmailWithAttachment(t *testing.T, to, subject, body, attachmentName, attachmentContent string) {
	t.Helper()
	skipIfNoSMTP(t)

	smtpHost, smtpPort := getSMTPConfig()
	from := "test@example.com"
	boundary := "boundary-attachment-67890"

	msg := fmt.Sprintf(`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="%s"

--%s
Content-Type: text/plain; charset=utf-8

%s

--%s
Content-Type: application/octet-stream; name="%s"
Content-Disposition: attachment; filename="%s"
Content-Transfer-Encoding: base64

%s

--%s--
`, from, to, subject, boundary, boundary, body, boundary, attachmentName, attachmentName, attachmentContent, boundary)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	if err := smtp.SendMail(addr, nil, from, []string{to}, []byte(msg)); err != nil {
		t.Fatalf("sendTestEmailWithAttachment() error = %v", err)
	}
	t.Logf("Sent email with attachment to %s", to)
}

// ============================================================================
// Output Parsing Helpers
// ============================================================================

// extractEmail extracts an email address from CLI output.
// Looks for patterns like "abc123@domain.com"
func extractEmail(output string) string {
	re := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

// ============================================================================
// Test Inbox Types (for JSON parsing)
// ============================================================================

// InboxJSON represents the JSON output from inbox commands.
type InboxJSON struct {
	Email     string `json:"email"`
	InboxHash string `json:"inbox_hash"`
	ExpiresAt string `json:"expires_at"`
	IsActive  bool   `json:"is_active,omitempty"`
}

// InboxListJSON represents the JSON output from inbox list command.
type InboxListJSON struct {
	Inboxes []InboxJSON `json:"inboxes"`
	Count   int         `json:"count"`
}

// EmailJSON represents the JSON output from email commands.
type EmailJSON struct {
	ID          string   `json:"id"`
	From        string   `json:"from"`
	To          []string `json:"to"`
	Subject     string   `json:"subject"`
	Text        string   `json:"text,omitempty"`
	HTML        string   `json:"html,omitempty"`
	ReceivedAt  string   `json:"received_at"`
	Links       []string `json:"links,omitempty"`
	Attachments []struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		Size        int    `json:"size"`
	} `json:"attachments,omitempty"`
}

// EmailListJSON represents the JSON output from email list command.
type EmailListJSON struct {
	Emails []EmailJSON `json:"emails"`
	Count  int         `json:"count"`
}

// WaitResultJSON represents the JSON output from wait command.
type WaitResultJSON struct {
	Email EmailJSON `json:"email,omitempty"`
	Link  string    `json:"link,omitempty"`
}
