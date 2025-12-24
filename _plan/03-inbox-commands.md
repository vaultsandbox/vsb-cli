# Phase 2.1: Inbox Commands

## Objective
Implement `vsb inbox create`, `vsb inbox list`, and `vsb inbox use` commands.

## Commands

| Command | Description |
|---------|-------------|
| `vsb inbox create [label]` | Create a new inbox with quantum-safe keys |
| `vsb inbox list` | List all stored inboxes |
| `vsb inbox use <email>` | Switch active inbox |
| `vsb inbox delete <email>` | Delete an inbox |

## Tasks

### 1. Inbox Parent Command

**File: `internal/cli/inbox.go`**

```go
package cli

import (
    "github.com/spf13/cobra"
)

var inboxCmd = &cobra.Command{
    Use:   "inbox",
    Short: "Manage temporary email inboxes",
    Long:  `Create, list, switch, and delete temporary email inboxes.`,
}

func init() {
    rootCmd.AddCommand(inboxCmd)
}
```

### 2. Inbox Create Command

**File: `internal/cli/inbox_create.go`**

```go
package cli

import (
    "context"
    "fmt"
    "time"

    "github.com/charmbracelet/lipgloss"
    "github.com/spf13/cobra"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/output"
)

var inboxCreateCmd = &cobra.Command{
    Use:   "create [label]",
    Short: "Create a new temporary inbox",
    Long: `Create a new temporary email inbox with quantum-safe encryption.

The inbox uses ML-KEM-768 for key encapsulation and ML-DSA-65 for signatures.
Your private key never leaves your machine - all decryption happens locally.

Examples:
  vsb inbox create
  vsb inbox create auth-tests
  vsb inbox create --ttl 1h`,
    Args: cobra.MaximumNArgs(1),
    RunE: runInboxCreate,
}

var (
    createTTL string
)

func init() {
    inboxCmd.AddCommand(inboxCreateCmd)

    inboxCreateCmd.Flags().StringVar(&createTTL, "ttl", "24h",
        "Inbox lifetime (e.g., 1h, 24h, 7d)")
}

func runInboxCreate(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Get optional label
    label := ""
    if len(args) > 0 {
        label = args[0]
    }

    // Parse TTL
    ttl, err := time.ParseDuration(createTTL)
    if err != nil {
        return fmt.Errorf("invalid TTL format: %w", err)
    }

    // Show spinner
    fmt.Println(output.Info("Generating quantum-safe keys..."))

    // Create client
    client, err := config.NewClient()
    if err != nil {
        return err
    }
    defer client.Close()

    // Create inbox with SDK
    fmt.Println(output.Info("Registering with VaultSandbox..."))

    inbox, err := client.CreateInbox(ctx, vaultsandbox.WithTTL(ttl))
    if err != nil {
        return fmt.Errorf("failed to create inbox: %w", err)
    }

    // Export inbox data for keystore
    exported := inbox.Export()

    // Save to keystore
    keystore, err := config.LoadKeystore()
    if err != nil {
        return fmt.Errorf("failed to load keystore: %w", err)
    }

    stored := config.StoredInboxFromExport(exported, label)
    if err := keystore.AddInbox(stored); err != nil {
        return fmt.Errorf("failed to save inbox: %w", err)
    }

    // Pretty output
    printInboxCreated(stored)

    return nil
}

func printInboxCreated(inbox config.StoredInbox) {
    // Title
    title := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#10B981")).
        Render("Inbox Ready!")

    // Email address box
    emailBox := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#FFFFFF")).
        Background(lipgloss.Color("#7C3AED")).
        Padding(0, 2).
        Render(inbox.Email)

    // Details
    expiry := inbox.ExpiresAt.Sub(time.Now()).Round(time.Hour)
    expiryStr := fmt.Sprintf("%v", expiry)

    labelStr := inbox.Label
    if labelStr == "" {
        labelStr = "(none)"
    }

    details := fmt.Sprintf(`
  Address:  %s
  Label:    %s
  Security: ML-KEM-768 (Quantum-Safe)
  Expires:  %s

Run 'vsb watch' to see emails arrive live.`,
        emailBox, labelStr, expiryStr)

    // Box it all
    box := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("#7C3AED")).
        Padding(1, 2).
        Render(title + details)

    fmt.Println()
    fmt.Println(box)
    fmt.Println()
}
```

### 3. Inbox List Command

**File: `internal/cli/inbox_list.go`**

