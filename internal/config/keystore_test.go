package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	vaultsandbox "github.com/vaultsandbox/client-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a test inbox
func testStoredInbox(email string, expiresIn time.Duration) StoredInbox {
	return StoredInbox{
		Email:     email,
		ID:        "hash-" + email,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(expiresIn),
		Keys: InboxKeys{
			KEMPrivate:  "priv-key",
			KEMPublic:   "pub-key",
			ServerSigPK: "server-sig",
		},
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
		data := `{"inboxes":[{"email":"test@example.com","id":"hash123","expiresAt":"2099-01-01T00:00:00Z"}],"active_inbox":"test@example.com"}`
		err := os.WriteFile(filepath.Join(dir, "keystore.json"), []byte(data), 0644)
		require.NoError(t, err)

		ks, err := LoadKeystore()
		require.NoError(t, err)
		assert.Len(t, ks.ListInboxes(), 1)
	})

	t.Run("prunes expired inboxes on load", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		// Write keystore with expired inbox
		expired := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		data := fmt.Sprintf(`{"inboxes":[{"email":"expired@example.com","expiresAt":"%s"}]}`, expired)
		err := os.WriteFile(filepath.Join(dir, "keystore.json"), []byte(data), 0644)
		require.NoError(t, err)

		ks, err := LoadKeystore()
		require.NoError(t, err)
		assert.Empty(t, ks.ListInboxes()) // Expired inbox pruned
	})

	t.Run("prunes expired active inbox and switches to remaining", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		// Write keystore with expired active inbox and one valid inbox
		expired := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		valid := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
		data := fmt.Sprintf(`{
			"active_inbox": "expired@example.com",
			"inboxes":[
				{"email":"expired@example.com","expiresAt":"%s"},
				{"email":"valid@example.com","expiresAt":"%s"}
			]
		}`, expired, valid)
		err := os.WriteFile(filepath.Join(dir, "keystore.json"), []byte(data), 0644)
		require.NoError(t, err)

		ks, err := LoadKeystore()
		require.NoError(t, err)
		assert.Len(t, ks.ListInboxes(), 1)
		assert.Equal(t, "valid@example.com", ks.ActiveInbox) // Should switch to valid inbox
	})

	t.Run("prunes all expired inboxes clears active", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		// Write keystore with all expired inboxes
		expired := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		data := fmt.Sprintf(`{
			"active_inbox": "expired1@example.com",
			"inboxes":[
				{"email":"expired1@example.com","expiresAt":"%s"},
				{"email":"expired2@example.com","expiresAt":"%s"}
			]
		}`, expired, expired)
		err := os.WriteFile(filepath.Join(dir, "keystore.json"), []byte(data), 0644)
		require.NoError(t, err)

		ks, err := LoadKeystore()
		require.NoError(t, err)
		assert.Empty(t, ks.ListInboxes())
		assert.Empty(t, ks.ActiveInbox) // Should be cleared
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VSB_CONFIG_DIR", dir)

		err := os.WriteFile(filepath.Join(dir, "keystore.json"), []byte("invalid json{"), 0644)
		require.NoError(t, err)

		_, err = LoadKeystore()
		assert.Error(t, err)
	})
}

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

	t.Run("replaces existing inbox with same email", func(t *testing.T) {
		ks, _ := setupKeystore(t)
		inbox1 := testStoredInbox("dup@example.com", 24*time.Hour)
		inbox1.ID = "first-hash"

		inbox2 := testStoredInbox("dup@example.com", 48*time.Hour)
		inbox2.ID = "second-hash"

		ks.AddInbox(inbox1)
		err := ks.AddInbox(inbox2)
		require.NoError(t, err)

		// Should only have one inbox
		assert.Len(t, ks.ListInboxes(), 1)
		// Should be the second version
		found, err := ks.GetInbox("dup@example.com")
		require.NoError(t, err)
		assert.Equal(t, "second-hash", found.ID)
	})

	t.Run("persists to disk", func(t *testing.T) {
		ks, dir := setupKeystore(t)
		inbox := testStoredInbox("persist@example.com", 24*time.Hour)

		ks.AddInbox(inbox)

		// Reload and verify
		ks2, err := LoadKeystore()
		require.NoError(t, err)
		assert.Len(t, ks2.ListInboxes(), 1)
		_ = dir
	})
}

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

