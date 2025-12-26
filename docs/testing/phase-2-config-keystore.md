# Phase 2: Config & Keystore Testing

**Goal**: Test configuration and keystore logic with filesystem mocking.
**Expected Coverage Gain**: +10-12%
**Effort**: Medium

## Overview

Phase 2 tests the `internal/config/` package which handles:
- Configuration file management (YAML)
- Keystore persistence (JSON)
- Environment variable precedence
- Thread-safe inbox storage

---

## 2.1 Config Package

**File to create**: `internal/config/config_test.go`

### Functions to Test

#### `Dir() string`

```go
func TestDir(t *testing.T) {
    t.Run("default directory", func(t *testing.T) {
        // Unset VSB_CONFIG_DIR
        t.Setenv("VSB_CONFIG_DIR", "")
        dir := Dir()
        assert.Contains(t, dir, ".config/vsb")
    })

    t.Run("custom directory from env", func(t *testing.T) {
        customDir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", customDir)
        dir := Dir()
        assert.Equal(t, customDir, dir)
    })
}
```

#### `Path() string`

```go
func TestPath(t *testing.T) {
    t.Run("returns config.yaml path", func(t *testing.T) {
        path := Path()
        assert.True(t, strings.HasSuffix(path, "config.yaml"))
    })
}
```

#### `Load() (*Config, error)`

```go
func TestLoad(t *testing.T) {
    t.Run("missing file returns empty config", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)

        cfg, err := Load()
        require.NoError(t, err)
        assert.Empty(t, cfg.APIKey)
        assert.Empty(t, cfg.BaseURL)
    })

    t.Run("valid YAML", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)

        configContent := `api_key: test-key
base_url: https://api.example.com
output: json`
        os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(configContent), 0644)

        cfg, err := Load()
        require.NoError(t, err)
        assert.Equal(t, "test-key", cfg.APIKey)
        assert.Equal(t, "https://api.example.com", cfg.BaseURL)
        assert.Equal(t, "json", cfg.Output)
    })

    t.Run("invalid YAML", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)

        os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("invalid: [yaml"), 0644)

        _, err := Load()
        assert.Error(t, err)
    })
}
```

#### `EnsureDir() error`

```go
func TestEnsureDir(t *testing.T) {
    t.Run("creates directory with correct permissions", func(t *testing.T) {
        base := t.TempDir()
        dir := filepath.Join(base, "new-config-dir")
        t.Setenv("VSB_CONFIG_DIR", dir)

        err := EnsureDir()
        require.NoError(t, err)

        info, err := os.Stat(dir)
        require.NoError(t, err)
        assert.True(t, info.IsDir())
        assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
    })

    t.Run("existing directory is ok", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)

        err := EnsureDir()
        assert.NoError(t, err)
    })
}
```

#### `GetAPIKey() string`

```go
func TestGetAPIKey(t *testing.T) {
    t.Run("env var takes precedence", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)
        t.Setenv("VSB_API_KEY", "env-key")

        // Write different key to config
        os.WriteFile(filepath.Join(dir, "config.yaml"),
            []byte("api_key: config-key"), 0644)

        key := GetAPIKey()
        assert.Equal(t, "env-key", key)
    })

    t.Run("falls back to config file", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)
        t.Setenv("VSB_API_KEY", "") // Clear env var

        os.WriteFile(filepath.Join(dir, "config.yaml"),
            []byte("api_key: config-key"), 0644)

        key := GetAPIKey()
        assert.Equal(t, "config-key", key)
    })

    t.Run("returns empty if not set", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)
        t.Setenv("VSB_API_KEY", "")

        key := GetAPIKey()
        assert.Empty(t, key)
    })
}
```

#### `GetBaseURL() string`

```go
func TestGetBaseURL(t *testing.T) {
    t.Run("env var takes precedence", func(t *testing.T) {
        t.Setenv("VSB_BASE_URL", "https://custom.api.com")
        url := GetBaseURL()
        assert.Equal(t, "https://custom.api.com", url)
    })

    t.Run("returns default if not set", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)
        t.Setenv("VSB_BASE_URL", "")

        url := GetBaseURL()
        assert.Equal(t, "https://api.vaultsandbox.com", url) // Default
    })
}
```

