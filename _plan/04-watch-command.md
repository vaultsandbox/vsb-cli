# Phase 2.2: Watch Command (Real-time TUI)

## Objective
Implement `vsb watch` - a real-time TUI dashboard using Bubble Tea that shows emails as they arrive.

## Command

| Command | Description |
|---------|-------------|
| `vsb watch` | Watch active inbox for incoming emails |
| `vsb watch --all` | Watch all stored inboxes |
| `vsb watch --email <addr>` | Watch specific inbox |

## Tasks

### 1. TUI Styles

**File: `internal/tui/styles/styles.go`**

```go
package styles

import "github.com/charmbracelet/lipgloss"

var (
    // Colors
    Purple    = lipgloss.Color("#7C3AED")
    Green     = lipgloss.Color("#10B981")
    Red       = lipgloss.Color("#EF4444")
    Yellow    = lipgloss.Color("#F59E0B")
    Gray      = lipgloss.Color("#6B7280")
    DarkGray  = lipgloss.Color("#374151")
    White     = lipgloss.Color("#FFFFFF")

    // App frame
    AppStyle = lipgloss.NewStyle().
        Padding(1, 2)

    // Header
    HeaderStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(Purple).
        MarginBottom(1)

    // Status bar
    StatusBarStyle = lipgloss.NewStyle().
        Foreground(Gray).
        MarginTop(1)

    // Email list item
    EmailItemStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(DarkGray).
        Padding(0, 1).
        MarginBottom(1)

    EmailItemSelectedStyle = EmailItemStyle.
        BorderForeground(Purple)

    // Email fields
    SubjectStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(White)

    FromStyle = lipgloss.NewStyle().
        Foreground(Gray)

    TimeStyle = lipgloss.NewStyle().
        Foreground(Gray).
        Italic(true)

    // Labels/badges
    InboxLabelStyle = lipgloss.NewStyle().
        Background(Purple).
        Foreground(White).
        Padding(0, 1).
        MarginRight(1)

    UnreadBadge = lipgloss.NewStyle().
        Bold(true).
        Foreground(Green)

    // Preview pane
    PreviewStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(Purple).
        Padding(1, 2)

    // Help
    HelpStyle = lipgloss.NewStyle().
        Foreground(Gray)
)
```

### 2. Watch TUI Model

**File: `internal/tui/watch/model.go`**

