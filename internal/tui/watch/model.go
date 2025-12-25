package watch

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/browser"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
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

// DetailView represents which tab is active in detail view
type DetailView int

const (
	ViewContent DetailView = iota
	ViewSecurity
	ViewLinks
	ViewRaw
)

// Model is the Bubble Tea model for the watch TUI
type Model struct {
	list     list.Model
	viewport viewport.Model
	emails   []EmailItem

	showAll    bool
	inboxLabel string

	// Detail view state
	viewing     bool
	viewedEmail *EmailItem
	detailView  DetailView

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
	Up        key.Binding
	Down      key.Binding
	Enter     key.Binding
	Back      key.Binding
	OpenLinks key.Binding
	ViewHTML  key.Binding
	Delete    key.Binding
	Refresh   key.Binding
	Quit      key.Binding
	Help      key.Binding
	Audit     key.Binding
	ListLinks key.Binding
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
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view email"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	OpenLinks: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open links"),
	),
	ViewHTML: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view html"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
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
	Audit: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "security audit"),
	),
	ListLinks: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "list links"),
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
		// Handle detail view keys
		if m.viewing {
			switch {
			case key.Matches(msg, DefaultKeyMap.Quit):
				m.cancel()
				return m, tea.Quit
			case key.Matches(msg, DefaultKeyMap.Back):
				m.viewing = false
				m.viewedEmail = nil
				m.detailView = ViewContent
				return m, nil
			case key.Matches(msg, DefaultKeyMap.OpenLinks):
				if m.viewedEmail != nil && len(m.viewedEmail.Email.Links) > 0 {
					return m, m.openLinks()
				}
			case key.Matches(msg, DefaultKeyMap.ViewHTML):
				if m.viewedEmail != nil && m.viewedEmail.Email.HTML != "" {
					return m, m.viewHTML()
				}
			case key.Matches(msg, DefaultKeyMap.Audit):
				if m.viewedEmail != nil {
					m.detailView = ViewSecurity
					m.viewport.SetContent(m.renderSecurityView())
					m.viewport.GotoTop()
				}
				return m, nil
			case key.Matches(msg, DefaultKeyMap.ListLinks):
				if m.viewedEmail != nil {
					m.detailView = ViewLinks
					m.viewport.SetContent(m.renderLinksView())
					m.viewport.GotoTop()
				}
				return m, nil
			case msg.String() == "1":
				if m.viewedEmail != nil {
					m.detailView = ViewContent
					m.viewport.SetContent(m.renderEmailDetail())
					m.viewport.GotoTop()
				}
				return m, nil
			case msg.String() == "2":
				if m.viewedEmail != nil {
					m.detailView = ViewSecurity
					m.viewport.SetContent(m.renderSecurityView())
					m.viewport.GotoTop()
				}
				return m, nil
			case msg.String() == "3":
				if m.viewedEmail != nil {
					m.detailView = ViewLinks
					m.viewport.SetContent(m.renderLinksView())
					m.viewport.GotoTop()
				}
				return m, nil
			case msg.String() == "4":
				if m.viewedEmail != nil {
					m.detailView = ViewRaw
					m.viewport.SetContent(m.renderRawView())
					m.viewport.GotoTop()
				}
				return m, nil
			}
			// Update viewport for scrolling
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		// Don't handle keys when filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, DefaultKeyMap.Quit):
			m.cancel()
			return m, tea.Quit
		case key.Matches(msg, DefaultKeyMap.Enter):
			if len(m.emails) > 0 {
				if i := m.list.Index(); i >= 0 && i < len(m.emails) {
					m.viewing = true
					m.viewedEmail = &m.emails[i]
					m.viewport.SetContent(m.renderEmailDetail())
					m.viewport.GotoTop()
				}
			}
			return m, nil
		case key.Matches(msg, DefaultKeyMap.OpenLinks):
			if len(m.emails) > 0 {
				return m, m.openLinks()
			}
		case key.Matches(msg, DefaultKeyMap.ViewHTML):
			if len(m.emails) > 0 {
				return m, m.viewHTML()
			}
		case key.Matches(msg, DefaultKeyMap.Delete):
			if len(m.emails) > 0 {
				return m, m.deleteEmail()
			}
		case key.Matches(msg, DefaultKeyMap.Audit):
			if len(m.emails) > 0 {
				if i := m.list.Index(); i >= 0 && i < len(m.emails) {
					m.viewing = true
					m.viewedEmail = &m.emails[i]
					m.detailView = ViewSecurity
					m.viewport.SetContent(m.renderSecurityView())
					m.viewport.GotoTop()
				}
			}
			return m, nil
		case key.Matches(msg, DefaultKeyMap.ListLinks):
			if len(m.emails) > 0 {
				if i := m.list.Index(); i >= 0 && i < len(m.emails) {
					m.viewing = true
					m.viewedEmail = &m.emails[i]
					m.detailView = ViewLinks
					m.viewport.SetContent(m.renderLinksView())
					m.viewport.GotoTop()
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-6)
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 8

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

	case emailDeletedMsg:
		if msg.err != nil {
			m.lastError = msg.err
			return m, nil
		}
		// Remove email from local state
		for i, e := range m.emails {
			if e.Email.ID == msg.emailID {
				m.emails = append(m.emails[:i], m.emails[i+1:]...)
				break
			}
		}
		// Update list items
		items := make([]list.Item, len(m.emails))
		for i, e := range m.emails {
			items[i] = e
		}
		m.list.SetItems(items)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.viewing {
		return m.viewDetail()
	}
	return m.viewList()
}

func (m Model) viewList() string {
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
	help := styles.HelpStyle.Render("q: quit • enter: view • a: audit • l: links • o: open • v: html • d: delete")

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left,
		m.list.View(),
		statusBar,
		help,
	)

	return styles.AppStyle.Render(content)
}

