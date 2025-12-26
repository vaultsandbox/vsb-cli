package watch

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vaultsandbox/vsb-cli/internal/security"
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

	if email.AuthResults != nil {
		auth := email.AuthResults

		// SPF
		if auth.SPF != nil {
			spfResult := styles.FormatAuthResult(auth.SPF.Status)
			sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("SPF:"), spfResult))
			if auth.SPF.Domain != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", auth.SPF.Domain))
			}
			sb.WriteString("\n")
		}

		// DKIM (it's a slice)
		if len(auth.DKIM) > 0 {
			dkim := auth.DKIM[0]
			dkimResult := styles.FormatAuthResult(dkim.Status)
			sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("DKIM:"), dkimResult))
			if dkim.Domain != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", dkim.Domain))
			}
			sb.WriteString("\n")
		}

		// DMARC
		if auth.DMARC != nil {
			dmarcResult := styles.FormatAuthResult(auth.DMARC.Status)
			sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("DMARC:"), dmarcResult))
			if auth.DMARC.Policy != "" {
				sb.WriteString(fmt.Sprintf(" (policy: %s)", auth.DMARC.Policy))
			}
			sb.WriteString("\n")
		}

		// Reverse DNS
		if auth.ReverseDNS != nil {
			rdnsResult := styles.FormatAuthResult(auth.ReverseDNS.Status())
			sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("Reverse DNS:"), rdnsResult))
			if auth.ReverseDNS.Hostname != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", auth.ReverseDNS.Hostname))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(styles.WarnStyle.Render("No authentication results available"))
		sb.WriteString("\n")
	}

	// Transport Security
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("TRANSPORT SECURITY"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("TLS:"), styles.PassStyle.Render("TLS 1.3")))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("E2E:"), styles.PassStyle.Render("ML-KEM-768 + AES-256-GCM")))

	// Security Score
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("SECURITY SCORE"))
	sb.WriteString("\n")
	score := security.CalculateScore(email)
	scoreStyle := styles.PassStyle
	if score < 80 {
		scoreStyle = styles.WarnStyle
	}
	if score < 60 {
		scoreStyle = styles.FailStyle
	}
	sb.WriteString(scoreStyle.Render(fmt.Sprintf("%d/100", score)))
	sb.WriteString("\n")

	return sb.String()
}

