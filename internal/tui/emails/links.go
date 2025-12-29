package emails

import (
	"fmt"
	"strings"

	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// renderLinksView renders the links list view
func (m Model) renderLinksView() string {
	return m.renderDetailView("No email selected", func(email *vaultsandbox.Email, b *strings.Builder) {
		if len(email.Links) == 0 {
			b.WriteString(styles.HelpStyle.Render("No links found in this email"))
			return
		}

		b.WriteString(styles.ListLabelStyle.Render(fmt.Sprintf("Found %d links:", len(email.Links))))
		b.WriteString("\n\n")

		for i, link := range email.Links {
			if i == m.selectedLink {
				b.WriteString(styles.ListSelectedStyle.Render(">"))
				b.WriteString(" " + link + "\n")
			} else {
				b.WriteString("  " + link + "\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render("↑/↓: select • enter: open"))
	})
}
