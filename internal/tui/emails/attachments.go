package emails

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/files"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// attachmentSavedMsg is sent after saving an attachment
type attachmentSavedMsg struct {
	filename string
	err      error
}

// renderAttachmentsView renders the attachments list view
func (m Model) renderAttachmentsView() string {
	return m.renderDetailView("No email selected", func(email *vaultsandbox.Email, b *strings.Builder) {
		if len(email.Attachments) == 0 {
			b.WriteString(styles.HelpStyle.Render("No attachments in this email"))
			return
		}

		b.WriteString(styles.ListLabelStyle.Render(fmt.Sprintf("Found %d attachments:", len(email.Attachments))))
		b.WriteString("\n\n")

		for i, att := range email.Attachments {
			info := fmt.Sprintf(" (%s, %s)", att.ContentType, humanize.Bytes(uint64(att.Size)))
			if i == m.selectedAttachment {
				b.WriteString(styles.ListSelectedStyle.Render(">"))
				b.WriteString(" " + att.Filename + styles.ListSizeStyle.Render(info) + "\n")
			} else {
				b.WriteString("  " + att.Filename + styles.ListSizeStyle.Render(info) + "\n")
			}
		}

		b.WriteString("\n")
		if m.lastSavedFile != "" {
			b.WriteString(styles.PassStyle.Render("Saved: " + m.lastSavedFile))
			b.WriteString("\n\n")
		}
		b.WriteString(styles.HelpStyle.Render("↑/↓: select • enter: save to current directory"))
	})
}

// saveAttachment saves the attachment at the given index
func (m Model) saveAttachment(index int) tea.Cmd {
	return func() tea.Msg {
		if m.viewedEmail == nil || index < 0 || index >= len(m.viewedEmail.Email.Attachments) {
			return nil
		}

		att := m.viewedEmail.Email.Attachments[index]
		path, err := files.SaveFile(".", att.Filename, att.Content)
		return attachmentSavedMsg{filename: path, err: err}
	}
}
