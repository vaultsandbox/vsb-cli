# Phase 4.1: Open and View Commands

## Objective
The `watch` TUI already has `o` (open links) and `v` (view HTML) keybindings implemented in `internal/tui/watch/browser.go`. This phase adds:

1. **CLI commands** for scripting/CI/CD use cases
2. **Shared browser utility** to consolidate browser opening logic
3. **Enhanced link handling** in the watch TUI (number keys to open specific links)

## Current State

The watch TUI already supports:
- `o` - Open first link in browser
- `v` - View HTML in browser

These work but use a simple inline implementation. We'll extract to a shared utility.

## Commands (CLI - for scripting)

| Command | Description |
|---------|-------------|
| `vsb open` | Extract first link from latest email and open in browser |
| `vsb open <id>` | Extract first link from specific email |
| `vsb open --list` | List all links without opening (returns JSON-friendly output) |
| `vsb open --nth 2` | Open the Nth link |
| `vsb view` | Open latest email HTML in browser |
| `vsb view <id>` | Open specific email HTML in browser |
| `vsb view --text` | Print plain text to terminal |
| `vsb view --raw` | Print raw email source (RFC 5322) |

## Watch TUI Enhancements

Enhance the existing keybindings:

```
Keybindings (in detail view):
  o       Open first link in browser (existing)
  1-9     Open specific link by number (NEW)
  l       Show links list view (NEW - from 06-audit)
  v       View HTML in browser (existing)
  r       View raw email source (NEW)
```

## Tasks

### 1. Shared Browser Utility

Extract browser logic to a shared package for use by both CLI and TUI.

**File: `internal/browser/browser.go`**

```go
package browser

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "time"
)

// Open opens a URL in the default browser
func Open(url string) error {
    var cmd *exec.Cmd

    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("open", url)
    case "linux":
        cmd = exec.Command("xdg-open", url)
    case "windows":
        cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
    default:
        return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }

    return cmd.Start()
}

// OpenHTML writes HTML content to a temp file and opens it in the browser
func OpenHTML(html string) error {
    // Create temp directory for vsb previews
    tmpDir := filepath.Join(os.TempDir(), "vsb-previews")
    if err := os.MkdirAll(tmpDir, 0755); err != nil {
        return err
    }

    // Create unique temp file
    filename := fmt.Sprintf("preview-%d.html", time.Now().UnixNano())
    tmpFile := filepath.Join(tmpDir, filename)

    // Write HTML
    if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
        return err
    }

    // Open in browser
    return Open("file://" + tmpFile)
}

// CleanupPreviews removes old preview files (older than 24 hours)
func CleanupPreviews() error {
    tmpDir := filepath.Join(os.TempDir(), "vsb-previews")

    entries, err := os.ReadDir(tmpDir)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }

    cutoff := time.Now().Add(-24 * time.Hour)

    for _, entry := range entries {
        info, err := entry.Info()
        if err != nil {
            continue
        }
        if info.ModTime().Before(cutoff) {
            os.Remove(filepath.Join(tmpDir, entry.Name()))
        }
    }

    return nil
}
```

### 2. Open Command (CLI)

**File: `internal/cli/open.go`**

```go
package cli

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/browser"
    "github.com/vaultsandbox/vsb-cli/internal/config"
)

var openCmd = &cobra.Command{
    Use:   "open [email-id]",
    Short: "Extract and open the first link from an email",
    Long: `Extract the first HTTP/HTTPS link from an email and open it in your browser.

This is useful for quickly following verification links, password reset links,
or any other actionable URLs in emails.

For interactive use, prefer 'vsb watch' and press 'o' to open links.

Examples:
  vsb open              # Open first link from latest email
  vsb open abc123       # Open first link from specific email
  vsb open --list       # List all links (for scripting)
  vsb open --nth 2      # Open the second link
  vsb open --json       # JSON output for CI/CD`,
    Args: cobra.MaximumNArgs(1),
    RunE: runOpen,
}

var (
    openList   bool
    openNth    int
    openEmail  string
    openJSON   bool
)

func init() {
    rootCmd.AddCommand(openCmd)

    openCmd.Flags().BoolVar(&openList, "list", false,
        "List all links without opening")
    openCmd.Flags().IntVar(&openNth, "nth", 1,
        "Open the Nth link (1-indexed)")
    openCmd.Flags().StringVar(&openEmail, "email", "",
        "Use specific inbox (default: active)")
    openCmd.Flags().BoolVar(&openJSON, "json", false,
        "Output as JSON")
}

