# Phase 3: CLI Business Logic Testing

**Goal**: Extract and test CLI business logic separate from Cobra commands.
**Expected Coverage Gain**: +10-15%
**Effort**: Medium-High

## Overview

Phase 3 requires some refactoring to extract testable interfaces from the CLI commands. The goal is to test business logic without needing real API calls or full Cobra command execution.

---

## 3.1 Interface Extraction

### Required Refactoring

Create interfaces in `internal/cli/interfaces.go`:

```go
package cli

import (
    "context"

    "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/config"
)

// KeystoreReader provides read access to stored inboxes
type KeystoreReader interface {
    GetActiveInbox() (*config.StoredInbox, error)
    FindInbox(partial string) (*config.StoredInbox, int, error)
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
    CreateInbox(ctx context.Context, opts ...vaultsandbox.CreateOption) (*vaultsandbox.Inbox, error)
    ImportInbox(ctx context.Context, exp vaultsandbox.ExportedInbox) (*vaultsandbox.Inbox, error)
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
```

---

## 3.2 Helpers Package Tests

**File to create**: `internal/cli/helpers_test.go`

### Mock Implementations

```go
package cli

import (
    "github.com/vaultsandbox/vsb-cli/internal/config"
)

// MockKeystore for testing
type MockKeystore struct {
    inboxes     []config.StoredInbox
    activeEmail string

    // Function overrides for custom behavior
    GetActiveInboxFunc func() (*config.StoredInbox, error)
    FindInboxFunc      func(partial string) (*config.StoredInbox, int, error)
    GetInboxFunc       func(email string) (*config.StoredInbox, error)
}

func (m *MockKeystore) GetActiveInbox() (*config.StoredInbox, error) {
    if m.GetActiveInboxFunc != nil {
        return m.GetActiveInboxFunc()
    }
    if m.activeEmail == "" {
        return nil, config.ErrNoActiveInbox
    }
    for _, inbox := range m.inboxes {
        if inbox.Email == m.activeEmail {
            return &inbox, nil
        }
    }
    return nil, config.ErrNoActiveInbox
}

func (m *MockKeystore) FindInbox(partial string) (*config.StoredInbox, int, error) {
    if m.FindInboxFunc != nil {
        return m.FindInboxFunc(partial)
    }
    var matches []config.StoredInbox
    for _, inbox := range m.inboxes {
        if inbox.Email == partial {
            return &inbox, 1, nil // Exact match
        }
        if strings.Contains(inbox.Email, partial) {
            matches = append(matches, inbox)
        }
    }
    if len(matches) == 0 {
        return nil, 0, config.ErrInboxNotFound
    }
    if len(matches) > 1 {
        return nil, len(matches), config.ErrMultipleMatches
    }
    return &matches[0], 1, nil
}

func (m *MockKeystore) GetInbox(email string) (*config.StoredInbox, error) {
    if m.GetInboxFunc != nil {
        return m.GetInboxFunc(email)
    }
    for _, inbox := range m.inboxes {
        if inbox.Email == email {
            return &inbox, nil
        }
    }
    return nil, config.ErrInboxNotFound
}

func (m *MockKeystore) ListInboxes() []config.StoredInbox {
    return m.inboxes
}
```

### Test Cases for `GetInbox`

```go
func TestGetInbox(t *testing.T) {
    inbox1 := config.StoredInbox{Email: "test1@example.com"}
    inbox2 := config.StoredInbox{Email: "test2@example.com"}

    t.Run("empty flag returns active inbox", func(t *testing.T) {
        ks := &MockKeystore{
            inboxes:     []config.StoredInbox{inbox1, inbox2},
            activeEmail: "test1@example.com",
        }

        result, err := GetInbox(ks, "")
        require.NoError(t, err)
        assert.Equal(t, "test1@example.com", result.Email)
    })

    t.Run("empty flag with no active returns error", func(t *testing.T) {
        ks := &MockKeystore{
            inboxes:     []config.StoredInbox{inbox1},
            activeEmail: "",
        }

        _, err := GetInbox(ks, "")
        assert.ErrorIs(t, err, config.ErrNoActiveInbox)
    })

    t.Run("exact email match", func(t *testing.T) {
        ks := &MockKeystore{
            inboxes: []config.StoredInbox{inbox1, inbox2},
        }

        result, err := GetInbox(ks, "test2@example.com")
        require.NoError(t, err)
        assert.Equal(t, "test2@example.com", result.Email)
    })

    t.Run("partial match", func(t *testing.T) {
        ks := &MockKeystore{
            inboxes: []config.StoredInbox{
                {Email: "unique123@example.com"},
            },
        }

        result, err := GetInbox(ks, "unique123")
        require.NoError(t, err)
        assert.Equal(t, "unique123@example.com", result.Email)
    })

    t.Run("multiple matches returns error", func(t *testing.T) {
        ks := &MockKeystore{
            inboxes: []config.StoredInbox{inbox1, inbox2},
        }

        _, err := GetInbox(ks, "test")
        assert.ErrorIs(t, err, config.ErrMultipleMatches)
    })

    t.Run("no match returns error", func(t *testing.T) {
        ks := &MockKeystore{
            inboxes: []config.StoredInbox{inbox1},
        }

        _, err := GetInbox(ks, "nonexistent")
        assert.ErrorIs(t, err, config.ErrInboxNotFound)
    })
}
```

