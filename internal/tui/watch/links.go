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

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Purple)
	linkStyle := lipgloss.NewStyle().Foreground(styles.White)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Purple)

	// Tab bar
	sb.WriteString(m.renderTabs())
	sb.WriteString("\n\n")

	if len(email.Links) == 0 {
		sb.WriteString(styles.HelpStyle.Render("No links found in this email"))
		return sb.String()
	}

	sb.WriteString(labelStyle.Render(fmt.Sprintf("Found %d links:\n\n", len(email.Links))))

	for i, link := range email.Links {
		if i == m.selectedLink {
			sb.WriteString(selectedStyle.Render("> " + link))
		} else {
			sb.WriteString(linkStyle.Render("  " + link))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styles.HelpStyle.Render("↑/↓: select • enter: open"))

	return sb.String()
}
