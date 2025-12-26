package emails

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vaultsandbox "github.com/vaultsandbox/client-go"
)

func TestUpdateEmailReceived(t *testing.T) {
	t.Run("adds new email to empty list", func(t *testing.T) {
		m := testModel([]EmailItem{})
		newEmail := testEmail("new-1", "New Email", "sender@example.com")

		newModel, _ := m.Update(emailReceivedMsg{email: newEmail, inboxLabel: "inbox@test.com"})

		updated := newModel.(Model)
		assert.Len(t, updated.emails, 1)
		assert.Equal(t, "new-1", updated.emails[0].Email.ID)
		assert.Equal(t, "inbox@test.com", updated.emails[0].InboxLabel)
	})

	t.Run("adds new email to front of list", func(t *testing.T) {
		existing := []EmailItem{
			testEmailItem("1", "Existing", "old@example.com", "inbox"),
		}
		m := testModel(existing)
		newEmail := testEmail("2", "New Email", "new@example.com")

		newModel, _ := m.Update(emailReceivedMsg{email: newEmail, inboxLabel: "inbox"})

		updated := newModel.(Model)
		assert.Len(t, updated.emails, 2)
		assert.Equal(t, "2", updated.emails[0].Email.ID) // New email at front
		assert.Equal(t, "1", updated.emails[1].Email.ID) // Old email at back
	})

	t.Run("does not add duplicate email", func(t *testing.T) {
		existing := []EmailItem{
			testEmailItem("1", "Existing", "old@example.com", "inbox"),
		}
		m := testModel(existing)
		duplicateEmail := testEmail("1", "Duplicate", "old@example.com")

		newModel, _ := m.Update(emailReceivedMsg{email: duplicateEmail, inboxLabel: "inbox"})

		updated := newModel.(Model)
		assert.Len(t, updated.emails, 1)
		assert.Equal(t, "Existing", updated.emails[0].Email.Subject)
	})
}

func TestUpdateErrorMsg(t *testing.T) {
	t.Run("sets lastError", func(t *testing.T) {
		m := testModel([]EmailItem{})
		testErr := errors.New("connection failed")

		newModel, _ := m.Update(errMsg{err: testErr})

		updated := newModel.(Model)
		assert.Equal(t, testErr, updated.lastError)
	})

	t.Run("sets connected to false", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.connected = true

		newModel, _ := m.Update(errMsg{err: errors.New("error")})

		updated := newModel.(Model)
		assert.False(t, updated.connected)
	})
}

func TestUpdateConnectedMsg(t *testing.T) {
	t.Run("sets connected to true", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.connected = false

		newModel, _ := m.Update(connectedMsg{})

		updated := newModel.(Model)
		assert.True(t, updated.connected)
	})
}

func TestUpdateWindowSize(t *testing.T) {
	t.Run("updates width and height", func(t *testing.T) {
		m := testModel([]EmailItem{})

		newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

		updated := newModel.(Model)
		assert.Equal(t, 120, updated.width)
		assert.Equal(t, 40, updated.height)
	})

	t.Run("updates viewport dimensions", func(t *testing.T) {
		m := testModel([]EmailItem{})

		newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

		updated := newModel.(Model)
		assert.Equal(t, 116, updated.viewport.Width)  // width - 4
		assert.Equal(t, 32, updated.viewport.Height)  // height - 8
	})
}

func TestUpdateEmailDeleted(t *testing.T) {
	t.Run("removes email from list on success", func(t *testing.T) {
		emails := []EmailItem{
			testEmailItem("1", "First", "a@x.com", "inbox"),
			testEmailItem("2", "Second", "b@x.com", "inbox"),
			testEmailItem("3", "Third", "c@x.com", "inbox"),
		}
		m := testModel(emails)

		newModel, _ := m.Update(emailDeletedMsg{emailID: "2", err: nil})

		updated := newModel.(Model)
		assert.Len(t, updated.emails, 2)
		for _, e := range updated.emails {
			assert.NotEqual(t, "2", e.Email.ID)
		}
	})

	t.Run("sets error on failure", func(t *testing.T) {
		m := testModel([]EmailItem{testEmailItem("1", "Test", "a@x.com", "inbox")})
		deleteErr := errors.New("delete failed")

		newModel, _ := m.Update(emailDeletedMsg{emailID: "1", err: deleteErr})

		updated := newModel.(Model)
		assert.Equal(t, deleteErr, updated.lastError)
		// Email should still be in list
		assert.Len(t, updated.emails, 1)
	})
}

func TestUpdateAttachmentSaved(t *testing.T) {
	email := EmailItem{
		Email: testEmailWithAttachments("1", "Test", "from@x.com", []vaultsandbox.Attachment{
			{Filename: "test.pdf", ContentType: "application/pdf", Size: 1024},
		}),
		InboxLabel: "inbox",
	}

	t.Run("sets lastSavedFile on success", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewAttachments

		newModel, _ := m.Update(attachmentSavedMsg{filename: "/tmp/test.pdf", err: nil})

		updated := newModel.(Model)
		assert.Equal(t, "/tmp/test.pdf", updated.lastSavedFile)
		assert.Nil(t, updated.lastError)
	})

	t.Run("sets error on failure", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewAttachments
		saveErr := errors.New("save failed")

		newModel, _ := m.Update(attachmentSavedMsg{filename: "", err: saveErr})

		updated := newModel.(Model)
		assert.Equal(t, saveErr, updated.lastError)
	})
}

