//go:build e2e

package e2e

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmailList tests listing emails in an inbox.
func TestEmailList(t *testing.T) {
	skipIfNoSMTP(t)
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

	t.Run("empty inbox", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
		require.Equal(t, 0, code, "list failed: stdout=%s, stderr=%s", stdout, stderr)

		var result []interface{}
		err := json.Unmarshal([]byte(stdout), &result)
		if err == nil {
			assert.Empty(t, result)
		}
	})

	t.Run("with emails", func(t *testing.T) {
		// Send test emails
		sendTestEmail(t, inboxEmail, "Test Subject 1", "Test body 1")
		sendTestEmail(t, inboxEmail, "Test Subject 2", "Test body 2")

		// Wait for emails to be received
		time.Sleep(2 * time.Second)

		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
		require.Equal(t, 0, code, "list failed: stdout=%s, stderr=%s", stdout, stderr)

		var result []struct {
			ID         string `json:"id"`
			Subject    string `json:"subject"`
			From       string `json:"from"`
			ReceivedAt string `json:"receivedAt"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.GreaterOrEqual(t, len(result), 2)

		// Verify email fields
		for _, email := range result {
			assert.NotEmpty(t, email.ID)
			assert.NotEmpty(t, email.From)
			assert.NotEmpty(t, email.ReceivedAt)
		}

		// Verify subjects are present
		subjects := make(map[string]bool)
		for _, email := range result {
			subjects[email.Subject] = true
		}
		assert.True(t, subjects["Test Subject 1"] || subjects["Test Subject 2"],
			"at least one of our test emails should be found")
	})
}

// TestEmailView tests viewing email content.
func TestEmailView(t *testing.T) {
	skipIfNoSMTP(t)
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

	// Send test email
	testSubject := "View Test Email"
	testBody := "This is the test email body for viewing."
	sendTestEmail(t, inboxEmail, testSubject, testBody)

	// Wait for email
	time.Sleep(2 * time.Second)

	t.Run("view latest email JSON", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "view", "--output", "json")
		require.Equal(t, 0, code, "view failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			ID         string   `json:"id"`
			Subject    string   `json:"subject"`
			From       string   `json:"from"`
			To         string   `json:"to"`
			ReceivedAt string   `json:"receivedAt"`
			Text       string   `json:"text"`
			HTML       string   `json:"html"`
			Links      []string `json:"links"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, testSubject, result.Subject)
		assert.Contains(t, result.Text, testBody)
		assert.NotEmpty(t, result.ID)
	})

	t.Run("view specific email by ID", func(t *testing.T) {
		// First get the email ID from list
		stdout, _, code := runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
		require.Equal(t, 0, code)

		var emails []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &emails))
		require.NotEmpty(t, emails)

		emailID := emails[0].ID

		// View by ID
		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "view", emailID, "--output", "json")
		require.Equal(t, 0, code, "view by ID failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, emailID, result.ID)
	})

	t.Run("view text only", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "view", "--text")
		require.Equal(t, 0, code, "view --text failed: stdout=%s, stderr=%s", stdout, stderr)

		assert.Contains(t, stdout, testBody)
		assert.Contains(t, stdout, testSubject)
	})

	t.Run("view raw RFC 5322", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "view", "--raw")
		require.Equal(t, 0, code, "view --raw failed: stdout=%s, stderr=%s", stdout, stderr)

		// Raw email should contain headers
		assert.Contains(t, stdout, "Subject:")
		assert.Contains(t, stdout, "From:")
		assert.Contains(t, stdout, "To:")
	})
}