func runOpen(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Get email
    email, err := getEmailFromArgs(ctx, args, openEmail)
    if err != nil {
        return err
    }

    // Check for links
    if len(email.Links) == 0 {
        if openJSON {
            fmt.Println("[]")
        } else {
            fmt.Println("No links found in email")
        }
        return nil
    }

    // List mode
    if openList {
        if openJSON {
            output, _ := json.MarshalIndent(email.Links, "", "  ")
            fmt.Println(string(output))
        } else {
            for i, link := range email.Links {
                fmt.Printf("%d. %s\n", i+1, link)
            }
        }
        return nil
    }

    // Get the requested link
    if openNth < 1 || openNth > len(email.Links) {
        return fmt.Errorf("link index %d out of range (1-%d)", openNth, len(email.Links))
    }
    link := email.Links[openNth-1]

    if openJSON {
        output, _ := json.Marshal(map[string]string{"url": link})
        fmt.Println(string(output))
    } else {
        fmt.Printf("Opening: %s\n", link)
    }

    return browser.Open(link)
}

// getEmailFromArgs is a helper to get an email from args or latest
func getEmailFromArgs(ctx context.Context, args []string, inboxEmail string) (*vaultsandbox.Email, error) {
    emailID := ""
    useLatest := true
    if len(args) > 0 {
        emailID = args[0]
        useLatest = false
    }

    keystore, err := config.LoadKeystore()
    if err != nil {
        return nil, err
    }

    var stored *config.StoredInbox
    if inboxEmail != "" {
        stored, err = keystore.GetInbox(inboxEmail)
    } else {
        stored, err = keystore.GetActiveInbox()
    }
    if err != nil {
        return nil, fmt.Errorf("no inbox found: %w", err)
    }

    client, err := config.NewClient()
    if err != nil {
        return nil, err
    }
    defer client.Close()

    inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
    if err != nil {
        return nil, err
    }

    if useLatest {
        emails, err := inbox.GetEmails(ctx)
        if err != nil {
            return nil, err
        }
        if len(emails) == 0 {
            return nil, fmt.Errorf("no emails in inbox")
        }
        return emails[0], nil
    }

    return inbox.GetEmail(ctx, emailID)
}
```

### 3. View Command (CLI)

**File: `internal/cli/view.go`**

```go
package cli

import (
    "context"
    "fmt"
    "html"
    "strings"

    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/browser"
    "github.com/vaultsandbox/vsb-cli/internal/config"
)

var viewCmd = &cobra.Command{
    Use:   "view [email-id]",
    Short: "Preview email content",
    Long: `View email content in various formats.

For interactive use, prefer 'vsb watch' and press 'v' to view HTML.

Examples:
  vsb view              # View latest email HTML in browser
  vsb view abc123       # View specific email
  vsb view --text       # Print plain text to terminal
  vsb view --raw        # Print raw email source (RFC 5322)`,
    Args: cobra.MaximumNArgs(1),
    RunE: runView,
}

var (
    viewText  bool
    viewRaw   bool
    viewEmail string
)

func init() {
    rootCmd.AddCommand(viewCmd)

    viewCmd.Flags().BoolVar(&viewText, "text", false,
        "Show plain text version in terminal")
    viewCmd.Flags().BoolVar(&viewRaw, "raw", false,
        "Show raw email source (RFC 5322)")
    viewCmd.Flags().StringVar(&viewEmail, "email", "",
        "Use specific inbox (default: active)")
}

func runView(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Get email
    email, inbox, err := getEmailAndInbox(ctx, args, viewEmail)
    if err != nil {
        return err
    }

    // Raw mode - show RFC 5322 source
    if viewRaw {
        raw, err := inbox.GetRawEmail(ctx, email.ID)
        if err != nil {
            return err
        }
        fmt.Println(raw)
        return nil
    }

    // Text mode - print to terminal
    if viewText {
        if email.Text == "" {
            fmt.Println("No plain text version available")
            return nil
        }
        fmt.Printf("Subject: %s\n", email.Subject)
        fmt.Printf("From: %s\n", email.From)
        fmt.Printf("Date: %s\n\n", email.ReceivedAt.Format("2006-01-02 15:04:05"))
        fmt.Println(email.Text)
        return nil
    }

    // HTML mode - open in browser
    if email.HTML == "" {
        fmt.Println("No HTML version, showing text:")
        fmt.Println(email.Text)
        return nil
    }

    // Wrap HTML with proper document structure
    wrappedHTML := wrapEmailHTML(email)

    fmt.Println("Opening email in browser...")

    // Cleanup old previews
    browser.CleanupPreviews()

    return browser.OpenHTML(wrappedHTML)
}