---

## 3.3 Wait Command Tests

**File to create**: `internal/cli/wait_test.go`

### Extract Filtering Logic

First, extract the email filtering logic from `wait.go`:

```go
// In wait.go - extract this function
func matchesFilter(email *vaultsandbox.Email, subjectExact, subjectRegex, fromExact, fromRegex string) (bool, error) {
    // Subject exact match
    if subjectExact != "" && email.Subject != subjectExact {
        return false, nil
    }

    // Subject regex match
    if subjectRegex != "" {
        re, err := regexp.Compile(subjectRegex)
        if err != nil {
            return false, fmt.Errorf("invalid subject regex: %w", err)
        }
        if !re.MatchString(email.Subject) {
            return false, nil
        }
    }

    // From exact match
    if fromExact != "" && email.From != fromExact {
        return false, nil
    }

    // From regex match
    if fromRegex != "" {
        re, err := regexp.Compile(fromRegex)
        if err != nil {
            return false, fmt.Errorf("invalid from regex: %w", err)
        }
        if !re.MatchString(email.From) {
            return false, nil
        }
    }

    return true, nil
}
```

### Test Cases

```go
func TestMatchesFilter(t *testing.T) {
    baseEmail := &vaultsandbox.Email{
        Subject: "Welcome to VaultSandbox",
        From:    "noreply@vaultsandbox.com",
    }

    t.Run("no filters matches all", func(t *testing.T) {
        match, err := matchesFilter(baseEmail, "", "", "", "")
        require.NoError(t, err)
        assert.True(t, match)
    })

    t.Run("subject exact match", func(t *testing.T) {
        match, _ := matchesFilter(baseEmail, "Welcome to VaultSandbox", "", "", "")
        assert.True(t, match)

        match, _ = matchesFilter(baseEmail, "Different Subject", "", "", "")
        assert.False(t, match)
    })

    t.Run("subject regex match", func(t *testing.T) {
        match, err := matchesFilter(baseEmail, "", "Welcome.*", "", "")
        require.NoError(t, err)
        assert.True(t, match)

        match, _ = matchesFilter(baseEmail, "", "^Goodbye", "", "")
        assert.False(t, match)
    })

    t.Run("invalid regex returns error", func(t *testing.T) {
        _, err := matchesFilter(baseEmail, "", "[invalid", "", "")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid subject regex")
    })

    t.Run("from exact match", func(t *testing.T) {
        match, _ := matchesFilter(baseEmail, "", "", "noreply@vaultsandbox.com", "")
        assert.True(t, match)

        match, _ = matchesFilter(baseEmail, "", "", "other@example.com", "")
        assert.False(t, match)
    })

    t.Run("from regex match", func(t *testing.T) {
        match, _ := matchesFilter(baseEmail, "", "", "", "@vaultsandbox\\.com$")
        assert.True(t, match)
    })

    t.Run("combined filters (AND logic)", func(t *testing.T) {
        // Both must match
        match, _ := matchesFilter(baseEmail, "Welcome to VaultSandbox", "", "noreply@vaultsandbox.com", "")
        assert.True(t, match)

        // Subject matches but from doesn't
        match, _ = matchesFilter(baseEmail, "Welcome to VaultSandbox", "", "wrong@example.com", "")
        assert.False(t, match)
    })
}
```

---

## 3.4 Audit Command Tests

**File to create**: `internal/cli/audit_test.go`

### Test `buildMIMETree`

```go
func TestBuildMIMETree(t *testing.T) {
    t.Run("simple text email", func(t *testing.T) {
        email := &vaultsandbox.Email{
            ContentType: "text/plain",
        }

        tree := buildMIMETree(email)
        assert.Contains(t, tree, "text/plain")
    })

    t.Run("multipart email", func(t *testing.T) {
        email := &vaultsandbox.Email{
            ContentType: "multipart/alternative",
            Text:        "Plain text version",
            HTML:        "<p>HTML version</p>",
        }

        tree := buildMIMETree(email)
        assert.Contains(t, tree, "multipart/alternative")
        assert.Contains(t, tree, "text/plain")
        assert.Contains(t, tree, "text/html")
    })

    t.Run("email with attachments", func(t *testing.T) {
        email := &vaultsandbox.Email{
            ContentType: "multipart/mixed",
            Attachments: []vaultsandbox.Attachment{
                {Filename: "doc.pdf", ContentType: "application/pdf", Size: 1024},
                {Filename: "image.png", ContentType: "image/png", Size: 2048},
            },
        }

        tree := buildMIMETree(email)
        assert.Contains(t, tree, "multipart/mixed")
        assert.Contains(t, tree, "application/pdf")
        assert.Contains(t, tree, "image/png")
        assert.Contains(t, tree, "doc.pdf")
    })
}
```