// TestEmailAudit tests email security auditing.
func TestEmailAudit(t *testing.T) {
	skipIfNoSMTP(t)
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

	// Send test email
	sendTestEmail(t, inboxEmail, "Audit Test", "Test body for audit")
	time.Sleep(2 * time.Second)

	t.Run("audit latest email", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "audit", "--output", "json")
		require.Equal(t, 0, code, "audit failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			ID            string `json:"id"`
			Subject       string `json:"subject"`
			From          string `json:"from"`
			To            []string `json:"to"`
			SecurityScore int    `json:"securityScore"`
			AuthResults   struct {
				SPF struct {
					Status string `json:"status"`
					Domain string `json:"domain"`
				} `json:"spf"`
				DKIM struct {
					Status   string `json:"status"`
					Selector string `json:"selector"`
					Domain   string `json:"domain"`
				} `json:"dkim"`
				DMARC struct {
					Status string `json:"status"`
					Policy string `json:"policy"`
				} `json:"dmarc"`
			} `json:"authResults"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, "Audit Test", result.Subject)
		assert.NotEmpty(t, result.ID)
		assert.GreaterOrEqual(t, result.SecurityScore, 0)
		assert.LessOrEqual(t, result.SecurityScore, 100)
	})
}

// TestEmailURL tests URL extraction from emails.
func TestEmailURL(t *testing.T) {
	skipIfNoSMTP(t)
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

	t.Run("extract URLs from HTML email", func(t *testing.T) {
		// Send HTML email with links
		htmlBody := `<html><body>
			<p>Click <a href="https://example.com/verify?token=abc123">here</a> to verify.</p>
			<p>Or visit <a href="https://example.org/signup">our signup page</a>.</p>
		</body></html>`
		textBody := "Click here to verify: https://example.com/verify?token=abc123"

		sendTestHTMLEmail(t, inboxEmail, "Email with URLs", textBody, htmlBody)
		time.Sleep(2 * time.Second)

		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "url", "--output", "json")
		require.Equal(t, 0, code, "url failed: stdout=%s, stderr=%s", stdout, stderr)

		var links []string
		require.NoError(t, json.Unmarshal([]byte(stdout), &links))

		// Should find at least one URL
		assert.NotEmpty(t, links)

		// Check for expected URLs
		foundVerify := false
		for _, link := range links {
			if strings.Contains(link, "example.com/verify") {
				foundVerify = true
			}
		}
		assert.True(t, foundVerify, "should find verify URL")
	})

	t.Run("no URLs in email", func(t *testing.T) {
		// Send plain email without links
		sendTestEmail(t, inboxEmail, "No Links Email", "This email has no links at all.")
		time.Sleep(2 * time.Second)

		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "url", "--output", "json")
		require.Equal(t, 0, code, "url failed: stdout=%s, stderr=%s", stdout, stderr)

		// Should return empty array
		assert.Equal(t, "[]\n", stdout)
	})
}

// TestEmailAttachment tests attachment listing and downloading.
func TestEmailAttachment(t *testing.T) {
	skipIfNoSMTP(t)
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

	t.Run("list attachments", func(t *testing.T) {
		// Send email with attachment
		attachmentContent := base64.StdEncoding.EncodeToString([]byte("Hello, this is a test file content!"))
		sendTestEmailWithAttachment(t, inboxEmail, "Email with Attachment", "See attached file.", "test.txt", attachmentContent)
		time.Sleep(2 * time.Second)

		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "attachment", "--output", "json")
		require.Equal(t, 0, code, "attachment failed: stdout=%s, stderr=%s", stdout, stderr)

		var result []struct {
			Index       int    `json:"index"`
			Filename    string `json:"filename"`
			ContentType string `json:"contentType"`
			Size        int    `json:"size"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.NotEmpty(t, result)
		assert.Equal(t, "test.txt", result[0].Filename)
		assert.Equal(t, 1, result[0].Index)
	})

	t.Run("download single attachment", func(t *testing.T) {
		downloadDir := t.TempDir()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "attachment", "--save", "1", "--dir", downloadDir)
		require.Equal(t, 0, code, "attachment --save failed: stdout=%s, stderr=%s", stdout, stderr)

		// Verify file was downloaded
		files, err := os.ReadDir(downloadDir)
		require.NoError(t, err)
		assert.NotEmpty(t, files)

		// Check that test.txt exists
		found := false
		for _, f := range files {
			if f.Name() == "test.txt" {
				found = true
				break
			}
		}
		assert.True(t, found, "test.txt should be downloaded")
	})

	t.Run("download all attachments", func(t *testing.T) {
		downloadDir := t.TempDir()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "attachment", "--all", "--dir", downloadDir)
		require.Equal(t, 0, code, "attachment --all failed: stdout=%s, stderr=%s", stdout, stderr)

		// Verify files were downloaded
		files, err := os.ReadDir(downloadDir)
		require.NoError(t, err)
		assert.NotEmpty(t, files)
	})

	t.Run("no attachments", func(t *testing.T) {
		// Send email without attachments
		sendTestEmail(t, inboxEmail, "No Attachments", "This email has no attachments.")
		time.Sleep(2 * time.Second)

		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "attachment", "--output", "json")
		require.Equal(t, 0, code, "attachment failed: stdout=%s, stderr=%s", stdout, stderr)

		// Should return empty array
		assert.Equal(t, "[]\n", stdout)
	})
}