func TestFindInbox(t *testing.T) {
	t.Run("exact match takes priority", func(t *testing.T) {
		ks, _ := setupKeystore(t)
		ks.AddInbox(testStoredInbox("abc@example.com", 24*time.Hour))
		ks.AddInbox(testStoredInbox("abcdef@example.com", 24*time.Hour))

		found, matches, err := ks.FindInbox("abc@example.com")
		require.NoError(t, err)
		assert.Nil(t, matches)
		assert.Equal(t, "abc@example.com", found.Email)
	})

	t.Run("partial match works", func(t *testing.T) {
		ks, _ := setupKeystore(t)
		ks.AddInbox(testStoredInbox("unique123@example.com", 24*time.Hour))

		found, matches, err := ks.FindInbox("unique123")
		require.NoError(t, err)
		assert.Nil(t, matches)
		assert.Equal(t, "unique123@example.com", found.Email)
	})

	t.Run("multiple matches returns error", func(t *testing.T) {
		ks, _ := setupKeystore(t)
		ks.AddInbox(testStoredInbox("test1@example.com", 24*time.Hour))
		ks.AddInbox(testStoredInbox("test2@example.com", 24*time.Hour))

		_, matches, err := ks.FindInbox("test")
		assert.ErrorIs(t, err, ErrMultipleMatches)
		assert.Len(t, matches, 2)
	})

	t.Run("no match returns error", func(t *testing.T) {
		ks, _ := setupKeystore(t)

		_, _, err := ks.FindInbox("nonexistent")
		assert.ErrorIs(t, err, ErrInboxNotFound)
	})
}

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

func TestSetActiveInbox(t *testing.T) {
	t.Run("switches active inbox", func(t *testing.T) {
		ks, _ := setupKeystore(t)
		ks.AddInbox(testStoredInbox("first@example.com", 24*time.Hour))
		ks.AddInbox(testStoredInbox("second@example.com", 24*time.Hour))

		err := ks.SetActiveInbox("first@example.com")
		require.NoError(t, err)

		active, _ := ks.GetActiveInbox()
		assert.Equal(t, "first@example.com", active.Email)
	})

	t.Run("returns error for nonexistent inbox", func(t *testing.T) {
		ks, _ := setupKeystore(t)

		err := ks.SetActiveInbox("nonexistent@example.com")
		assert.ErrorIs(t, err, ErrInboxNotFound)
	})

	t.Run("persists active inbox", func(t *testing.T) {
		ks, _ := setupKeystore(t)
		ks.AddInbox(testStoredInbox("first@example.com", 24*time.Hour))
		ks.AddInbox(testStoredInbox("second@example.com", 24*time.Hour))

		ks.SetActiveInbox("first@example.com")

		// Reload and verify
		ks2, err := LoadKeystore()
		require.NoError(t, err)
		active, _ := ks2.GetActiveInbox()
		assert.Equal(t, "first@example.com", active.Email)
	})
}

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

	t.Run("clears active when removing last inbox", func(t *testing.T) {
		ks, _ := setupKeystore(t)
		ks.AddInbox(testStoredInbox("only@example.com", 24*time.Hour))

		ks.RemoveInbox("only@example.com")

		_, err := ks.GetActiveInbox()
		assert.ErrorIs(t, err, ErrNoActiveInbox)
	})

	t.Run("returns error for nonexistent", func(t *testing.T) {
		ks, _ := setupKeystore(t)

		err := ks.RemoveInbox("nonexistent@example.com")
		assert.ErrorIs(t, err, ErrInboxNotFound)
	})
}

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

	t.Run("returns empty slice for empty keystore", func(t *testing.T) {
		ks, _ := setupKeystore(t)

		list := ks.ListInboxes()
		assert.NotNil(t, list)
		assert.Empty(t, list)
	})
}