---

## 3.5 Attachment Command Tests

**File to create**: `internal/cli/attachment_test.go`

### Test Download Functions

```go
func TestDownloadAttachment(t *testing.T) {
    t.Run("saves file successfully", func(t *testing.T) {
        dir := t.TempDir()
        content := []byte("test attachment content")

        err := downloadAttachment(dir, "test.txt", content)
        require.NoError(t, err)

        saved, _ := os.ReadFile(filepath.Join(dir, "test.txt"))
        assert.Equal(t, content, saved)
    })

    t.Run("handles filename collision", func(t *testing.T) {
        dir := t.TempDir()

        // Create first file
        downloadAttachment(dir, "test.txt", []byte("first"))

        // Save with same name
        err := downloadAttachment(dir, "test.txt", []byte("second"))
        require.NoError(t, err)

        // Should create test_1.txt
        _, err = os.Stat(filepath.Join(dir, "test_1.txt"))
        assert.NoError(t, err)
    })
}

func TestDownloadAllAttachments(t *testing.T) {
    t.Run("downloads multiple attachments", func(t *testing.T) {
        dir := t.TempDir()
        attachments := []vaultsandbox.Attachment{
            {Filename: "file1.txt", Content: []byte("content1")},
            {Filename: "file2.txt", Content: []byte("content2")},
        }

        err := downloadAllAttachments(dir, attachments)
        require.NoError(t, err)

        assert.FileExists(t, filepath.Join(dir, "file1.txt"))
        assert.FileExists(t, filepath.Join(dir, "file2.txt"))
    })
}
```

---

## 3.6 JSON Output Tests

**File to create**: `internal/cli/json_test.go`

### Test JSON Formatting

```go
func TestEmailSummaryJSON(t *testing.T) {
    email := &vaultsandbox.Email{
        ID:         "msg-123",
        From:       "sender@example.com",
        To:         []string{"recipient@test.com"},
        Subject:    "Test Subject",
        ReceivedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
    }

    result := EmailSummaryJSON(email)

    assert.Equal(t, "msg-123", result["id"])
    assert.Equal(t, "sender@example.com", result["from"])
    assert.Equal(t, []string{"recipient@test.com"}, result["to"])
    assert.Equal(t, "Test Subject", result["subject"])
}

func TestEmailFullJSON(t *testing.T) {
    email := &vaultsandbox.Email{
        ID:      "msg-456",
        Subject: "Full Email",
        Links:   []string{"https://example.com", "https://test.com"},
        Attachments: []vaultsandbox.Attachment{
            {Filename: "doc.pdf", ContentType: "application/pdf", Size: 1024},
        },
    }

    result := EmailFullJSON(email)

    links := result["links"].([]string)
    assert.Len(t, links, 2)

    attachments := result["attachments"].([]map[string]interface{})
    assert.Len(t, attachments, 1)
    assert.Equal(t, "doc.pdf", attachments[0]["filename"])
}

func TestInboxSummaryJSON(t *testing.T) {
    inbox := &config.StoredInbox{
        Email:     "test@example.com",
        InboxHash: "hash123",
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }

    result := InboxSummaryJSON(inbox, true, time.Now())

    assert.Equal(t, "test@example.com", result["email"])
    assert.Equal(t, true, result["is_active"])
    assert.Contains(t, result, "expires_at")
}

func TestInboxFullJSON(t *testing.T) {
    inbox := &config.StoredInbox{
        Email: "test@example.com",
    }

    t.Run("without error", func(t *testing.T) {
        result := InboxFullJSON(inbox, true, 5, nil)
        assert.Equal(t, 5, result["email_count"])
        assert.Nil(t, result["sync_error"])
    })

    t.Run("with sync error", func(t *testing.T) {
        err := errors.New("connection timeout")
        result := InboxFullJSON(inbox, true, 0, err)
        assert.Equal(t, "connection timeout", result["sync_error"])
    })
}
```

---

## Refactoring Checklist

Before writing tests, these refactoring changes are needed:

- [ ] Create `internal/cli/interfaces.go` with Keystore and Client interfaces
- [ ] Extract `matchesFilter()` function from `wait.go`
- [ ] Make `buildMIMETree()` exported or move to testable location
- [ ] Ensure `downloadAttachment()` accepts directory parameter
- [ ] Update helper functions to accept interfaces instead of concrete types

## Test Files Checklist

- [ ] Create `internal/cli/interfaces.go`
- [ ] Create `internal/cli/helpers_test.go`
- [ ] Create `internal/cli/wait_test.go`
- [ ] Create `internal/cli/audit_test.go`
- [ ] Create `internal/cli/attachment_test.go`
- [ ] Create `internal/cli/json_test.go`
- [ ] Run `go test ./internal/cli/...`

## Commands

```bash
# Run Phase 3 tests
go test -v ./internal/cli/...

# Check coverage
go test -coverprofile=phase3.out ./internal/cli/...
go tool cover -func=phase3.out

# View coverage in browser
go tool cover -html=phase3.out
```
