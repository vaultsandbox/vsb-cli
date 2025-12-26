package emails

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// renderSecurityView renders the security audit view for an email
func (m Model) renderSecurityView() string {
	if m.viewedEmail == nil {
		return ""
	}

	email := m.viewedEmail.Email
	var sb strings.Builder

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Width(16)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.White).MarginTop(1)

	// Tab bar
	sb.WriteString(m.renderTabs())
	sb.WriteString("\n\n")

	// Authentication
	sb.WriteString(sectionStyle.Render("AUTHENTICATION"))
	sb.WriteString("\n")
	sb.WriteString(styles.RenderAuthResults(email.AuthResults, labelStyle, true))
	sb.WriteString("\n")

	// Transport Security
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("TRANSPORT SECURITY"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("TLS:"), styles.PassStyle.Render("TLS 1.3")))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("E2E:"), styles.PassStyle.Render(styles.EncryptionLabel)))

	// Security Score
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("SECURITY SCORE"))
	sb.WriteString("\n")
	score := styles.CalculateScore(email)
	sb.WriteString(styles.ScoreStyle(score).Render(fmt.Sprintf("%d/100", score)))
	sb.WriteString("\n")

	return sb.String()
}

