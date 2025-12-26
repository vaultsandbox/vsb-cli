package emails

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	vaultsandbox "github.com/vaultsandbox/client-go"
)

func TestView(t *testing.T) {
	emails := []EmailItem{
		testEmailItem("1", "Test Subject", "sender@example.com", "inbox@test.com"),
	}

	t.Run("list view renders email list", func(t *testing.T) {
		m := testModel(emails)
		m.viewing = false

		output := m.View()
		// List should contain help text
		assert.Contains(t, output, "quit")
		assert.Contains(t, output, "view")
	})

	t.Run("detail view renders email detail", func(t *testing.T) {
		m := testModelDetailView(emails[0])

		output := m.View()
		// Should contain detail view help
		assert.Contains(t, output, "tabs")
		assert.Contains(t, output, "back")
	})

	t.Run("detail view returns empty for nil email", func(t *testing.T) {
		m := testModel(emails)
		m.viewing = true
		m.viewedEmail = nil

		output := m.viewDetail()
		assert.Empty(t, output)
	})
}

func TestViewList(t *testing.T) {
	t.Run("shows help text", func(t *testing.T) {
		m := testModel([]EmailItem{})

		output := m.viewList()
		assert.Contains(t, output, "q: quit")
		assert.Contains(t, output, "enter: view")
		assert.Contains(t, output, "o: open")
		assert.Contains(t, output, "v: html")
		assert.Contains(t, output, "d: delete")
		assert.Contains(t, output, "n: new")
	})
}

func TestRenderTabs(t *testing.T) {
	email := testEmailItem("1", "Test", "from@example.com", "inbox")

	t.Run("renders all tab names", func(t *testing.T) {
		m := testModelDetailView(email)

		output := m.renderTabs()
		assert.Contains(t, output, "Content")
		assert.Contains(t, output, "Security")
		assert.Contains(t, output, "Links")
		assert.Contains(t, output, "Attach")
		assert.Contains(t, output, "Raw")
	})

	t.Run("renders tab numbers", func(t *testing.T) {
		m := testModelDetailView(email)

		output := m.renderTabs()
		assert.Contains(t, output, "1")
		assert.Contains(t, output, "2")
		assert.Contains(t, output, "3")
		assert.Contains(t, output, "4")
		assert.Contains(t, output, "5")
	})
}

func TestRenderEmailDetail(t *testing.T) {
	t.Run("returns empty for nil email", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.viewing = true
		m.viewedEmail = nil

		output := m.renderEmailDetail()
		assert.Empty(t, output)
	})

	t.Run("includes from address", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "sender@example.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "From:")
		assert.Contains(t, output, "sender@example.com")
	})

	t.Run("includes to address", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "sender@example.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "To:")
		assert.Contains(t, output, "test@example.com")
	})

	t.Run("includes subject", func(t *testing.T) {
		email := testEmailItem("1", "Test Subject", "sender@example.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "Subject:")
		assert.Contains(t, output, "Test Subject")
	})

	t.Run("shows placeholder for empty subject", func(t *testing.T) {
		email := testEmailItem("1", "", "sender@example.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "(no subject)")
	})

	t.Run("includes date", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "sender@example.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "Date:")
	})

	t.Run("shows links count when present", func(t *testing.T) {
		email := EmailItem{
			Email:      testEmailWithLinks("1", "Subject", "from@x.com", []string{"http://a.com", "http://b.com"}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "Links:")
		assert.Contains(t, output, "2 found")
	})

	t.Run("shows attachments count when present", func(t *testing.T) {
		email := EmailItem{
			Email: testEmailWithAttachments("1", "Subject", "from@x.com", []vaultsandbox.Attachment{
				{Filename: "doc.pdf"},
				{Filename: "image.png"},
				{Filename: "file.txt"},
			}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "Attach:")
		assert.Contains(t, output, "3 files")
	})

	t.Run("includes body text", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.Text = "This is the email body content."
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "This is the email body content.")
	})

	t.Run("shows placeholder for empty body", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.Text = ""
		m := testModelDetailView(email)

		output := m.renderEmailDetail()
		assert.Contains(t, output, "(no text content)")
	})
}

