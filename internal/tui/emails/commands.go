package emails

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vaultsandbox/vsb-cli/internal/browser"
)

func (m Model) openFirstURL() tea.Cmd {
	return func() tea.Msg {
		if email := m.selectedEmail(); email != nil && len(email.Links) > 0 {
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

func (m Model) viewHTML() tea.Cmd {
	return func() tea.Msg {
		if email := m.selectedEmail(); email != nil && email.HTML != "" {
			browser.ViewEmailHTML(email.Subject, email.From, email.ReceivedAt, email.HTML)
		}
		return nil
	}
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
