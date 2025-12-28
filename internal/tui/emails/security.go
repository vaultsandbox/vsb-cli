package emails

import (
	"fmt"
	"regexp"
	"strings"

	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// tlsVersionRegex extracts TLS version from Received header
// e.g., "with ESMTPS (version=TLSv1.3 cipher=...)"
var tlsVersionRegex = regexp.MustCompile(`version=(TLSv[\d.]+)`)

// renderSecurityView renders the security audit view for an email
func (m Model) renderSecurityView() string {
	return m.renderDetailView("No email selected", func(email *vaultsandbox.Email, b *strings.Builder) {
		labelStyle := styles.DetailLabelStyle.Width(16)

		// Authentication
		b.WriteString(styles.DetailSectionStyle.Render("AUTHENTICATION"))
		b.WriteString("\n")
		b.WriteString(styles.RenderAuthResults(email.AuthResults, labelStyle, false))
		b.WriteString("\n")

		// Transport Security
		b.WriteString("\n")
		b.WriteString(styles.DetailSectionStyle.Render("TRANSPORT SECURITY"))
		b.WriteString("\n")
		if tlsVersion := extractTLSVersion(email.Headers["received"]); tlsVersion != "" {
			b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("TLS:"), styles.PassStyle.Render(tlsVersion)))
		} else {
			b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("TLS:"), styles.WarnStyle.Render("unknown")))
		}
		b.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("E2E:"), styles.PassStyle.Render(styles.EncryptionLabel)))

		// Security Score
		b.WriteString("\n")
		b.WriteString(styles.DetailSectionStyle.Render("SECURITY SCORE"))
		b.WriteString("\n")
		score := styles.CalculateScore(email)
		b.WriteString(styles.ScoreStyle(score).Render(fmt.Sprintf("%d/100", score)))
		b.WriteString("\n")
	})
}

// extractTLSVersion parses TLS version from Received header.
// Returns empty string if not found.
func extractTLSVersion(received string) string {
	if match := tlsVersionRegex.FindStringSubmatch(received); len(match) > 1 {
		return match[1]
	}
	return ""
}

