# Phase 4: TUI Testing

**Goal**: Make TUI testable without running interactive program.
**Expected Coverage Gain**: +8-12%
**Effort**: High

## Overview

Bubble Tea's architecture is actually very testable because:
- `Update(msg) (Model, Cmd)` is a pure function
- `View() string` is a pure function
- Commands are deferred side effects (returned, not executed)

The strategy is to:
1. Extract pure helper functions
2. Test `Update()` with mock messages
3. Test `View()` output contains expected strings
4. Mock external dependencies (SDK client, keystore)

---

## 4.1 Understanding the TUI Architecture

### Model Structure

```go
type Model struct {
    // List view state
    list            list.Model
    emails          []EmailItem
    currentInboxIdx int

    // Detail view state
    viewing            bool
    viewedEmail        *EmailItem
    detailView         DetailView
    selectedLink       int
    selectedAttachment int

    // Connection state
    connected bool
    lastError error

    // Dependencies (need mocking)
    client   *vaultsandbox.Client
    inboxes  []*vaultsandbox.Inbox
    keystore Keystore
}
```

### Message Types

```go
type emailReceivedMsg struct { email *EmailItem }
type errMsg struct { err error }
type connectedMsg struct{}
type inboxCreatedMsg struct { inbox *vaultsandbox.Inbox; err error }
type emailDeletedMsg struct { err error }
type attachmentSavedMsg struct { path string; err error }
```

---

## 4.2 Mock Infrastructure

**File to create**: `internal/tui/emails/mocks_test.go`

```go
package emails

import (
    "context"
    "time"

    "github.com/vaultsandbox/client-go"
)

// MockClient implements a test double for vaultsandbox.Client
type MockClient struct {
    WatchInboxesFunc func(ctx context.Context, inboxes ...*vaultsandbox.Inbox) chan *vaultsandbox.Event
    CreateInboxFunc  func(ctx context.Context, opts ...vaultsandbox.CreateOption) (*vaultsandbox.Inbox, error)
}

func (m *MockClient) WatchInboxes(ctx context.Context, inboxes ...*vaultsandbox.Inbox) chan *vaultsandbox.Event {
    if m.WatchInboxesFunc != nil {
        return m.WatchInboxesFunc(ctx, inboxes...)
    }
    ch := make(chan *vaultsandbox.Event)
    close(ch)
    return ch
}

func (m *MockClient) CreateInbox(ctx context.Context, opts ...vaultsandbox.CreateOption) (*vaultsandbox.Inbox, error) {
    if m.CreateInboxFunc != nil {
        return m.CreateInboxFunc(ctx, opts...)
    }
    return nil, nil
}

// MockKeystore implements Keystore interface
type MockKeystore struct {
    SaveInboxFunc func(exp *vaultsandbox.ExportedInbox) error
}

func (m *MockKeystore) SaveInbox(exp *vaultsandbox.ExportedInbox) error {
    if m.SaveInboxFunc != nil {
        return m.SaveInboxFunc(exp)
    }
    return nil
}

// MockInbox provides a test double for vaultsandbox.Inbox
type MockInbox struct {
    EmailAddr    string
    GetEmailsFunc func(ctx context.Context) ([]*vaultsandbox.Email, error)
    DeleteEmailFunc func(ctx context.Context, id string) error
}

func (m *MockInbox) EmailAddress() string {
    return m.EmailAddr
}

func (m *MockInbox) GetEmails(ctx context.Context) ([]*vaultsandbox.Email, error) {
    if m.GetEmailsFunc != nil {
        return m.GetEmailsFunc(ctx)
    }
    return nil, nil
}

func (m *MockInbox) Export() *vaultsandbox.ExportedInbox {
    return &vaultsandbox.ExportedInbox{Email: m.EmailAddr}
}

// Test fixtures
func testEmail(id, subject, from string) *vaultsandbox.Email {
    return &vaultsandbox.Email{
        ID:         id,
        Subject:    subject,
        From:       from,
        To:         []string{"test@example.com"},
        ReceivedAt: time.Now(),
    }
}

func testEmailItem(id, subject, from, inboxLabel string) EmailItem {
    return EmailItem{
        Email:      testEmail(id, subject, from),
        InboxLabel: inboxLabel,
    }
}

// Helper to create a model with test data
func testModel(emails []EmailItem) Model {
    m := Model{
        emails:    emails,
        connected: true,
        width:     80,
        height:    24,
    }
    // Initialize list with emails
    items := make([]list.Item, len(emails))
    for i, e := range emails {
        items[i] = e
    }
    m.list = list.New(items, list.NewDefaultDelegate(), 80, 20)
    return m
}
```

