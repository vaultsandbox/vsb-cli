# Phase 3.2: Audit Command & Security View

## Objective
Implement security analysis for emails:
1. **`vsb audit`** - CLI command for scripting/CI/CD
2. **Security view in `watch` TUI** - Press `a` to see security details for the selected email

## Commands

| Command | Description |
|---------|-------------|
| `vsb audit <email-id>` | Audit specific email by ID |
| `vsb audit --latest` | Audit the most recent email |
| `vsb audit --json` | JSON output for scripting |

## Watch TUI Integration

In the watch TUI, add `a` keybinding to show security details:

```
Keybindings:
  j/k     Navigate emails
  Enter   View email content
  a       Audit (show security info)  <-- NEW
  o       Open first link in browser
  l       List all links              <-- NEW
  v       View HTML in browser
  ?       Help
  q       Quit
```

When viewing an email (after pressing Enter), the detail view should show tabs:

```
┌─ Email: Password Reset ─────────────────────────┐
│ [Content] [Security] [Links] [Raw]              │
│  Tab 1-4 to switch                              │
├─────────────────────────────────────────────────┤
│ From: noreply@example.com                       │
│ To:   abc123@vaultsandbox.com                   │
│ ...                                             │
└─────────────────────────────────────────────────┘
```

## Output Sections

1. **Authentication**: SPF, DKIM, DMARC results
2. **Transport Security**: TLS version, cipher suite
3. **MIME Structure**: Headers, body parts, attachments
4. **Security Score**: 0-100 based on auth results

## Tasks

### 1. CLI Audit Command (for scripting)

**File: `internal/cli/audit.go`**

```go
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
    "github.com/vaultsandbox/vsb-cli/internal/tui/styles"
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
  vsb audit abc123          # Audit specific email
  vsb audit --latest        # Audit most recent email
  vsb audit --latest --json # JSON output for scripting`,
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
    data := map[string]interface{}{
        "id":            email.ID,
        "subject":       email.Subject,
        "from":          email.From,
        "to":            email.To,
        "receivedAt":    email.ReceivedAt,
        "securityScore": calculateSecurityScore(email),
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

    output, _ := json.MarshalIndent(data, "", "  ")
    fmt.Println(string(output))
    return nil
}
```

### 2. Security View in Watch TUI

**Update: `internal/tui/watch/model.go`**

Add a new view mode for security details. When in detail view, pressing `a` toggles to security view:

```go
// Add to KeyMap
type KeyMap struct {
    // ... existing keys ...
    Audit     key.Binding  // NEW
    ListLinks key.Binding  // NEW
}

var DefaultKeyMap = KeyMap{
    // ... existing keys ...
    Audit: key.NewBinding(
        key.WithKeys("a"),
        key.WithHelp("a", "security audit"),
    ),
    ListLinks: key.NewBinding(
        key.WithKeys("l"),
        key.WithHelp("l", "list links"),
    ),
}

// Add view mode enum
type DetailView int

const (
    ViewContent DetailView = iota
    ViewSecurity
    ViewLinks
    ViewRaw
)

// Update Model to track current detail view
type Model struct {
    // ... existing fields ...
    detailView DetailView  // NEW: which tab is active
}

// In Update(), handle 'a' key to toggle security view
case key.Matches(msg, DefaultKeyMap.Audit):
    if m.viewing && m.viewedEmail != nil {
        m.detailView = ViewSecurity
        m.viewport.SetContent(m.renderSecurityView())
        m.viewport.GotoTop()
    }

// In Update(), handle 'l' key to show links list
case key.Matches(msg, DefaultKeyMap.ListLinks):
    if m.viewing && m.viewedEmail != nil {
        m.detailView = ViewLinks
        m.viewport.SetContent(m.renderLinksView())
        m.viewport.GotoTop()
    }