// TestEmailDelete tests deleting emails.
func TestEmailDelete(t *testing.T) {
	skipIfNoSMTP(t)
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

	t.Run("delete email by ID", func(t *testing.T) {
		// Send test email
		sendTestEmail(t, inboxEmail, "Delete Test", "This email will be deleted.")
		time.Sleep(2 * time.Second)

		// Get email ID
		stdout, _, code := runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
		require.Equal(t, 0, code)

		var emails []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &emails))
		require.NotEmpty(t, emails)

		// Find the "Delete Test" email
		var emailID string
		for _, email := range emails {
			if email.Subject == "Delete Test" {
				emailID = email.ID
				break
			}
		}
		require.NotEmpty(t, emailID, "should find Delete Test email")

		// Delete the email
		_, stderr, code := runVSBWithConfig(t, configDir, "email", "delete", emailID)
		require.Equal(t, 0, code, "delete failed: stderr=%s", stderr)

		// Verify email is deleted
		stdout, _, code = runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
		require.Equal(t, 0, code)

		var remainingEmails []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		}
		json.Unmarshal([]byte(stdout), &remainingEmails)

		// Check that "Delete Test" is not in the list
		found := false
		for _, email := range remainingEmails {
			if email.Subject == "Delete Test" {
				found = true
				break
			}
		}
		assert.False(t, found, "Delete Test email should be deleted")
	})

	t.Run("error on non-existent ID", func(t *testing.T) {
		_, _, code := runVSBWithConfig(t, configDir, "email", "delete", "nonexistent-id-12345")
		assert.NotEqual(t, 0, code, "should fail for non-existent email ID")
	})
}

