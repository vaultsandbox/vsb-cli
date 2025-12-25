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
	indexStyle := lipgloss.NewStyle().Foreground(styles.Gray)

	// Tab indicator
	sb.WriteString(styles.HelpStyle.Render("[1:Content] [2:Security] [3:Links] [4:Raw]"))
	sb.WriteString("\n")
	sb.WriteString(styles.HelpStyle.Render("                         ^^^^^^^"))
	sb.WriteString("\n\n")

	if len(email.Links) == 0 {
		sb.WriteString(styles.HelpStyle.Render("No links found in this email"))
		return sb.String()
	}

	sb.WriteString(labelStyle.Render(fmt.Sprintf("Found %d links:\n\n", len(email.Links))))

	for i, link := range email.Links {
		sb.WriteString(indexStyle.Render(fmt.Sprintf("%2d. ", i+1)))
		sb.WriteString(linkStyle.Render(link))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styles.HelpStyle.Render("Press 'o' to open first link, or number key (1-9) to open specific link"))

	return sb.String()
}