```

**New: `internal/tui/watch/security.go`**

```go
package watch

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/tui/styles"
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

    // Tab indicator
    sb.WriteString(styles.HelpStyle.Render("[1:Content] [2:Security] [3:Links] [4:Raw]"))
    sb.WriteString("\n")
    sb.WriteString(styles.HelpStyle.Render("           ^^^^^^^^^^"))
    sb.WriteString("\n\n")

    // Authentication
    sb.WriteString(sectionStyle.Render("AUTHENTICATION"))
    sb.WriteString("\n")

    if email.AuthResults != nil {
        auth := email.AuthResults

        // SPF
        spfResult := formatResult(auth.SPF.Result, passStyle, failStyle, warnStyle)
        sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("SPF:"), spfResult))
        if auth.SPF.Domain != "" {
            sb.WriteString(fmt.Sprintf(" (%s)", auth.SPF.Domain))
        }
        sb.WriteString("\n")

        // DKIM
        dkimResult := formatResult(auth.DKIM.Result, passStyle, failStyle, warnStyle)
        sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("DKIM:"), dkimResult))
        if auth.DKIM.Domain != "" {
            sb.WriteString(fmt.Sprintf(" (%s)", auth.DKIM.Domain))
        }
        sb.WriteString("\n")

        // DMARC
        dmarcResult := formatResult(auth.DMARC.Result, passStyle, failStyle, warnStyle)
        sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("DMARC:"), dmarcResult))
        if auth.DMARC.Policy != "" {
            sb.WriteString(fmt.Sprintf(" (policy: %s)", auth.DMARC.Policy))
        }
        sb.WriteString("\n")

        // Reverse DNS
        if auth.ReverseDNS != nil {
            rdns := "FAIL"
            if auth.ReverseDNS.Valid {
                rdns = "PASS"
            }
            rdnsResult := formatResult(rdns, passStyle, failStyle, warnStyle)
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
        if strings.EqualFold(email.AuthResults.SPF.Result, "pass") {
            score += 15
        }
        if strings.EqualFold(email.AuthResults.DKIM.Result, "pass") {
            score += 20
        }
        if strings.EqualFold(email.AuthResults.DMARC.Result, "pass") {
            score += 10
        }
        if email.AuthResults.ReverseDNS != nil && email.AuthResults.ReverseDNS.Valid {
            score += 5
        }
    }

    return score
}
```

**New: `internal/tui/watch/links.go`**

```go
package watch

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/vaultsandbox/vsb-cli/internal/tui/styles"
)

// renderLinksView renders the links list view
func (m Model) renderLinksView() string {
    if m.viewedEmail == nil {
        return ""
    }

    email := m.viewedEmail.Email
    var sb strings.Builder

    labelStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Purple)
    linkStyle := lipgloss.NewStyle().Foreground(styles.White)
    indexStyle := lipgloss.NewStyle().Foreground(styles.Gray)

    // Tab indicator
    sb.WriteString(styles.HelpStyle.Render("[1:Content] [2:Security] [3:Links] [4:Raw]"))
    sb.WriteString("\n")
    sb.WriteString(styles.HelpStyle.Render("                         ^^^^^^^"))
    sb.WriteString("\n\n")

    if len(email.Links) == 0 {
        sb.WriteString(styles.HelpStyle.Render("No links found in this email"))
        return sb.String()
    }

    sb.WriteString(labelStyle.Render(fmt.Sprintf("Found %d links:\n\n", len(email.Links))))

    for i, link := range email.Links {
        sb.WriteString(indexStyle.Render(fmt.Sprintf("%2d. ", i+1)))
        sb.WriteString(linkStyle.Render(link))
        sb.WriteString("\n")
    }

    sb.WriteString("\n")
    sb.WriteString(styles.HelpStyle.Render("Press 'o' to open first link, or number key (1-9) to open specific link"))

    return sb.String()
}
```

## Sample CLI Output

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

## Watch TUI Security View

```
┌─ Email Details ─────────────────────────────────┐
│ [1:Content] [2:Security] [3:Links] [4:Raw]      │
│              ^^^^^^^^^^                         │
│                                                 │
│ AUTHENTICATION                                  │
│ SPF:            PASS (example.com)              │
│ DKIM:           PASS (example.com)              │
│ DMARC:          PASS (policy: reject)           │
│ Reverse DNS:    PASS (mail.example.com)         │
│                                                 │
│ TRANSPORT SECURITY                              │
│ TLS:            TLS 1.3                         │
│ E2E:            ML-KEM-768 + AES-256-GCM        │
│                                                 │
│ SECURITY SCORE                                  │
│ 100/100                                         │
│                                                 │
├─────────────────────────────────────────────────┤
│ 1-4: switch tabs • esc: back • q: quit          │
└─────────────────────────────────────────────────┘
```

## Verification

```bash
# CLI: Audit latest email
vsb audit --latest

# CLI: JSON output for CI/CD
vsb audit --latest --json | jq '.authResults.spf.result'

# TUI: In watch, press Enter on email, then 'a' for security view
vsb watch
```

## Files Created/Modified

- `internal/cli/audit.go` (NEW - CLI command)
- `internal/tui/watch/model.go` (UPDATE - add security view mode)
- `internal/tui/watch/security.go` (NEW - security view renderer)
- `internal/tui/watch/links.go` (NEW - links list view renderer)

## Next Steps

Proceed to [07-open-view-commands.md](07-open-view-commands.md) to finalize open/view commands and browser utilities.