#### `GetDefaultOutput() string`

```go
func TestGetDefaultOutput(t *testing.T) {
    t.Run("defaults to pretty", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)
        t.Setenv("VSB_OUTPUT", "")

        output := GetDefaultOutput()
        assert.Equal(t, "pretty", output)
    })

    t.Run("env var override", func(t *testing.T) {
        t.Setenv("VSB_OUTPUT", "json")
        output := GetDefaultOutput()
        assert.Equal(t, "json", output)
    })
}
```

#### `Save(cfg *Config) error`

```go
func TestSave(t *testing.T) {
    t.Run("saves config to file", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)

        cfg := &Config{
            APIKey:  "test-key",
            BaseURL: "https://api.example.com",
            Output:  "json",
        }

        err := Save(cfg)
        require.NoError(t, err)

        // Read back and verify
        loaded, err := Load()
        require.NoError(t, err)
        assert.Equal(t, cfg.APIKey, loaded.APIKey)
        assert.Equal(t, cfg.BaseURL, loaded.BaseURL)
    })
}
```

---

## 2.2 Keystore Package

**File to create**: `internal/config/keystore_test.go`

### Test Helpers

```go
// Helper to create a test inbox
func testStoredInbox(email string, expiresIn time.Duration) StoredInbox {
    return StoredInbox{
        Email:            email,
        InboxHash:        "hash-" + email,
        ExpiresAt:        time.Now().Add(expiresIn),
        EncryptionPubKey: []byte("pub-key"),
        EncryptionSecKey: []byte("sec-key"),
        SigningPubKey:    []byte("sign-pub"),
        SigningSecKey:    []byte("sign-sec"),
    }
}

// Helper to create keystore with temp directory
func setupKeystore(t *testing.T) (*Keystore, string) {
    dir := t.TempDir()
    t.Setenv("VSB_CONFIG_DIR", dir)
    ks, err := LoadKeystore()
    require.NoError(t, err)
    return ks, dir
}
```

### Functions to Test

#### `LoadKeystore() (*Keystore, error)`

```go
func TestLoadKeystore(t *testing.T) {
    t.Run("missing file creates empty keystore", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)

        ks, err := LoadKeystore()
        require.NoError(t, err)
        assert.Empty(t, ks.ListInboxes())
    })

    t.Run("loads existing inboxes", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)

        // Write keystore JSON manually
        data := `{"inboxes":[{"email":"test@example.com","inbox_hash":"hash123","expires_at":"2099-01-01T00:00:00Z"}],"active_inbox":"test@example.com"}`
        os.WriteFile(filepath.Join(dir, "keystore.json"), []byte(data), 0644)

        ks, err := LoadKeystore()
        require.NoError(t, err)
        assert.Len(t, ks.ListInboxes(), 1)
    })

    t.Run("prunes expired inboxes on load", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)

        // Write keystore with expired inbox
        expired := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
        data := fmt.Sprintf(`{"inboxes":[{"email":"expired@example.com","expires_at":"%s"}]}`, expired)
        os.WriteFile(filepath.Join(dir, "keystore.json"), []byte(data), 0644)

        ks, err := LoadKeystore()
        require.NoError(t, err)
        assert.Empty(t, ks.ListInboxes()) // Expired inbox pruned
    })
}
```

#### `AddInbox(inbox StoredInbox) error`

```go
func TestAddInbox(t *testing.T) {
    t.Run("adds new inbox", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        inbox := testStoredInbox("new@example.com", 24*time.Hour)

        err := ks.AddInbox(inbox)
        require.NoError(t, err)

        assert.Len(t, ks.ListInboxes(), 1)
    })

    t.Run("sets first inbox as active", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        inbox := testStoredInbox("first@example.com", 24*time.Hour)

        ks.AddInbox(inbox)

        active, err := ks.GetActiveInbox()
        require.NoError(t, err)
        assert.Equal(t, "first@example.com", active.Email)
    })

    t.Run("duplicate inbox returns error", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        inbox := testStoredInbox("dup@example.com", 24*time.Hour)

        ks.AddInbox(inbox)
        err := ks.AddInbox(inbox)

        assert.Error(t, err)
    })

    t.Run("persists to disk", func(t *testing.T) {
        ks, dir := setupKeystore(t)
        inbox := testStoredInbox("persist@example.com", 24*time.Hour)

        ks.AddInbox(inbox)

        // Reload and verify
        ks2, _ := LoadKeystore()
        assert.Len(t, ks2.ListInboxes(), 1)
        _ = dir
    })
}
```

