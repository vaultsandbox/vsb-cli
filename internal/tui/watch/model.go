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

type inboxCreatedMsg struct {
	inbox *vaultsandbox.Inbox
	err   error
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

// Model is the Bubble Tea model for the watch TUI
type Model struct {
	list     list.Model
	viewport viewport.Model
	emails   []EmailItem

	currentInboxIdx int // index into inboxes slice

	// Detail view state
	viewing            bool
	viewedEmail        *EmailItem
	detailView         DetailView
	selectedLink       int    // selected link index in links view
	selectedAttachment int    // selected attachment index in attachments view
	lastSavedFile      string // last saved attachment filename

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
	OpenURL   key.Binding
	ViewHTML  key.Binding
	Delete    key.Binding
	Refresh   key.Binding
	Quit      key.Binding
	Help      key.Binding
	PrevInbox key.Binding
	NextInbox key.Binding
	NewInbox  key.Binding
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
	OpenURL: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open url"),
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
	PrevInbox: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "prev inbox"),
	),
	NextInbox: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next inbox"),
	),
	NewInbox: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new inbox"),
	),
}

// NewModel creates a new watch TUI model
// activeIdx is the index of the initially selected inbox
func NewModel(client *vaultsandbox.Client, inboxes []*vaultsandbox.Inbox, activeIdx int) Model {
	ctx, cancel := context.WithCancel(context.Background())

	// Create list with custom delegate
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(styles.Purple).
		BorderForeground(styles.Purple)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(styles.Gray)

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
				p.Send(emailReceivedMsg{
					email:      email,
					inboxLabel: inbox.EmailAddress(),
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
			case key.Matches(msg, DefaultKeyMap.ViewHTML):
				if m.viewedEmail != nil && m.viewedEmail.Email.HTML != "" {
					return m, m.viewHTML()
				}
			// Links/Attachments view: up/down to navigate, enter to open/save
			case key.Matches(msg, DefaultKeyMap.Up):
				if m.viewedEmail != nil {
					if m.detailView == ViewLinks && len(m.viewedEmail.Email.Links) > 0 {
						m.selectedLink--
						if m.selectedLink < 0 {
							m.selectedLink = len(m.viewedEmail.Email.Links) - 1
						}
						m.viewport.SetContent(m.renderLinksView())
						return m, nil
					}
					if m.detailView == ViewAttachments && len(m.viewedEmail.Email.Attachments) > 0 {
						m.selectedAttachment--
						if m.selectedAttachment < 0 {
							m.selectedAttachment = len(m.viewedEmail.Email.Attachments) - 1
						}
						m.viewport.SetContent(m.renderAttachmentsView())
						return m, nil
					}
				}
			case key.Matches(msg, DefaultKeyMap.Down):
				if m.viewedEmail != nil {
					if m.detailView == ViewLinks && len(m.viewedEmail.Email.Links) > 0 {
						m.selectedLink++
						if m.selectedLink >= len(m.viewedEmail.Email.Links) {
							m.selectedLink = 0
						}
						m.viewport.SetContent(m.renderLinksView())
						return m, nil
					}
					if m.detailView == ViewAttachments && len(m.viewedEmail.Email.Attachments) > 0 {
						m.selectedAttachment++
						if m.selectedAttachment >= len(m.viewedEmail.Email.Attachments) {
							m.selectedAttachment = 0
						}
						m.viewport.SetContent(m.renderAttachmentsView())
						return m, nil
					}
				}
			case key.Matches(msg, DefaultKeyMap.Enter):
				if m.viewedEmail != nil {
					if m.detailView == ViewLinks && len(m.viewedEmail.Email.Links) > 0 {
						return m, m.openLinkByIndex(m.selectedLink)
					}
					if m.detailView == ViewAttachments && len(m.viewedEmail.Email.Attachments) > 0 {
						return m, m.saveAttachment(m.selectedAttachment)
					}
				}
			// Number keys: switch tabs
			default:
				if m.viewedEmail != nil && len(msg.String()) == 1 {
					r := msg.String()[0]
					switch r {
					case '1':
						m.detailView = ViewContent
						m.viewport.SetContent(m.renderEmailDetail())
						m.viewport.GotoTop()
					case '2':
						m.detailView = ViewSecurity
						m.viewport.SetContent(m.renderSecurityView())
						m.viewport.GotoTop()
					case '3':
						m.detailView = ViewLinks
						m.selectedLink = 0
						m.viewport.SetContent(m.renderLinksView())
						m.viewport.GotoTop()
					case '4':
						m.detailView = ViewAttachments
						m.selectedAttachment = 0
						m.viewport.SetContent(m.renderAttachmentsView())
						m.viewport.GotoTop()
					case '5':
						m.detailView = ViewRaw
						m.viewport.SetContent(m.renderRawView())
						m.viewport.GotoTop()
					}
					return m, nil
				}
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
			if len(m.filteredEmails()) > 0 {
				if i := m.list.Index(); i >= 0 && i < len(m.filteredEmails()) {
					filtered := m.filteredEmails()
					m.viewing = true
					m.viewedEmail = &filtered[i]
					m.viewport.SetContent(m.renderEmailDetail())
					m.viewport.GotoTop()
				}
			}
			return m, nil
		case key.Matches(msg, DefaultKeyMap.OpenURL):
			if len(m.filteredEmails()) > 0 {
				return m, m.openFirstURL()
			}
		case key.Matches(msg, DefaultKeyMap.ViewHTML):
			if len(m.filteredEmails()) > 0 {
				return m, m.viewHTML()
			}
		case key.Matches(msg, DefaultKeyMap.Delete):
			if len(m.filteredEmails()) > 0 {
				return m, m.deleteEmail()
			}
		case key.Matches(msg, DefaultKeyMap.PrevInbox):
			if len(m.inboxes) > 0 {
				m.currentInboxIdx--
				if m.currentInboxIdx < 0 {
					m.currentInboxIdx = len(m.inboxes) - 1
				}
				m.updateFilteredList()
			}
			return m, nil
		case key.Matches(msg, DefaultKeyMap.NextInbox):
			if len(m.inboxes) > 0 {
				m.currentInboxIdx++
				if m.currentInboxIdx >= len(m.inboxes) {
					m.currentInboxIdx = 0
				}
				m.updateFilteredList()
			}
			return m, nil
		case key.Matches(msg, DefaultKeyMap.NewInbox):
			return m, m.createNewInbox()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-6)
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 8
		// Refresh list after sizing
		if m.connected {
			m.updateFilteredList()
		}

	case connectedMsg:
		m.connected = true
		m.updateFilteredList()

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
		m.updateFilteredList()

	case errMsg:
		m.lastError = msg.err
		m.connected = false
		m.updateTitle()

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
		m.updateFilteredList()

	case attachmentSavedMsg:
		if msg.err != nil {
			m.lastError = msg.err
		} else {
			m.lastSavedFile = msg.filename
		}
		m.viewport.SetContent(m.renderAttachmentsView())
		return m, nil

	case inboxCreatedMsg:
		if msg.err != nil {
			m.lastError = msg.err
			return m, nil
		}
		// Add inbox and switch to it
		m.inboxes = append(m.inboxes, msg.inbox)
		m.currentInboxIdx = len(m.inboxes) - 1
		m.updateFilteredList()
		// Start watching the new inbox
		return m, m.watchNewInbox(msg.inbox)
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
	help := styles.HelpStyle.Render("q: quit • enter: view • o: open • v: html • d: delete • ←/→: inbox • n: new")

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.list.View(),
		help,
	)

	return styles.AppStyle.Render(content)
}

