package cli

import (
	"context"
	"fmt"

	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// LoadKeystoreOrError loads keystore with a consistent error message.
func LoadKeystoreOrError() (*config.Keystore, error) {
	ks, err := config.LoadKeystore()
	if err != nil {
		return nil, fmt.Errorf("failed to load keystore: %w", err)
	}
	return ks, nil
}

// GetInbox returns an inbox by email flag, or the active inbox if emailFlag is empty.
func GetInbox(ks *config.Keystore, emailFlag string) (*config.StoredInbox, error) {
	if emailFlag != "" {
		inbox, err := ks.GetInbox(emailFlag)
		if err != nil {
			return nil, fmt.Errorf("inbox not found: %s", emailFlag)
		}
		return inbox, nil
	}

	inbox, err := ks.GetActiveInbox()
	if err != nil {
		return nil, fmt.Errorf("no active inbox. Create one with 'vsb inbox create' or set with 'vsb inbox use'")
	}
	return inbox, nil
}

// GetEmailByIDOrLatest fetches an email by ID if provided, otherwise returns the latest email.
// Returns the email, the imported inbox, a cleanup function (closes client), and any error.
// The caller must call the cleanup function when done.
func GetEmailByIDOrLatest(ctx context.Context, emailID, emailFlag string) (*vaultsandbox.Email, *vaultsandbox.Inbox, func(), error) {
	noop := func() {}

	// Load keystore
	ks, err := LoadKeystoreOrError()
	if err != nil {
		return nil, nil, noop, err
	}

	// Get inbox
	stored, err := GetInbox(ks, emailFlag)
	if err != nil {
		return nil, nil, noop, err
	}

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return nil, nil, noop, err
	}

	cleanup := func() {
		client.Close()
	}

	// Import inbox
	inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
	if err != nil {
		cleanup()
		return nil, nil, noop, fmt.Errorf("failed to import inbox: %w", err)
	}

	// Fetch email
	var email *vaultsandbox.Email
	if emailID != "" {
		email, err = inbox.GetEmail(ctx, emailID)
		if err != nil {
			cleanup()
			return nil, nil, noop, fmt.Errorf("failed to get email %s: %w", emailID, err)
		}
	} else {
		emails, err := inbox.GetEmails(ctx)
		if err != nil {
			cleanup()
			return nil, nil, noop, fmt.Errorf("failed to get emails: %w", err)
		}
		if len(emails) == 0 {
			cleanup()
			return nil, nil, noop, fmt.Errorf("no emails found in inbox")
		}
		email = emails[0]
	}

	return email, inbox, cleanup, nil
}
