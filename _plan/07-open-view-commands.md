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

### 1. Shared Browser Utility (Already Exists)

The browser utility already exists at `internal/browser/browser.go` with:

- `OpenURL(url string) error` - Opens URL in default browser (with scheme validation)
- `ViewHTML(html string) error` - Writes HTML to temp file and opens in browser
- `CleanupPreviews(olderThan time.Duration) error` - Removes old preview files

**No changes needed** - the TUI already uses this package.

### 2. Open Command (CLI)

**File: `internal/cli/open.go`**

```go
package cli

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/spf13/cobra"
    "github.com/vaultsandbox/vsb-cli/internal/browser"
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
    openList  bool
    openNth   int
    openEmail string
    openJSON  bool
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

    // Get email ID (empty = latest)
    emailID := ""
    if len(args) > 0 {
        emailID = args[0]
    }

    // Use shared helper
    email, _, cleanup, err := GetEmailByIDOrLatest(ctx, emailID, openEmail)
    if err != nil {
        return err
    }
    defer cleanup()

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
            data, _ := json.MarshalIndent(email.Links, "", "  ")
            fmt.Println(string(data))
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
        data, _ := json.Marshal(map[string]string{"url": link})
        fmt.Println(string(data))
    } else {
        fmt.Printf("Opening: %s\n", link)
    }

    return browser.OpenURL(link)
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
    "time"

    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/browser"
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

    // Get email ID (empty = latest)
    emailID := ""
    if len(args) > 0 {
        emailID = args[0]
    }

    // Use shared helper (returns email, inbox, cleanup, error)
    email, inbox, cleanup, err := GetEmailByIDOrLatest(ctx, emailID, viewEmail)
    if err != nil {
        return err
    }
    defer cleanup()

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

    // Cleanup old previews (older than 1 hour)
    browser.CleanupPreviews(time.Hour)

    return browser.ViewHTML(wrappedHTML)
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

### 4. Watch TUI Browser Integration (Already Done)

The watch TUI model (`internal/tui/watch/model.go`) already uses the shared browser package directly:

```go
// In openLinks() method:
browser.OpenURL(email.Links[0])

// In viewHTML() method:
browser.ViewHTML(email.HTML)
```

**No separate browser.go file needed** - the model uses `internal/browser` directly.

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

- `internal/cli/open.go` (NEW - CLI command, uses existing helpers)
- `internal/cli/view.go` (NEW - CLI command, uses existing helpers)
- `internal/tui/watch/model.go` (UPDATE - add number key handling for links)

**Existing files used (no changes needed):**
- `internal/browser/browser.go` - Already has OpenURL, ViewHTML, CleanupPreviews
- `internal/cli/helpers.go` - Already has GetEmailByIDOrLatest

## Next Steps

Proceed to [08-export-import-commands.md](08-export-import-commands.md) to implement export/import functionality.
