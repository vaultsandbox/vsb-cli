package emails

import (
	"strings"

	vaultsandbox "github.com/vaultsandbox/client-go"
)

// renderDetailView handles common view rendering with a content function.
// It renders the tab bar, checks for nil email, and delegates content rendering
// to the provided callback.
func (m Model) renderDetailView(emptyMsg string, renderContent func(*vaultsandbox.Email, *strings.Builder)) string {
	var b strings.Builder
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	if m.viewedEmail == nil {
		b.WriteString("  ")
		b.WriteString(emptyMsg)
		return b.String()
	}

	renderContent(m.viewedEmail.Email, &b)
	return b.String()
}
