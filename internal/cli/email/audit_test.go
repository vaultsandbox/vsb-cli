package email

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/client-go/authresults"
)

// captureStdout captures stdout during function execution
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

func TestBuildMIMETree(t *testing.T) {
	t.Run("simple text email", func(t *testing.T) {
		email := &vaultsandbox.Email{
			Text: "Plain text content",
		}

		tree := buildMIMETree(email)

		assert.Contains(t, tree, "message/rfc822")
		assert.Contains(t, tree, "headers")
		assert.Contains(t, tree, "body")
		assert.Contains(t, tree, "text/plain")
	})

	t.Run("simple html email", func(t *testing.T) {
		email := &vaultsandbox.Email{
			HTML: "<p>HTML content</p>",
		}

		tree := buildMIMETree(email)

		assert.Contains(t, tree, "message/rfc822")
		assert.Contains(t, tree, "body")
		assert.Contains(t, tree, "text/html")
		assert.NotContains(t, tree, "text/plain")
	})

	t.Run("multipart alternative email", func(t *testing.T) {
		email := &vaultsandbox.Email{
			Text: "Plain text version",
			HTML: "<p>HTML version</p>",
		}

		tree := buildMIMETree(email)

		assert.Contains(t, tree, "message/rfc822")
		assert.Contains(t, tree, "body")
		assert.Contains(t, tree, "text/plain")
		assert.Contains(t, tree, "text/html")
	})

	t.Run("email with single attachment", func(t *testing.T) {
		email := &vaultsandbox.Email{
			Text: "Message with attachment",
			Attachments: []vaultsandbox.Attachment{
				{Filename: "doc.pdf", ContentType: "application/pdf", Size: 1024},
			},
		}

		tree := buildMIMETree(email)

		assert.Contains(t, tree, "attachments")
		assert.Contains(t, tree, "application/pdf")
		assert.Contains(t, tree, "doc.pdf")
		assert.Contains(t, tree, "1024 bytes")
	})

	t.Run("email with multiple attachments", func(t *testing.T) {
		email := &vaultsandbox.Email{
			HTML: "<p>Message</p>",
			Attachments: []vaultsandbox.Attachment{
				{Filename: "doc.pdf", ContentType: "application/pdf", Size: 1024},
				{Filename: "image.png", ContentType: "image/png", Size: 2048},
				{Filename: "data.csv", ContentType: "text/csv", Size: 512},
			},
		}

		tree := buildMIMETree(email)

		assert.Contains(t, tree, "attachments")
		assert.Contains(t, tree, "application/pdf")
		assert.Contains(t, tree, "image/png")
		assert.Contains(t, tree, "text/csv")
		assert.Contains(t, tree, "doc.pdf")
		assert.Contains(t, tree, "image.png")
		assert.Contains(t, tree, "data.csv")
	})

	t.Run("empty email shows headers only", func(t *testing.T) {
		email := &vaultsandbox.Email{}

		tree := buildMIMETree(email)

		assert.Contains(t, tree, "message/rfc822")
		assert.Contains(t, tree, "headers")
		assert.Contains(t, tree, "From")
		assert.Contains(t, tree, "To")
		assert.Contains(t, tree, "Subject")
		assert.Contains(t, tree, "Date")
		assert.Contains(t, tree, "Message-ID")
	})

	t.Run("tree structure includes correct prefixes", func(t *testing.T) {
		email := &vaultsandbox.Email{
			Text: "Text",
			HTML: "HTML",
			Attachments: []vaultsandbox.Attachment{
				{Filename: "file.txt", ContentType: "text/plain", Size: 100},
			},
		}

		tree := buildMIMETree(email)

		// Check tree structure characters are present
		assert.Contains(t, tree, "├──")
		assert.Contains(t, tree, "└──")
		assert.Contains(t, tree, "│")
	})

	t.Run("attachments only (no body)", func(t *testing.T) {
		email := &vaultsandbox.Email{
			Attachments: []vaultsandbox.Attachment{
				{Filename: "file.bin", ContentType: "application/octet-stream", Size: 4096},
			},
		}

		tree := buildMIMETree(email)

		assert.Contains(t, tree, "attachments")
		assert.Contains(t, tree, "file.bin")
		assert.NotContains(t, tree, "body")
	})
}

