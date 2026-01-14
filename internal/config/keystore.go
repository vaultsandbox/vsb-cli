package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	vaultsandbox "github.com/vaultsandbox/client-go"
)

var (
	ErrNoActiveInbox = errors.New("no active inbox set")
	ErrInboxNotFound = errors.New("inbox not found in keystore")
)

// StoredInbox represents an inbox persisted in the keystore
type StoredInbox struct {
	Email     string    `json:"email"`
	ID        string    `json:"id"`        // inbox hash
	Label     string    `json:"label"`     // user-defined label
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
	Keys      InboxKeys `json:"keys"`
	Encrypted bool      `json:"encrypted"`  // whether inbox uses encryption
	EmailAuth bool      `json:"emailAuth"`  // whether email auth is enabled
}

// InboxKeys contains the cryptographic keys for an inbox
type InboxKeys struct {
	KEMPrivate  string `json:"kem_private"`   // base64 encoded
	KEMPublic   string `json:"kem_public"`    // base64 encoded
	ServerSigPK string `json:"server_sig_pk"` // pinned server key
}

// ExportedInboxFile is the file format for exported inboxes
type ExportedInboxFile struct {
	Version      int          `json:"version"`
	EmailAddress string       `json:"emailAddress"`
	InboxHash    string       `json:"inboxHash"`
	ExpiresAt    time.Time    `json:"expiresAt"`
	ExportedAt   time.Time    `json:"exportedAt"`
	Keys         ExportedKeys `json:"keys"`
	Encrypted    bool         `json:"encrypted"`
	EmailAuth    bool         `json:"emailAuth"`
}

// ExportedKeys contains the cryptographic keys in an export file
type ExportedKeys struct {
	KEMPrivate  string `json:"kemPrivate"`
	KEMPublic   string `json:"kemPublic"`
	ServerSigPK string `json:"serverSigPk"`
}

// Keystore manages inbox persistence
type Keystore struct {
	Inboxes     []StoredInbox `json:"inboxes"`
	ActiveInbox string        `json:"active_inbox"` // email address

	mu   sync.RWMutex
	path string
}

// keystorePath returns the path to keystore.json
func keystorePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "keystore.json"), nil
}

// LoadKeystore reads the keystore from disk
func LoadKeystore() (*Keystore, error) {
	path, err := keystorePath()
	if err != nil {
		return nil, err
	}

	ks := &Keystore{
		Inboxes: []StoredInbox{},
		path:    path,
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// New keystore
		return ks, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, ks); err != nil {
		return nil, err
	}
	ks.path = path

	// Auto-prune expired inboxes on load
	ks.pruneExpired()

	return ks, nil
}

// Save writes the keystore to disk with secure permissions
func (ks *Keystore) Save() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	return ks.saveLocked()
}

// AddInbox adds a new inbox to the keystore
func (ks *Keystore) AddInbox(inbox StoredInbox) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Remove existing inbox with same email (update)
	ks.removeInboxLocked(inbox.Email)

	ks.Inboxes = append(ks.Inboxes, inbox)
	ks.ActiveInbox = inbox.Email

	return ks.saveLocked()
}

// SaveInbox saves an exported inbox to the keystore
func (ks *Keystore) SaveInbox(exported *vaultsandbox.ExportedInbox) error {
	stored := StoredInboxFromExport(exported)
	return ks.AddInbox(stored)
}

// GetInbox retrieves an inbox by email address (exact match)
func (ks *Keystore) GetInbox(email string) (*StoredInbox, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	for i := range ks.Inboxes {
		if ks.Inboxes[i].Email == email {
			return &ks.Inboxes[i], nil
		}
	}
	return nil, ErrInboxNotFound
}

// ErrMultipleMatches is returned when a partial match finds multiple inboxes
var ErrMultipleMatches = errors.New("multiple inboxes match")

// FindInbox retrieves an inbox by partial email match.
// Returns the inbox if exactly one matches, error if none or multiple match.
func (ks *Keystore) FindInbox(partial string) (*StoredInbox, []string, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	// Single-pass: check exact and collect partial matches together
	var matches []*StoredInbox
	var matchEmails []string
	for i := range ks.Inboxes {
		if ks.Inboxes[i].Email == partial {
			// Exact match - return immediately
			return &ks.Inboxes[i], nil, nil
		}
		if strings.Contains(ks.Inboxes[i].Email, partial) {
			matches = append(matches, &ks.Inboxes[i])
			matchEmails = append(matchEmails, ks.Inboxes[i].Email)
		}
	}

	if len(matches) == 0 {
		return nil, nil, ErrInboxNotFound
	}
	if len(matches) == 1 {
		return matches[0], nil, nil
	}
	return nil, matchEmails, ErrMultipleMatches
}

// GetActiveInbox returns the currently active inbox
func (ks *Keystore) GetActiveInbox() (*StoredInbox, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if ks.ActiveInbox == "" {
		return nil, ErrNoActiveInbox
	}

	for i := range ks.Inboxes {
		if ks.Inboxes[i].Email == ks.ActiveInbox {
			return &ks.Inboxes[i], nil
		}
	}
	return nil, ErrNoActiveInbox
}

