package inbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// TestInboxUseWorkflow tests the inbox use workflow:
// GetInbox (find/match) -> SetActiveInbox (persist)
func TestInboxUseWorkflow(t *testing.T) {
	inbox1 := config.StoredInbox{Email: "test1@example.com"}
	inbox2 := config.StoredInbox{Email: "test2@example.com"}
	uniqueInbox := config.StoredInbox{Email: "unique123@example.com"}

	t.Run("sets active inbox by exact match", func(t *testing.T) {
		ks := &cliutil.MockKeystore{
			Inboxes: []config.StoredInbox{inbox1, inbox2},
		}

		// Simulate the workflow in runInboxUse
		inbox, err := cliutil.GetInbox(ks, "test1@example.com")
		require.NoError(t, err)

		err = ks.SetActiveInbox(inbox.Email)
		require.NoError(t, err)

		assert.Equal(t, "test1@example.com", ks.ActiveEmail)
	})

	t.Run("sets active inbox by partial match", func(t *testing.T) {
		ks := &cliutil.MockKeystore{
			Inboxes: []config.StoredInbox{uniqueInbox},
		}

		inbox, err := cliutil.GetInbox(ks, "unique123")
		require.NoError(t, err)

		err = ks.SetActiveInbox(inbox.Email)
		require.NoError(t, err)

		assert.Equal(t, "unique123@example.com", ks.ActiveEmail)
	})

	t.Run("errors on no match", func(t *testing.T) {
		ks := &cliutil.MockKeystore{
			Inboxes: []config.StoredInbox{inbox1},
		}

		_, err := cliutil.GetInbox(ks, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("errors on multiple matches", func(t *testing.T) {
		ks := &cliutil.MockKeystore{
			Inboxes: []config.StoredInbox{inbox1, inbox2},
		}

		_, err := cliutil.GetInbox(ks, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "multiple inboxes match")
	})

	t.Run("preserves active after workflow", func(t *testing.T) {
		ks := &cliutil.MockKeystore{
			Inboxes:     []config.StoredInbox{inbox1, inbox2},
			ActiveEmail: "test1@example.com",
		}

		// Switch to inbox2
		inbox, err := cliutil.GetInbox(ks, "test2@example.com")
		require.NoError(t, err)

		err = ks.SetActiveInbox(inbox.Email)
		require.NoError(t, err)

		// Verify switch happened
		assert.Equal(t, "test2@example.com", ks.ActiveEmail)

		// Verify we can get the new active
		active, err := ks.GetActiveInbox()
		require.NoError(t, err)
		assert.Equal(t, "test2@example.com", active.Email)
	})
}
