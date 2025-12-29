package cliutil

import (
	"strings"

	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// MockKeystore implements KeystoreReader and SetActiveInbox for testing
type MockKeystore struct {
	Inboxes     []config.StoredInbox
	ActiveEmail string

	// Function overrides for custom behavior
	GetActiveInboxFunc func() (*config.StoredInbox, error)
	FindInboxFunc      func(partial string) (*config.StoredInbox, []string, error)
	GetInboxFunc       func(email string) (*config.StoredInbox, error)
	SetActiveInboxFunc func(email string) error
}

func (m *MockKeystore) GetActiveInbox() (*config.StoredInbox, error) {
	if m.GetActiveInboxFunc != nil {
		return m.GetActiveInboxFunc()
	}
	if m.ActiveEmail == "" {
		return nil, config.ErrNoActiveInbox
	}
	for i := range m.Inboxes {
		if m.Inboxes[i].Email == m.ActiveEmail {
			return &m.Inboxes[i], nil
		}
	}
	return nil, config.ErrNoActiveInbox
}

func (m *MockKeystore) FindInbox(partial string) (*config.StoredInbox, []string, error) {
	if m.FindInboxFunc != nil {
		return m.FindInboxFunc(partial)
	}
	var matches []config.StoredInbox
	var matchEmails []string
	for i := range m.Inboxes {
		if m.Inboxes[i].Email == partial {
			return &m.Inboxes[i], nil, nil // Exact match
		}
		if strings.Contains(m.Inboxes[i].Email, partial) {
			matches = append(matches, m.Inboxes[i])
			matchEmails = append(matchEmails, m.Inboxes[i].Email)
		}
	}
	if len(matches) == 0 {
		return nil, nil, config.ErrInboxNotFound
	}
	if len(matches) > 1 {
		return nil, matchEmails, config.ErrMultipleMatches
	}
	return &matches[0], nil, nil
}

func (m *MockKeystore) GetInbox(email string) (*config.StoredInbox, error) {
	if m.GetInboxFunc != nil {
		return m.GetInboxFunc(email)
	}
	for i := range m.Inboxes {
		if m.Inboxes[i].Email == email {
			return &m.Inboxes[i], nil
		}
	}
	return nil, config.ErrInboxNotFound
}

func (m *MockKeystore) ListInboxes() []config.StoredInbox {
	return m.Inboxes
}

func (m *MockKeystore) SetActiveInbox(email string) error {
	if m.SetActiveInboxFunc != nil {
		return m.SetActiveInboxFunc(email)
	}
	// Verify inbox exists
	for _, inbox := range m.Inboxes {
		if inbox.Email == email {
			m.ActiveEmail = email
			return nil
		}
	}
	return config.ErrInboxNotFound
}