func (m Model) viewDetail() string {
	if m.viewedEmail == nil {
		return ""
	}

	// Header
	header := styles.HeaderStyle.Render("Email Details")

	// Help text
	help := styles.HelpStyle.Render("1-4: tabs • a: audit • l: links • o: open • v: html • esc: back • q: quit")

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		m.viewport.View(),
		help,
	)

	return styles.AppStyle.Render(content)
}

func (m Model) renderEmailDetail() string {
	if m.viewedEmail == nil {
		return ""
	}

	email := m.viewedEmail.Email
	var sb strings.Builder

	// Tab indicator
	sb.WriteString(styles.HelpStyle.Render("[1:Content] [2:Security] [3:Links] [4:Raw]"))
	sb.WriteString("\n")
	sb.WriteString(styles.HelpStyle.Render("^^^^^^^^^"))
	sb.WriteString("\n\n")

	// Field styles
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Purple)
	valueStyle := lipgloss.NewStyle().Foreground(styles.White)

	// From
	sb.WriteString(labelStyle.Render("From:    "))
	sb.WriteString(valueStyle.Render(email.From))
	sb.WriteString("\n")

	// To
	sb.WriteString(labelStyle.Render("To:      "))
	sb.WriteString(valueStyle.Render(strings.Join(email.To, ", ")))
	sb.WriteString("\n")

	// Date
	sb.WriteString(labelStyle.Render("Date:    "))
	sb.WriteString(valueStyle.Render(email.ReceivedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString("\n")

	// Subject
	sb.WriteString(labelStyle.Render("Subject: "))
	subject := email.Subject
	if subject == "" {
		subject = "(no subject)"
	}
	sb.WriteString(valueStyle.Render(subject))
	sb.WriteString("\n")

	// Links (if any)
	if len(email.Links) > 0 {
		sb.WriteString(labelStyle.Render("Links:   "))
		sb.WriteString(valueStyle.Render(fmt.Sprintf("%d found", len(email.Links))))
		sb.WriteString("\n")
	}

	// Attachments (if any)
	if len(email.Attachments) > 0 {
		sb.WriteString(labelStyle.Render("Attach:  "))
		sb.WriteString(valueStyle.Render(fmt.Sprintf("%d files", len(email.Attachments))))
		sb.WriteString("\n")
	}

	// Separator
	sb.WriteString("\n")
	sb.WriteString(styles.HelpStyle.Render(strings.Repeat("─", 60)))
	sb.WriteString("\n\n")

	// Body
	body := email.Text
	if body == "" {
		body = "(no text content)"
	}
	sb.WriteString(body)

	return sb.String()
}

func (m Model) openLinks() tea.Cmd {
	return func() tea.Msg {
		var email *vaultsandbox.Email
		if m.viewing && m.viewedEmail != nil {
			email = m.viewedEmail.Email
		} else if i := m.list.Index(); i >= 0 && i < len(m.emails) {
			email = m.emails[i].Email
		}
		if email != nil && len(email.Links) > 0 {
			browser.OpenURL(email.Links[0])
		}
		return nil
	}
}

func (m Model) viewHTML() tea.Cmd {
	return func() tea.Msg {
		var email *vaultsandbox.Email
		if m.viewing && m.viewedEmail != nil {
			email = m.viewedEmail.Email
		} else if i := m.list.Index(); i >= 0 && i < len(m.emails) {
			email = m.emails[i].Email
		}
		if email != nil && email.HTML != "" {
			browser.ViewHTML(email.HTML)
		}
		return nil
	}
}

// emailDeletedMsg is sent after an email is deleted
type emailDeletedMsg struct {
	emailID string
	err     error
}

func (m Model) deleteEmail() tea.Cmd {
	return func() tea.Msg {
		if i := m.list.Index(); i >= 0 && i < len(m.emails) {
			emailItem := m.emails[i]
			// Find inbox for this email
			for _, inbox := range m.inboxes {
				if m.showAll || len(m.inboxes) > 1 {
					if inbox.EmailAddress() == emailItem.InboxLabel {
						err := inbox.DeleteEmail(m.ctx, emailItem.Email.ID)
						return emailDeletedMsg{emailID: emailItem.Email.ID, err: err}
					}
				} else {
					err := inbox.DeleteEmail(m.ctx, emailItem.Email.ID)
					return emailDeletedMsg{emailID: emailItem.Email.ID, err: err}
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
