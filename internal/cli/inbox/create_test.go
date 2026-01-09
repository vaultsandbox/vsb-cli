package inbox

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// mockInbox implements ExportableInbox for testing
type mockInbox struct {
	exported *vaultsandbox.ExportedInbox
}

func (m *mockInbox) Export() *vaultsandbox.ExportedInbox {
	return m.exported
}

// mockClient implements InboxCreator for testing
type mockClient struct {
	inbox     ExportableInbox
	createErr error
	closed    bool
}

func (m *mockClient) CreateInbox(ctx context.Context, opts ...vaultsandbox.InboxOption) (ExportableInbox, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.inbox, nil
}

func (m *mockClient) Close() error {
	m.closed = true
	return nil
}

// mockKeystore implements KeystoreWriter for testing
type mockKeystore struct {
	addedInbox *config.StoredInbox
	addErr     error
}

func (m *mockKeystore) AddInbox(inbox config.StoredInbox) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.addedInbox = &inbox
	return nil
}

// captureCreateStdout captures stdout during function execution
func captureCreateStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	return buf.String()
}

// createTestCommand creates a test cobra command with the output flag
func createTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "test",
		RunE: runCreate,
	}
	cmd.Flags().StringP("output", "o", "", "Output format")
	return cmd
}

// resetCreateTestState resets global state after each test
func resetCreateTestState(oldClientFunc func() (InboxCreator, error), oldKeystoreFunc func() (KeystoreWriter, error), oldTTL string) {
	newClientFunc = oldClientFunc
	loadKeystoreFunc = oldKeystoreFunc
	createTTL = oldTTL
}

func TestParseTTL(t *testing.T) {
	t.Run("parses hours", func(t *testing.T) {
		d, err := parseTTL("1h")
		require.NoError(t, err)
		assert.Equal(t, time.Hour, d)
	})

	t.Run("parses multiple hours", func(t *testing.T) {
		d, err := parseTTL("24h")
		require.NoError(t, err)
		assert.Equal(t, 24*time.Hour, d)
	})

	t.Run("parses minutes", func(t *testing.T) {
		d, err := parseTTL("30m")
		require.NoError(t, err)
		assert.Equal(t, 30*time.Minute, d)
	})

	t.Run("parses days", func(t *testing.T) {
		d, err := parseTTL("7d")
		require.NoError(t, err)
		assert.Equal(t, 7*24*time.Hour, d)
	})

	t.Run("parses single day", func(t *testing.T) {
		d, err := parseTTL("1d")
		require.NoError(t, err)
		assert.Equal(t, 24*time.Hour, d)
	})

	t.Run("parses complex duration", func(t *testing.T) {
		d, err := parseTTL("1h30m")
		require.NoError(t, err)
		assert.Equal(t, time.Hour+30*time.Minute, d)
	})

	t.Run("returns error for invalid day value", func(t *testing.T) {
		_, err := parseTTL("abcd")
		require.Error(t, err)
	})

	t.Run("returns error for invalid day number", func(t *testing.T) {
		_, err := parseTTL("xyzd")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid day value")
	})

	t.Run("returns error for empty string", func(t *testing.T) {
		_, err := parseTTL("")
		require.Error(t, err)
	})
}