---

## 4.3 Pure Function Tests

**File to create**: `internal/tui/emails/helpers_test.go`

### Test `wrapIndex`

```go
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
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := wrapIndex(tt.current, tt.delta, tt.length)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Test `EmailItem` Methods

```go
func TestEmailItemTitle(t *testing.T) {
    t.Run("returns subject", func(t *testing.T) {
        item := testEmailItem("1", "Test Subject", "from@example.com", "inbox")
        assert.Equal(t, "Test Subject", item.Title())
    })

    t.Run("returns placeholder for empty subject", func(t *testing.T) {
        item := testEmailItem("1", "", "from@example.com", "inbox")
        assert.Equal(t, "(no subject)", item.Title())
    })
}

func TestEmailItemDescription(t *testing.T) {
    t.Run("includes from and inbox", func(t *testing.T) {
        item := testEmailItem("1", "Subject", "sender@example.com", "inbox@test.com")
        desc := item.Description()
        assert.Contains(t, desc, "sender@example.com")
        assert.Contains(t, desc, "inbox@test.com")
    })
}

func TestEmailItemFilterValue(t *testing.T) {
    t.Run("searchable by subject and from", func(t *testing.T) {
        item := testEmailItem("1", "Welcome Email", "support@company.com", "inbox")
        filter := item.FilterValue()
        assert.Contains(t, filter, "Welcome Email")
        assert.Contains(t, filter, "support@company.com")
    })
}
```

### Test `selectedEmail`

```go
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
        assert.Equal(t, "2", selected.ID)
    })

    t.Run("returns list selection in list view", func(t *testing.T) {
        m := testModel(emails)
        m.viewing = false
        m.list.Select(1)

        selected := m.selectedEmail()
        assert.Equal(t, "2", selected.ID)
    })

    t.Run("returns nil for empty list", func(t *testing.T) {
        m := testModel([]EmailItem{})
        selected := m.selectedEmail()
        assert.Nil(t, selected)
    })
}
```

### Test `filteredEmails`

```go
func TestFilteredEmails(t *testing.T) {
    inbox1 := &MockInbox{EmailAddr: "inbox1@example.com"}
    inbox2 := &MockInbox{EmailAddr: "inbox2@example.com"}

    emails := []EmailItem{
        testEmailItem("1", "Email 1", "a@x.com", "inbox1@example.com"),
        testEmailItem("2", "Email 2", "b@x.com", "inbox2@example.com"),
        testEmailItem("3", "Email 3", "c@x.com", "inbox1@example.com"),
    }

    t.Run("filters by current inbox (index 0 = all)", func(t *testing.T) {
        m := testModel(emails)
        m.inboxes = []*vaultsandbox.Inbox{inbox1, inbox2}
        m.currentInboxIdx = 0 // "All" inbox

        filtered := m.filteredEmails()
        assert.Len(t, filtered, 3)
    })

    t.Run("filters by specific inbox", func(t *testing.T) {
        m := testModel(emails)
        m.inboxes = []*vaultsandbox.Inbox{inbox1, inbox2}
        m.currentInboxIdx = 1 // inbox1

        filtered := m.filteredEmails()
        assert.Len(t, filtered, 2)
        for _, e := range filtered {
            assert.Equal(t, "inbox1@example.com", e.InboxLabel)
        }
    })
}
```

---

## 4.4 Update Function Tests

**File to create**: `internal/tui/emails/update_test.go`

### Test Message Handling

```go
func TestUpdateEmailReceived(t *testing.T) {
    m := testModel([]EmailItem{})
    newEmail := testEmailItem("new-1", "New Email", "sender@example.com", "inbox@test.com")

    newModel, _ := m.Update(emailReceivedMsg{email: &newEmail})

    updated := newModel.(Model)
    assert.Len(t, updated.emails, 1)
    assert.Equal(t, "new-1", updated.emails[0].Email.ID)
}

