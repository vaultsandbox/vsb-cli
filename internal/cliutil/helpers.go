package cliutil

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

// GetInbox returns an inbox by email flag (with partial matching), or the active inbox if emailFlag is empty.
// Accepts KeystoreReader interface to allow testing with mock implementations.
func GetInbox(ks KeystoreReader, emailFlag string) (*config.StoredInbox, error) {
	if emailFlag != "" {
		inbox, matches, err := ks.FindInbox(emailFlag)
		if err == config.ErrMultipleMatches {
			return nil, fmt.Errorf("multiple inboxes match '%s': %v", emailFlag, matches)
		}
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

// LoadAndImportInbox loads the keystore, gets an inbox (by emailFlag or active),
// creates a client, and imports the inbox into the SDK.
// Returns the imported inbox, a cleanup function (closes client), and any error.
// The caller must call the cleanup function when done.
func LoadAndImportInbox(ctx context.Context, emailFlag string) (*vaultsandbox.Inbox, func(), error) {
	// Load keystore
	ks, err := LoadKeystoreOrError()
	if err != nil {
		return nil, func() {}, err
	}

	// Get stored inbox
	stored, err := GetInbox(ks, emailFlag)
	if err != nil {
		return nil, func() {}, err
	}

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return nil, func() {}, err
	}

	cleanup := func() {
		client.Close()
	}

	// Import inbox
	inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to import inbox: %w", err)
	}

	return inbox, cleanup, nil
}

// GetEmailByIDOrLatest fetches an email by ID if provided, otherwise returns the latest email.
// Returns the email, the imported inbox, a cleanup function (closes client), and any error.
// The caller must call the cleanup function when done.
func GetEmailByIDOrLatest(ctx context.Context, emailID, emailFlag string) (*vaultsandbox.Email, *vaultsandbox.Inbox, func(), error) {
	inbox, cleanup, err := LoadAndImportInbox(ctx, emailFlag)
	if err != nil {
		return nil, nil, func() {}, err
	}

	// Fetch email
	var email *vaultsandbox.Email
	if emailID != "" {
		email, err = inbox.GetEmail(ctx, emailID)
		if err != nil {
			cleanup()
			return nil, nil, func() {}, fmt.Errorf("failed to get email %s: %w", emailID, err)
		}
	} else {
		emails, err := inbox.GetEmails(ctx)
		if err != nil {
			cleanup()
			return nil, nil, func() {}, fmt.Errorf("failed to get emails: %w", err)
		}
		if len(emails) == 0 {
			cleanup()
			return nil, nil, func() {}, fmt.Errorf("no emails found in inbox")
		}
		email = emails[0]
	}

	return email, inbox, cleanup, nil
}