func TestStoredInboxConversions(t *testing.T) {
	t.Run("StoredInboxFromExport", func(t *testing.T) {
		exported := &vaultsandbox.ExportedInbox{
			Version:      1,
			EmailAddress: "test@example.com",
			InboxHash:    "hash123",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			ExportedAt:   time.Now(),
			SecretKey:    "sec-key-b64",
			ServerSigPk:  "server-sig",
		}

		stored := StoredInboxFromExport(exported)
		assert.Equal(t, exported.EmailAddress, stored.Email)
		assert.Equal(t, exported.InboxHash, stored.ID)
		assert.Empty(t, stored.Keys.KEMPublic) // Public key derived from secret key per spec
		assert.Equal(t, exported.SecretKey, stored.Keys.KEMPrivate)
		assert.Equal(t, exported.ServerSigPk, stored.Keys.ServerSigPK)
	})

	t.Run("ToExportedInbox roundtrip", func(t *testing.T) {
		original := testStoredInbox("roundtrip@example.com", 24*time.Hour)

		exported := original.ToExportedInbox()
		back := StoredInboxFromExport(exported)

		assert.Equal(t, original.Email, back.Email)
		assert.Equal(t, original.ID, back.ID)
		// Public key is not preserved in SDK format (derived from secret key per spec)
		assert.Empty(t, back.Keys.KEMPublic)
		assert.Equal(t, original.Keys.KEMPrivate, back.Keys.KEMPrivate)
	})
}

func TestExportedInboxFile(t *testing.T) {
	t.Run("ToExportFile", func(t *testing.T) {
		stored := testStoredInbox("export@example.com", 24*time.Hour)

		exportFile := stored.ToExportFile()

		assert.Equal(t, 1, exportFile.Version)
		assert.Equal(t, stored.Email, exportFile.EmailAddress)
		assert.Equal(t, stored.ID, exportFile.InboxHash)
		assert.Equal(t, stored.Keys.KEMPrivate, exportFile.Keys.KEMPrivate)
		assert.Equal(t, stored.Keys.KEMPublic, exportFile.Keys.KEMPublic)
	})

	t.Run("ToStoredInbox", func(t *testing.T) {
		exportFile := ExportedInboxFile{
			Version:      1,
			EmailAddress: "import@example.com",
			InboxHash:    "hash-import",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			ExportedAt:   time.Now(),
			Keys: ExportedKeys{
				KEMPrivate:  "priv",
				KEMPublic:   "pub",
				ServerSigPK: "sig",
			},
		}

		stored := exportFile.ToStoredInbox()

		assert.Equal(t, exportFile.EmailAddress, stored.Email)
		assert.Equal(t, exportFile.InboxHash, stored.ID)
		assert.Equal(t, exportFile.Keys.KEMPrivate, stored.Keys.KEMPrivate)
	})

	t.Run("ExportFile roundtrip", func(t *testing.T) {
		original := testStoredInbox("roundtrip@example.com", 24*time.Hour)

		exportFile := original.ToExportFile()
		back := exportFile.ToStoredInbox()

		assert.Equal(t, original.Email, back.Email)
		assert.Equal(t, original.ID, back.ID)
	})
}

func TestKeystoreSave(t *testing.T) {
	t.Run("saves keystore to disk", func(t *testing.T) {
		ks, dir := setupKeystore(t)
		ks.AddInbox(testStoredInbox("save@example.com", 24*time.Hour))

		err := ks.Save()
		require.NoError(t, err)

		// Verify file exists
		path := filepath.Join(dir, "keystore.json")
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var loaded Keystore
		err = json.Unmarshal(data, &loaded)
		require.NoError(t, err)
		assert.Len(t, loaded.Inboxes, 1)
	})
}

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

func TestSaveInbox(t *testing.T) {
	t.Run("saves exported inbox to keystore", func(t *testing.T) {
		ks, _ := setupKeystore(t)

		exported := &vaultsandbox.ExportedInbox{
			Version:      1,
			EmailAddress: "sdk@example.com",
			InboxHash:    "sdk-hash",
			ExpiresAt:    time.Now().Add(24 * time.Hour),
			ExportedAt:   time.Now(),
			SecretKey:    "sec",
			ServerSigPk:  "sig",
		}

		err := ks.SaveInbox(exported)
		require.NoError(t, err)

		found, err := ks.GetInbox("sdk@example.com")
		require.NoError(t, err)
		assert.Equal(t, "sdk-hash", found.ID)
	})
}