func TestUpdateErrorMsg(t *testing.T) {
    m := testModel([]EmailItem{})
    testErr := errors.New("connection failed")

    newModel, _ := m.Update(errMsg{err: testErr})

    updated := newModel.(Model)
    assert.Equal(t, testErr, updated.lastError)
}

func TestUpdateConnectedMsg(t *testing.T) {
    m := testModel([]EmailItem{})
    m.connected = false

    newModel, _ := m.Update(connectedMsg{})

    updated := newModel.(Model)
    assert.True(t, updated.connected)
}

func TestUpdateWindowSize(t *testing.T) {
    m := testModel([]EmailItem{})

    newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

    updated := newModel.(Model)
    assert.Equal(t, 120, updated.width)
    assert.Equal(t, 40, updated.height)
}
```

### Test Key Navigation

```go
func TestUpdateKeyNavigation(t *testing.T) {
    emails := []EmailItem{
        testEmailItem("1", "First", "a@x.com", "inbox"),
        testEmailItem("2", "Second", "b@x.com", "inbox"),
        testEmailItem("3", "Third", "c@x.com", "inbox"),
    }

    t.Run("j key moves down", func(t *testing.T) {
        m := testModel(emails)
        m.list.Select(0)

        newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

        updated := newModel.(Model)
        assert.Equal(t, 1, updated.list.Index())
    })

    t.Run("k key moves up", func(t *testing.T) {
        m := testModel(emails)
        m.list.Select(1)

        newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

        updated := newModel.(Model)
        assert.Equal(t, 0, updated.list.Index())
    })

    t.Run("enter key opens detail view", func(t *testing.T) {
        m := testModel(emails)
        m.list.Select(1)

        newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

        updated := newModel.(Model)
        assert.True(t, updated.viewing)
        assert.Equal(t, "2", updated.viewedEmail.Email.ID)
    })

    t.Run("escape closes detail view", func(t *testing.T) {
        m := testModel(emails)
        m.viewing = true
        m.viewedEmail = &emails[0]

        newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})

        updated := newModel.(Model)
        assert.False(t, updated.viewing)
    })

    t.Run("q key returns quit command", func(t *testing.T) {
        m := testModel(emails)

        _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

        // Cmd should be tea.Quit
        assert.NotNil(t, cmd)
    })
}
```

### Test Tab Switching

```go
func TestUpdateTabSwitch(t *testing.T) {
    emails := []EmailItem{
        testEmailItem("1", "Test", "a@x.com", "inbox"),
    }

    t.Run("tab cycles through views", func(t *testing.T) {
        m := testModel(emails)
        m.viewing = true
        m.viewedEmail = &emails[0]
        m.detailView = ContentView

        // Press tab
        newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
        updated := newModel.(Model)
        assert.Equal(t, SecurityView, updated.detailView)

        // Press tab again
        newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyTab})
        updated = newModel.(Model)
        assert.Equal(t, LinksView, updated.detailView)
    })

    t.Run("number keys switch to specific tabs", func(t *testing.T) {
        m := testModel(emails)
        m.viewing = true
        m.viewedEmail = &emails[0]

        testCases := []struct {
            key  rune
            want DetailView
        }{
            {'1', ContentView},
            {'2', SecurityView},
            {'3', LinksView},
            {'4', AttachmentsView},
            {'5', RawView},
        }

        for _, tc := range testCases {
            newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
            updated := newModel.(Model)
            assert.Equal(t, tc.want, updated.detailView, "key %c", tc.key)
        }
    })
}
```

### Test Inbox Switching

```go
func TestUpdateInboxSwitch(t *testing.T) {
    inbox1 := &MockInbox{EmailAddr: "inbox1@example.com"}
    inbox2 := &MockInbox{EmailAddr: "inbox2@example.com"}

    emails := []EmailItem{
        testEmailItem("1", "Email 1", "a@x.com", "inbox1@example.com"),
        testEmailItem("2", "Email 2", "b@x.com", "inbox2@example.com"),
    }

    t.Run("left/right arrows switch inbox", func(t *testing.T) {
        m := testModel(emails)
        m.inboxes = []*vaultsandbox.Inbox{inbox1, inbox2}
        m.currentInboxIdx = 0

        // Press right
        newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
        updated := newModel.(Model)
        assert.Equal(t, 1, updated.currentInboxIdx)

        // Press left
        newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
        updated = newModel.(Model)
        assert.Equal(t, 0, updated.currentInboxIdx)
    })
}
```

---

## 4.5 View Function Tests

**File to create**: `internal/tui/emails/view_test.go`

### Test View Output

```go
func TestView(t *testing.T) {
    emails := []EmailItem{
        testEmailItem("1", "Test Subject", "sender@example.com", "inbox@test.com"),
    }

    t.Run("list view contains email info", func(t *testing.T) {
        m := testModel(emails)
        m.viewing = false

        output := m.View()
        assert.Contains(t, output, "Test Subject")
    })

    t.Run("detail view shows tabs", func(t *testing.T) {
        m := testModel(emails)
        m.viewing = true
        m.viewedEmail = &emails[0]

        output := m.View()
        assert.Contains(t, output, "Content")
        assert.Contains(t, output, "Security")
        assert.Contains(t, output, "Links")
    })

    t.Run("shows connection status", func(t *testing.T) {
        m := testModel(emails)
        m.connected = false

        output := m.View()
        assert.Contains(t, output, "Connecting") // Or similar indicator
    })

    t.Run("shows error message", func(t *testing.T) {
        m := testModel(emails)
        m.lastError = errors.New("test error")

        output := m.View()
        assert.Contains(t, output, "test error")
    })
}
```

### Test Tab Rendering

```go
func TestRenderTabs(t *testing.T) {
    m := testModel([]EmailItem{})

    t.Run("highlights active tab", func(t *testing.T) {
        m.detailView = SecurityView
        output := m.renderTabs()

        // Active tab should be styled differently
        // This test checks structure, not exact styling
        assert.Contains(t, output, "Content")
        assert.Contains(t, output, "Security")
    })
}

