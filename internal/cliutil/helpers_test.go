package cliutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

func TestGetArg(t *testing.T) {
	t.Run("returns value at index", func(t *testing.T) {
		args := []string{"first", "second", "third"}
		assert.Equal(t, "first", GetArg(args, 0, "default"))
		assert.Equal(t, "second", GetArg(args, 1, "default"))
		assert.Equal(t, "third", GetArg(args, 2, "default"))
	})

	t.Run("returns default when index out of range", func(t *testing.T) {
		args := []string{"first"}
		assert.Equal(t, "default", GetArg(args, 1, "default"))
		assert.Equal(t, "default", GetArg(args, 5, "default"))
	})

	t.Run("returns default for empty slice", func(t *testing.T) {
		var args []string
		assert.Equal(t, "default", GetArg(args, 0, "default"))
	})

	t.Run("returns empty default when specified", func(t *testing.T) {
		var args []string
		assert.Equal(t, "", GetArg(args, 0, ""))
	})
}

func TestGetInbox(t *testing.T) {
	inbox1 := config.StoredInbox{Email: "test1@example.com"}
	inbox2 := config.StoredInbox{Email: "test2@example.com"}

	t.Run("empty flag returns active inbox", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes:     []config.StoredInbox{inbox1, inbox2},
			ActiveEmail: "test1@example.com",
		}

		result, err := GetInbox(ks, "")
		require.NoError(t, err)
		assert.Equal(t, "test1@example.com", result.Email)
	})

	t.Run("empty flag with no active returns error", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes:     []config.StoredInbox{inbox1},
			ActiveEmail: "",
		}

		_, err := GetInbox(ks, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active inbox")
	})

	t.Run("exact email match", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes: []config.StoredInbox{inbox1, inbox2},
		}

		result, err := GetInbox(ks, "test2@example.com")
		require.NoError(t, err)
		assert.Equal(t, "test2@example.com", result.Email)
	})

	t.Run("partial match", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes: []config.StoredInbox{
				{Email: "unique123@example.com"},
			},
		}

		result, err := GetInbox(ks, "unique123")
		require.NoError(t, err)
		assert.Equal(t, "unique123@example.com", result.Email)
	})

	t.Run("multiple matches returns error", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes: []config.StoredInbox{inbox1, inbox2},
		}

		_, err := GetInbox(ks, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "multiple inboxes match")
	})

	t.Run("no match returns error", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes: []config.StoredInbox{inbox1},
		}

		_, err := GetInbox(ks, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestMockKeystoreGetActiveInbox(t *testing.T) {
	t.Run("returns active inbox when set", func(t *testing.T) {
		inbox := config.StoredInbox{Email: "active@example.com"}
		ks := &MockKeystore{
			Inboxes:     []config.StoredInbox{inbox},
			ActiveEmail: "active@example.com",
		}

		result, err := ks.GetActiveInbox()
		require.NoError(t, err)
		assert.Equal(t, "active@example.com", result.Email)
	})

	t.Run("returns error when no active", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes:     []config.StoredInbox{{Email: "test@example.com"}},
			ActiveEmail: "",
		}

		_, err := ks.GetActiveInbox()
		assert.ErrorIs(t, err, config.ErrNoActiveInbox)
	})

	t.Run("uses override function when provided", func(t *testing.T) {
		customInbox := &config.StoredInbox{Email: "custom@example.com"}
		ks := &MockKeystore{
			GetActiveInboxFunc: func() (*config.StoredInbox, error) {
				return customInbox, nil
			},
		}

		result, err := ks.GetActiveInbox()
		require.NoError(t, err)
		assert.Equal(t, "custom@example.com", result.Email)
	})
}

func TestMockKeystoreFindInbox(t *testing.T) {
	t.Run("exact match returns immediately", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes: []config.StoredInbox{
				{Email: "exact@example.com"},
				{Email: "exactmore@example.com"},
			},
		}

		result, matches, err := ks.FindInbox("exact@example.com")
		require.NoError(t, err)
		assert.Nil(t, matches)
		assert.Equal(t, "exact@example.com", result.Email)
	})

	t.Run("returns multiple match emails", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes: []config.StoredInbox{
				{Email: "test1@example.com"},
				{Email: "test2@example.com"},
			},
		}

		_, matches, err := ks.FindInbox("test")
		assert.ErrorIs(t, err, config.ErrMultipleMatches)
		assert.Contains(t, matches, "test1@example.com")
		assert.Contains(t, matches, "test2@example.com")
	})
}

func TestMockKeystoreListInboxes(t *testing.T) {
	inboxes := []config.StoredInbox{
		{Email: "inbox1@example.com"},
		{Email: "inbox2@example.com"},
	}
	ks := &MockKeystore{Inboxes: inboxes}

	result := ks.ListInboxes()
	assert.Len(t, result, 2)
	assert.Equal(t, "inbox1@example.com", result[0].Email)
	assert.Equal(t, "inbox2@example.com", result[1].Email)
}

func TestMockKeystoreSetActiveInbox(t *testing.T) {
	t.Run("sets active inbox when exists", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes: []config.StoredInbox{
				{Email: "test@example.com"},
			},
		}

		err := ks.SetActiveInbox("test@example.com")
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", ks.ActiveEmail)
	})

	t.Run("returns error for non-existent inbox", func(t *testing.T) {
		ks := &MockKeystore{
			Inboxes: []config.StoredInbox{
				{Email: "test@example.com"},
			},
		}

		err := ks.SetActiveInbox("nonexistent@example.com")
		assert.ErrorIs(t, err, config.ErrInboxNotFound)
	})

	t.Run("uses override function when provided", func(t *testing.T) {
		customErr := config.ErrMultipleMatches
		ks := &MockKeystore{
			SetActiveInboxFunc: func(email string) error {
				return customErr
			},
		}

		err := ks.SetActiveInbox("any@example.com")
		assert.ErrorIs(t, err, customErr)
	})
}
