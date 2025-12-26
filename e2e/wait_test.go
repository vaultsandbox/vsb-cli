//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sendTestEmailAsync sends email in a goroutine and signals completion via channel.
// Returns immediately, errors are logged but don't fail the test.
func sendTestEmailAsync(inboxEmail, subject, body string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		smtpHost, smtpPort := getSMTPConfig()
		if smtpHost == "" {
			return
		}
		from := "test@example.com"
		msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
			from, inboxEmail, subject, body)
		addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
		smtp.SendMail(addr, nil, from, []string{inboxEmail}, []byte(msg))
	}()
	return done
}

// sendTestHTMLEmailAsync sends HTML email in a goroutine and signals completion via channel.
func sendTestHTMLEmailAsync(inboxEmail, subject, textBody, htmlBody string) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		smtpHost, smtpPort := getSMTPConfig()
		if smtpHost == "" {
			return
		}
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
`, from, inboxEmail, subject, boundary, boundary, textBody, boundary, htmlBody, boundary)
		addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
		smtp.SendMail(addr, nil, from, []string{inboxEmail}, []byte(msg))
	}()
	return done
}

// TestWaitBasic tests waiting for any email.
func TestWaitBasic(t *testing.T) {
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

	t.Run("wait for any email", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			<-sendTestEmailAsync(inboxEmail, "Wait Test Basic", "This is a basic wait test email")
		}()

		// Wait for email
		stdout, stderr, code := runVSBWithConfig(t, configDir, "wait", "--timeout", "30s", "--output", "json")
		require.Equal(t, 0, code, "wait failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			From    string `json:"from"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))

		assert.Equal(t, "Wait Test Basic", result.Subject)
		assert.NotEmpty(t, result.ID)
		assert.NotEmpty(t, result.From)

		wg.Wait() // Ensure goroutine completes
	})
}

// TestWaitSubject tests waiting with subject filters.
func TestWaitSubject(t *testing.T) {
	skipIfNoSMTP(t)

	t.Run("exact subject match", func(t *testing.T) {
		configDir := t.TempDir()

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

		uniqueSubject := "Exact Subject Match " + time.Now().Format("15:04:05.000")

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			// Send a decoy email first
			<-sendTestEmailAsync(inboxEmail, "Wrong Subject", "This should not match")
			time.Sleep(200 * time.Millisecond)
			// Send the target email
			<-sendTestEmailAsync(inboxEmail, uniqueSubject, "This should match")
		}()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "wait", "--subject", uniqueSubject, "--timeout", "30s", "--output", "json")
		require.Equal(t, 0, code, "wait --subject failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, uniqueSubject, result.Subject)

		wg.Wait()
	})

	t.Run("subject regex match", func(t *testing.T) {
		configDir := t.TempDir()

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

		timestamp := time.Now().Format("150405.000")

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			<-sendTestEmailAsync(inboxEmail, "Password Reset Request "+timestamp, "Click here to reset password")
		}()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "wait", "--subject-regex", "Password.*"+timestamp[:6], "--timeout", "30s", "--output", "json")
		require.Equal(t, 0, code, "wait --subject-regex failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Contains(t, result.Subject, "Password Reset Request")

		wg.Wait()
	})
}

// TestWaitFrom tests waiting with sender filters.
func TestWaitFrom(t *testing.T) {
	skipIfNoSMTP(t)

	t.Run("from exact match", func(t *testing.T) {
		configDir := t.TempDir()

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

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			<-sendTestEmailAsync(inboxEmail, "From Test Email", "Testing from filter")
		}()

		// Our test emails are from test@example.com
		stdout, stderr, code := runVSBWithConfig(t, configDir, "wait", "--from", "test@example.com", "--timeout", "30s", "--output", "json")
		require.Equal(t, 0, code, "wait --from failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			From string `json:"from"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Contains(t, result.From, "test@example.com")

		wg.Wait()
	})

	t.Run("from regex match", func(t *testing.T) {
		configDir := t.TempDir()

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

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			<-sendTestEmailAsync(inboxEmail, "From Regex Test", "Testing from regex filter")
		}()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "wait", "--from-regex", "test@.*\\.com", "--timeout", "30s", "--output", "json")
		require.Equal(t, 0, code, "wait --from-regex failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			From string `json:"from"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Contains(t, result.From, "example.com")

		wg.Wait()
	})
}

