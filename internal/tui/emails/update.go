package emails

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.viewing {
			return m.handleDetailViewUpdate(msg)
		}
		if m.list.FilterState() == list.Filtering {
			break
		}
		return m.handleListViewUpdate(msg)

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
		// Save to keystore
		if m.keystore != nil {
			exported := msg.inbox.Export()
			if err := m.keystore.SaveInbox(exported); err != nil {
				m.lastError = err
				return m, nil
			}
		}
		// Add inbox and switch to it
		m.inboxes = append(m.inboxes, msg.inbox)
		m.currentInboxIdx = len(m.inboxes) - 1
		m.updateFilteredList()
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
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

// handleListNavigation handles up/down navigation in links and attachments views.
// Returns true if navigation was handled, false otherwise.
func (m *Model) handleListNavigation(delta int) bool {
	if m.viewedEmail == nil {
		return false
	}
	switch m.detailView {
	case ViewLinks:
		if len(m.viewedEmail.Email.Links) > 0 {
			m.selectedLink = wrapIndex(m.selectedLink, delta, len(m.viewedEmail.Email.Links))
			m.viewport.SetContent(m.renderLinksView())
			return true
		}
	case ViewAttachments:
		if len(m.viewedEmail.Email.Attachments) > 0 {
			m.selectedAttachment = wrapIndex(m.selectedAttachment, delta, len(m.viewedEmail.Email.Attachments))
			m.viewport.SetContent(m.renderAttachmentsView())
			return true
		}
	}
	return false
}

// handleDetailViewUpdate handles key events when viewing an email detail
func (m Model) handleDetailViewUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case key.Matches(msg, DefaultKeyMap.Up):
		if m.handleListNavigation(-1) {
			return m, nil
		}
	case key.Matches(msg, DefaultKeyMap.Down):
		if m.handleListNavigation(1) {
			return m, nil
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
	default:
		// Number keys: switch tabs
		if m.viewedEmail != nil && len(msg.String()) == 1 {
			if cmd := m.handleTabSwitch(msg.String()[0]); cmd != nil {
				return m, cmd
			}
		}
	}
	// Update viewport for scrolling
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleListViewUpdate handles key events when viewing the email list
func (m Model) handleListViewUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredEmails()
	hasEmails := len(filtered) > 0

	switch {
	case key.Matches(msg, DefaultKeyMap.Quit):
		m.cancel()
		return m, tea.Quit
	case key.Matches(msg, DefaultKeyMap.Enter):
		if i := m.list.Index(); i >= 0 && i < len(filtered) {
			m.viewing = true
			m.viewedEmail = &filtered[i]
			m.viewport.SetContent(m.renderEmailDetail())
			m.viewport.GotoTop()
		}
		return m, nil
	case key.Matches(msg, DefaultKeyMap.OpenURL):
		if hasEmails {
			return m, m.openFirstURL()
		}
	case key.Matches(msg, DefaultKeyMap.ViewHTML):
		if hasEmails {
			return m, m.viewHTML()
		}
	case key.Matches(msg, DefaultKeyMap.Delete):
		if hasEmails {
			return m, m.deleteEmail()
		}
	case key.Matches(msg, DefaultKeyMap.PrevInbox):
		if len(m.inboxes) > 0 {
			m.currentInboxIdx = wrapIndex(m.currentInboxIdx, -1, len(m.inboxes))
			m.updateFilteredList()
		}
		return m, nil
	case key.Matches(msg, DefaultKeyMap.NextInbox):
		if len(m.inboxes) > 0 {
			m.currentInboxIdx = wrapIndex(m.currentInboxIdx, 1, len(m.inboxes))
			m.updateFilteredList()
		}
		return m, nil
	case key.Matches(msg, DefaultKeyMap.NewInbox):
		return m, m.createNewInbox()
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// handleTabSwitch handles number key presses to switch tabs in detail view
func (m *Model) handleTabSwitch(r byte) tea.Cmd {
	type tabConfig struct {
		view       DetailView
		render     func() string
		resetIndex *int
	}

	tabs := map[byte]tabConfig{
		'1': {ViewContent, m.renderEmailDetail, nil},
		'2': {ViewSecurity, m.renderSecurityView, nil},
		'3': {ViewLinks, m.renderLinksView, &m.selectedLink},
		'4': {ViewAttachments, m.renderAttachmentsView, &m.selectedAttachment},
		'5': {ViewRaw, m.renderRawView, nil},
	}

	cfg, ok := tabs[r]
	if !ok {
		return nil
	}

	m.detailView = cfg.view
	if cfg.resetIndex != nil {
		*cfg.resetIndex = 0
	}
	m.viewport.SetContent(cfg.render())
	m.viewport.GotoTop()
	return nil
}
