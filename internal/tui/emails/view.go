package emails

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

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
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
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
		subject = noSubject
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
