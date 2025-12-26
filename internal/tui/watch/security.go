package watch

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// renderSecurityView renders the security audit view for an email
func (m Model) renderSecurityView() string {
	if m.viewedEmail == nil {
		return ""
	}

	email := m.viewedEmail.Email
	var sb strings.Builder

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Purple).Width(16)
	passStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Green)
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Red)
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Yellow)
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
			spfResult := formatResult(auth.SPF.Status, passStyle, failStyle, warnStyle)
			sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("SPF:"), spfResult))
			if auth.SPF.Domain != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", auth.SPF.Domain))
			}
			sb.WriteString("\n")
		}

		// DKIM (it's a slice)
		if len(auth.DKIM) > 0 {
			dkim := auth.DKIM[0]
			dkimResult := formatResult(dkim.Status, passStyle, failStyle, warnStyle)
			sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("DKIM:"), dkimResult))
			if dkim.Domain != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", dkim.Domain))
			}
			sb.WriteString("\n")
		}

		// DMARC
		if auth.DMARC != nil {
			dmarcResult := formatResult(auth.DMARC.Status, passStyle, failStyle, warnStyle)
			sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("DMARC:"), dmarcResult))
			if auth.DMARC.Policy != "" {
				sb.WriteString(fmt.Sprintf(" (policy: %s)", auth.DMARC.Policy))
			}
			sb.WriteString("\n")
		}

		// Reverse DNS
		if auth.ReverseDNS != nil {
			rdnsResult := formatResult(auth.ReverseDNS.Status(), passStyle, failStyle, warnStyle)
			sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("Reverse DNS:"), rdnsResult))
			if auth.ReverseDNS.Hostname != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", auth.ReverseDNS.Hostname))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(warnStyle.Render("No authentication results available"))
		sb.WriteString("\n")
	}

	// Transport Security
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("TRANSPORT SECURITY"))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("TLS:"), passStyle.Render("TLS 1.3")))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("E2E:"), passStyle.Render("ML-KEM-768 + AES-256-GCM")))

	// Security Score
	sb.WriteString("\n")
	sb.WriteString(sectionStyle.Render("SECURITY SCORE"))
	sb.WriteString("\n")
	score := calculateScore(email)
	scoreStyle := passStyle
	if score < 80 {
		scoreStyle = warnStyle
	}
	if score < 60 {
		scoreStyle = failStyle
	}
	sb.WriteString(scoreStyle.Render(fmt.Sprintf("%d/100", score)))
	sb.WriteString("\n")

	return sb.String()
}

func formatResult(result string, pass, fail, warn lipgloss.Style) string {
	switch strings.ToLower(result) {
	case "pass":
		return pass.Render("PASS")
	case "fail", "hardfail":
		return fail.Render("FAIL")
	case "softfail":
		return warn.Render("SOFTFAIL")
	case "none", "neutral":
		return warn.Render(strings.ToUpper(result))
	default:
		return result
	}
}

func calculateScore(email *vaultsandbox.Email) int {
	score := 50 // Base for E2E

	if email.AuthResults != nil {
		if email.AuthResults.SPF != nil && strings.EqualFold(email.AuthResults.SPF.Status, "pass") {
			score += 15
		}
		if len(email.AuthResults.DKIM) > 0 && strings.EqualFold(email.AuthResults.DKIM[0].Status, "pass") {
			score += 20
		}
		if email.AuthResults.DMARC != nil && strings.EqualFold(email.AuthResults.DMARC.Status, "pass") {
			score += 10
		}
		if email.AuthResults.ReverseDNS != nil && strings.EqualFold(email.AuthResults.ReverseDNS.Status(), "pass") {
			score += 5
		}
	}

	return score
}
