package emails

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/client-go/authresults"
)

// MockKeystore implements Keystore interface for testing
type MockKeystore struct {
	SaveInboxFunc func(exp *vaultsandbox.ExportedInbox) error
	SavedInboxes  []*vaultsandbox.ExportedInbox
}

func (m *MockKeystore) SaveInbox(exp *vaultsandbox.ExportedInbox) error {
	m.SavedInboxes = append(m.SavedInboxes, exp)
	if m.SaveInboxFunc != nil {
		return m.SaveInboxFunc(exp)
	}
	return nil
}

// testEmail creates a test email with the given parameters
func testEmail(id, subject, from string) *vaultsandbox.Email {
	return &vaultsandbox.Email{
		ID:         id,
		Subject:    subject,
		From:       from,
		To:         []string{"test@example.com"},
		ReceivedAt: time.Now(),
		Headers:    map[string]string{},
	}
}

// testEmailWithLinks creates a test email with links
func testEmailWithLinks(id, subject, from string, links []string) *vaultsandbox.Email {
	email := testEmail(id, subject, from)
	email.Links = links
	return email
}

// testEmailWithAttachments creates a test email with attachments
func testEmailWithAttachments(id, subject, from string, attachments []vaultsandbox.Attachment) *vaultsandbox.Email {
	email := testEmail(id, subject, from)
	email.Attachments = attachments
	return email
}

// testEmailWithAuth creates a test email with authentication results
func testEmailWithAuth(id, subject, from string, spfResult, dkimResult, dmarcResult string) *vaultsandbox.Email {
	email := testEmail(id, subject, from)
	email.AuthResults = &authresults.AuthResults{}

	if spfResult != "" {
		email.AuthResults.SPF = &authresults.SPFResult{
			Result: spfResult,
			Domain: "example.com",
		}
	}
	if dkimResult != "" {
		email.AuthResults.DKIM = []authresults.DKIMResult{
			{Result: dkimResult, Domain: "example.com", Selector: "default"},
		}
	}
	if dmarcResult != "" {
		email.AuthResults.DMARC = &authresults.DMARCResult{
			Result: dmarcResult,
			Policy: "reject",
		}
	}
	return email
}

// testEmailItem creates a test EmailItem
func testEmailItem(id, subject, from, inboxLabel string) EmailItem {
	return EmailItem{
		Email:      testEmail(id, subject, from),
		InboxLabel: inboxLabel,
	}
}

// testModel creates a Model with test data for list view testing
func testModel(emails []EmailItem) Model {
	delegate := list.NewDefaultDelegate()
	items := make([]list.Item, len(emails))
	for i, e := range emails {
		items[i] = e
	}

	l := list.New(items, delegate, 80, 20)
	l.SetShowStatusBar(false)

	ctx, cancel := context.WithCancel(context.Background())

	m := Model{
		list:      l,
		emails:    emails,
		connected: true,
		width:     80,
		height:    24,
		viewport:  viewport.New(76, 16),
		ctx:       ctx,
		cancel:    cancel,
	}
	return m
}

// testModelDetailView creates a Model in detail view mode
func testModelDetailView(email EmailItem) Model {
	m := testModel([]EmailItem{email})
	m.viewing = true
	m.viewedEmail = &email
	m.detailView = ViewContent
	m.viewport.SetContent(m.renderEmailDetail())
	return m
}