```go
package cli

import (
    "fmt"
    "strings"
    "time"

    "github.com/charmbracelet/lipgloss"
    "github.com/spf13/cobra"
    "github.com/vaultsandbox/vsb-cli/internal/config"
)

var inboxListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all stored inboxes",
    Long:  `Display all inboxes stored in the local keystore.`,
    Aliases: []string{"ls"},
    RunE:  runInboxList,
}

var (
    listShowExpired bool
)

func init() {
    inboxCmd.AddCommand(inboxListCmd)

    inboxListCmd.Flags().BoolVar(&listShowExpired, "all", false,
        "Show expired inboxes too")
}

func runInboxList(cmd *cobra.Command, args []string) error {
    keystore, err := config.LoadKeystore()
    if err != nil {
        return fmt.Errorf("failed to load keystore: %w", err)
    }

    inboxes := keystore.ListInboxes()
    if len(inboxes) == 0 {
        fmt.Println("No inboxes found. Create one with 'vsb inbox create'")
        return nil
    }

    // Styles
    headerStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#7C3AED"))

    activeStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#10B981"))

    expiredStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#6B7280")).
        Strikethrough(true)

    now := time.Now()

    // Header
    fmt.Println()
    fmt.Printf("%s  %-35s  %-12s  %s\n",
        headerStyle.Render(" "),
        headerStyle.Render("EMAIL"),
        headerStyle.Render("LABEL"),
        headerStyle.Render("EXPIRES"))
    fmt.Println(strings.Repeat("─", 70))

    for _, inbox := range inboxes {
        isActive := inbox.Email == keystore.ActiveInbox
        isExpired := inbox.ExpiresAt.Before(now)

        if isExpired && !listShowExpired {
            continue
        }

        // Active marker
        marker := "  "
        if isActive {
            marker = activeStyle.Render("→ ")
        }

        // Email
        email := inbox.Email
        if isExpired {
            email = expiredStyle.Render(email)
        } else if isActive {
            email = activeStyle.Render(email)
        }

        // Label
        label := inbox.Label
        if label == "" {
            label = "-"
        }

        // Expiry
        var expiry string
        if isExpired {
            expiry = expiredStyle.Render("expired")
        } else {
            remaining := inbox.ExpiresAt.Sub(now).Round(time.Minute)
            expiry = formatDuration(remaining)
        }

        fmt.Printf("%s%-35s  %-12s  %s\n", marker, email, label, expiry)
    }

    fmt.Println()
    return nil
}

func formatDuration(d time.Duration) string {
    if d < time.Hour {
        return fmt.Sprintf("%dm", int(d.Minutes()))
    }
    if d < 24*time.Hour {
        return fmt.Sprintf("%dh", int(d.Hours()))
    }
    return fmt.Sprintf("%dd", int(d.Hours()/24))
}
```

### 4. Inbox Use Command

**File: `internal/cli/inbox_use.go`**

```go
package cli

import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/output"
)

var inboxUseCmd = &cobra.Command{
    Use:   "use <email>",
    Short: "Switch active inbox",
    Long: `Set the active inbox for commands like 'watch', 'wait-for', etc.

Examples:
  vsb inbox use abc123@vaultsandbox.com`,
    Args: cobra.ExactArgs(1),
    RunE: runInboxUse,
}

func init() {
    inboxCmd.AddCommand(inboxUseCmd)
}

func runInboxUse(cmd *cobra.Command, args []string) error {
    email := args[0]

    keystore, err := config.LoadKeystore()
    if err != nil {
        return fmt.Errorf("failed to load keystore: %w", err)
    }

    if err := keystore.SetActiveInbox(email); err != nil {
        if err == config.ErrInboxNotFound {
            return fmt.Errorf("inbox not found: %s", email)
        }
        return err
    }

    fmt.Println(output.Success(fmt.Sprintf("Active inbox set to %s", email)))
    return nil
}
```

### 5. Inbox Delete Command

**File: `internal/cli/inbox_delete.go`**

```go
package cli

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/output"
)

var inboxDeleteCmd = &cobra.Command{
    Use:   "delete <email>",
    Short: "Delete an inbox",
    Long: `Delete an inbox from both the server and local keystore.

Examples:
  vsb inbox delete abc123@vaultsandbox.com
  vsb inbox delete --local-only abc123@vaultsandbox.com`,
    Aliases: []string{"rm"},
    Args:    cobra.ExactArgs(1),
    RunE:    runInboxDelete,
}

var (
    deleteLocalOnly bool
)

func init() {
    inboxCmd.AddCommand(inboxDeleteCmd)

    inboxDeleteCmd.Flags().BoolVar(&deleteLocalOnly, "local-only", false,
        "Only remove from local keystore, don't delete on server")
}

func runInboxDelete(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    email := args[0]

    keystore, err := config.LoadKeystore()
    if err != nil {
        return fmt.Errorf("failed to load keystore: %w", err)
    }

    // Delete from server unless --local-only
    if !deleteLocalOnly {
        client, err := config.NewClient()
        if err != nil {
            return err
        }
        defer client.Close()

        if err := client.DeleteInbox(ctx, email); err != nil {
            // Continue with local deletion even if server fails
            fmt.Println(output.Error(fmt.Sprintf("Warning: server deletion failed: %v", err)))
        } else {
            fmt.Println(output.Success("Deleted from server"))
        }
    }

    // Delete from keystore
    if err := keystore.RemoveInbox(email); err != nil {
        if err == config.ErrInboxNotFound {
            return fmt.Errorf("inbox not found in keystore: %s", email)
        }
        return err
    }

    fmt.Println(output.Success("Deleted from keystore"))
    return nil
}
```

## Verification

```bash
# Create an inbox
vsb inbox create test-inbox

# List inboxes
vsb inbox list

# Switch active inbox
vsb inbox use <email-from-list>

# Delete inbox
vsb inbox delete <email>
```

## Files Created

- `internal/cli/inbox.go`
- `internal/cli/inbox_create.go`
- `internal/cli/inbox_list.go`
- `internal/cli/inbox_use.go`
- `internal/cli/inbox_delete.go`

## Next Steps

Proceed to [04-watch-command.md](04-watch-command.md) to implement the real-time TUI dashboard.
