package emails

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// renderRawView renders the raw email headers and content
func (m Model) renderRawView() string {
	return m.renderDetailView("No email selected", func(email *vaultsandbox.Email, b *strings.Builder) {
		labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
		headerKeyStyle := lipgloss.NewStyle().Foreground(styles.Gray)
		headerValStyle := lipgloss.NewStyle().Foreground(styles.White)

		// Raw Headers
		b.WriteString(labelStyle.Render("RAW HEADERS"))
		b.WriteString("\n\n")

		if len(email.Headers) > 0 {
			// Sort headers for consistent display
			keys := make([]string, 0, len(email.Headers))
			for k := range email.Headers {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				v := email.Headers[k]
				b.WriteString(headerKeyStyle.Render(k + ": "))
				b.WriteString(headerValStyle.Render(v))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(styles.HelpStyle.Render("No raw headers available"))
			b.WriteString("\n")
		}

		// Raw body
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("RAW TEXT BODY"))
		b.WriteString("\n\n")

		if email.Text != "" {
			b.WriteString(email.Text)
		} else {
			b.WriteString(styles.HelpStyle.Render("No text body available"))
		}

		// HTML indicator
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("HTML BODY"))
		b.WriteString("\n\n")

		if email.HTML != "" {
			// Show first 500 chars of HTML
			html := email.HTML
			if len(html) > 500 {
				html = html[:500] + fmt.Sprintf("\n... (%d more bytes)", len(email.HTML)-500)
			}
			b.WriteString(styles.HelpStyle.Render(html))
			b.WriteString("\n\n")
			b.WriteString(styles.HelpStyle.Render("Press 'v' to view full HTML in browser"))
		} else {
			b.WriteString(styles.HelpStyle.Render("No HTML body available"))
		}
	})
}