func getEmailAndInbox(ctx context.Context, args []string, inboxEmail string) (*vaultsandbox.Email, *vaultsandbox.Inbox, error) {
    emailID := ""
    useLatest := true
    if len(args) > 0 {
        emailID = args[0]
        useLatest = false
    }

    keystore, err := config.LoadKeystore()
    if err != nil {
        return nil, nil, err
    }

    var stored *config.StoredInbox
    if inboxEmail != "" {
        stored, err = keystore.GetInbox(inboxEmail)
    } else {
        stored, err = keystore.GetActiveInbox()
    }
    if err != nil {
        return nil, nil, fmt.Errorf("no inbox found: %w", err)
    }

    client, err := config.NewClient()
    if err != nil {
        return nil, nil, err
    }
    defer client.Close()

    inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
    if err != nil {
        return nil, nil, err
    }

    var email *vaultsandbox.Email
    if useLatest {
        emails, err := inbox.GetEmails(ctx)
        if err != nil {
            return nil, nil, err
        }
        if len(emails) == 0 {
            return nil, nil, fmt.Errorf("no emails in inbox")
        }
        email = emails[0]
    } else {
        email, err = inbox.GetEmail(ctx, emailID)
        if err != nil {
            return nil, nil, err
        }
    }

    return email, inbox, nil
}

func wrapEmailHTML(email *vaultsandbox.Email) string {
    return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .header {
            background: #7C3AED;
            color: white;
            padding: 20px;
            border-radius: 8px 8px 0 0;
        }
        .header h1 {
            margin: 0 0 10px 0;
            font-size: 1.2em;
        }
        .header .meta {
            font-size: 0.9em;
            opacity: 0.9;
        }
        .content {
            background: white;
            padding: 20px;
            border-radius: 0 0 8px 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .vsb-badge {
            background: #10B981;
            color: white;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 0.8em;
            margin-left: 10px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>%s <span class="vsb-badge">VaultSandbox</span></h1>
        <div class="meta">
            <strong>From:</strong> %s<br>
            <strong>Date:</strong> %s
        </div>
    </div>
    <div class="content">
        %s
    </div>
</body>
</html>`,
        html.EscapeString(email.Subject),
        html.EscapeString(email.Subject),
        html.EscapeString(email.From),
        email.ReceivedAt.Format("January 2, 2006 at 3:04 PM"),
        email.HTML,
    )
}
```

### 4. Update Watch TUI to Use Shared Browser

**Update: `internal/tui/watch/browser.go`**

Replace the inline implementation with calls to the shared browser package:

```go
package watch

import (
    "github.com/vaultsandbox/vsb-cli/internal/browser"
)

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
    return browser.Open(url)
}

// viewInBrowser opens HTML in the browser with wrapper
func viewInBrowser(html string) error {
    return browser.OpenHTML(html)
}
```

### 5. Add Number Keys for Link Selection (Watch TUI)

**Update: `internal/tui/watch/model.go`**

In the Update() function, add handling for number keys 1-9 to open specific links:

```go
// In the detail view key handling section:
case msg.Type == tea.KeyRunes:
    if len(msg.Runes) == 1 && m.viewing && m.viewedEmail != nil {
        n := int(msg.Runes[0] - '1') // '1' -> 0, '2' -> 1, etc.
        if n >= 0 && n < len(m.viewedEmail.Email.Links) {
            return m, func() tea.Msg {
                browser.Open(m.viewedEmail.Email.Links[n])
                return nil
            }
        }
    }
```

## Usage Examples

### CLI (Scripting/CI/CD)

```bash
# Get verification link in CI/CD
LINK=$(vsb open --list --json | jq -r '.[0]')
curl "$LINK"

# Or use built-in nth
vsb open --nth 1

# View email text in terminal
vsb view --text

# Pipe raw email for debugging
vsb view --raw | grep "X-Spam-Score"
```

### Watch TUI (Interactive)

```bash
vsb watch
# Navigate to email, press Enter to view
# Press 'l' to see links list
# Press '1' to open first link, '2' for second, etc.
# Press 'v' to view HTML in browser
# Press 'a' to see security audit
```

## Verification

```bash
# CLI commands
vsb open --list              # List all links
vsb open                     # Open first link
vsb view --text              # View in terminal
vsb view                     # View in browser

# Watch TUI
vsb watch
# Press Enter on email, then:
#   'o' - open first link
#   '2' - open second link
#   'l' - list links
#   'v' - view HTML
```

## Files Created/Modified

- `internal/browser/browser.go` (NEW - shared browser utility)
- `internal/cli/open.go` (NEW - CLI command)
- `internal/cli/view.go` (NEW - CLI command)
- `internal/tui/watch/browser.go` (UPDATE - use shared browser)
- `internal/tui/watch/model.go` (UPDATE - add number key handling)

## Next Steps

Proceed to [08-export-import-commands.md](08-export-import-commands.md) to implement export/import functionality.
