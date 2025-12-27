package cli

import (
	"context"

	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// KeystoreReader provides read access to stored inboxes
type KeystoreReader interface {
	GetActiveInbox() (*config.StoredInbox, error)
	FindInbox(partial string) (*config.StoredInbox, []string, error)
	GetInbox(email string) (*config.StoredInbox, error)
	ListInboxes() []config.StoredInbox
}

// KeystoreWriter provides write access to stored inboxes
type KeystoreWriter interface {
	AddInbox(inbox config.StoredInbox) error
	RemoveInbox(email string) error
	SetActiveInbox(email string) error
	SaveInbox(exported *vaultsandbox.ExportedInbox) error
}

// Keystore combines read and write access
type Keystore interface {
	KeystoreReader
	KeystoreWriter
}

// InboxClient provides inbox operations
type InboxClient interface {
	CreateInbox(ctx context.Context, opts ...vaultsandbox.InboxOption) (*vaultsandbox.Inbox, error)
	ImportInbox(ctx context.Context, exp *vaultsandbox.ExportedInbox) (*vaultsandbox.Inbox, error)
	DeleteInbox(ctx context.Context, email string) error
	Close()
}

// EmailOperations provides email-level operations
type EmailOperations interface {
	GetEmails(ctx context.Context) ([]*vaultsandbox.Email, error)
	GetEmail(ctx context.Context, id string) (*vaultsandbox.Email, error)
	DeleteEmail(ctx context.Context, id string) error
	GetRawEmail(ctx context.Context, id string) (string, error)
	WaitForEmail(ctx context.Context, opts ...vaultsandbox.WaitOption) (*vaultsandbox.Email, error)
}
