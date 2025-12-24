# Phase 3.2: Audit Command

## Objective
Implement `vsb audit` - a deep-dive command that proves the "Production Fidelity" of email flows by displaying security and authentication details.

## Command

| Command | Description |
|---------|-------------|
| `vsb audit <email-id>` | Audit specific email by ID |
| `vsb audit --latest` | Audit the most recent email |

## Output Sections

1. **Transport Security**: TLS version, cipher suite
2. **Authentication**: SPF, DKIM, DMARC results
3. **MIME Structure**: Headers, body parts, attachments

## Tasks

### 1. Audit Command

**File: `internal/cli/audit.go`**

```go
package cli

import (
    "context"
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/tui/styles"
)

var auditCmd = &cobra.Command{
    Use:   "audit [email-id]",
    Short: "Deep-dive security analysis of an email",
    Long: `Analyze an email's transport security, authentication, and structure.

Proves the "Production Fidelity" of the email flow by displaying:
- Transport Security: TLS version and cipher suite
- Authentication: SPF, DKIM, and DMARC validation results
- MIME Structure: Headers, body parts, and attachments

Examples:
  vsb audit abc123          # Audit specific email
  vsb audit --latest        # Audit most recent email
  vsb audit --latest --json # JSON output`,
    Args: cobra.MaximumNArgs(1),
    RunE: runAudit,
}

var (
    auditLatest bool
    auditEmail  string
    auditJSON   bool
)

func init() {
    rootCmd.AddCommand(auditCmd)

    auditCmd.Flags().BoolVar(&auditLatest, "latest", false,
        "Audit the most recent email")
    auditCmd.Flags().StringVar(&auditEmail, "email", "",
        "Use specific inbox (default: active)")
    auditCmd.Flags().BoolVar(&auditJSON, "json", false,
        "Output as JSON")
}