func TestRenderAuditReport(t *testing.T) {
	t.Run("renders basic email info", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "test-id-123",
			Subject:    "Test Subject",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Text:       "Hello, world!",
		}

		output := captureStdout(t, func() {
			err := renderAuditReport(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "EMAIL AUDIT REPORT")
		assert.Contains(t, output, "BASIC INFO")
		assert.Contains(t, output, "Test Subject")
		assert.Contains(t, output, "sender@example.com")
		assert.Contains(t, output, "recipient@example.com")
	})

	t.Run("renders authentication results with all passing", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "auth-test-id",
			Subject:    "Auth Test",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Content",
			AuthResults: &authresults.AuthResults{
				SPF: &authresults.SPFResult{
					Result: "pass",
					Domain: "example.com",
				},
				DKIM: []authresults.DKIMResult{
					{Result: "pass", Selector: "default", Domain: "example.com"},
				},
				DMARC: &authresults.DMARCResult{
					Result: "pass",
					Policy: "reject",
				},
				ReverseDNS: &authresults.ReverseDNSResult{
					Verified: true,
					Hostname: "mail.example.com",
				},
			},
		}

		output := captureStdout(t, func() {
			err := renderAuditReport(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "AUTHENTICATION")
		assert.Contains(t, output, "SPF")
		assert.Contains(t, output, "DKIM")
		assert.Contains(t, output, "DMARC")
		assert.Contains(t, output, "Security Score")
		assert.Contains(t, output, "100/100") // All auth passing = 100
	})

	t.Run("renders authentication results with failures", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "fail-auth-id",
			Subject:    "Failed Auth",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Content",
			AuthResults: &authresults.AuthResults{
				SPF: &authresults.SPFResult{
					Result: "fail",
					Domain: "example.com",
				},
				DKIM: []authresults.DKIMResult{
					{Result: "fail", Domain: "example.com"},
				},
				DMARC: &authresults.DMARCResult{
					Result: "fail",
					Policy: "none",
				},
			},
		}

		output := captureStdout(t, func() {
			err := renderAuditReport(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "AUTHENTICATION")
		assert.Contains(t, output, "Security Score")
		assert.Contains(t, output, "50/100") // No auth passing = base 50
	})

	t.Run("renders transport security with TLS info", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "tls-test-id",
			Subject:    "TLS Test",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Content",
			Headers: map[string]string{
				"received": "from mail.example.com (version=TLSv1.3 cipher=TLS_AES_256_GCM_SHA384)",
			},
		}

		output := captureStdout(t, func() {
			err := renderAuditReport(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "TRANSPORT SECURITY")
		assert.Contains(t, output, "TLSv1.3")
		assert.Contains(t, output, "TLS_AES_256_GCM_SHA384")
	})

	t.Run("renders transport security without TLS info", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "no-tls-id",
			Subject:    "No TLS",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Content",
		}

		output := captureStdout(t, func() {
			err := renderAuditReport(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "TRANSPORT SECURITY")
		assert.Contains(t, output, "unknown")
	})

	t.Run("renders MIME structure section", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "mime-test-id",
			Subject:    "MIME Test",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Plain text",
			HTML:       "<p>HTML content</p>",
			Attachments: []vaultsandbox.Attachment{
				{Filename: "doc.pdf", ContentType: "application/pdf", Size: 1024},
			},
		}

		output := captureStdout(t, func() {
			err := renderAuditReport(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "MIME STRUCTURE")
		assert.Contains(t, output, "message/rfc822")
		assert.Contains(t, output, "text/plain")
		assert.Contains(t, output, "text/html")
		assert.Contains(t, output, "doc.pdf")
	})

	t.Run("renders email with no auth results", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "no-auth-id",
			Subject:    "No Auth",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Content",
		}

		output := captureStdout(t, func() {
			err := renderAuditReport(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "AUTHENTICATION")
		assert.Contains(t, output, "No authentication results available")
	})

	t.Run("renders multiple recipients", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "multi-rcpt-id",
			Subject:    "Multiple Recipients",
			From:       "sender@example.com",
			To:         []string{"alice@example.com", "bob@example.com", "carol@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Hello all",
		}

		output := captureStdout(t, func() {
			err := renderAuditReport(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, "alice@example.com")
		assert.Contains(t, output, "bob@example.com")
		assert.Contains(t, output, "carol@example.com")
	})
}

func TestRenderAuditJSON(t *testing.T) {
	t.Run("outputs valid JSON for basic email", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "json-test-id",
			Subject:    "JSON Test",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Text:       "Content",
		}

		output := captureStdout(t, func() {
			err := renderAuditJSON(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, `"id": "json-test-id"`)
		assert.Contains(t, output, `"subject": "JSON Test"`)
		assert.Contains(t, output, `"from": "sender@example.com"`)
		assert.Contains(t, output, `"to"`)
		assert.Contains(t, output, `"securityScore"`)
	})

	t.Run("outputs auth results in JSON", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "json-auth-id",
			Subject:    "JSON Auth Test",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Content",
			AuthResults: &authresults.AuthResults{
				SPF: &authresults.SPFResult{
					Result: "pass",
					Domain: "example.com",
				},
				DKIM: []authresults.DKIMResult{
					{Result: "pass", Selector: "default", Domain: "example.com"},
				},
				DMARC: &authresults.DMARCResult{
					Result: "pass",
					Policy: "reject",
				},
			},
		}

		output := captureStdout(t, func() {
			err := renderAuditJSON(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, `"authResults"`)
		assert.Contains(t, output, `"spf"`)
		assert.Contains(t, output, `"dkim"`)
		assert.Contains(t, output, `"dmarc"`)
		// Score: 50 base + 15 SPF + 20 DKIM + 10 DMARC = 95 (no ReverseDNS)
		assert.Contains(t, output, `"securityScore": 95`)
	})

	t.Run("outputs security score in JSON", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "score-test-id",
			Subject:    "Score Test",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Content",
		}

		output := captureStdout(t, func() {
			err := renderAuditJSON(email)
			require.NoError(t, err)
		})

		// Base score of 50 when no auth results
		assert.Contains(t, output, `"securityScore": 50`)
	})

	t.Run("outputs partial auth results", func(t *testing.T) {
		email := &vaultsandbox.Email{
			ID:         "partial-auth-id",
			Subject:    "Partial Auth",
			From:       "sender@example.com",
			To:         []string{"recipient@example.com"},
			ReceivedAt: time.Now(),
			Text:       "Content",
			AuthResults: &authresults.AuthResults{
				SPF: &authresults.SPFResult{
					Result: "pass",
					Domain: "example.com",
				},
				// No DKIM or DMARC
			},
		}

		output := captureStdout(t, func() {
			err := renderAuditJSON(email)
			require.NoError(t, err)
		})

		assert.Contains(t, output, `"authResults"`)
		assert.Contains(t, output, `"spf"`)
		// Score should be 50 + 15 = 65
		assert.Contains(t, output, `"securityScore": 65`)
	})
}
