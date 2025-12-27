package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vaultsandbox "github.com/vaultsandbox/client-go"
)

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
