package cliutil

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

func TestEmailSummaryJSON(t *testing.T) {
	now := time.Now()
	email := &vaultsandbox.Email{
		ID:         "msg-123",
		From:       "sender@example.com",
		To:         []string{"recipient@test.com"},
		Subject:    "Test Subject",
		ReceivedAt: now,
	}

	result := EmailSummaryJSON(email)

	assert.Equal(t, "msg-123", result["id"])
	assert.Equal(t, "sender@example.com", result["from"])
	assert.Equal(t, "Test Subject", result["subject"])
	assert.Equal(t, now.Format(time.RFC3339), result["receivedAt"])
}

func TestEmailSummaryJSON_EmptyFields(t *testing.T) {
	email := &vaultsandbox.Email{}

	result := EmailSummaryJSON(email)

	assert.Equal(t, "", result["id"])
	assert.Equal(t, "", result["from"])
	assert.Equal(t, "", result["subject"])
}

func TestEmailFullJSON(t *testing.T) {
	now := time.Now()
	email := &vaultsandbox.Email{
		ID:         "msg-456",
		From:       "sender@example.com",
		To:         []string{"recipient1@test.com", "recipient2@test.com"},
		Subject:    "Full Test",
		Text:       "Plain text content",
		HTML:       "<p>HTML content</p>",
		ReceivedAt: now,
		Links:      []string{"https://example.com", "https://test.com"},
		Headers:    map[string]string{"X-Custom": "value"},
	}

	result := EmailFullJSON(email)

	assert.Equal(t, "msg-456", result["id"])
	assert.Equal(t, "sender@example.com", result["from"])
	assert.Equal(t, "recipient1@test.com, recipient2@test.com", result["to"])
	assert.Equal(t, "Full Test", result["subject"])
	assert.Equal(t, "Plain text content", result["text"])
	assert.Equal(t, "<p>HTML content</p>", result["html"])
	assert.Equal(t, now.Format(time.RFC3339), result["receivedAt"])
	assert.Equal(t, []string{"https://example.com", "https://test.com"}, result["links"])
	assert.Equal(t, map[string]string{"X-Custom": "value"}, result["headers"])
}

func TestEmailFullJSON_EmptyTo(t *testing.T) {
	email := &vaultsandbox.Email{
		ID: "msg-789",
		To: []string{},
	}

	result := EmailFullJSON(email)

	assert.Equal(t, "", result["to"])
}

func TestEmailFullJSON_SingleRecipient(t *testing.T) {
	email := &vaultsandbox.Email{
		ID: "msg-abc",
		To: []string{"single@test.com"},
	}

	result := EmailFullJSON(email)

	assert.Equal(t, "single@test.com", result["to"])
}

func TestInboxSummaryJSON(t *testing.T) {
	now := time.Now()
	inbox := &config.StoredInbox{
		Email:     "test@example.com",
		ExpiresAt: now.Add(24 * time.Hour),
	}

	t.Run("active inbox not expired", func(t *testing.T) {
		result := InboxSummaryJSON(inbox, true, now)

		assert.Equal(t, "test@example.com", result["email"])
		assert.Equal(t, inbox.ExpiresAt.Format(time.RFC3339), result["expiresAt"])
		assert.Equal(t, true, result["isActive"])
		assert.Equal(t, false, result["isExpired"])
	})

	t.Run("inactive inbox", func(t *testing.T) {
		result := InboxSummaryJSON(inbox, false, now)

		assert.Equal(t, false, result["isActive"])
	})

	t.Run("expired inbox", func(t *testing.T) {
		expiredInbox := &config.StoredInbox{
			Email:     "expired@example.com",
			ExpiresAt: now.Add(-1 * time.Hour),
		}
		result := InboxSummaryJSON(expiredInbox, false, now)

		assert.Equal(t, true, result["isExpired"])
	})
}

func TestInboxFullJSON(t *testing.T) {
	now := time.Now()
	inbox := &config.StoredInbox{
		Email:     "full@example.com",
		ID:        "inbox-hash-123",
		CreatedAt: now.Add(-48 * time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	}

	t.Run("full inbox info without error", func(t *testing.T) {
		result := InboxFullJSON(inbox, true, 5, nil)

		assert.Equal(t, "full@example.com", result["email"])
		assert.Equal(t, "inbox-hash-123", result["id"])
		assert.Equal(t, inbox.CreatedAt.Format(time.RFC3339), result["createdAt"])
		assert.Equal(t, inbox.ExpiresAt.Format(time.RFC3339), result["expiresAt"])
		assert.Equal(t, false, result["isExpired"])
		assert.Equal(t, true, result["isActive"])
		assert.Equal(t, 5, result["emailCount"])
		assert.Nil(t, result["syncError"])
	})

	t.Run("full inbox info with sync error", func(t *testing.T) {
		syncErr := errors.New("connection timeout")
		result := InboxFullJSON(inbox, false, 0, syncErr)

		assert.Equal(t, false, result["isActive"])
		assert.Equal(t, 0, result["emailCount"])
		assert.Equal(t, "connection timeout", result["syncError"])
	})

	t.Run("expired inbox", func(t *testing.T) {
		expiredInbox := &config.StoredInbox{
			Email:     "expired@example.com",
			ID:        "expired-hash",
			CreatedAt: now.Add(-72 * time.Hour),
			ExpiresAt: now.Add(-24 * time.Hour),
		}
		result := InboxFullJSON(expiredInbox, false, 0, nil)

		assert.Equal(t, true, result["isExpired"])
	})
}