func TestRenderSecurityView(t *testing.T) {
    email := testEmail("1", "Test", "sender@example.com")
    email.AuthResults = &authresults.AuthResults{
        SPF:  &authresults.SPFResult{Result: "pass"},
        DKIM: []authresults.DKIMResult{{Result: "pass"}},
    }

    item := EmailItem{Email: email, InboxLabel: "inbox"}
    m := testModel([]EmailItem{item})
    m.viewing = true
    m.viewedEmail = &item

    output := m.renderSecurityView()
    assert.Contains(t, output, "SPF")
    assert.Contains(t, output, "DKIM")
    assert.Contains(t, output, "PASS") // Formatted result
}

func TestRenderLinksView(t *testing.T) {
    email := testEmail("1", "Test", "sender@example.com")
    email.Links = []string{
        "https://example.com/link1",
        "https://example.com/link2",
    }

    item := EmailItem{Email: email, InboxLabel: "inbox"}
    m := testModel([]EmailItem{item})
    m.viewing = true
    m.viewedEmail = &item
    m.selectedLink = 0

    output := m.renderLinksView()
    assert.Contains(t, output, "https://example.com/link1")
    assert.Contains(t, output, "https://example.com/link2")
    assert.Contains(t, output, "[1]") // Numbered links
}

func TestRenderAttachmentsView(t *testing.T) {
    email := testEmail("1", "Test", "sender@example.com")
    email.Attachments = []vaultsandbox.Attachment{
        {Filename: "document.pdf", ContentType: "application/pdf", Size: 1024},
        {Filename: "image.png", ContentType: "image/png", Size: 2048},
    }

    item := EmailItem{Email: email, InboxLabel: "inbox"}
    m := testModel([]EmailItem{item})
    m.viewing = true
    m.viewedEmail = &item

    output := m.renderAttachmentsView()
    assert.Contains(t, output, "document.pdf")
    assert.Contains(t, output, "image.png")
    assert.Contains(t, output, "1.0 KB") // Formatted size
}