func TestPrintInboxCreated(t *testing.T) {
	t.Run("prints inbox information", func(t *testing.T) {
		inbox := config.StoredInbox{
			Email:     "test@example.vaultsandbox.com",
			ID:        "inbox-123",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		output := captureCreateStdout(t, func() {
			printInboxCreated(inbox)
		})

		assert.Contains(t, output, "Inbox Ready!")
		assert.Contains(t, output, "test@example.vaultsandbox.com")
		assert.Contains(t, output, "Expires")
	})

	t.Run("prints with short expiry", func(t *testing.T) {
		inbox := config.StoredInbox{
			Email:     "short@example.com",
			ID:        "inbox-short",
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		output := captureCreateStdout(t, func() {
			printInboxCreated(inbox)
		})

		assert.Contains(t, output, "Inbox Ready!")
		assert.Contains(t, output, "short@example.com")
	})
}

func TestRunCreate(t *testing.T) {
	t.Run("creates inbox successfully", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "24h"

		mockKS := &mockKeystore{}
		mockInb := &mockInbox{
			exported: &vaultsandbox.ExportedInbox{
				Version:      1,
				EmailAddress: "test@example.vaultsandbox.com",
				InboxHash:    "hash123",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				ExportedAt:   time.Now(),
				SecretKey:    "secret-key-base64",
				ServerSigPk:  "server-sig-pk",
			},
		}
		mockCl := &mockClient{inbox: mockInb}

		newClientFunc = func() (InboxCreator, error) {
			return mockCl, nil
		}
		loadKeystoreFunc = func() (KeystoreWriter, error) {
			return mockKS, nil
		}

		cmd := createTestCommand()
		output := captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.NoError(t, err)
		})

		assert.True(t, mockCl.closed)
		assert.NotNil(t, mockKS.addedInbox)
		assert.Equal(t, "test@example.vaultsandbox.com", mockKS.addedInbox.Email)
		assert.Contains(t, output, "Inbox Ready!")
	})

	t.Run("outputs JSON when requested", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "24h"

		mockKS := &mockKeystore{}
		expiresAt := time.Now().Add(24 * time.Hour)
		createdAt := time.Now()
		mockInb := &mockInbox{
			exported: &vaultsandbox.ExportedInbox{
				Version:      1,
				EmailAddress: "json@example.vaultsandbox.com",
				InboxHash:    "hash456",
				ExpiresAt:    expiresAt,
				ExportedAt:   createdAt,
				SecretKey:    "secret-key",
				ServerSigPk:  "server-sig",
			},
		}
		mockCl := &mockClient{inbox: mockInb}

		newClientFunc = func() (InboxCreator, error) {
			return mockCl, nil
		}
		loadKeystoreFunc = func() (KeystoreWriter, error) {
			return mockKS, nil
		}

		cmd := createTestCommand()
		cmd.Flags().Set("output", "json")

		output := captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "json@example.vaultsandbox.com")
		assert.Contains(t, output, "email")
		assert.Contains(t, output, "expiresAt")
		assert.Contains(t, output, "createdAt")
		assert.NotContains(t, output, "Inbox Ready!")
	})

	t.Run("returns error for invalid TTL", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "invalidttl"

		cmd := createTestCommand()
		err := runCreate(cmd, []string{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid TTL format")
	})

	t.Run("returns error when client creation fails", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "24h"
		newClientFunc = func() (InboxCreator, error) {
			return nil, errors.New("no API key configured")
		}

		cmd := createTestCommand()
		captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no API key configured")
		})
	})

	t.Run("returns error when inbox creation fails", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "24h"
		mockCl := &mockClient{createErr: errors.New("server error")}

		newClientFunc = func() (InboxCreator, error) {
			return mockCl, nil
		}

		cmd := createTestCommand()
		captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create inbox")
		})

		assert.True(t, mockCl.closed)
	})

	t.Run("returns error when keystore load fails", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "24h"

		mockInb := &mockInbox{
			exported: &vaultsandbox.ExportedInbox{
				Version:      1,
				EmailAddress: "test@example.com",
				InboxHash:    "hash",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				ExportedAt:   time.Now(),
				SecretKey:    "key",
				ServerSigPk:  "sig",
			},
		}
		mockCl := &mockClient{inbox: mockInb}

		newClientFunc = func() (InboxCreator, error) {
			return mockCl, nil
		}
		loadKeystoreFunc = func() (KeystoreWriter, error) {
			return nil, errors.New("keystore corrupted")
		}

		cmd := createTestCommand()
		captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "keystore corrupted")
		})
	})

	t.Run("returns error when inbox save fails", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "24h"

		mockKS := &mockKeystore{addErr: errors.New("disk full")}
		mockInb := &mockInbox{
			exported: &vaultsandbox.ExportedInbox{
				Version:      1,
				EmailAddress: "test@example.com",
				InboxHash:    "hash",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				ExportedAt:   time.Now(),
				SecretKey:    "key",
				ServerSigPk:  "sig",
			},
		}
		mockCl := &mockClient{inbox: mockInb}

		newClientFunc = func() (InboxCreator, error) {
			return mockCl, nil
		}
		loadKeystoreFunc = func() (KeystoreWriter, error) {
			return mockKS, nil
		}

		cmd := createTestCommand()
		captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to save inbox")
		})
	})

	t.Run("shows progress messages in non-JSON mode", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "24h"

		mockKS := &mockKeystore{}
		mockInb := &mockInbox{
			exported: &vaultsandbox.ExportedInbox{
				Version:      1,
				EmailAddress: "test@example.com",
				InboxHash:    "hash",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				ExportedAt:   time.Now(),
				SecretKey:    "key",
				ServerSigPk:  "sig",
			},
		}
		mockCl := &mockClient{inbox: mockInb}

		newClientFunc = func() (InboxCreator, error) {
			return mockCl, nil
		}
		loadKeystoreFunc = func() (KeystoreWriter, error) {
			return mockKS, nil
		}

		cmd := createTestCommand()
		output := captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.NoError(t, err)
		})

		assert.Contains(t, output, "Generating keys")
		assert.Contains(t, output, "Registering with VaultSandbox")
	})

	t.Run("hides progress messages in JSON mode", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "24h"

		mockKS := &mockKeystore{}
		mockInb := &mockInbox{
			exported: &vaultsandbox.ExportedInbox{
				Version:      1,
				EmailAddress: "test@example.com",
				InboxHash:    "hash",
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				ExportedAt:   time.Now(),
				SecretKey:    "key",
				ServerSigPk:  "sig",
			},
		}
		mockCl := &mockClient{inbox: mockInb}

		newClientFunc = func() (InboxCreator, error) {
			return mockCl, nil
		}
		loadKeystoreFunc = func() (KeystoreWriter, error) {
			return mockKS, nil
		}

		cmd := createTestCommand()
		cmd.Flags().Set("output", "json")

		output := captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.NoError(t, err)
		})

		assert.NotContains(t, output, "Generating keys")
		assert.NotContains(t, output, "Registering with VaultSandbox")
	})

	t.Run("uses custom TTL", func(t *testing.T) {
		oldClientFunc := newClientFunc
		oldKeystoreFunc := loadKeystoreFunc
		oldTTL := createTTL
		defer resetCreateTestState(oldClientFunc, oldKeystoreFunc, oldTTL)

		createTTL = "7d"

		mockKS := &mockKeystore{}
		mockInb := &mockInbox{
			exported: &vaultsandbox.ExportedInbox{
				Version:      1,
				EmailAddress: "test@example.com",
				InboxHash:    "hash",
				ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
				ExportedAt:   time.Now(),
				SecretKey:    "key",
				ServerSigPk:  "sig",
			},
		}
		mockCl := &mockClient{inbox: mockInb}

		newClientFunc = func() (InboxCreator, error) {
			return mockCl, nil
		}
		loadKeystoreFunc = func() (KeystoreWriter, error) {
			return mockKS, nil
		}

		cmd := createTestCommand()
		captureCreateStdout(t, func() {
			err := runCreate(cmd, []string{})
			require.NoError(t, err)
		})

		assert.NotNil(t, mockKS.addedInbox)
	})
}
