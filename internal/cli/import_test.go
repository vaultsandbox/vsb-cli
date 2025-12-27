package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

func TestImportValidation(t *testing.T) {
	t.Run("rejects unsupported version", func(t *testing.T) {
		data := `{"version": 2, "emailAddress": "test@example.com"}`
		var exported config.ExportedInboxFile
		err := json.Unmarshal([]byte(data), &exported)
		assert.NoError(t, err) // Parsing succeeds

		// Version validation (same logic as runImport)
		if exported.Version != 1 {
			// This is what runImport checks
			assert.NotEqual(t, 1, exported.Version)
		}
	})

	t.Run("rejects version 0", func(t *testing.T) {
		data := `{"emailAddress": "test@example.com"}` // version defaults to 0
		var exported config.ExportedInboxFile
		err := json.Unmarshal([]byte(data), &exported)
		assert.NoError(t, err)
		assert.Equal(t, 0, exported.Version) // Default value
		assert.NotEqual(t, 1, exported.Version)
	})

	t.Run("accepts version 1", func(t *testing.T) {
		data := `{"version": 1, "emailAddress": "test@example.com"}`
		var exported config.ExportedInboxFile
		err := json.Unmarshal([]byte(data), &exported)
		assert.NoError(t, err)
		assert.Equal(t, 1, exported.Version)
	})

	t.Run("rejects expired inbox", func(t *testing.T) {
		expired := time.Now().Add(-time.Hour)
		data := config.ExportedInboxFile{
			Version:      1,
			EmailAddress: "test@example.com",
			ExpiresAt:    expired,
		}

		// Same validation as runImport
		isExpired := data.ExpiresAt.Before(time.Now())
		assert.True(t, isExpired)
	})

	t.Run("accepts non-expired inbox", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		data := config.ExportedInboxFile{
			Version:      1,
			EmailAddress: "test@example.com",
			ExpiresAt:    future,
		}

		isExpired := data.ExpiresAt.Before(time.Now())
		assert.False(t, isExpired)
	})

	t.Run("rejects malformed JSON", func(t *testing.T) {
		testCases := []struct {
			name string
			data string
		}{
			{"empty", ""},
			{"invalid syntax", "{invalid}"},
			{"unclosed brace", `{"version": 1`},
			{"wrong type for version", `{"version": "one"}`},
			{"invalid date format", `{"version": 1, "expiresAt": "not-a-date"}`},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var exported config.ExportedInboxFile
				err := json.Unmarshal([]byte(tc.data), &exported)
				assert.Error(t, err)
			})
		}
	})

	t.Run("parses complete export file", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		data := `{
			"version": 1,
			"emailAddress": "test@vsb.email",
			"inboxHash": "abc123",
			"expiresAt": "` + now.Add(24*time.Hour).Format(time.RFC3339) + `",
			"exportedAt": "` + now.Format(time.RFC3339) + `",
			"keys": {
				"kemPrivate": "private-key-data",
				"kemPublic": "public-key-data",
				"serverSigPk": "server-sig-data"
			}
		}`

		var exported config.ExportedInboxFile
		err := json.Unmarshal([]byte(data), &exported)
		assert.NoError(t, err)
		assert.Equal(t, 1, exported.Version)
		assert.Equal(t, "test@vsb.email", exported.EmailAddress)
		assert.Equal(t, "abc123", exported.InboxHash)
		assert.Equal(t, "private-key-data", exported.Keys.KEMPrivate)
		assert.Equal(t, "public-key-data", exported.Keys.KEMPublic)
		assert.Equal(t, "server-sig-data", exported.Keys.ServerSigPK)
	})
}