func TestUpdateKeyNavigation(t *testing.T) {
	emails := []EmailItem{
		testEmailItem("1", "First", "a@x.com", "inbox"),
		testEmailItem("2", "Second", "b@x.com", "inbox"),
		testEmailItem("3", "Third", "c@x.com", "inbox"),
	}

	t.Run("enter key opens detail view", func(t *testing.T) {
		m := testModel(emails)
		m.list.Select(1)

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

		updated := newModel.(Model)
		assert.True(t, updated.viewing)
		require.NotNil(t, updated.viewedEmail)
		assert.Equal(t, "2", updated.viewedEmail.Email.ID)
	})

	t.Run("enter key does nothing on empty list", func(t *testing.T) {
		m := testModel([]EmailItem{})

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

		updated := newModel.(Model)
		assert.False(t, updated.viewing)
	})

	t.Run("q key returns quit command", func(t *testing.T) {
		m := testModel(emails)

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

		assert.NotNil(t, cmd)
	})
}

func TestUpdateDetailViewKeys(t *testing.T) {
	email := testEmailItem("1", "Test", "from@example.com", "inbox")

	t.Run("escape closes detail view", func(t *testing.T) {
		m := testModelDetailView(email)

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})

		updated := newModel.(Model)
		assert.False(t, updated.viewing)
		assert.Nil(t, updated.viewedEmail)
		assert.Equal(t, ViewContent, updated.detailView)
	})

	t.Run("backspace closes detail view", func(t *testing.T) {
		m := testModelDetailView(email)

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})

		updated := newModel.(Model)
		assert.False(t, updated.viewing)
	})
}

func TestUpdateTabSwitch(t *testing.T) {
	email := testEmailItem("1", "Test", "from@example.com", "inbox")

	t.Run("1 key switches to content view", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewSecurity

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

		updated := newModel.(Model)
		assert.Equal(t, ViewContent, updated.detailView)
	})

	t.Run("2 key switches to security view", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewContent

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})

		updated := newModel.(Model)
		assert.Equal(t, ViewSecurity, updated.detailView)
	})

	t.Run("3 key switches to links view", func(t *testing.T) {
		m := testModelDetailView(email)

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})

		updated := newModel.(Model)
		assert.Equal(t, ViewLinks, updated.detailView)
		assert.Equal(t, 0, updated.selectedLink) // Reset selection
	})

	t.Run("4 key switches to attachments view", func(t *testing.T) {
		m := testModelDetailView(email)

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})

		updated := newModel.(Model)
		assert.Equal(t, ViewAttachments, updated.detailView)
		assert.Equal(t, 0, updated.selectedAttachment) // Reset selection
	})

	t.Run("5 key switches to raw view", func(t *testing.T) {
		m := testModelDetailView(email)

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})

		updated := newModel.(Model)
		assert.Equal(t, ViewRaw, updated.detailView)
	})
}

func TestUpdateLinksNavigation(t *testing.T) {
	email := EmailItem{
		Email:      testEmailWithLinks("1", "Test", "from@x.com", []string{"http://a.com", "http://b.com", "http://c.com"}),
		InboxLabel: "inbox",
	}

	t.Run("down arrow navigates links", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewLinks
		m.selectedLink = 0

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})

		updated := newModel.(Model)
		assert.Equal(t, 1, updated.selectedLink)
	})

	t.Run("up arrow navigates links", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewLinks
		m.selectedLink = 1

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})

		updated := newModel.(Model)
		assert.Equal(t, 0, updated.selectedLink)
	})

	t.Run("wraps around at end", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewLinks
		m.selectedLink = 2

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})

		updated := newModel.(Model)
		assert.Equal(t, 0, updated.selectedLink)
	})

	t.Run("wraps around at start", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewLinks
		m.selectedLink = 0

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})

		updated := newModel.(Model)
		assert.Equal(t, 2, updated.selectedLink)
	})
}

func TestUpdateAttachmentsNavigation(t *testing.T) {
	email := EmailItem{
		Email: testEmailWithAttachments("1", "Test", "from@x.com", []vaultsandbox.Attachment{
			{Filename: "a.pdf"},
			{Filename: "b.pdf"},
		}),
		InboxLabel: "inbox",
	}

	t.Run("down arrow navigates attachments", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewAttachments
		m.selectedAttachment = 0

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})

		updated := newModel.(Model)
		assert.Equal(t, 1, updated.selectedAttachment)
	})

	t.Run("up arrow navigates attachments", func(t *testing.T) {
		m := testModelDetailView(email)
		m.detailView = ViewAttachments
		m.selectedAttachment = 1

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})

		updated := newModel.(Model)
		assert.Equal(t, 0, updated.selectedAttachment)
	})
}

func TestUpdateInboxSwitch(t *testing.T) {
	// Note: Full inbox switching tests require real Inbox objects which need API access.
	// These tests verify the index switching behavior without triggering list filtering.

	t.Run("does nothing when no inboxes", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.inboxes = nil
		m.currentInboxIdx = 0

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})

		updated := newModel.(Model)
		assert.Equal(t, 0, updated.currentInboxIdx)
	})

	t.Run("left does nothing when no inboxes", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.inboxes = nil
		m.currentInboxIdx = 0

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})

		updated := newModel.(Model)
		assert.Equal(t, 0, updated.currentInboxIdx)
	})
}
