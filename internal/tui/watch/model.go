package watch

import (
	"context"
	"fmt"

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
	if e.Email.Subject == "" {
		return "(no subject)"
	}
	return e.Email.Subject
}

func (e EmailItem) Description() string {
	desc := fmt.Sprintf("From: %s", e.Email.From)
	if e.InboxLabel != "" {
		desc = fmt.Sprintf("[%s] %s", e.InboxLabel, desc)
	}
	desc += fmt.Sprintf(" • %s", e.Email.ReceivedAt.Format("15:04:05"))
	return desc
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

type connectedMsg struct{}

// Model is the Bubble Tea model for the watch TUI
type Model struct {
	list   list.Model
	emails []EmailItem

	showAll    bool
	inboxLabel string

	// Connection status
	connected bool
	lastError error

	// Dimensions
	width  int
	height int

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// SDK components
	client  *vaultsandbox.Client
	inboxes []*vaultsandbox.Inbox
}

// KeyMap defines the keybindings
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	View    key.Binding
	Delete  key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Help    key.Binding
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

	// Create list with custom delegate
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
	)
}

func (m *Model) startWatching() tea.Cmd {
	return func() tea.Msg {
		return connectedMsg{}
	}
}

// WatchEmails starts watching for emails and sends them to the program
func (m *Model) WatchEmails(p *tea.Program) {
	if m.showAll || len(m.inboxes) > 1 {
		// Watch multiple inboxes
		eventCh := m.client.WatchInboxes(m.ctx, m.inboxes...)
		go func() {
			for {
				select {
				case <-m.ctx.Done():
					return
				case event := <-eventCh:
					if event != nil {
						p.Send(emailReceivedMsg{
							email:      event.Email,
							inboxLabel: event.Inbox.EmailAddress(),
						})
					}
				}
			}
		}()
	} else if len(m.inboxes) == 1 {
		// Watch single inbox
		emailCh := m.inboxes[0].Watch(m.ctx)
		go func() {
			for {
				select {
				case <-m.ctx.Done():
					return
				case email := <-emailCh:
					if email != nil {
						p.Send(emailReceivedMsg{
							email:      email,
							inboxLabel: m.inboxLabel,
						})
					}
				}
			}
		}()
	}
}

// LoadExistingEmails fetches existing emails and sends them to the program
func (m *Model) LoadExistingEmails(p *tea.Program) {
	go func() {
		for _, inbox := range m.inboxes {
			emails, err := inbox.GetEmails(m.ctx)
			if err != nil {
				p.Send(errMsg{err: err})
				continue
			}
			for _, email := range emails {
				label := ""
				if m.showAll || len(m.inboxes) > 1 {
					label = inbox.EmailAddress()
				}
				p.Send(emailReceivedMsg{
					email:      email,
					inboxLabel: label,
				})
			}
		}
	}()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

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
		case key.Matches(msg, DefaultKeyMap.Delete):
			if len(m.emails) > 0 {
				return m, m.deleteEmail()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-6)

	case connectedMsg:
		m.connected = true

	case emailReceivedMsg:
		// Check if email already exists (avoid duplicates)
		for _, existing := range m.emails {
			if existing.Email.ID == msg.email.ID {
				return m, nil
			}
		}

		item := EmailItem{
			Email:      msg.email,
			InboxLabel: msg.inboxLabel,
		}
		// Add to front (newest first)
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
	} else if m.inboxLabel != "" {
		status += " " + m.inboxLabel
	}
	if !m.connected {
		status = styles.HelpStyle.Foreground(styles.Red).Render("Disconnected")
	}
	if m.lastError != nil {
		status = styles.HelpStyle.Foreground(styles.Red).Render("Error: " + m.lastError.Error())
	}

	statusBar := styles.StatusBarStyle.Render(
		fmt.Sprintf("%s • %d emails • Press ? for help",
			status, len(m.emails)))

	// Help text
	help := styles.HelpStyle.Render("q: quit • o: open links • v: view html • d: delete • /: filter")

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

func (m Model) deleteEmail() tea.Cmd {
	return func() tea.Msg {
		if i := m.list.Index(); i >= 0 && i < len(m.emails) {
			emailItem := m.emails[i]
			// Find inbox for this email
			for _, inbox := range m.inboxes {
				if m.showAll || len(m.inboxes) > 1 {
					if inbox.EmailAddress() == emailItem.InboxLabel {
						inbox.DeleteEmail(m.ctx, emailItem.Email.ID)
						break
					}
				} else {
					inbox.DeleteEmail(m.ctx, emailItem.Email.ID)
					break
				}
			}
		}
		return nil
	}
}

// Cancel stops watching
func (m *Model) Cancel() {
	m.cancel()
}
