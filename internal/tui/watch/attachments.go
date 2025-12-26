package watch

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	if m.viewedEmail == nil {
		return ""
	}

	email := m.viewedEmail.Email
	var sb strings.Builder

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	sizeStyle := lipgloss.NewStyle().Foreground(styles.Gray)

	// Tab bar
	sb.WriteString(m.renderTabs())
	sb.WriteString("\n\n")

	if len(email.Attachments) == 0 {
		sb.WriteString(styles.HelpStyle.Render("No attachments in this email"))
		return sb.String()
	}

	sb.WriteString(labelStyle.Render(fmt.Sprintf("Found %d attachments:", len(email.Attachments))))
	sb.WriteString("\n\n")

	for i, att := range email.Attachments {
		info := fmt.Sprintf(" (%s, %s)", att.ContentType, formatSize(att.Size))
		if i == m.selectedAttachment {
			sb.WriteString(selectedStyle.Render(">"))
			sb.WriteString(" " + att.Filename + sizeStyle.Render(info) + "\n")
		} else {
			sb.WriteString("  " + att.Filename + sizeStyle.Render(info) + "\n")
		}
	}

	sb.WriteString("\n")
	if m.lastSavedFile != "" {
		savedStyle := lipgloss.NewStyle().Foreground(styles.Green)
		sb.WriteString(savedStyle.Render("Saved: " + m.lastSavedFile))
		sb.WriteString("\n\n")
	}
	sb.WriteString(styles.HelpStyle.Render("↑/↓: select • enter: save to current directory"))

	return sb.String()
}

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
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