#### `GetInbox(email string) (*StoredInbox, error)`

```go
func TestGetInbox(t *testing.T) {
    t.Run("returns inbox by exact email", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        inbox := testStoredInbox("find@example.com", 24*time.Hour)
        ks.AddInbox(inbox)

        found, err := ks.GetInbox("find@example.com")
        require.NoError(t, err)
        assert.Equal(t, "find@example.com", found.Email)
    })

    t.Run("returns error for not found", func(t *testing.T) {
        ks, _ := setupKeystore(t)

        _, err := ks.GetInbox("notfound@example.com")
        assert.ErrorIs(t, err, ErrInboxNotFound)
    })
}
```

#### `FindInbox(partial string) (*StoredInbox, int, error)`

```go
func TestFindInbox(t *testing.T) {
    t.Run("exact match takes priority", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        ks.AddInbox(testStoredInbox("abc@example.com", 24*time.Hour))
        ks.AddInbox(testStoredInbox("abcdef@example.com", 24*time.Hour))

        found, count, err := ks.FindInbox("abc@example.com")
        require.NoError(t, err)
        assert.Equal(t, 1, count)
        assert.Equal(t, "abc@example.com", found.Email)
    })

    t.Run("partial match works", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        ks.AddInbox(testStoredInbox("unique123@example.com", 24*time.Hour))

        found, count, err := ks.FindInbox("unique123")
        require.NoError(t, err)
        assert.Equal(t, 1, count)
        assert.Equal(t, "unique123@example.com", found.Email)
    })

    t.Run("multiple matches returns error", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        ks.AddInbox(testStoredInbox("test1@example.com", 24*time.Hour))
        ks.AddInbox(testStoredInbox("test2@example.com", 24*time.Hour))

        _, count, err := ks.FindInbox("test")
        assert.ErrorIs(t, err, ErrMultipleMatches)
        assert.Equal(t, 2, count)
    })

    t.Run("no match returns error", func(t *testing.T) {
        ks, _ := setupKeystore(t)

        _, _, err := ks.FindInbox("nonexistent")
        assert.ErrorIs(t, err, ErrInboxNotFound)
    })
}
```

#### `GetActiveInbox() (*StoredInbox, error)`

```go
func TestGetActiveInbox(t *testing.T) {
    t.Run("returns active inbox", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        ks.AddInbox(testStoredInbox("active@example.com", 24*time.Hour))

        active, err := ks.GetActiveInbox()
        require.NoError(t, err)
        assert.Equal(t, "active@example.com", active.Email)
    })

    t.Run("returns error when no active", func(t *testing.T) {
        ks, _ := setupKeystore(t)

        _, err := ks.GetActiveInbox()
        assert.ErrorIs(t, err, ErrNoActiveInbox)
    })
}
```

#### `SetActiveInbox(email string) error`

```go
func TestSetActiveInbox(t *testing.T) {
    t.Run("switches active inbox", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        ks.AddInbox(testStoredInbox("first@example.com", 24*time.Hour))
        ks.AddInbox(testStoredInbox("second@example.com", 24*time.Hour))

        err := ks.SetActiveInbox("second@example.com")
        require.NoError(t, err)

        active, _ := ks.GetActiveInbox()
        assert.Equal(t, "second@example.com", active.Email)
    })

    t.Run("returns error for nonexistent inbox", func(t *testing.T) {
        ks, _ := setupKeystore(t)

        err := ks.SetActiveInbox("nonexistent@example.com")
        assert.ErrorIs(t, err, ErrInboxNotFound)
    })
}
```

#### `RemoveInbox(email string) error`

