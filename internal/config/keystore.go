package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
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
}

// InboxKeys contains the cryptographic keys for an inbox
type InboxKeys struct {
	KEMPrivate  string `json:"kem_private"`   // base64 encoded
	KEMPublic   string `json:"kem_public"`    // base64 encoded
	ServerSigPK string `json:"server_sig_pk"` // pinned server key
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

	return ks, nil
}

// Save writes the keystore to disk with secure permissions
func (ks *Keystore) Save() error {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if err := EnsureDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ks, "", "  ")
	if err != nil {
		return err
	}

	// Write with restrictive permissions (owner read/write only)
	return os.WriteFile(ks.path, data, 0600)
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

// GetInbox retrieves an inbox by email address
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

	// Verify inbox exists
	found := false
	for _, inbox := range ks.Inboxes {
		if inbox.Email == email {
			found = true
			break
		}
	}
	if !found {
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

// PruneExpired removes expired inboxes
func (ks *Keystore) PruneExpired() (int, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	now := time.Now()
	removed := 0
	active := []StoredInbox{}

	for _, inbox := range ks.Inboxes {
		if inbox.ExpiresAt.After(now) {
			active = append(active, inbox)
		} else {
			removed++
		}
	}

	if removed > 0 {
		ks.Inboxes = active

		// Fix active inbox if it was pruned
		if ks.ActiveInbox != "" {
			found := false
			for _, inbox := range ks.Inboxes {
				if inbox.Email == ks.ActiveInbox {
					found = true
					break
				}
			}
			if !found {
				if len(ks.Inboxes) > 0 {
					ks.ActiveInbox = ks.Inboxes[0].Email
				} else {
					ks.ActiveInbox = ""
				}
			}
		}

		return removed, ks.saveLocked()
	}

	return 0, nil
}

// Internal helpers

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
