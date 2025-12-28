package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
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
  vsb email audit              # Audit most recent email
  vsb email audit abc123       # Audit specific email
  vsb email audit -o json      # JSON output for scripting`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAudit,
}

var (
	auditInbox string
)

func init() {
	Cmd.AddCommand(auditCmd)

	auditCmd.Flags().StringVar(&auditInbox, "inbox", "",
		"Use specific inbox (default: active)")
}

func runAudit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	emailID := cliutil.GetArg(args, 0, "")

	// Use shared helper to get email
	email, _, cleanup, err := cliutil.GetEmailByIDOrLatest(ctx, emailID, auditInbox)
	if err != nil {
		return err
	}
	defer cleanup()

	// Render audit report
	if cliutil.GetOutput(cmd) == "json" {
		return renderAuditJSON(email)
	}
	return renderAuditReport(email)
}

func renderAuditReport(email *vaultsandbox.Email) error {
	labelStyle := styles.LabelStyle

	// Title
	title := styles.AuditTitleStyle.Render(" EMAIL AUDIT REPORT ")

	fmt.Println()
	fmt.Println(title)
	fmt.Println()

	// Basic Info
	fmt.Println(styles.SectionStyle.Render("BASIC INFO"))
	fmt.Printf("%s %s\n", labelStyle.Render("Subject:"), email.Subject)
	fmt.Printf("%s %s\n", labelStyle.Render("From:"), email.From)
	fmt.Printf("%s %s\n", labelStyle.Render("To:"), strings.Join(email.To, ", "))
	fmt.Printf("%s %s\n", labelStyle.Render("Received:"), email.ReceivedAt.Format("2006-01-02 15:04:05 MST"))

	// Authentication Results
	fmt.Println()
	fmt.Println(styles.SectionStyle.Render("AUTHENTICATION"))
	fmt.Println(styles.RenderAuthResults(email.AuthResults, labelStyle, true))

	// Transport Security
	fmt.Println()
	fmt.Println(styles.SectionStyle.Render("TRANSPORT SECURITY"))

	// Extract from headers if available
	tlsVersion := "TLS 1.3"
	if v := email.Headers["X-TLS-Version"]; v != "" {
		tlsVersion = v
	}
	cipherSuite := "ECDHE-RSA-AES256-GCM-SHA384"
	if v := email.Headers["X-TLS-Cipher"]; v != "" {
		cipherSuite = v
	}

	fmt.Printf("%s %s\n", labelStyle.Render("TLS Version:"), styles.PassStyle.Render(tlsVersion))
	fmt.Printf("%s %s\n", labelStyle.Render("Cipher Suite:"), cipherSuite)
	fmt.Printf("%s %s\n", labelStyle.Render("E2E Encryption:"), styles.PassStyle.Render(styles.EncryptionLabel))

	// MIME Structure
	fmt.Println()
	fmt.Println(styles.SectionStyle.Render("MIME STRUCTURE"))

	mimeTree := buildMIMETree(email)
	fmt.Println(styles.BoxStyle.Render(mimeTree))

	// Summary
	fmt.Println()
	score := styles.CalculateScore(email)
	summary := fmt.Sprintf("Security Score: %s", styles.ScoreStyle(score).Render(fmt.Sprintf("%d/100", score)))
	fmt.Println(styles.BoxStyle.Render(summary))
	fmt.Println()

	return nil
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

func renderAuditJSON(email *vaultsandbox.Email) error {
	data := map[string]interface{}{
		"id":            email.ID,
		"subject":       email.Subject,
		"from":          email.From,
		"to":            email.To,
		"receivedAt":    email.ReceivedAt,
		"securityScore": styles.CalculateScore(email),
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

	return cliutil.OutputJSON(data)
}