```go
func TestRemoveInbox(t *testing.T) {
    t.Run("removes inbox", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        ks.AddInbox(testStoredInbox("remove@example.com", 24*time.Hour))

        err := ks.RemoveInbox("remove@example.com")
        require.NoError(t, err)

        assert.Empty(t, ks.ListInboxes())
    })

    t.Run("adjusts active when removing active inbox", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        ks.AddInbox(testStoredInbox("first@example.com", 24*time.Hour))
        ks.AddInbox(testStoredInbox("second@example.com", 24*time.Hour))
        ks.SetActiveInbox("first@example.com")

        ks.RemoveInbox("first@example.com")

        active, err := ks.GetActiveInbox()
        require.NoError(t, err)
        assert.Equal(t, "second@example.com", active.Email)
    })

    t.Run("returns error for nonexistent", func(t *testing.T) {
        ks, _ := setupKeystore(t)

        err := ks.RemoveInbox("nonexistent@example.com")
        assert.ErrorIs(t, err, ErrInboxNotFound)
    })
}
```

#### `ListInboxes() []StoredInbox`

```go
func TestListInboxes(t *testing.T) {
    t.Run("returns copy (mutation safe)", func(t *testing.T) {
        ks, _ := setupKeystore(t)
        ks.AddInbox(testStoredInbox("test@example.com", 24*time.Hour))

        list := ks.ListInboxes()
        list[0].Email = "mutated@example.com"

        // Original should be unchanged
        original := ks.ListInboxes()
        assert.Equal(t, "test@example.com", original[0].Email)
    })
}
```

### Conversion Functions

```go
func TestStoredInboxConversions(t *testing.T) {
    t.Run("StoredInboxFromExport", func(t *testing.T) {
        exported := &vaultsandbox.ExportedInbox{
            Email:     "test@example.com",
            InboxHash: "hash123",
            ExpiresAt: time.Now().Add(24 * time.Hour),
            // ... other fields
        }

        stored := StoredInboxFromExport(exported)
        assert.Equal(t, exported.Email, stored.Email)
        assert.Equal(t, exported.InboxHash, stored.InboxHash)
    })

    t.Run("ToExportedInbox roundtrip", func(t *testing.T) {
        original := testStoredInbox("roundtrip@example.com", 24*time.Hour)

        exported := original.ToExportedInbox()
        back := StoredInboxFromExport(exported)

        assert.Equal(t, original.Email, back.Email)
        assert.Equal(t, original.InboxHash, back.InboxHash)
    })
}
```

---

## 2.3 Client Package

**File to create**: `internal/config/client_test.go`

### Functions to Test

#### `NewClient() (*vaultsandbox.Client, error)`

```go
func TestNewClient(t *testing.T) {
    t.Run("returns error when no API key", func(t *testing.T) {
        dir := t.TempDir()
        t.Setenv("VSB_CONFIG_DIR", dir)
        t.Setenv("VSB_API_KEY", "")

        _, err := NewClient()
        assert.ErrorIs(t, err, ErrNoAPIKey)
    })

    t.Run("creates client with valid config", func(t *testing.T) {
        t.Setenv("VSB_API_KEY", "test-api-key")
        t.Setenv("VSB_BASE_URL", "https://api.example.com")

        client, err := NewClient()
        require.NoError(t, err)
        assert.NotNil(t, client)
        client.Close()
    })
}
```

---

## Thread Safety Tests

```go
func TestKeystoreConcurrency(t *testing.T) {
    ks, _ := setupKeystore(t)

    // Add initial inbox
    ks.AddInbox(testStoredInbox("initial@example.com", 24*time.Hour))

    var wg sync.WaitGroup
    errors := make(chan error, 100)

    // Concurrent reads
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = ks.ListInboxes()
            _, _ = ks.GetActiveInbox()
        }()
    }

    // Concurrent writes
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            email := fmt.Sprintf("concurrent%d@example.com", n)
            if err := ks.AddInbox(testStoredInbox(email, 24*time.Hour)); err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Errorf("concurrent operation failed: %v", err)
    }
}
```

---

## Checklist

- [ ] Create `internal/config/config_test.go`
- [ ] Create `internal/config/keystore_test.go`
- [ ] Create `internal/config/client_test.go`
- [ ] Run `go test -race ./internal/config/...` (race detection)
- [ ] Verify coverage increase

## Commands

```bash
# Run Phase 2 tests with race detection
go test -race -v ./internal/config/...

# Check coverage
go test -coverprofile=phase2.out ./internal/config/...
go tool cover -func=phase2.out
```