// SetActiveInbox changes the active inbox
func (ks *Keystore) SetActiveInbox(email string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.inboxExistsLocked(email) {
		return ErrInboxNotFound
	}

	ks.ActiveInbox = email
	return ks.saveLocked()
}

// RemoveInbox removes an inbox by email address
func (ks *Keystore) RemoveInbox(email string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if !ks.removeInboxLocked(email) {
		return ErrInboxNotFound
	}

	// Clear active if it was this inbox
	if ks.ActiveInbox == email {
		if len(ks.Inboxes) > 0 {
			ks.ActiveInbox = ks.Inboxes[0].Email
		} else {
			ks.ActiveInbox = ""
		}
	}

	return ks.saveLocked()
}

// ListInboxes returns all stored inboxes
func (ks *Keystore) ListInboxes() []StoredInbox {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	// Return copy to avoid race conditions
	result := make([]StoredInbox, len(ks.Inboxes))
	copy(result, ks.Inboxes)
	return result
}

// pruneExpired removes expired inboxes (internal, no locking - used during load)
func (ks *Keystore) pruneExpired() {
	now := time.Now()
	active := []StoredInbox{}

	for _, inbox := range ks.Inboxes {
		if inbox.ExpiresAt.After(now) {
			active = append(active, inbox)
		}
	}

	if len(active) < len(ks.Inboxes) {
		ks.Inboxes = active

		// Fix active inbox if it was pruned
		if ks.ActiveInbox != "" && !ks.inboxExistsLocked(ks.ActiveInbox) {
			if len(ks.Inboxes) > 0 {
				ks.ActiveInbox = ks.Inboxes[0].Email
			} else {
				ks.ActiveInbox = ""
			}
		}

		// Save changes silently
		ks.saveLocked()
	}
}

// Internal helpers

func (ks *Keystore) inboxExistsLocked(email string) bool {
	for i := range ks.Inboxes {
		if ks.Inboxes[i].Email == email {
			return true
		}
	}
	return false
}

func (ks *Keystore) removeInboxLocked(email string) bool {
	for i, inbox := range ks.Inboxes {
		if inbox.Email == email {
			ks.Inboxes = append(ks.Inboxes[:i], ks.Inboxes[i+1:]...)
			return true
		}
	}
	return false
}

func (ks *Keystore) saveLocked() error {
	if err := EnsureDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ks.path, data, 0600)
}

// StoredInboxFromExport converts SDK ExportedInbox to StoredInbox
func StoredInboxFromExport(exp *vaultsandbox.ExportedInbox) StoredInbox {
	return StoredInbox{
		Email:     exp.EmailAddress,
		ID:        exp.InboxHash,
		CreatedAt: exp.ExportedAt,
		ExpiresAt: exp.ExpiresAt,
		Keys: InboxKeys{
			KEMPrivate:  exp.SecretKey,
			KEMPublic:   "", // Public key is derived from secret key per spec Section 4.2
			ServerSigPK: exp.ServerSigPk,
		},
		Encrypted: exp.Encrypted,
		EmailAuth: exp.EmailAuth,
	}
}

// ToExportedInbox converts StoredInbox to SDK ExportedInbox for import
func (s *StoredInbox) ToExportedInbox() *vaultsandbox.ExportedInbox {
	return &vaultsandbox.ExportedInbox{
		Version:      1,
		EmailAddress: s.Email,
		ExpiresAt:    s.ExpiresAt,
		InboxHash:    s.ID,
		ServerSigPk:  s.Keys.ServerSigPK,
		SecretKey:    s.Keys.KEMPrivate,
		ExportedAt:   s.CreatedAt,
		Encrypted:    s.Encrypted,
		EmailAuth:    s.EmailAuth,
	}
}

// ToExportFile converts StoredInbox to ExportedInboxFile for file export
func (s *StoredInbox) ToExportFile() ExportedInboxFile {
	return ExportedInboxFile{
		Version:      1,
		EmailAddress: s.Email,
		InboxHash:    s.ID,
		ExpiresAt:    s.ExpiresAt,
		ExportedAt:   time.Now(),
		Keys: ExportedKeys{
			KEMPrivate:  s.Keys.KEMPrivate,
			KEMPublic:   s.Keys.KEMPublic,
			ServerSigPK: s.Keys.ServerSigPK,
		},
		Encrypted: s.Encrypted,
		EmailAuth: s.EmailAuth,
	}
}

// ToStoredInbox converts ExportedInboxFile to StoredInbox for import
func (e *ExportedInboxFile) ToStoredInbox() StoredInbox {
	return StoredInbox{
		Email:     e.EmailAddress,
		ID:        e.InboxHash,
		CreatedAt: e.ExportedAt,
		ExpiresAt: e.ExpiresAt,
		Keys: InboxKeys{
			KEMPrivate:  e.Keys.KEMPrivate,
			KEMPublic:   e.Keys.KEMPublic,
			ServerSigPK: e.Keys.ServerSigPK,
		},
		Encrypted: e.Encrypted,
		EmailAuth: e.EmailAuth,
	}
}
