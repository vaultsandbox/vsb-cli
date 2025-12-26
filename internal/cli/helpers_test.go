package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// MockKeystore implements KeystoreReader for testing
type MockKeystore struct {
	inboxes     []config.StoredInbox
	activeEmail string

	// Function overrides for custom behavior
	GetActiveInboxFunc func() (*config.StoredInbox, error)
	FindInboxFunc      func(partial string) (*config.StoredInbox, []string, error)
	GetInboxFunc       func(email string) (*config.StoredInbox, error)
}

func (m *MockKeystore) GetActiveInbox() (*config.StoredInbox, error) {
	if m.GetActiveInboxFunc != nil {
		return m.GetActiveInboxFunc()
	}
	if m.activeEmail == "" {
		return nil, config.ErrNoActiveInbox
	}
	for i := range m.inboxes {
		if m.inboxes[i].Email == m.activeEmail {
			return &m.inboxes[i], nil
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
	for i := range m.inboxes {
		if m.inboxes[i].Email == partial {
			return &m.inboxes[i], nil, nil // Exact match
		}
		if strings.Contains(m.inboxes[i].Email, partial) {
			matches = append(matches, m.inboxes[i])
			matchEmails = append(matchEmails, m.inboxes[i].Email)
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
	for i := range m.inboxes {
		if m.inboxes[i].Email == email {
			return &m.inboxes[i], nil
		}
	}
	return nil, config.ErrInboxNotFound
}

func (m *MockKeystore) ListInboxes() []config.StoredInbox {
	return m.inboxes
}

func TestGetInbox(t *testing.T) {
	inbox1 := config.StoredInbox{Email: "test1@example.com"}
	inbox2 := config.StoredInbox{Email: "test2@example.com"}

	t.Run("empty flag returns active inbox", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes:     []config.StoredInbox{inbox1, inbox2},
			activeEmail: "test1@example.com",
		}

		result, err := GetInbox(ks, "")
		require.NoError(t, err)
		assert.Equal(t, "test1@example.com", result.Email)
	})

	t.Run("empty flag with no active returns error", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes:     []config.StoredInbox{inbox1},
			activeEmail: "",
		}

		_, err := GetInbox(ks, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active inbox")
	})

	t.Run("exact email match", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes: []config.StoredInbox{inbox1, inbox2},
		}

		result, err := GetInbox(ks, "test2@example.com")
		require.NoError(t, err)
		assert.Equal(t, "test2@example.com", result.Email)
	})

	t.Run("partial match", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes: []config.StoredInbox{
				{Email: "unique123@example.com"},
			},
		}

		result, err := GetInbox(ks, "unique123")
		require.NoError(t, err)
		assert.Equal(t, "unique123@example.com", result.Email)
	})

	t.Run("multiple matches returns error", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes: []config.StoredInbox{inbox1, inbox2},
		}

		_, err := GetInbox(ks, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "multiple inboxes match")
	})

	t.Run("no match returns error", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes: []config.StoredInbox{inbox1},
		}

		_, err := GetInbox(ks, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestMockKeystoreGetActiveInbox(t *testing.T) {
	t.Run("returns active inbox when set", func(t *testing.T) {
		inbox := config.StoredInbox{Email: "active@example.com"}
		ks := &MockKeystore{
			inboxes:     []config.StoredInbox{inbox},
			activeEmail: "active@example.com",
		}

		result, err := ks.GetActiveInbox()
		require.NoError(t, err)
		assert.Equal(t, "active@example.com", result.Email)
	})

	t.Run("returns error when no active", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes:     []config.StoredInbox{{Email: "test@example.com"}},
			activeEmail: "",
		}

		_, err := ks.GetActiveInbox()
		assert.ErrorIs(t, err, config.ErrNoActiveInbox)
	})

	t.Run("uses override function when provided", func(t *testing.T) {
		customInbox := &config.StoredInbox{Email: "custom@example.com"}
		ks := &MockKeystore{
			GetActiveInboxFunc: func() (*config.StoredInbox, error) {
				return customInbox, nil
			},
		}

		result, err := ks.GetActiveInbox()
		require.NoError(t, err)
		assert.Equal(t, "custom@example.com", result.Email)
	})
}

func TestMockKeystoreFindInbox(t *testing.T) {
	t.Run("exact match returns immediately", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes: []config.StoredInbox{
				{Email: "exact@example.com"},
				{Email: "exactmore@example.com"},
			},
		}

		result, matches, err := ks.FindInbox("exact@example.com")
		require.NoError(t, err)
		assert.Nil(t, matches)
		assert.Equal(t, "exact@example.com", result.Email)
	})

	t.Run("returns multiple match emails", func(t *testing.T) {
		ks := &MockKeystore{
			inboxes: []config.StoredInbox{
				{Email: "test1@example.com"},
				{Email: "test2@example.com"},
			},
		}

		_, matches, err := ks.FindInbox("test")
		assert.ErrorIs(t, err, config.ErrMultipleMatches)
		assert.Contains(t, matches, "test1@example.com")
		assert.Contains(t, matches, "test2@example.com")
	})
}

func TestMockKeystoreListInboxes(t *testing.T) {
	inboxes := []config.StoredInbox{
		{Email: "inbox1@example.com"},
		{Email: "inbox2@example.com"},
	}
	ks := &MockKeystore{inboxes: inboxes}

	result := ks.ListInboxes()
	assert.Len(t, result, 2)
	assert.Equal(t, "inbox1@example.com", result[0].Email)
	assert.Equal(t, "inbox2@example.com", result[1].Email)
}
