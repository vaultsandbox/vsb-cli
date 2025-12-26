package watch

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// renderRawView renders the raw email headers and content
func (m Model) renderRawView() string {
	if m.viewedEmail == nil {
		return ""
	}

	email := m.viewedEmail.Email
	var sb strings.Builder

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	headerKeyStyle := lipgloss.NewStyle().Foreground(styles.Gray)
	headerValStyle := lipgloss.NewStyle().Foreground(styles.White)

	// Tab bar
	sb.WriteString(m.renderTabs())
	sb.WriteString("\n\n")

	// Raw Headers
	sb.WriteString(labelStyle.Render("RAW HEADERS"))
	sb.WriteString("\n\n")

	if len(email.Headers) > 0 {
		// Sort headers for consistent display
		keys := make([]string, 0, len(email.Headers))
		for k := range email.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := email.Headers[k]
			sb.WriteString(headerKeyStyle.Render(k + ": "))
			sb.WriteString(headerValStyle.Render(v))
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(styles.HelpStyle.Render("No raw headers available"))
		sb.WriteString("\n")
	}

	// Raw body
	sb.WriteString("\n")
	sb.WriteString(labelStyle.Render("RAW TEXT BODY"))
	sb.WriteString("\n\n")

	if email.Text != "" {
		sb.WriteString(email.Text)
	} else {
		sb.WriteString(styles.HelpStyle.Render("No text body available"))
	}

	// HTML indicator
	sb.WriteString("\n\n")
	sb.WriteString(labelStyle.Render("HTML BODY"))
	sb.WriteString("\n\n")

	if email.HTML != "" {
		// Show first 500 chars of HTML
		html := email.HTML
		if len(html) > 500 {
			html = html[:500] + fmt.Sprintf("\n... (%d more bytes)", len(email.HTML)-500)
		}
		sb.WriteString(styles.HelpStyle.Render(html))
		sb.WriteString("\n\n")
		sb.WriteString(styles.HelpStyle.Render("Press 'v' to view full HTML in browser"))
	} else {
		sb.WriteString(styles.HelpStyle.Render("No HTML body available"))
	}

	return sb.String()
}