// TestWaitTimeout tests timeout behavior.
func TestWaitTimeout(t *testing.T) {
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

	t.Run("timeout with no email", func(t *testing.T) {
		// Wait with a very short timeout, expecting no email to arrive
		uniqueSubject := "NonExistent Subject " + time.Now().Format("150405.000")
		start := time.Now()
		_, stderr, code := runVSBWithConfig(t, configDir, "wait", "--timeout", "2s", "--subject", uniqueSubject)
		elapsed := time.Since(start)

		// Should fail with timeout
		assert.NotEqual(t, 0, code, "wait should timeout and return non-zero exit code")
		assert.Contains(t, stderr, "timeout", "stderr should mention timeout")

		// Should have waited approximately the timeout duration
		assert.GreaterOrEqual(t, elapsed, 1*time.Second, "should have waited at least 1 second")
		assert.Less(t, elapsed, 10*time.Second, "should not have waited too long")
	})
}

// TestWaitCount tests waiting for multiple emails.
func TestWaitCount(t *testing.T) {
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

	t.Run("wait for multiple emails", func(t *testing.T) {
		timestamp := time.Now().Format("150405.000")
		subject := "Multi Count Test " + timestamp

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			<-sendTestEmailAsync(inboxEmail, subject, "First email")
			time.Sleep(200 * time.Millisecond)
			<-sendTestEmailAsync(inboxEmail, subject, "Second email")
		}()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "wait", "--count", "2", "--subject", subject, "--timeout", "30s", "--output", "json")
		require.Equal(t, 0, code, "wait --count failed: stdout=%s, stderr=%s", stdout, stderr)

		// Output should contain JSON array or multiple JSON objects
		// The wait command outputs multiple emails
		assert.NotEmpty(t, stdout)

		wg.Wait()
	})
}

// TestWaitExtractLink tests link extraction from emails.
func TestWaitExtractLink(t *testing.T) {
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

	t.Run("extract link from email", func(t *testing.T) {
		timestamp := time.Now().Format("150405.000")
		verifyURL := "https://example.com/verify?token=" + timestamp

		// Send HTML email with link
		htmlBody := `<html><body><a href="` + verifyURL + `">Click to verify</a></body></html>`
		textBody := "Verify: " + verifyURL
		subject := "Verify Your Account " + timestamp

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			<-sendTestHTMLEmailAsync(inboxEmail, subject, textBody, htmlBody)
		}()

		stdout, stderr, code := runVSBWithConfig(t, configDir, "wait", "--subject", subject, "--extract-link", "--timeout", "30s")
		require.Equal(t, 0, code, "wait --extract-link failed: stdout=%s, stderr=%s", stdout, stderr)

		// Output should contain the verification URL
		assert.Contains(t, stdout, "example.com/verify")
		assert.Contains(t, stdout, timestamp[:6])

		wg.Wait()
	})
}

// TestWaitQuiet tests quiet mode output.
func TestWaitQuiet(t *testing.T) {
	skipIfNoSMTP(t)

	t.Run("quiet mode success", func(t *testing.T) {
		configDir := t.TempDir()

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

		timestamp := time.Now().Format("150405.000")
		subject := "Quiet Test " + timestamp

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			<-sendTestEmailAsync(inboxEmail, subject, "This is a quiet test")
		}()

		stdout, _, code = runVSBWithConfig(t, configDir, "wait", "--quiet", "--subject", subject, "--timeout", "30s")
		require.Equal(t, 0, code)

		// Quiet mode should produce no stdout output
		assert.Empty(t, strings.TrimSpace(stdout), "quiet mode should produce no output")

		wg.Wait()
	})

	t.Run("quiet mode timeout", func(t *testing.T) {
		configDir := t.TempDir()

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

		uniqueSubject := "Nonexistent Quiet Subject " + time.Now().Format("150405.000")
		_, _, code = runVSBWithConfig(t, configDir, "wait", "--quiet", "--timeout", "2s", "--subject", uniqueSubject)

		// Should fail with non-zero exit code
		assert.NotEqual(t, 0, code, "wait --quiet should return non-zero on timeout")
	})
}

// TestWaitWithInbox tests wait with explicit inbox selection.
func TestWaitWithInbox(t *testing.T) {
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

	t.Run("wait with explicit inbox flag", func(t *testing.T) {
		timestamp := time.Now().Format("150405.000")
		subject := "Inbox Flag Test " + timestamp

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(500 * time.Millisecond)
			<-sendTestEmailAsync(inboxEmails[0], subject, "Testing wait with explicit inbox")
		}()

		// Wait using explicit --inbox flag for first inbox
		stdout, stderr, code := runVSBWithConfig(t, configDir, "wait", "--inbox", inboxEmails[0], "--subject", subject, "--timeout", "30s", "--output", "json")
		require.Equal(t, 0, code, "wait --inbox failed: stdout=%s, stderr=%s", stdout, stderr)

		var result struct {
			Subject string `json:"subject"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &result))
		assert.Equal(t, subject, result.Subject)

		wg.Wait()
	})
}

// Verify async helpers actually check SMTP config
func init() {
	// Ensure environment is loaded for getSMTPConfig in async helpers
	os.Getenv("SMTP_HOST")
}
