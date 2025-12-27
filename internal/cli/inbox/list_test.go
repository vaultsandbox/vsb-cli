package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

func TestFilterInboxes(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	activeInbox := config.StoredInbox{Email: "active@example.com", ExpiresAt: future}
	expiredInbox := config.StoredInbox{Email: "expired@example.com", ExpiresAt: past}
	anotherActive := config.StoredInbox{Email: "another@example.com", ExpiresAt: future}

	t.Run("hides expired by default", func(t *testing.T) {
		inboxes := []config.StoredInbox{activeInbox, expiredInbox, anotherActive}
		result := filterInboxes(inboxes, false, now)

		assert.Len(t, result, 2)
		for _, inbox := range result {
			assert.NotEqual(t, "expired@example.com", inbox.Email)
		}
	})

	t.Run("shows expired with showExpired flag", func(t *testing.T) {
		inboxes := []config.StoredInbox{activeInbox, expiredInbox, anotherActive}
		result := filterInboxes(inboxes, true, now)

		assert.Len(t, result, 3)
		emails := make([]string, len(result))
		for i, inbox := range result {
			emails[i] = inbox.Email
		}
		assert.Contains(t, emails, "expired@example.com")
	})

	t.Run("returns empty for no inboxes", func(t *testing.T) {
		result := filterInboxes(nil, false, now)
		assert.Empty(t, result)

		result = filterInboxes([]config.StoredInbox{}, true, now)
		assert.Empty(t, result)
	})

	t.Run("returns empty when all expired and showExpired is false", func(t *testing.T) {
		inboxes := []config.StoredInbox{
			{Email: "old1@example.com", ExpiresAt: past},
			{Email: "old2@example.com", ExpiresAt: past},
		}
		result := filterInboxes(inboxes, false, now)
		assert.Empty(t, result)
	})

	t.Run("returns all when all active", func(t *testing.T) {
		inboxes := []config.StoredInbox{activeInbox, anotherActive}
		result := filterInboxes(inboxes, false, now)
		assert.Len(t, result, 2)
	})

	t.Run("preserves order", func(t *testing.T) {
		inboxes := []config.StoredInbox{anotherActive, activeInbox}
		result := filterInboxes(inboxes, false, now)

		assert.Equal(t, "another@example.com", result[0].Email)
		assert.Equal(t, "active@example.com", result[1].Email)
	})
}
