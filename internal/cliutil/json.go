package cliutil

import (
	"strings"
	"time"

	"github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// EmailJSONOptions controls which fields to include in email JSON output.
type EmailJSONOptions struct {
	IncludeTo      bool
	IncludeBody    bool // text and html
	IncludeLinks   bool
	IncludeHeaders bool
}

// EmailJSON returns a map for JSON output with configurable fields.
func EmailJSON(email *vaultsandbox.Email, opts EmailJSONOptions) map[string]interface{} {
	m := map[string]interface{}{
		"id":         email.ID,
		"subject":    email.Subject,
		"from":       email.From,
		"receivedAt": email.ReceivedAt.Format(time.RFC3339),
	}

	if opts.IncludeTo {
		if len(email.To) > 0 {
			m["to"] = strings.Join(email.To, ", ")
		} else {
			m["to"] = ""
		}
	}
	if opts.IncludeBody {
		m["text"] = email.Text
		m["html"] = email.HTML
	}
	if opts.IncludeLinks {
		m["links"] = email.Links
	}
	if opts.IncludeHeaders {
		m["headers"] = email.Headers
	}

	return m
}

// EmailSummaryJSON returns a map for JSON output of email list items.
// Used by list command for compact email representation.
func EmailSummaryJSON(email *vaultsandbox.Email) map[string]interface{} {
	return EmailJSON(email, EmailJSONOptions{})
}

// EmailFullJSON returns a map for JSON output of full email details.
// Used by view and wait commands.
func EmailFullJSON(email *vaultsandbox.Email) map[string]interface{} {
	return EmailJSON(email, EmailJSONOptions{
		IncludeTo:      true,
		IncludeBody:    true,
		IncludeLinks:   true,
		IncludeHeaders: true,
	})
}

// InboxJSONOptions controls which fields to include in inbox JSON output.
type InboxJSONOptions struct {
	IncludeID         bool
	IncludeCreatedAt  bool
	IncludeEmailCount bool
	EmailCount        int
	SyncErr           error
}

// InboxJSON returns a map for JSON output with configurable fields.
func InboxJSON(inbox *config.StoredInbox, isActive bool, now time.Time, opts InboxJSONOptions) map[string]interface{} {
	m := map[string]interface{}{
		"email":     inbox.Email,
		"expiresAt": inbox.ExpiresAt.Format(time.RFC3339),
		"isActive":  isActive,
		"isExpired": inbox.ExpiresAt.Before(now),
	}

	if opts.IncludeID {
		m["id"] = inbox.ID
	}
	if opts.IncludeCreatedAt {
		m["createdAt"] = inbox.CreatedAt.Format(time.RFC3339)
	}
	if opts.IncludeEmailCount {
		m["emailCount"] = opts.EmailCount
	}
	if opts.SyncErr != nil {
		m["syncError"] = opts.SyncErr.Error()
	}

	return m
}

// InboxSummaryJSON returns a map for JSON output of inbox list items.
// Used by inbox list command.
func InboxSummaryJSON(inbox *config.StoredInbox, isActive bool, now time.Time) map[string]interface{} {
	return InboxJSON(inbox, isActive, now, InboxJSONOptions{})
}

// InboxFullJSON returns a map for JSON output of full inbox details.
// Used by inbox info command.
func InboxFullJSON(inbox *config.StoredInbox, isActive bool, emailCount int, syncErr error) map[string]interface{} {
	return InboxJSON(inbox, isActive, time.Now(), InboxJSONOptions{
		IncludeID:         true,
		IncludeCreatedAt:  true,
		IncludeEmailCount: true,
		EmailCount:        emailCount,
		SyncErr:           syncErr,
	})
}