func TestRenderSecurityView(t *testing.T) {
	t.Run("returns empty for nil email", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.viewing = true
		m.viewedEmail = nil

		output := m.renderSecurityView()
		assert.Empty(t, output)
	})

	t.Run("shows authentication section", func(t *testing.T) {
		email := EmailItem{
			Email:      testEmailWithAuth("1", "Subject", "from@x.com", "pass", "pass", "pass"),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)
		m.detailView = ViewSecurity

		output := m.renderSecurityView()
		assert.Contains(t, output, "AUTHENTICATION")
	})

	t.Run("shows transport security section", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderSecurityView()
		assert.Contains(t, output, "TRANSPORT SECURITY")
		assert.Contains(t, output, "TLS")
		assert.Contains(t, output, "E2E")
	})

	t.Run("shows security score section", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderSecurityView()
		assert.Contains(t, output, "SECURITY SCORE")
		assert.Contains(t, output, "/100")
	})

	t.Run("includes encryption label", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderSecurityView()
		assert.Contains(t, output, "ML-KEM-768")
	})
}

func TestRenderLinksView(t *testing.T) {
	t.Run("returns empty for nil email", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.viewing = true
		m.viewedEmail = nil

		output := m.renderLinksView()
		assert.Empty(t, output)
	})

	t.Run("shows message when no links", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderLinksView()
		assert.Contains(t, output, "No links found")
	})

	t.Run("lists all links", func(t *testing.T) {
		email := EmailItem{
			Email:      testEmailWithLinks("1", "Subject", "from@x.com", []string{"http://a.com", "http://b.com"}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderLinksView()
		assert.Contains(t, output, "http://a.com")
		assert.Contains(t, output, "http://b.com")
	})

	t.Run("shows link count", func(t *testing.T) {
		email := EmailItem{
			Email:      testEmailWithLinks("1", "Subject", "from@x.com", []string{"http://a.com", "http://b.com"}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderLinksView()
		assert.Contains(t, output, "Found 2 links")
	})

	t.Run("highlights selected link", func(t *testing.T) {
		email := EmailItem{
			Email:      testEmailWithLinks("1", "Subject", "from@x.com", []string{"http://a.com", "http://b.com"}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)
		m.detailView = ViewLinks
		m.selectedLink = 0

		output := m.renderLinksView()
		// Selected link should have ">" indicator
		assert.Contains(t, output, ">")
	})

	t.Run("shows navigation help", func(t *testing.T) {
		email := EmailItem{
			Email:      testEmailWithLinks("1", "Subject", "from@x.com", []string{"http://a.com"}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderLinksView()
		assert.Contains(t, output, "select")
		assert.Contains(t, output, "open")
	})
}

func TestRenderAttachmentsView(t *testing.T) {
	t.Run("returns empty for nil email", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.viewing = true
		m.viewedEmail = nil

		output := m.renderAttachmentsView()
		assert.Empty(t, output)
	})

	t.Run("shows message when no attachments", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		m := testModelDetailView(email)

		output := m.renderAttachmentsView()
		assert.Contains(t, output, "No attachments")
	})

	t.Run("lists all attachments", func(t *testing.T) {
		email := EmailItem{
			Email: testEmailWithAttachments("1", "Subject", "from@x.com", []vaultsandbox.Attachment{
				{Filename: "document.pdf", ContentType: "application/pdf", Size: 1024},
				{Filename: "image.png", ContentType: "image/png", Size: 2048},
			}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderAttachmentsView()
		assert.Contains(t, output, "document.pdf")
		assert.Contains(t, output, "image.png")
	})

	t.Run("shows attachment count", func(t *testing.T) {
		email := EmailItem{
			Email: testEmailWithAttachments("1", "Subject", "from@x.com", []vaultsandbox.Attachment{
				{Filename: "doc.pdf"},
				{Filename: "img.png"},
			}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderAttachmentsView()
		assert.Contains(t, output, "Found 2 attachments")
	})

	t.Run("shows content type and size", func(t *testing.T) {
		email := EmailItem{
			Email: testEmailWithAttachments("1", "Subject", "from@x.com", []vaultsandbox.Attachment{
				{Filename: "doc.pdf", ContentType: "application/pdf", Size: 1024},
			}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderAttachmentsView()
		assert.Contains(t, output, "application/pdf")
		assert.Contains(t, output, "1.0 kB")
	})

	t.Run("shows saved file indicator", func(t *testing.T) {
		email := EmailItem{
			Email: testEmailWithAttachments("1", "Subject", "from@x.com", []vaultsandbox.Attachment{
				{Filename: "doc.pdf"},
			}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)
		m.lastSavedFile = "/tmp/doc.pdf"

		output := m.renderAttachmentsView()
		assert.Contains(t, output, "Saved:")
		assert.Contains(t, output, "/tmp/doc.pdf")
	})

	t.Run("shows navigation help", func(t *testing.T) {
		email := EmailItem{
			Email: testEmailWithAttachments("1", "Subject", "from@x.com", []vaultsandbox.Attachment{
				{Filename: "doc.pdf"},
			}),
			InboxLabel: "inbox",
		}
		m := testModelDetailView(email)

		output := m.renderAttachmentsView()
		assert.Contains(t, output, "select")
		assert.Contains(t, output, "save")
	})
}

func TestRenderRawView(t *testing.T) {
	t.Run("returns empty for nil email", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.viewing = true
		m.viewedEmail = nil

		output := m.renderRawView()
		assert.Empty(t, output)
	})

	t.Run("shows raw headers section", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.Headers = map[string]string{
			"From":    "sender@example.com",
			"Subject": "Test Subject",
		}
		m := testModelDetailView(email)

		output := m.renderRawView()
		assert.Contains(t, output, "RAW HEADERS")
		assert.Contains(t, output, "From:")
		assert.Contains(t, output, "Subject:")
	})

	t.Run("shows message when no headers", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.Headers = nil
		m := testModelDetailView(email)

		output := m.renderRawView()
		assert.Contains(t, output, "No raw headers available")
	})

	t.Run("shows raw text body", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.Text = "This is the raw text body"
		m := testModelDetailView(email)

		output := m.renderRawView()
		assert.Contains(t, output, "RAW TEXT BODY")
		assert.Contains(t, output, "This is the raw text body")
	})

	t.Run("shows message when no text body", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.Text = ""
		m := testModelDetailView(email)

		output := m.renderRawView()
		assert.Contains(t, output, "No text body available")
	})

	t.Run("shows HTML body section", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.HTML = "<p>HTML content</p>"
		m := testModelDetailView(email)

		output := m.renderRawView()
		assert.Contains(t, output, "HTML BODY")
		assert.Contains(t, output, "<p>HTML content</p>")
	})

	t.Run("truncates long HTML", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		// Create HTML longer than 500 chars
		longHTML := "<html>" + string(make([]byte, 600)) + "</html>"
		email.Email.HTML = longHTML
		m := testModelDetailView(email)

		output := m.renderRawView()
		assert.Contains(t, output, "more bytes")
	})

	t.Run("shows message when no HTML body", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.HTML = ""
		m := testModelDetailView(email)

		output := m.renderRawView()
		assert.Contains(t, output, "No HTML body available")
	})

	t.Run("shows hint to view HTML in browser", func(t *testing.T) {
		email := testEmailItem("1", "Subject", "from@x.com", "inbox")
		email.Email.HTML = "<p>content</p>"
		m := testModelDetailView(email)

		output := m.renderRawView()
		assert.Contains(t, output, "Press 'v' to view full HTML in browser")
	})
}

func TestUpdateTitle(t *testing.T) {
	t.Run("shows Disconnected when not connected", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.connected = false

		m.updateTitle()
		assert.Equal(t, "Disconnected", m.list.Title)
	})

	t.Run("shows error message when lastError set", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.connected = true
		m.lastError = errors.New("test error message")

		m.updateTitle()
		assert.Contains(t, m.list.Title, "Error:")
		assert.Contains(t, m.list.Title, "test error message")
	})

	t.Run("shows No inboxes when empty", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.connected = true
		m.inboxes = nil

		m.updateTitle()
		assert.Equal(t, "No inboxes", m.list.Title)
	})
}