// TestEmailViewWithSpecificInbox tests viewing emails with --inbox flag.
func TestEmailViewWithSpecificInbox(t *testing.T) {
	skipIfNoSMTP(t)
	configDir := t.TempDir()

	// Create two inboxes
	var inboxEmails []string
	for i := 0; i < 2; i++ {
		stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create", "--output", "json")
		require.Equal(t, 0, code)

		var result struct {
			Email string `json:"email"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		inboxEmails = append(inboxEmails, result.Email)
	}

	t.Cleanup(func() {
		for _, email := range inboxEmails {
			runVSBWithConfig(t, configDir, "inbox", "delete", email)
		}
	})

	// Send email to first inbox
	sendTestEmail(t, inboxEmails[0], "Inbox 1 Email", "This is in inbox 1")
	time.Sleep(2 * time.Second)

	// Active inbox is the second one (last created)
	// But we should be able to view emails in the first inbox with --inbox flag
	t.Run("view with explicit inbox flag", func(t *testing.T) {
		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "list", "--inbox", inboxEmails[0], "--output", "json")
		require.Equal(t, 0, code, "list --inbox failed: stdout=%s, stderr=%s", stdout, stderr)

		var emails []struct {
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &emails))
		require.NotEmpty(t, emails)

		found := false
		for _, e := range emails {
			if e.Subject == "Inbox 1 Email" {
				found = true
				break
			}
		}
		assert.True(t, found, "should find email in first inbox")
	})

	t.Run("view with partial inbox match", func(t *testing.T) {
		// Use partial match for first inbox
		parts := strings.Split(inboxEmails[0], "@")
		partial := parts[0][:6]

		stdout, stderr, code := runVSBWithConfig(t, configDir, "email", "list", "--inbox", partial, "--output", "json")
		require.Equal(t, 0, code, "list --inbox partial failed: stdout=%s, stderr=%s", stdout, stderr)

		var emails []struct {
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &emails))
		require.NotEmpty(t, emails)

		found := false
		for _, e := range emails {
			if e.Subject == "Inbox 1 Email" {
				found = true
				break
			}
		}
		assert.True(t, found, "should find email with partial inbox match")
	})
}

// TestEmailWorkflow tests a complete email workflow.
func TestEmailWorkflow(t *testing.T) {
	skipIfNoSMTP(t)
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

	// Send a realistic verification email
	htmlBody := `<!DOCTYPE html>
<html>
<head><title>Verify Your Account</title></head>
<body>
	<h1>Welcome!</h1>
	<p>Please click the link below to verify your account:</p>
	<a href="https://example.com/verify?token=abc123xyz">Verify Account</a>
	<p>This link expires in 24 hours.</p>
</body>
</html>`
	textBody := "Welcome! Verify your account: https://example.com/verify?token=abc123xyz"

	sendTestHTMLEmail(t, inboxEmail, "Verify Your Account", textBody, htmlBody)
	time.Sleep(2 * time.Second)

	// Step 1: List emails and find our email
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var emails []struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emails))
	require.NotEmpty(t, emails)

	var emailID string
	for _, e := range emails {
		if e.Subject == "Verify Your Account" {
			emailID = e.ID
			break
		}
	}
	require.NotEmpty(t, emailID)

	// Step 2: View the email
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "view", emailID, "--output", "json")
	require.Equal(t, 0, code)

	var emailView struct {
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
		Links   []string `json:"links"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &emailView))
	assert.Equal(t, "Verify Your Account", emailView.Subject)

	// Step 3: Extract URLs
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "url", emailID, "--output", "json")
	require.Equal(t, 0, code)

	var links []string
	require.NoError(t, json.Unmarshal([]byte(stdout), &links))

	foundVerifyLink := false
	for _, link := range links {
		if strings.Contains(link, "verify?token=") {
			foundVerifyLink = true
			break
		}
	}
	assert.True(t, foundVerifyLink, "should find verification link")

	// Step 4: Run audit
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "audit", emailID, "--output", "json")
	require.Equal(t, 0, code)

	var audit struct {
		SecurityScore int `json:"securityScore"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &audit))
	assert.GreaterOrEqual(t, audit.SecurityScore, 0)

	// Step 5: Delete the email
	_, _, code = runVSBWithConfig(t, configDir, "email", "delete", emailID)
	require.Equal(t, 0, code)

	// Verify deletion
	stdout, _, code = runVSBWithConfig(t, configDir, "email", "list", "--output", "json")
	require.Equal(t, 0, code)

	var remainingEmails []struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(stdout), &remainingEmails)

	found := false
	for _, e := range remainingEmails {
		if e.ID == emailID {
			found = true
			break
		}
	}
	assert.False(t, found, "deleted email should not be in list")
}

// downloadDirHasFile checks if a directory contains a file with the given name.
func downloadDirHasFile(dir, filename string) bool {
	files, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, f := range files {
		if f.Name() == filename {
			return true
		}
	}
	return false
}

// getDownloadedFilePath returns the path to a downloaded file.
func getDownloadedFilePath(dir, filename string) string {
	return filepath.Join(dir, filename)
}