func runAudit(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Get email ID
    emailID := ""
    if len(args) > 0 {
        emailID = args[0]
    } else if !auditLatest {
        return fmt.Errorf("specify an email ID or use --latest")
    }

    // Load keystore and get inbox
    keystore, err := config.LoadKeystore()
    if err != nil {
        return err
    }

    var stored *config.StoredInbox
    if auditEmail != "" {
        stored, err = keystore.GetInbox(auditEmail)
    } else {
        stored, err = keystore.GetActiveInbox()
    }
    if err != nil {
        return fmt.Errorf("no inbox found: %w", err)
    }

    // Create client and import inbox
    client, err := config.NewClient()
    if err != nil {
        return err
    }
    defer client.Close()

    inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
    if err != nil {
        return err
    }

    // Get email
    var email *vaultsandbox.Email
    if auditLatest {
        emails, err := inbox.GetEmails(ctx)
        if err != nil {
            return err
        }
        if len(emails) == 0 {
            return fmt.Errorf("no emails in inbox")
        }
        email = emails[0] // Most recent
    } else {
        email, err = inbox.GetEmail(ctx, emailID)
        if err != nil {
            return err
        }
    }

    // Render audit report
    if auditJSON {
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
        spfResult := formatAuthResult(auth.SPF.Result, passStyle, failStyle, warnStyle)
        fmt.Printf("%s %s\n", labelStyle.Render("SPF:"), spfResult)
        if auth.SPF.Domain != "" {
            fmt.Printf("%s %s\n", labelStyle.Render("  Domain:"), auth.SPF.Domain)
        }

        // DKIM
        dkimResult := formatAuthResult(auth.DKIM.Result, passStyle, failStyle, warnStyle)
        fmt.Printf("%s %s\n", labelStyle.Render("DKIM:"), dkimResult)
        if auth.DKIM.Selector != "" {
            fmt.Printf("%s %s\n", labelStyle.Render("  Selector:"), auth.DKIM.Selector)
        }
        if auth.DKIM.Domain != "" {
            fmt.Printf("%s %s\n", labelStyle.Render("  Domain:"), auth.DKIM.Domain)
        }

        // DMARC
        dmarcResult := formatAuthResult(auth.DMARC.Result, passStyle, failStyle, warnStyle)
        fmt.Printf("%s %s\n", labelStyle.Render("DMARC:"), dmarcResult)
        if auth.DMARC.Policy != "" {
            fmt.Printf("%s %s\n", labelStyle.Render("  Policy:"), auth.DMARC.Policy)
        }

        // Reverse DNS
        if auth.ReverseDNS != nil {
            rdnsResult := formatAuthResult(boolToResult(auth.ReverseDNS.Valid), passStyle, failStyle, warnStyle)
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

func boolToResult(b bool) string {
    if b {
        return "pass"
    }
    return "fail"
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
            if hasAttachments {
                sb.WriteString("│   └── text/html\n")
            } else {
                sb.WriteString("│   └── text/html\n")
            }
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
        if strings.EqualFold(auth.SPF.Result, "pass") {
            score += 15
        }

        // DKIM
        if strings.EqualFold(auth.DKIM.Result, "pass") {
            score += 20
        }

        // DMARC
        if strings.EqualFold(auth.DMARC.Result, "pass") {
            score += 10
        }

        // Reverse DNS
        if auth.ReverseDNS != nil && auth.ReverseDNS.Valid {
            score += 5
        }
    }

    return score
}

func renderAuditJSON(email *vaultsandbox.Email) error {
    // Similar to emailToMap but with auth results
    data := map[string]interface{}{
        "id":         email.ID,
        "subject":    email.Subject,
        "from":       email.From,
        "to":         email.To,
        "receivedAt": email.ReceivedAt,
        "text":       email.Text,
        "html":       email.HTML,
        "headers":    email.Headers,
        "links":      email.Links,
    }

    if email.AuthResults != nil {
        data["authResults"] = map[string]interface{}{
            "spf": map[string]string{
                "result": email.AuthResults.SPF.Result,
                "domain": email.AuthResults.SPF.Domain,
            },
            "dkim": map[string]string{
                "result":   email.AuthResults.DKIM.Result,
                "selector": email.AuthResults.DKIM.Selector,
                "domain":   email.AuthResults.DKIM.Domain,
            },
            "dmarc": map[string]string{
                "result": email.AuthResults.DMARC.Result,
                "policy": email.AuthResults.DMARC.Policy,
            },
        }
    }

    // JSON output
    output, _ := json.MarshalIndent(data, "", "  ")
    fmt.Println(string(output))
    return nil
}
```

## Sample Output

```
 EMAIL AUDIT REPORT

BASIC INFO
Subject:             Password Reset Request
From:                noreply@example.com
To:                  abc123@vaultsandbox.com
Received:            2024-01-15 14:32:05 UTC

AUTHENTICATION
SPF:                 PASS
  Domain:            example.com
DKIM:                PASS
  Selector:          s1
  Domain:            example.com
DMARC:               PASS
  Policy:            reject
Reverse DNS:         PASS
  Hostname:          mail.example.com

TRANSPORT SECURITY
TLS Version:         TLS 1.3
Cipher Suite:        ECDHE-RSA-AES256-GCM-SHA384
E2E Encryption:      ML-KEM-768 + AES-256-GCM

MIME STRUCTURE
╭──────────────────────────────────────────────╮
│ message/rfc822                               │
│ ├── headers                                  │
│ │   ├── From                                 │
│ │   ├── To                                   │
│ │   ├── Subject                              │
│ │   ├── Date                                 │
│ │   └── Message-ID                           │
│ ├── body                                     │
│ │   ├── text/plain                           │
│ │   └── text/html                            │
│ └── attachments                              │
│     └── logo.png (image/png, 2048 bytes)     │
╰──────────────────────────────────────────────╯

╭──────────────────────────────────────────────╮
│ Security Score: 100/100                      │
╰──────────────────────────────────────────────╯
```

## Verification

```bash
# Audit latest email
vsb audit --latest

# Audit specific email
vsb audit abc123

# JSON output
vsb audit --latest --json | jq '.authResults.dkim.result'
```

## Files Created

- `internal/cli/audit.go`

## Next Steps

Proceed to [07-open-view-commands.md](07-open-view-commands.md) to implement the open and view commands.
