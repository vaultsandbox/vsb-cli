package inbox

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

func TestFormatInboxInfoContent(t *testing.T) {
	now := time.Now()
	baseInbox := &config.StoredInbox{
		Email:     "test@example.com",
		ID:        "inbox-abc123",
		CreatedAt: now.Add(-24 * time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	}

	t.Run("active inbox shows ACTIVE badge", func(t *testing.T) {
		content := formatInboxInfoContent(baseInbox, true, false, 5, nil)

		assert.Contains(t, content, "test@example.com")
		assert.Contains(t, content, "ACTIVE")
		assert.Contains(t, content, "inbox-abc123")
	})

	t.Run("inactive inbox does not show ACTIVE badge", func(t *testing.T) {
		content := formatInboxInfoContent(baseInbox, false, false, 5, nil)

		assert.Contains(t, content, "test@example.com")
		assert.NotContains(t, content, "ACTIVE")
	})

	t.Run("expired inbox shows EXPIRED", func(t *testing.T) {
		expiredInbox := &config.StoredInbox{
			Email:     "expired@example.com",
			ID:        "inbox-expired",
			CreatedAt: now.Add(-72 * time.Hour),
			ExpiresAt: now.Add(-24 * time.Hour),
		}

		content := formatInboxInfoContent(expiredInbox, false, true, 0, nil)

		assert.Contains(t, content, "EXPIRED")
	})

	t.Run("non-expired inbox shows remaining time", func(t *testing.T) {
		content := formatInboxInfoContent(baseInbox, false, false, 5, nil)

		assert.Contains(t, content, "(1d)")
		assert.NotContains(t, content, "EXPIRED")
	})

	t.Run("sync error shows error message", func(t *testing.T) {
		syncErr := errors.New("connection failed")
		content := formatInboxInfoContent(baseInbox, false, false, 0, syncErr)

		assert.Contains(t, content, "(sync error)")
	})

	t.Run("no sync error shows email count", func(t *testing.T) {
		content := formatInboxInfoContent(baseInbox, false, false, 42, nil)

		assert.Contains(t, content, "42")
		assert.NotContains(t, content, "(sync error)")
	})

	t.Run("shows created date formatted", func(t *testing.T) {
		content := formatInboxInfoContent(baseInbox, false, false, 0, nil)

		expectedDate := baseInbox.CreatedAt.Format("2006-01-02 15:04")
		assert.Contains(t, content, expectedDate)
	})

	t.Run("shows expiry date when not expired", func(t *testing.T) {
		content := formatInboxInfoContent(baseInbox, false, false, 0, nil)

		expectedDate := baseInbox.ExpiresAt.Format("2006-01-02 15:04")
		assert.Contains(t, content, expectedDate)
	})

	t.Run("active and expired inbox shows both badges", func(t *testing.T) {
		expiredInbox := &config.StoredInbox{
			Email:     "active-expired@example.com",
			ID:        "inbox-ae",
			CreatedAt: now.Add(-72 * time.Hour),
			ExpiresAt: now.Add(-24 * time.Hour),
		}

		content := formatInboxInfoContent(expiredInbox, true, true, 0, nil)

		assert.Contains(t, content, "ACTIVE")
		assert.Contains(t, content, "EXPIRED")
	})

	t.Run("zero email count", func(t *testing.T) {
		content := formatInboxInfoContent(baseInbox, false, false, 0, nil)

		// Should show 0 emails, not sync error
		assert.NotContains(t, content, "(sync error)")
	})

	t.Run("short remaining time", func(t *testing.T) {
		shortInbox := &config.StoredInbox{
			Email:     "short@example.com",
			ID:        "inbox-short",
			CreatedAt: now.Add(-24 * time.Hour),
			ExpiresAt: now.Add(30 * time.Minute),
		}

		content := formatInboxInfoContent(shortInbox, false, false, 0, nil)

		assert.Contains(t, content, "(30m)")
	})
}