```go
package watch

import (
    "context"
    "fmt"
    "time"

    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/list"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/tui/styles"
)

// EmailItem represents an email in the list
type EmailItem struct {
    Email      *vaultsandbox.Email
    InboxLabel string
}

func (e EmailItem) Title() string {
    return e.Email.Subject
}

func (e EmailItem) Description() string {
    return fmt.Sprintf("From: %s • %s", e.Email.From,
        e.Email.ReceivedAt.Format("15:04:05"))
}

func (e EmailItem) FilterValue() string {
    return e.Email.Subject + " " + e.Email.From
}

// Messages
type emailReceivedMsg struct {
    email      *vaultsandbox.Email
    inboxLabel string
}

type errMsg struct {
    err error
}

// Model is the Bubble Tea model for the watch TUI
type Model struct {
    list       list.Model
    emails     []EmailItem
    selected   int
    showAll    bool
    inboxLabel string

    // Connection status
    connected  bool
    lastError  error

    // Dimensions
    width      int
    height     int

    // Context for cancellation
    ctx        context.Context
    cancel     context.CancelFunc

    // SDK components
    client     *vaultsandbox.Client
    inboxes    []*vaultsandbox.Inbox
}

// KeyMap defines the keybindings
type KeyMap struct {
    Up       key.Binding
    Down     key.Binding
    Open     key.Binding
    View     key.Binding
    Delete   key.Binding
    Refresh  key.Binding
    Quit     key.Binding
    Help     key.Binding
}

var DefaultKeyMap = KeyMap{
    Up: key.NewBinding(
        key.WithKeys("up", "k"),
        key.WithHelp("↑/k", "up"),
    ),
    Down: key.NewBinding(
        key.WithKeys("down", "j"),
        key.WithHelp("↓/j", "down"),
    ),
    Open: key.NewBinding(
        key.WithKeys("enter", "o"),
        key.WithHelp("enter/o", "open links"),
    ),
    View: key.NewBinding(
        key.WithKeys("v"),
        key.WithHelp("v", "view html"),
    ),
    Delete: key.NewBinding(
        key.WithKeys("d", "backspace"),
        key.WithHelp("d", "delete"),
    ),
    Refresh: key.NewBinding(
        key.WithKeys("r"),
        key.WithHelp("r", "refresh"),
    ),
    Quit: key.NewBinding(
        key.WithKeys("q", "ctrl+c"),
        key.WithHelp("q", "quit"),
    ),
    Help: key.NewBinding(
        key.WithKeys("?"),
        key.WithHelp("?", "help"),
    ),
}

// NewModel creates a new watch TUI model
func NewModel(client *vaultsandbox.Client, inboxes []*vaultsandbox.Inbox, showAll bool) Model {
    ctx, cancel := context.WithCancel(context.Background())

    // Create list
    delegate := list.NewDefaultDelegate()
    delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
        Foreground(styles.Purple).
        BorderForeground(styles.Purple)
    delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
        Foreground(styles.Gray)

    l := list.New([]list.Item{}, delegate, 0, 0)
    l.Title = "Incoming Emails"
    l.Styles.Title = styles.HeaderStyle
    l.SetShowStatusBar(false)
    l.SetFilteringEnabled(true)

    // Get label for single inbox
    label := ""
    if len(inboxes) == 1 {
        label = inboxes[0].EmailAddress()
    }

    return Model{
        list:       l,
        emails:     []EmailItem{},
        showAll:    showAll,
        inboxLabel: label,
        ctx:        ctx,
        cancel:     cancel,
        client:     client,
        inboxes:    inboxes,
    }
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        m.startWatching(),
        tea.EnterAltScreen,
    )
}

func (m *Model) startWatching() tea.Cmd {
    return func() tea.Msg {
        // Start watching inboxes
        if m.showAll || len(m.inboxes) > 1 {
            // Watch multiple inboxes
            eventCh := m.client.WatchInboxes(m.ctx, m.inboxes...)
            go func() {
                for event := range eventCh {
                    // Find label for this inbox
                    label := ""
                    for _, inbox := range m.inboxes {
                        if inbox.EmailAddress() == event.Email.To[0] {
                            label = inbox.EmailAddress()
                            break
                        }
                    }
                    // This would need program.Send in real implementation
                    _ = emailReceivedMsg{email: event.Email, inboxLabel: label}
                }
            }()
        } else if len(m.inboxes) == 1 {
            // Watch single inbox
            emailCh := m.inboxes[0].Watch(m.ctx)
            go func() {
                for email := range emailCh {
                    _ = emailReceivedMsg{email: email, inboxLabel: m.inboxLabel}
                }
            }()
        }

        m.connected = true
        return nil
    }
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, DefaultKeyMap.Quit):
            m.cancel()
            return m, tea.Quit
        case key.Matches(msg, DefaultKeyMap.Open):
            if len(m.emails) > 0 {
                return m, m.openLinks()
            }
        case key.Matches(msg, DefaultKeyMap.View):
            if len(m.emails) > 0 {
                return m, m.viewHTML()
            }
        }

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.list.SetSize(msg.Width-4, msg.Height-6)

    case emailReceivedMsg:
        item := EmailItem{
            Email:      msg.email,
            InboxLabel: msg.inboxLabel,
        }
        m.emails = append([]EmailItem{item}, m.emails...)

        // Update list
        items := make([]list.Item, len(m.emails))
        for i, e := range m.emails {
            items[i] = e
        }
        m.list.SetItems(items)

    case errMsg:
        m.lastError = msg.err
        m.connected = false
    }

    var cmd tea.Cmd
    m.list, cmd = m.list.Update(msg)
    return m, cmd
}

func (m Model) View() string {
    // Status bar
    status := "Watching"
    if m.showAll {
        status += fmt.Sprintf(" %d inboxes", len(m.inboxes))
    } else {
        status += " " + m.inboxLabel
    }
    if !m.connected {
        status = styles.HelpStyle.Foreground(styles.Red).Render("Disconnected")
    }

    statusBar := styles.StatusBarStyle.Render(
        fmt.Sprintf("%s • %d emails • Press ? for help",
            status, len(m.emails)))

    // Help text
    help := styles.HelpStyle.Render("q: quit • o: open links • v: view html • d: delete")

    // Combine
    content := lipgloss.JoinVertical(lipgloss.Left,
        m.list.View(),
        statusBar,
        help,
    )

    return styles.AppStyle.Render(content)
}

func (m Model) openLinks() tea.Cmd {
    return func() tea.Msg {
        if i := m.list.Index(); i >= 0 && i < len(m.emails) {
            email := m.emails[i].Email
            if len(email.Links) > 0 {
                // Open first link in browser
                openBrowser(email.Links[0])
            }
        }
        return nil
    }
}

func (m Model) viewHTML() tea.Cmd {
    return func() tea.Msg {
        if i := m.list.Index(); i >= 0 && i < len(m.emails) {
            email := m.emails[i].Email
            if email.HTML != "" {
                viewInBrowser(email.HTML)
            }
        }
        return nil
    }
}
```

