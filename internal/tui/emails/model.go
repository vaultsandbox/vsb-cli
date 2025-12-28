package emails

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// EmailItem represents an email in the list
type EmailItem struct {
	Email      *vaultsandbox.Email
	InboxLabel string
}

func (e EmailItem) Title() string {
	return cliutil.SubjectOrDefault(e.Email.Subject)
}

func (e EmailItem) Description() string {
	desc := fmt.Sprintf("From: %s", e.Email.From)
	if e.InboxLabel != "" {
		desc = fmt.Sprintf("[%s] %s", e.InboxLabel, desc)
	}
	desc += fmt.Sprintf(" • %s", e.Email.ReceivedAt.Format(cliutil.TimeFormatTimeOnly))
	return desc
}

func (e EmailItem) FilterValue() string {
	return e.Email.Subject + " " + e.Email.From
}

// Keystore interface for saving inboxes
type Keystore interface {
	SaveInbox(exported *vaultsandbox.ExportedInbox) error
}

// Messages
type emailReceivedMsg struct {
	email      *vaultsandbox.Email
	inboxLabel string
}

type errMsg struct {
	err error
}

type connectedMsg struct{}

type inboxCreatedMsg struct {
	inbox *vaultsandbox.Inbox
	err   error
}

type emailDeletedMsg struct {
	emailID string
	err     error
}

// DetailView represents which tab is active in detail view
type DetailView int

const (
	ViewContent DetailView = iota
	ViewSecurity
	ViewLinks
	ViewAttachments
	ViewRaw
)

// Model is the Bubble Tea model for the watch TUI.
// Fields are grouped by concern: list state, detail view state, connection state, and dependencies.
type Model struct {
	// List view state
	list            list.Model
	viewport        viewport.Model
	emails          []EmailItem
	currentInboxIdx int // index into inboxes slice

	// Detail view state
	viewing            bool
	viewedEmail        *EmailItem
	detailView         DetailView
	selectedLink       int
	selectedAttachment int
	lastSavedFile      string

	// Connection state
	connected bool
	lastError error

	// Layout
	width  int
	height int

	// Context
	ctx    context.Context
	cancel context.CancelFunc

	// Dependencies
	client   *vaultsandbox.Client
	inboxes  []*vaultsandbox.Inbox
	keystore Keystore
}

// NewModel creates a new watch TUI model
// activeIdx is the index of the initially selected inbox
func NewModel(client *vaultsandbox.Client, inboxes []*vaultsandbox.Inbox, activeIdx int, keystore Keystore) Model {
	ctx, cancel := context.WithCancel(context.Background())

	// Create list with custom delegate
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(styles.White).
		BorderForeground(styles.DarkGray)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(styles.Gray).
		BorderForeground(styles.DarkGray)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(styles.Primary).
		BorderForeground(styles.Primary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(styles.Gray).
		BorderForeground(styles.Primary)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Connecting..."
	l.Styles.Title = styles.HeaderStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	// Clamp activeIdx to valid range
	if activeIdx < 0 || activeIdx >= len(inboxes) {
		activeIdx = 0
	}

	return Model{
		list:            l,
		emails:          []EmailItem{},
		currentInboxIdx: activeIdx,
		ctx:             ctx,
		cancel:          cancel,
		client:          client,
		inboxes:         inboxes,
		keystore:        keystore,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.startWatching(),
	)
}

func (m *Model) startWatching() tea.Cmd {
	return func() tea.Msg {
		return connectedMsg{}
	}
}

// WatchEmails starts watching for emails and sends them to the program
func (m *Model) WatchEmails(p *tea.Program) {
	if len(m.inboxes) == 0 {
		return
	}
	eventCh := m.client.WatchInboxes(m.ctx, m.inboxes...)
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				if event != nil {
					p.Send(emailReceivedMsg{
						email:      event.Email,
						inboxLabel: event.Inbox.EmailAddress(),
					})
				}
			}
		}
	}()
}

// LoadExistingEmails fetches existing emails and sends them to the program.
func (m *Model) LoadExistingEmails(p *tea.Program) {
	go func() {
		for _, inbox := range m.inboxes {
			emails, err := inbox.GetEmails(m.ctx)
			if err != nil {
				p.Send(errMsg{err: err})
				continue
			}
			for _, email := range emails {
				p.Send(emailReceivedMsg{
					email:      email,
					inboxLabel: inbox.EmailAddress(),
				})
			}
		}
	}()
}

// Cancel stops watching
func (m *Model) Cancel() {
	m.cancel()
}

// selectedEmail returns the currently selected or viewed email
func (m Model) selectedEmail() *vaultsandbox.Email {
	if m.viewing && m.viewedEmail != nil {
		return m.viewedEmail.Email
	}
	filtered := m.filteredEmails()
	if i := m.list.Index(); i >= 0 && i < len(filtered) {
		return filtered[i].Email
	}
	return nil
}

// wrapIndex wraps an index with delta within the given length
func wrapIndex(current, delta, length int) int {
	if length == 0 {
		return 0
	}
	return (current + delta + length) % length
}
