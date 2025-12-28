package cliutil

import (
	"time"

	"github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// EmailJSONOptions controls which fields to include in email JSON output.
type EmailJSONOptions struct {
	IncludeTo          bool
	IncludeBody        bool // text and html
	IncludeLinks       bool
	IncludeHeaders     bool
	IncludeAuthResults bool
	IncludeScore       bool
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
		m["to"] = email.To
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
	if opts.IncludeAuthResults && email.AuthResults != nil {
		m["authResults"] = buildAuthResultsJSON(email)
	}
	if opts.IncludeScore {
		m["securityScore"] = styles.CalculateScore(email)
	}

	return m
}

// buildAuthResultsJSON builds auth results map for JSON output.
func buildAuthResultsJSON(email *vaultsandbox.Email) map[string]interface{} {
	auth := email.AuthResults
	authData := map[string]interface{}{}

	if auth.SPF != nil {
		authData["spf"] = map[string]string{
			"status": auth.SPF.Status,
			"domain": auth.SPF.Domain,
		}
	}
	if len(auth.DKIM) > 0 {
		dkim := auth.DKIM[0]
		authData["dkim"] = map[string]string{
			"status":   dkim.Status,
			"selector": dkim.Selector,
			"domain":   dkim.Domain,
		}
	}
	if auth.DMARC != nil {
		authData["dmarc"] = map[string]string{
			"status": auth.DMARC.Status,
			"policy": auth.DMARC.Policy,
		}
	}

	return authData
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

// EmailAuditJSON returns a map for JSON output of email audit.
// Used by audit command.
func EmailAuditJSON(email *vaultsandbox.Email) map[string]interface{} {
	return EmailJSON(email, EmailJSONOptions{
		IncludeTo:          true,
		IncludeAuthResults: true,
		IncludeScore:       true,
	})
}

// InboxJSONOptions controls which fields to include in inbox JSON output.
// Pointer fields are included when non-nil.
type InboxJSONOptions struct {
	IncludeID        bool
	IncludeCreatedAt bool
	EmailCount       *int  // nil = don't include
	SyncErr          error // nil = don't include
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
	if opts.EmailCount != nil {
		m["emailCount"] = *opts.EmailCount
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
func InboxFullJSON(inbox *config.StoredInbox, isActive bool, emailCount int, syncErr error, now time.Time) map[string]interface{} {
	return InboxJSON(inbox, isActive, now, InboxJSONOptions{
		IncludeID:        true,
		IncludeCreatedAt: true,
		EmailCount:       &emailCount,
		SyncErr:          syncErr,
	})
}