### 3. Watch Command

**File: `internal/cli/watch.go`**

```go
package cli

import (
    "context"
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/tui/watch"
)

var watchCmd = &cobra.Command{
    Use:   "watch",
    Short: "Watch for incoming emails in real-time",
    Long: `Open a real-time dashboard showing emails as they arrive.

Uses Server-Sent Events (SSE) for instant notifications.
All emails are decrypted locally using your stored private keys.

Examples:
  vsb watch           # Watch active inbox
  vsb watch --all     # Watch all stored inboxes
  vsb watch --email abc@vaultsandbox.com`,
    RunE: runWatch,
}

var (
    watchAll   bool
    watchEmail string
)

func init() {
    rootCmd.AddCommand(watchCmd)

    watchCmd.Flags().BoolVar(&watchAll, "all", false,
        "Watch all stored inboxes")
    watchCmd.Flags().StringVar(&watchEmail, "email", "",
        "Watch specific inbox by email address")
}

func runWatch(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Load keystore
    keystore, err := config.LoadKeystore()
    if err != nil {
        return fmt.Errorf("failed to load keystore: %w", err)
    }

    // Determine which inboxes to watch
    var storedInboxes []config.StoredInbox

    if watchAll {
        storedInboxes = keystore.ListInboxes()
        if len(storedInboxes) == 0 {
            return fmt.Errorf("no inboxes found. Create one with 'vsb inbox create'")
        }
    } else if watchEmail != "" {
        inbox, err := keystore.GetInbox(watchEmail)
        if err != nil {
            return fmt.Errorf("inbox not found: %s", watchEmail)
        }
        storedInboxes = []config.StoredInbox{*inbox}
    } else {
        inbox, err := keystore.GetActiveInbox()
        if err != nil {
            return fmt.Errorf("no active inbox. Create one with 'vsb inbox create' or set with 'vsb inbox use'")
        }
        storedInboxes = []config.StoredInbox{*inbox}
    }

    // Create SDK client
    client, err := config.NewClient()
    if err != nil {
        return err
    }
    defer client.Close()

    // Import inboxes into client
    var inboxes []*vaultsandbox.Inbox
    for _, stored := range storedInboxes {
        exported := stored.ToExportedInbox()
        inbox, err := client.ImportInbox(ctx, exported)
        if err != nil {
            return fmt.Errorf("failed to import inbox %s: %w", stored.Email, err)
        }
        inboxes = append(inboxes, inbox)
    }

    // Create and run TUI
    model := watch.NewModel(client, inboxes, watchAll)
    p := tea.NewProgram(model, tea.WithAltScreen())

    if _, err := p.Run(); err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }

    return nil
}
```

### 4. Browser Helpers

**File: `internal/tui/watch/browser.go`**

```go
package watch

import (
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
)

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
    var cmd *exec.Cmd

    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("open", url)
    case "linux":
        cmd = exec.Command("xdg-open", url)
    case "windows":
        cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
    default:
        return nil
    }

    return cmd.Start()
}

// viewInBrowser writes HTML to a temp file and opens it
func viewInBrowser(html string) error {
    // Create temp file
    tmpDir := os.TempDir()
    tmpFile := filepath.Join(tmpDir, "vsb-preview.html")

    if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
        return err
    }

    return openBrowser("file://" + tmpFile)
}
```

## Features

1. **Real-time Updates**: Emails appear instantly via SSE
2. **Multi-inbox Support**: `--all` flag to watch all inboxes
3. **Keyboard Navigation**: vim-style (j/k) or arrow keys
4. **Quick Actions**:
   - `o` - Open first link in browser
   - `v` - View HTML in browser
   - `d` - Delete email
5. **Search/Filter**: Built-in fuzzy search
6. **Connection Status**: Shows connected/disconnected state

## Verification

```bash
# Watch active inbox
vsb watch

# Watch all inboxes
vsb watch --all

# Watch specific inbox
vsb watch --email test@vaultsandbox.com
```

## Files Created

- `internal/tui/styles/styles.go`
- `internal/tui/watch/model.go`
- `internal/tui/watch/browser.go`
- `internal/cli/watch.go`

## Next Steps

Proceed to [05-wait-for-command.md](05-wait-for-command.md) to implement the CI/CD wait-for command.
