# Phase 4.1: Open and View Commands

## Objective
Implement `vsb open` (extract and open first link) and `vsb view` (preview HTML in browser).

## Commands

| Command | Description |
|---------|-------------|
| `vsb open` | Extract first link from latest email and open in browser |
| `vsb open <id>` | Extract first link from specific email |
| `vsb open --list` | List all links without opening |
| `vsb view` | Open latest email HTML in browser |
| `vsb view <id>` | Open specific email HTML in browser |

## Tasks

### 1. Browser Utility (Shared)

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

// CleanupPreviews removes old preview files
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

### 2. Open Command

**File: `internal/cli/open.go`**

```go
package cli

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/browser"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/output"
)

var openCmd = &cobra.Command{
    Use:   "open [email-id]",
    Short: "Extract and open the first link from an email",
    Long: `Extract the first HTTP/HTTPS link from an email and open it in your browser.

This is useful for quickly following verification links, password reset links,
or any other actionable URLs in emails.

Examples:
  vsb open              # Open first link from latest email
  vsb open abc123       # Open first link from specific email
  vsb open --list       # List all links without opening
  vsb open --nth 2      # Open the second link`,
    Args: cobra.MaximumNArgs(1),
    RunE: runOpen,
}

var (
    openList   bool
    openNth    int
    openEmail  string
    openLatest bool
)

func init() {
    rootCmd.AddCommand(openCmd)

    openCmd.Flags().BoolVar(&openList, "list", false,
        "List all links without opening")
    openCmd.Flags().IntVar(&openNth, "nth", 1,
        "Open the Nth link (1-indexed)")
    openCmd.Flags().StringVar(&openEmail, "email", "",
        "Use specific inbox (default: active)")
    openCmd.Flags().BoolVar(&openLatest, "latest", true,
        "Use latest email (default: true)")
}

func runOpen(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Get email ID or use latest
    emailID := ""
    if len(args) > 0 {
        emailID = args[0]
        openLatest = false
    }

    // Get inbox
    keystore, err := config.LoadKeystore()
    if err != nil {
        return err
    }

    var stored *config.StoredInbox
    if openEmail != "" {
        stored, err = keystore.GetInbox(openEmail)
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
    if openLatest {
        emails, err := inbox.GetEmails(ctx)
        if err != nil {
            return err
        }
        if len(emails) == 0 {
            return fmt.Errorf("no emails in inbox")
        }
        email = emails[0]
    } else {
        email, err = inbox.GetEmail(ctx, emailID)
        if err != nil {
            return err
        }
    }

    // Check for links
    if len(email.Links) == 0 {
        fmt.Println(output.Info("No links found in email"))
        return nil
    }

    // List mode
    if openList {
        fmt.Printf("Found %d links in email:\n\n", len(email.Links))
        for i, link := range email.Links {
            fmt.Printf("  %d. %s\n", i+1, link)
        }
        return nil
    }

    // Get the requested link
    if openNth < 1 || openNth > len(email.Links) {
        return fmt.Errorf("link index %d out of range (1-%d)", openNth, len(email.Links))
    }
    link := email.Links[openNth-1]

    // Open in browser
    fmt.Println(output.Info(fmt.Sprintf("Opening: %s", link)))
    return browser.Open(link)
}
```

### 3. View Command

**File: `internal/cli/view.go`**

```go
package cli

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/browser"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/output"
)

var viewCmd = &cobra.Command{
    Use:   "view [email-id]",
    Short: "Preview email HTML in browser",
    Long: `Open the HTML content of an email in your default web browser.

The HTML is saved to a temporary file and opened. This allows you to
see the full rendered email as it would appear in an email client.

Note: External images may not load due to tracking pixel protections.

Examples:
  vsb view              # View latest email
  vsb view abc123       # View specific email
  vsb view --text       # View plain text version`,
    Args: cobra.MaximumNArgs(1),
    RunE: runView,
}

var (
    viewText    bool
    viewEmail   string
    viewLatest  bool
    viewRaw     bool
)

func init() {
    rootCmd.AddCommand(viewCmd)

    viewCmd.Flags().BoolVar(&viewText, "text", false,
        "Show plain text version instead of HTML")
    viewCmd.Flags().StringVar(&viewEmail, "email", "",
        "Use specific inbox (default: active)")
    viewCmd.Flags().BoolVar(&viewLatest, "latest", true,
        "Use latest email (default: true)")
    viewCmd.Flags().BoolVar(&viewRaw, "raw", false,
        "Show raw email source")
}

func runView(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Get email ID or use latest
    emailID := ""
    if len(args) > 0 {
        emailID = args[0]
        viewLatest = false
    }

    // Get inbox
    keystore, err := config.LoadKeystore()
    if err != nil {
        return err
    }

    var stored *config.StoredInbox
    if viewEmail != "" {
        stored, err = keystore.GetInbox(viewEmail)
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
    if viewLatest {
        emails, err := inbox.GetEmails(ctx)
        if err != nil {
            return err
        }
        if len(emails) == 0 {
            return fmt.Errorf("no emails in inbox")
        }
        email = emails[0]
    } else {
        email, err = inbox.GetEmail(ctx, emailID)
        if err != nil {
            return err
        }
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
            fmt.Println(output.Info("No plain text version available"))
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
        // No HTML, show text instead
        fmt.Println(output.Info("No HTML version, showing text:"))
        fmt.Println(email.Text)
        return nil
    }

    // Wrap HTML with proper document structure
    html := wrapHTML(email)

    // Open in browser
    fmt.Println(output.Info("Opening email in browser..."))

    // Cleanup old previews
    browser.CleanupPreviews()

    return browser.OpenHTML(html)
}

func wrapHTML(email *vaultsandbox.Email) string {
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
            margin-bottom: 0;
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
        escapeHTML(email.Subject),
        escapeHTML(email.Subject),
        escapeHTML(email.From),
        email.ReceivedAt.Format("January 2, 2006 at 3:04 PM"),
        email.HTML,
    )
}

func escapeHTML(s string) string {
    replacer := strings.NewReplacer(
        "&", "&amp;",
        "<", "&lt;",
        ">", "&gt;",
        `"`, "&quot;",
        "'", "&#39;",
    )
    return replacer.Replace(s)
}
```

## Usage Examples

### Quick Link Opening
```bash
# Open verification link immediately
vsb open

# Open second link in email
vsb open --nth 2

# See all links first
vsb open --list
```

### Email Preview
```bash
# View HTML in browser
vsb view

# View plain text in terminal
vsb view --text

# View raw email source
vsb view --raw
```

### CI/CD Integration
```bash
# Get verification link
LINK=$(vsb wait-for --subject "Verify" --json | jq -r '.links[0]')
curl "$LINK"

# Or use the built-in extraction
vsb wait-for --subject "Verify" --extract-link
```

## Verification

```bash
# Create inbox and trigger an email
vsb inbox create

# Once email arrives...
vsb open --list          # List links
vsb open                 # Open first link
vsb view                 # View in browser
vsb view --text          # View in terminal
```

## Files Created

- `internal/browser/browser.go`
- `internal/cli/open.go`
- `internal/cli/view.go`

## Next Steps

Proceed to [08-export-import-commands.md](08-export-import-commands.md) to implement export/import functionality.
