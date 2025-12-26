package watch

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// renderLinksView renders the links list view
func (m Model) renderLinksView() string {
	if m.viewedEmail == nil {
		return ""
	}

	email := m.viewedEmail.Email
	var sb strings.Builder

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)

	// Tab bar
	sb.WriteString(m.renderTabs())
	sb.WriteString("\n\n")

	if len(email.Links) == 0 {
		sb.WriteString(styles.HelpStyle.Render("No links found in this email"))
		return sb.String()
	}

	sb.WriteString(labelStyle.Render(fmt.Sprintf("Found %d links:", len(email.Links))))
	sb.WriteString("\n\n")

	for i, link := range email.Links {
		if i == m.selectedLink {
			sb.WriteString(selectedStyle.Render(">"))
			sb.WriteString(" " + link + "\n")
		} else {
			sb.WriteString("  " + link + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(styles.HelpStyle.Render("↑/↓: select • enter: open"))

	return sb.String()
}
