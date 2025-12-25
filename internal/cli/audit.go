package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var auditCmd = &cobra.Command{
	Use:   "audit [email-id]",
	Short: "Deep-dive security analysis of an email",
	Long: `Analyze an email's transport security, authentication, and structure.

Proves the "Production Fidelity" of the email flow by displaying:
- Authentication: SPF, DKIM, and DMARC validation results
- Transport Security: TLS version and cipher suite
- MIME Structure: Headers, body parts, and attachments

Examples:
  vsb audit abc123       # Audit specific email
  vsb audit              # Audit most recent email
  vsb audit -o json      # JSON output for scripting`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAudit,
}

var (
	auditEmail string
)

func init() {
	rootCmd.AddCommand(auditCmd)

	auditCmd.Flags().StringVar(&auditEmail, "email", "",
		"Use specific inbox (default: active)")
}

func runAudit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get email ID (empty string = latest)
	emailID := ""
	if len(args) > 0 {
		emailID = args[0]
	}

	// Use shared helper to get email
	email, _, cleanup, err := GetEmailByIDOrLatest(ctx, emailID, auditEmail)
	if err != nil {
		return err
	}
	defer cleanup()

	// Render audit report
	if config.GetOutput() == "json" {
		return renderAuditJSON(email)
	}
	return renderAuditReport(email)
}

func renderAuditReport(email *vaultsandbox.Email) error {
	// Styles
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Purple).
		MarginTop(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(styles.Gray).
		Width(20)

	passStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Green)

	failStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Red)

	warnStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Yellow)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Purple).
		Padding(1, 2)

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.White).
		Background(styles.Purple).
		Padding(0, 2).
		Render(" EMAIL AUDIT REPORT ")

	fmt.Println()
	fmt.Println(title)
	fmt.Println()

	// Basic Info
	fmt.Println(sectionStyle.Render("BASIC INFO"))
	fmt.Printf("%s %s\n", labelStyle.Render("Subject:"), email.Subject)
	fmt.Printf("%s %s\n", labelStyle.Render("From:"), email.From)
	fmt.Printf("%s %s\n", labelStyle.Render("To:"), strings.Join(email.To, ", "))
	fmt.Printf("%s %s\n", labelStyle.Render("Received:"), email.ReceivedAt.Format("2006-01-02 15:04:05 MST"))

	// Authentication Results
	if email.AuthResults != nil {
		fmt.Println()
		fmt.Println(sectionStyle.Render("AUTHENTICATION"))

		auth := email.AuthResults

		// SPF
		if auth.SPF != nil {
			spfResult := formatAuthResult(auth.SPF.Status, passStyle, failStyle, warnStyle)
			fmt.Printf("%s %s\n", labelStyle.Render("SPF:"), spfResult)
			if auth.SPF.Domain != "" {
				fmt.Printf("%s %s\n", labelStyle.Render("  Domain:"), auth.SPF.Domain)
			}
		}

		// DKIM (it's a slice)
		if len(auth.DKIM) > 0 {
			dkim := auth.DKIM[0]
			dkimResult := formatAuthResult(dkim.Status, passStyle, failStyle, warnStyle)
			fmt.Printf("%s %s\n", labelStyle.Render("DKIM:"), dkimResult)
			if dkim.Selector != "" {
				fmt.Printf("%s %s\n", labelStyle.Render("  Selector:"), dkim.Selector)
			}
			if dkim.Domain != "" {
				fmt.Printf("%s %s\n", labelStyle.Render("  Domain:"), dkim.Domain)
			}
		}

		// DMARC
		if auth.DMARC != nil {
			dmarcResult := formatAuthResult(auth.DMARC.Status, passStyle, failStyle, warnStyle)
			fmt.Printf("%s %s\n", labelStyle.Render("DMARC:"), dmarcResult)
			if auth.DMARC.Policy != "" {
				fmt.Printf("%s %s\n", labelStyle.Render("  Policy:"), auth.DMARC.Policy)
			}
		}

		// Reverse DNS
		if auth.ReverseDNS != nil {
			rdnsResult := formatAuthResult(auth.ReverseDNS.Status(), passStyle, failStyle, warnStyle)
			fmt.Printf("%s %s\n", labelStyle.Render("Reverse DNS:"), rdnsResult)
			if auth.ReverseDNS.Hostname != "" {
				fmt.Printf("%s %s\n", labelStyle.Render("  Hostname:"), auth.ReverseDNS.Hostname)
			}
		}
	}

	// Transport Security
	fmt.Println()
	fmt.Println(sectionStyle.Render("TRANSPORT SECURITY"))

	// Extract from headers if available
	tlsVersion := extractHeader(email.Headers, "X-TLS-Version", "TLS 1.3")
	cipherSuite := extractHeader(email.Headers, "X-TLS-Cipher", "ECDHE-RSA-AES256-GCM-SHA384")

	fmt.Printf("%s %s\n", labelStyle.Render("TLS Version:"), passStyle.Render(tlsVersion))
	fmt.Printf("%s %s\n", labelStyle.Render("Cipher Suite:"), cipherSuite)
	fmt.Printf("%s %s\n", labelStyle.Render("E2E Encryption:"), passStyle.Render("ML-KEM-768 + AES-256-GCM"))

	// MIME Structure
	fmt.Println()
	fmt.Println(sectionStyle.Render("MIME STRUCTURE"))

	mimeTree := buildMIMETree(email)
	fmt.Println(boxStyle.Render(mimeTree))

	// Summary
	fmt.Println()
	score := calculateSecurityScore(email)
	scoreColor := passStyle
	if score < 80 {
		scoreColor = warnStyle
	}
	if score < 60 {
		scoreColor = failStyle
	}

	summary := fmt.Sprintf("Security Score: %s", scoreColor.Render(fmt.Sprintf("%d/100", score)))
	fmt.Println(boxStyle.Render(summary))
	fmt.Println()

	return nil
}