func TestRenderRawView(t *testing.T) {
    email := testEmail("1", "Test Subject", "sender@example.com")
    email.Headers = map[string][]string{
        "From":    {"sender@example.com"},
        "Subject": {"Test Subject"},
    }
    email.Text = "Raw email body content"

    item := EmailItem{Email: email, InboxLabel: "inbox"}
    m := testModel([]EmailItem{item})
    m.viewing = true
    m.viewedEmail = &item

    output := m.renderRawView()
    assert.Contains(t, output, "From:")
    assert.Contains(t, output, "Subject:")
    assert.Contains(t, output, "Raw email body content")
}
```

---

## 4.6 Command Function Tests

Test that command functions return appropriate commands without executing them:

```go
func TestCommandFunctions(t *testing.T) {
    email := testEmail("1", "Test", "sender@example.com")
    email.Links = []string{"https://example.com"}

    item := EmailItem{Email: email, InboxLabel: "inbox"}
    m := testModel([]EmailItem{item})
    m.viewing = true
    m.viewedEmail = &item

    t.Run("openFirstURL returns command", func(t *testing.T) {
        cmd := m.openFirstURL()
        assert.NotNil(t, cmd)
    })

    t.Run("openLinkByIndex returns command for valid index", func(t *testing.T) {
        cmd := m.openLinkByIndex(0)
        assert.NotNil(t, cmd)
    })

    t.Run("openLinkByIndex returns nil for invalid index", func(t *testing.T) {
        cmd := m.openLinkByIndex(99)
        assert.Nil(t, cmd)
    })

    t.Run("viewHTML returns command", func(t *testing.T) {
        email.HTML = "<p>HTML content</p>"
        cmd := m.viewHTML()
        assert.NotNil(t, cmd)
    })
}
```

---

## Checklist

### Refactoring (if needed)
- [ ] Export message types for testing (or use internal tests)
- [ ] Ensure `wrapIndex` is accessible for testing
- [ ] Extract `filteredEmails` if currently inline

### Test Files
- [ ] Create `internal/tui/emails/mocks_test.go`
- [ ] Create `internal/tui/emails/helpers_test.go`
- [ ] Create `internal/tui/emails/update_test.go`
- [ ] Create `internal/tui/emails/view_test.go`
- [ ] Run `go test ./internal/tui/...`

## Commands

```bash
# Run Phase 4 tests
go test -v ./internal/tui/...

# Check coverage
go test -coverprofile=phase4.out ./internal/tui/...
go tool cover -func=phase4.out

# View coverage report
go tool cover -html=phase4.out
```

## Notes

- TUI tests focus on **state transitions** and **output content**, not visual appearance
- Bubble Tea's design makes testing straightforward once you understand the pattern
- Mock the SDK client and keystore, but test real model logic
- Commands are returned but not executed in tests - this is by design
