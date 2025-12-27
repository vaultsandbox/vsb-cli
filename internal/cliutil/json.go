package cli

import (
	"strings"
	"time"

	"github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// EmailSummaryJSON returns a map for JSON output of email list items.
// Used by list command for compact email representation.
func EmailSummaryJSON(email *vaultsandbox.Email) map[string]interface{} {
	return map[string]interface{}{
		"id":         email.ID,
		"subject":    email.Subject,
		"from":       email.From,
		"receivedAt": email.ReceivedAt.Format(time.RFC3339),
	}
}

// EmailFullJSON returns a map for JSON output of full email details.
// Used by view and wait commands.
func EmailFullJSON(email *vaultsandbox.Email) map[string]interface{} {
	var to interface{}
	if len(email.To) > 0 {
		to = strings.Join(email.To, ", ")
	} else {
		to = ""
	}

	return map[string]interface{}{
		"id":         email.ID,
		"subject":    email.Subject,
		"from":       email.From,
		"to":         to,
		"receivedAt": email.ReceivedAt.Format(time.RFC3339),
		"text":       email.Text,
		"html":       email.HTML,
		"links":      email.Links,
		"headers":    email.Headers,
	}
}

// InboxSummaryJSON returns a map for JSON output of inbox list items.
// Used by inbox list command.
func InboxSummaryJSON(inbox *config.StoredInbox, isActive bool, now time.Time) map[string]interface{} {
	return map[string]interface{}{
		"email":     inbox.Email,
		"expiresAt": inbox.ExpiresAt.Format(time.RFC3339),
		"isActive":  isActive,
		"isExpired": inbox.ExpiresAt.Before(now),
	}
}

// InboxFullJSON returns a map for JSON output of full inbox details.
// Used by inbox info command.
func InboxFullJSON(inbox *config.StoredInbox, isActive bool, emailCount int, syncErr error) map[string]interface{} {
	data := map[string]interface{}{
		"email":      inbox.Email,
		"id":         inbox.ID,
		"createdAt":  inbox.CreatedAt.Format(time.RFC3339),
		"expiresAt":  inbox.ExpiresAt.Format(time.RFC3339),
		"isExpired":  inbox.ExpiresAt.Before(time.Now()),
		"isActive":   isActive,
		"emailCount": emailCount,
	}
	if syncErr != nil {
		data["syncError"] = syncErr.Error()
	}
	return data
}