func (m Model) viewDetail() string {
	if m.viewedEmail == nil {
		return ""
	}

	// Help text
	help := styles.HelpStyle.Render("1-5: tabs • v: html • esc: back • q: quit")

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left,
		m.viewport.View(),
		help,
	)

	return styles.AppStyle.Render(content)
}

// renderTabs renders the tab bar with the active tab highlighted
func (m Model) renderTabs() string {
	tabs := []string{"Content", "Security", "Links", "Attach", "Raw"}
	var rendered []string
	for i, tab := range tabs {
		if DetailView(i) == m.detailView {
			rendered = append(rendered, styles.TabActiveStyle.Render(fmt.Sprintf("%d %s", i+1, tab)))
		} else {
			rendered = append(rendered, styles.TabStyle.Render(fmt.Sprintf("%d %s", i+1, tab)))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m Model) renderEmailDetail() string {
	if m.viewedEmail == nil {
		return ""
	}

	email := m.viewedEmail.Email
	var sb strings.Builder

	// Tab bar
	sb.WriteString(m.renderTabs())
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

// filteredEmails returns emails for the current inbox filter
func (m Model) filteredEmails() []EmailItem {
	if m.currentInboxIdx < 0 || m.currentInboxIdx >= len(m.inboxes) {
		return m.emails // show all
	}
	currentInbox := m.inboxes[m.currentInboxIdx].EmailAddress()
	var filtered []EmailItem
	for _, e := range m.emails {
		if e.InboxLabel == currentInbox {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// updateFilteredList updates the list with filtered emails
func (m *Model) updateFilteredList() {
	filtered := m.filteredEmails()
	items := make([]list.Item, len(filtered))
	for i, e := range filtered {
		items[i] = e
	}
	m.list.SetItems(items)
	m.updateTitle()
}

// updateTitle updates the list title with current inbox info
func (m *Model) updateTitle() {
	var title string
	if !m.connected {
		title = "Disconnected"
	} else if m.lastError != nil {
		title = "Error: " + m.lastError.Error()
	} else if len(m.inboxes) > 1 {
		title = fmt.Sprintf("[%d/%d] %s • %d emails", m.currentInboxIdx+1, len(m.inboxes), m.currentInboxLabel(), len(m.filteredEmails()))
	} else if len(m.inboxes) == 1 {
		title = fmt.Sprintf("%s • %d emails", m.currentInboxLabel(), len(m.filteredEmails()))
	} else {
		title = "No inboxes"
	}
	m.list.Title = title
}

// currentInboxLabel returns the label for the current inbox
func (m Model) currentInboxLabel() string {
	if m.currentInboxIdx >= 0 && m.currentInboxIdx < len(m.inboxes) {
		return m.inboxes[m.currentInboxIdx].EmailAddress()
	}
	return "all"
}

func (m Model) openFirstURL() tea.Cmd {
	return func() tea.Msg {
		var email *vaultsandbox.Email
		if m.viewing && m.viewedEmail != nil {
			email = m.viewedEmail.Email
		} else {
			filtered := m.filteredEmails()
			if i := m.list.Index(); i >= 0 && i < len(filtered) {
				email = filtered[i].Email
			}
		}
		if email != nil && len(email.Links) > 0 {
			browser.OpenURL(email.Links[0])
		}
		return nil
	}
}

func (m Model) openLinkByIndex(index int) tea.Cmd {
	return func() tea.Msg {
		if m.viewedEmail != nil && index >= 0 && index < len(m.viewedEmail.Email.Links) {
			browser.OpenURL(m.viewedEmail.Email.Links[index])
		}
		return nil
	}
}

func (m Model) createNewInbox() tea.Cmd {
	return func() tea.Msg {
		inbox, err := m.client.CreateInbox(m.ctx)
		return inboxCreatedMsg{inbox: inbox, err: err}
	}
}

func (m Model) watchNewInbox(inbox *vaultsandbox.Inbox) tea.Cmd {
	// Note: New inbox watching requires program reference
	// For now, new inboxes are added but won't receive real-time emails
	// until the watch command is restarted
	return nil
}

func (m Model) viewHTML() tea.Cmd {
	return func() tea.Msg {
		var email *vaultsandbox.Email
		if m.viewing && m.viewedEmail != nil {
			email = m.viewedEmail.Email
		} else {
			filtered := m.filteredEmails()
			if i := m.list.Index(); i >= 0 && i < len(filtered) {
				email = filtered[i].Email
			}
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
		filtered := m.filteredEmails()
		if i := m.list.Index(); i >= 0 && i < len(filtered) {
			emailItem := filtered[i]
			// Find inbox for this email
			for _, inbox := range m.inboxes {
				if len(m.inboxes) > 1 {
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
