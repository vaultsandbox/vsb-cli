# Phase 1.2: Configuration & Keystore Management

## Objective
Implement the configuration system (`config.yaml`) and secure keystore (`keystore.json`) for persisting inboxes and private keys.

## Tasks

### 1. Configuration Manager

**File: `internal/config/config.go`**

```go
package config

import (
    "os"
    "path/filepath"

    "github.com/spf13/viper"
)

type Config struct {
    APIKey        string `mapstructure:"api_key"`
    BaseURL       string `mapstructure:"base_url"`
    DefaultOutput string `mapstructure:"default_output"`
}

// DefaultBaseURL is the production API endpoint
const DefaultBaseURL = "https://api.vaultsandbox.com"

// Dir returns the vsb config directory path
func Dir() (string, error) {
    configDir, err := os.UserConfigDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(configDir, "vsb"), nil
}

// EnsureDir creates the config directory if it doesn't exist
func EnsureDir() error {
    dir, err := Dir()
    if err != nil {
        return err
    }
    return os.MkdirAll(dir, 0700)
}

// Load reads configuration from viper (already initialized in root.go)
func Load() *Config {
    return &Config{
        APIKey:        viper.GetString("api_key"),
        BaseURL:       viper.GetString("base_url"),
        DefaultOutput: viper.GetString("default_output"),
    }
}

// GetAPIKey returns the API key, checking env vars and config
func GetAPIKey() string {
    // Priority: flag > env > config file
    if key := viper.GetString("api_key"); key != "" {
        return key
    }
    return os.Getenv("VSB_API_KEY")
}

// GetBaseURL returns the base URL with default fallback
func GetBaseURL() string {
    if url := viper.GetString("base_url"); url != "" {
        return url
    }
    return DefaultBaseURL
}

// Save writes the current config to disk
func Save(cfg *Config) error {
    if err := EnsureDir(); err != nil {
        return err
    }

    dir, _ := Dir()
    configPath := filepath.Join(dir, "config.yaml")

    viper.Set("api_key", cfg.APIKey)
    viper.Set("base_url", cfg.BaseURL)
    viper.Set("default_output", cfg.DefaultOutput)

    return viper.WriteConfigAs(configPath)
}
```

### 2. Keystore Manager

**File: `internal/config/keystore.go`**

```go
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
    Email       string    `json:"email"`
    ID          string    `json:"id"`           // inbox hash
    Label       string    `json:"label"`        // user-defined label
    CreatedAt   time.Time `json:"createdAt"`
    ExpiresAt   time.Time `json:"expiresAt"`
    Keys        InboxKeys `json:"keys"`
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
    data, err := json.MarshalIndent(ks, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(ks.path, data, 0600)
}
```

### 3. SDK Client Factory

**File: `internal/config/client.go`**

```go
package config

import (
    "errors"

    vaultsandbox "github.com/vaultsandbox/client-go"
)

var ErrNoAPIKey = errors.New("API key not configured. Set VSB_API_KEY or run 'vsb config set api_key <key>'")

// NewClient creates a VaultSandbox client using current configuration
func NewClient() (*vaultsandbox.Client, error) {
    apiKey := GetAPIKey()
    if apiKey == "" {
        return nil, ErrNoAPIKey
    }

    opts := []vaultsandbox.Option{}

    if baseURL := GetBaseURL(); baseURL != "" {
        opts = append(opts, vaultsandbox.WithBaseURL(baseURL))
    }

    return vaultsandbox.New(apiKey, opts...)
}

// NewClientWithKeystore creates a client and loads the keystore
func NewClientWithKeystore() (*vaultsandbox.Client, *Keystore, error) {
    client, err := NewClient()
    if err != nil {
        return nil, nil, err
    }

    keystore, err := LoadKeystore()
    if err != nil {
        client.Close()
        return nil, nil, err
    }

    return client, keystore, nil
}
```

### 4. Conversion Helpers

**File: `internal/config/convert.go`**

```go
package config

import (
    vaultsandbox "github.com/vaultsandbox/client-go"
)

// StoredInboxFromExport converts SDK ExportedInbox to StoredInbox
func StoredInboxFromExport(exp *vaultsandbox.ExportedInbox, label string) StoredInbox {
    return StoredInbox{
        Email:     exp.EmailAddress,
        ID:        exp.InboxHash,
        Label:     label,
        CreatedAt: exp.ExportedAt,
        ExpiresAt: exp.ExpiresAt,
        Keys: InboxKeys{
            KEMPrivate:  exp.SecretKeyB64,
            KEMPublic:   exp.PublicKeyB64,
            ServerSigPK: exp.ServerSigPk,
        },
    }
}

// ToExportedInbox converts StoredInbox to SDK ExportedInbox for import
func (s *StoredInbox) ToExportedInbox() *vaultsandbox.ExportedInbox {
    return &vaultsandbox.ExportedInbox{
        EmailAddress: s.Email,
        ExpiresAt:    s.ExpiresAt,
        InboxHash:    s.ID,
        ServerSigPk:  s.Keys.ServerSigPK,
        PublicKeyB64: s.Keys.KEMPublic,
        SecretKeyB64: s.Keys.KEMPrivate,
        ExportedAt:   s.CreatedAt,
    }
}
```

## Security Considerations

1. **File Permissions**: Keystore is written with `0600` (owner read/write only)
2. **Private Keys**: KEMPrivate contains sensitive key material
3. **No Logging**: Never log private keys or full keystore contents

## Verification

Create a simple test command to verify keystore operations:

```bash
# Test creating and saving an inbox
go run ./cmd/vsb inbox create --label test

# Check file permissions
ls -la ~/.config/vsb/keystore.json
# Should show: -rw------- (0600)
```

## Files Created

- `internal/config/config.go`
- `internal/config/keystore.go`
- `internal/config/client.go`
- `internal/config/convert.go`

## Next Steps

Proceed to [03-inbox-commands.md](03-inbox-commands.md) to implement inbox management commands.