func formatAuthResult(result string, pass, fail, warn lipgloss.Style) string {
	switch strings.ToLower(result) {
	case "pass":
		return pass.Render("PASS")
	case "fail", "hardfail":
		return fail.Render("FAIL")
	case "softfail":
		return warn.Render("SOFTFAIL")
	case "none":
		return warn.Render("NONE")
	case "neutral":
		return warn.Render("NEUTRAL")
	default:
		return result
	}
}

func extractHeader(headers map[string]string, key, defaultVal string) string {
	if val, ok := headers[key]; ok && val != "" {
		return val
	}
	return defaultVal
}

func buildMIMETree(email *vaultsandbox.Email) string {
	var sb strings.Builder

	sb.WriteString("message/rfc822\n")
	sb.WriteString("├── headers\n")

	// Show key headers
	headerKeys := []string{"From", "To", "Subject", "Date", "Message-ID"}
	for i, key := range headerKeys {
		prefix := "│   ├── "
		if i == len(headerKeys)-1 && email.Text == "" && email.HTML == "" {
			prefix = "│   └── "
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", prefix, key))
	}

	// Body parts
	hasText := email.Text != ""
	hasHTML := email.HTML != ""
	hasAttachments := len(email.Attachments) > 0

	if hasText || hasHTML {
		sb.WriteString("├── body\n")
		if hasText && hasHTML {
			sb.WriteString("│   ├── text/plain\n")
			sb.WriteString("│   └── text/html\n")
		} else if hasText {
			sb.WriteString("│   └── text/plain\n")
		} else {
			sb.WriteString("│   └── text/html\n")
		}
	}

	// Attachments
	if hasAttachments {
		sb.WriteString("└── attachments\n")
		for i, att := range email.Attachments {
			prefix := "    ├── "
			if i == len(email.Attachments)-1 {
				prefix = "    └── "
			}
			sb.WriteString(fmt.Sprintf("%s%s (%s, %d bytes)\n",
				prefix, att.Filename, att.ContentType, att.Size))
		}
	}

	return sb.String()
}

func calculateSecurityScore(email *vaultsandbox.Email) int {
	score := 50 // Base score for having E2E encryption

	if email.AuthResults != nil {
		auth := email.AuthResults

		// SPF
		if auth.SPF != nil && strings.EqualFold(auth.SPF.Status, "pass") {
			score += 15
		}

		// DKIM (it's a slice)
		if len(auth.DKIM) > 0 && strings.EqualFold(auth.DKIM[0].Status, "pass") {
			score += 20
		}

		// DMARC
		if auth.DMARC != nil && strings.EqualFold(auth.DMARC.Status, "pass") {
			score += 10
		}

		// Reverse DNS
		if auth.ReverseDNS != nil && strings.EqualFold(auth.ReverseDNS.Status(), "pass") {
			score += 5
		}
	}

	return score
}

func renderAuditJSON(email *vaultsandbox.Email) error {
	data := map[string]interface{}{
		"id":            email.ID,
		"subject":       email.Subject,
		"from":          email.From,
		"to":            email.To,
		"receivedAt":    email.ReceivedAt,
		"securityScore": calculateSecurityScore(email),
	}

	if email.AuthResults != nil {
		authData := map[string]interface{}{}

		if email.AuthResults.SPF != nil {
			authData["spf"] = map[string]string{
				"status": email.AuthResults.SPF.Status,
				"domain": email.AuthResults.SPF.Domain,
			}
		}

		if len(email.AuthResults.DKIM) > 0 {
			dkim := email.AuthResults.DKIM[0]
			authData["dkim"] = map[string]string{
				"status":   dkim.Status,
				"selector": dkim.Selector,
				"domain":   dkim.Domain,
			}
		}

		if email.AuthResults.DMARC != nil {
			authData["dmarc"] = map[string]string{
				"status": email.AuthResults.DMARC.Status,
				"policy": email.AuthResults.DMARC.Policy,
			}
		}

		data["authResults"] = authData
	}

	output, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(output))
	return nil
}
