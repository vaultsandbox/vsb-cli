package emails

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapIndex(t *testing.T) {
	tests := []struct {
		name    string
		current int
		delta   int
		length  int
		want    int
	}{
		{"no wrap forward", 0, 1, 5, 1},
		{"no wrap backward", 2, -1, 5, 1},
		{"wrap forward at end", 4, 1, 5, 0},
		{"wrap backward at start", 0, -1, 5, 4},
		{"multiple wrap forward", 3, 3, 5, 1},
		{"single item", 0, 1, 1, 0},
		{"empty list", 0, 1, 0, 0},
		{"negative index handling", 0, -2, 5, 3},
		{"wrap around multiple times", 0, 7, 5, 2},
		{"large negative delta", 2, -7, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapIndex(tt.current, tt.delta, tt.length)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEmailItemTitle(t *testing.T) {
	t.Run("returns subject when present", func(t *testing.T) {
		item := testEmailItem("1", "Test Subject", "from@example.com", "inbox")
		assert.Equal(t, "Test Subject", item.Title())
	})

	t.Run("returns placeholder for empty subject", func(t *testing.T) {
		item := testEmailItem("1", "", "from@example.com", "inbox")
		assert.Equal(t, "(no subject)", item.Title())
	})

	t.Run("handles special characters in subject", func(t *testing.T) {
		item := testEmailItem("1", "Re: [URGENT] Test & Verification <test>", "from@example.com", "inbox")
		assert.Equal(t, "Re: [URGENT] Test & Verification <test>", item.Title())
	})
}

func TestEmailItemDescription(t *testing.T) {
	t.Run("includes from address", func(t *testing.T) {
		item := testEmailItem("1", "Subject", "sender@example.com", "inbox@test.com")
		desc := item.Description()
		assert.Contains(t, desc, "sender@example.com")
	})

	t.Run("includes inbox label", func(t *testing.T) {
		item := testEmailItem("1", "Subject", "sender@example.com", "inbox@test.com")
		desc := item.Description()
		assert.Contains(t, desc, "[inbox@test.com]")
	})

	t.Run("empty inbox label", func(t *testing.T) {
		item := testEmailItem("1", "Subject", "sender@example.com", "")
		desc := item.Description()
		assert.Contains(t, desc, "From: sender@example.com")
		assert.NotContains(t, desc, "[]")
	})

	t.Run("includes timestamp", func(t *testing.T) {
		item := testEmailItem("1", "Subject", "sender@example.com", "inbox")
		desc := item.Description()
		// Should contain time in format "HH:MM:SS"
		assert.Contains(t, desc, ":")
	})
}

func TestEmailItemFilterValue(t *testing.T) {
	t.Run("searchable by subject", func(t *testing.T) {
		item := testEmailItem("1", "Welcome Email", "support@company.com", "inbox")
		filter := item.FilterValue()
		assert.Contains(t, filter, "Welcome Email")
	})

	t.Run("searchable by from address", func(t *testing.T) {
		item := testEmailItem("1", "Welcome Email", "support@company.com", "inbox")
		filter := item.FilterValue()
		assert.Contains(t, filter, "support@company.com")
	})

	t.Run("combined subject and from", func(t *testing.T) {
		item := testEmailItem("1", "Test", "test@example.com", "inbox")
		filter := item.FilterValue()
		assert.Equal(t, "Test test@example.com", filter)
	})
}

func TestSelectedEmail(t *testing.T) {
	emails := []EmailItem{
		testEmailItem("1", "First", "a@example.com", "inbox"),
		testEmailItem("2", "Second", "b@example.com", "inbox"),
	}

	t.Run("returns viewed email in detail view", func(t *testing.T) {
		m := testModel(emails)
		m.viewing = true
		m.viewedEmail = &emails[1]

		selected := m.selectedEmail()
		assert.NotNil(t, selected)
		assert.Equal(t, "2", selected.ID)
	})

	t.Run("falls back to list selection when viewedEmail is nil in detail view", func(t *testing.T) {
		m := testModel(emails)
		m.viewing = true
		m.viewedEmail = nil
		m.list.Select(0)

		// Falls back to list selection when viewedEmail is nil
		selected := m.selectedEmail()
		assert.NotNil(t, selected)
		assert.Equal(t, "1", selected.ID)
	})

	t.Run("returns list selection in list view", func(t *testing.T) {
		m := testModel(emails)
		m.viewing = false
		m.list.Select(1)

		selected := m.selectedEmail()
		assert.NotNil(t, selected)
		assert.Equal(t, "2", selected.ID)
	})

	t.Run("returns first email when no selection", func(t *testing.T) {
		m := testModel(emails)
		m.viewing = false
		m.list.Select(0)

		selected := m.selectedEmail()
		assert.NotNil(t, selected)
		assert.Equal(t, "1", selected.ID)
	})

	t.Run("returns nil for empty list", func(t *testing.T) {
		m := testModel([]EmailItem{})
		selected := m.selectedEmail()
		assert.Nil(t, selected)
	})
}

func TestFilteredEmails(t *testing.T) {
	emails := []EmailItem{
		testEmailItem("1", "Email 1", "a@x.com", "inbox1@example.com"),
		testEmailItem("2", "Email 2", "b@x.com", "inbox2@example.com"),
		testEmailItem("3", "Email 3", "c@x.com", "inbox1@example.com"),
	}

	t.Run("returns all emails when no inboxes configured", func(t *testing.T) {
		m := testModel(emails)
		m.inboxes = nil
		m.currentInboxIdx = 0

		filtered := m.filteredEmails()
		assert.Len(t, filtered, 3)
	})

	t.Run("returns all emails when currentInboxIdx is negative", func(t *testing.T) {
		m := testModel(emails)
		m.currentInboxIdx = -1

		filtered := m.filteredEmails()
		assert.Len(t, filtered, 3)
	})

	t.Run("returns all emails when currentInboxIdx is out of bounds", func(t *testing.T) {
		m := testModel(emails)
		m.inboxes = nil
		m.currentInboxIdx = 10

		filtered := m.filteredEmails()
		assert.Len(t, filtered, 3)
	})
}

func TestCurrentInboxLabel(t *testing.T) {
	t.Run("returns 'all' when no inboxes", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.inboxes = nil
		m.currentInboxIdx = 0

		label := m.currentInboxLabel()
		assert.Equal(t, "all", label)
	})

	t.Run("returns 'all' when index out of bounds", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.inboxes = nil
		m.currentInboxIdx = 5

		label := m.currentInboxLabel()
		assert.Equal(t, "all", label)
	})

	t.Run("returns 'all' when index is negative", func(t *testing.T) {
		m := testModel([]EmailItem{})
		m.currentInboxIdx = -1

		label := m.currentInboxLabel()
		assert.Equal(t, "all", label)
	})
}

func TestDetailViewConstants(t *testing.T) {
	t.Run("view constants are sequential", func(t *testing.T) {
		assert.Equal(t, DetailView(0), ViewContent)
		assert.Equal(t, DetailView(1), ViewSecurity)
		assert.Equal(t, DetailView(2), ViewLinks)
		assert.Equal(t, DetailView(3), ViewAttachments)
		assert.Equal(t, DetailView(4), ViewRaw)
	})
}
