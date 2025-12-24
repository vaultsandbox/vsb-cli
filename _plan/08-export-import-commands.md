# Phase 4.2: Export and Import Commands

## Objective
Implement `vsb export` and `vsb import` for portable identity and team collaboration.

## Commands

| Command | Description |
|---------|-------------|
| `vsb export [email]` | Export inbox with keys to file |
| `vsb export --out file.json` | Export to specific file |
| `vsb import <file>` | Import inbox from file |

## Security Considerations

1. **Private Key Exposure**: Export files contain private keys
2. **File Permissions**: Exported files should be 0600
3. **Security Warnings**: Always warn users about key material

## Tasks

### 1. Export Command

**File: `internal/cli/export.go`**

```go
package cli

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/charmbracelet/lipgloss"
    "github.com/spf13/cobra"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/output"
    "github.com/vaultsandbox/vsb-cli/internal/tui/styles"
)

var exportCmd = &cobra.Command{
    Use:   "export [email-address]",
    Short: "Export inbox with private keys",
    Long: `Export an inbox's keys and metadata to a JSON file for backup or sharing.

WARNING: The exported file contains your PRIVATE KEY. Anyone with this file
can read emails sent to your inbox. Handle it securely!

Use cases:
- Backup inbox before it expires
- Share inbox with CI/CD systems
- Transfer inbox to another machine/team member

Examples:
  vsb export                     # Export active inbox
  vsb export abc@vsb.com         # Export specific inbox
  vsb export --out ~/backup.json # Specify output file`,
    Args: cobra.MaximumNArgs(1),
    RunE: runExport,
}

var (
    exportOut string
)

func init() {
    rootCmd.AddCommand(exportCmd)

    exportCmd.Flags().StringVarP(&exportOut, "out", "o", "",
        "Output file path (default: <email>.json)")
}

func runExport(cmd *cobra.Command, args []string) error {
    // Get inbox to export
    keystore, err := config.LoadKeystore()
    if err != nil {
        return err
    }

    var stored *config.StoredInbox
    if len(args) > 0 {
        stored, err = keystore.GetInbox(args[0])
    } else {
        stored, err = keystore.GetActiveInbox()
    }
    if err != nil {
        return fmt.Errorf("inbox not found: %w", err)
    }

    // Check if expired
    if stored.ExpiresAt.Before(time.Now()) {
        warningBox := lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(styles.Yellow).
            Padding(0, 1).
            Render(styles.Yellow.Render("Warning: This inbox has expired"))
        fmt.Println(warningBox)
    }

    // Determine output file
    outPath := exportOut
    if outPath == "" {
        // Default to email.json in current directory
        safeEmail := sanitizeFilename(stored.Email)
        outPath = safeEmail + ".json"
    }

    // Convert to absolute path
    absPath, err := filepath.Abs(outPath)
    if err != nil {
        return err
    }

    // Check if file exists
    if _, err := os.Stat(absPath); err == nil {
        return fmt.Errorf("file already exists: %s (use --out to specify different path)", absPath)
    }

    // Create export data
    exportData := ExportedInboxFile{
        Version:      1,
        EmailAddress: stored.Email,
        InboxHash:    stored.ID,
        Label:        stored.Label,
        ExpiresAt:    stored.ExpiresAt,
        ExportedAt:   time.Now(),
        Keys: ExportedKeys{
            KEMPrivate:  stored.Keys.KEMPrivate,
            KEMPublic:   stored.Keys.KEMPublic,
            ServerSigPK: stored.Keys.ServerSigPK,
        },
    }

    // Marshal to JSON
    data, err := json.MarshalIndent(exportData, "", "  ")
    if err != nil {
        return err
    }

    // Write with secure permissions
    if err := os.WriteFile(absPath, data, 0600); err != nil {
        return err
    }

    // Security warning
    printExportWarning(absPath, stored.Email)

    return nil
}

// ExportedInboxFile is the file format for exported inboxes
type ExportedInboxFile struct {
    Version      int          `json:"version"`
    EmailAddress string       `json:"emailAddress"`
    InboxHash    string       `json:"inboxHash"`
    Label        string       `json:"label,omitempty"`
    ExpiresAt    time.Time    `json:"expiresAt"`
    ExportedAt   time.Time    `json:"exportedAt"`
    Keys         ExportedKeys `json:"keys"`
}

type ExportedKeys struct {
    KEMPrivate  string `json:"kemPrivate"`
    KEMPublic   string `json:"kemPublic"`
    ServerSigPK string `json:"serverSigPk"`
}

func sanitizeFilename(email string) string {
    // Replace @ and . with underscores for safe filename
    result := ""
    for _, r := range email {
        if r == '@' || r == '.' {
            result += "_"
        } else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
                  (r >= '0' && r <= '9') || r == '-' || r == '_' {
            result += string(r)
        }
    }
    return result
}

func printExportWarning(path, email string) {
    warningStyle := lipgloss.NewStyle().
        Bold(true).
        Foreground(styles.Yellow)

    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(styles.Yellow).
        Padding(1, 2)

    warning := fmt.Sprintf(`%s

This file contains your PRIVATE KEY for:
  %s

Anyone with this file can read emails sent to this inbox.
Keep it secure and do not commit it to version control!

File: %s`,
        warningStyle.Render("SECURITY WARNING"),
        email,
        path)

    fmt.Println()
    fmt.Println(boxStyle.Render(warning))
    fmt.Println()
    fmt.Println(output.Success("Export complete"))
}
```

### 2. Import Command

**File: `internal/cli/import.go`**

```go
package cli

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/charmbracelet/lipgloss"
    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/output"
    "github.com/vaultsandbox/vsb-cli/internal/tui/styles"
)

var importCmd = &cobra.Command{
    Use:   "import <file>",
    Short: "Import inbox from export file",
    Long: `Import an inbox from a previously exported JSON file.

This adds the inbox to your local keystore and optionally verifies
it's still valid on the server.

Examples:
  vsb import backup.json          # Import and verify
  vsb import backup.json --local  # Skip server verification
  vsb import backup.json --label "shared-inbox"`,
    Args: cobra.ExactArgs(1),
    RunE: runImport,
}

var (
    importLocal bool
    importLabel string
    importForce bool
)

func init() {
    rootCmd.AddCommand(importCmd)

    importCmd.Flags().BoolVar(&importLocal, "local", false,
        "Skip server verification")
    importCmd.Flags().StringVar(&importLabel, "label", "",
        "Override the label for imported inbox")
    importCmd.Flags().BoolVar(&importForce, "force", false,
        "Overwrite existing inbox with same email")
}

func runImport(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    filePath := args[0]

    // Read file
    data, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }

    // Parse JSON
    var exported ExportedInboxFile
    if err := json.Unmarshal(data, &exported); err != nil {
        return fmt.Errorf("invalid export file format: %w", err)
    }

    // Validate version
    if exported.Version != 1 {
        return fmt.Errorf("unsupported export file version: %d", exported.Version)
    }

    // Check if expired
    if exported.ExpiresAt.Before(time.Now()) {
        warningBox := lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(styles.Red).
            Padding(0, 1).
            Render(styles.Red.Render("Error: This inbox has expired"))
        fmt.Println(warningBox)
        return fmt.Errorf("inbox expired on %s", exported.ExpiresAt.Format("2006-01-02"))
    }

    // Load keystore
    keystore, err := config.LoadKeystore()
    if err != nil {
        return err
    }

    // Check for existing inbox
    existing, _ := keystore.GetInbox(exported.EmailAddress)
    if existing != nil && !importForce {
        return fmt.Errorf("inbox already exists: %s (use --force to overwrite)", exported.EmailAddress)
    }

    // Server verification (unless --local)
    if !importLocal {
        fmt.Println(output.Info("Verifying with server..."))

        client, err := config.NewClient()
        if err != nil {
            return err
        }
        defer client.Close()

        // Try to import into SDK to verify
        sdkExport := &vaultsandbox.ExportedInbox{
            EmailAddress: exported.EmailAddress,
            ExpiresAt:    exported.ExpiresAt,
            InboxHash:    exported.InboxHash,
            ServerSigPk:  exported.Keys.ServerSigPK,
            PublicKeyB64: exported.Keys.KEMPublic,
            SecretKeyB64: exported.Keys.KEMPrivate,
            ExportedAt:   exported.ExportedAt,
        }

        inbox, err := client.ImportInbox(ctx, sdkExport)
        if err != nil {
            return fmt.Errorf("server verification failed: %w", err)
        }

        // Check sync status
        status, err := inbox.GetSyncStatus(ctx)
        if err != nil {
            fmt.Println(output.Info("Warning: Could not verify sync status"))
        } else {
            fmt.Println(output.Success(fmt.Sprintf("Inbox verified: %d emails", status.EmailCount)))
        }
    }

    // Determine label
    label := exported.Label
    if importLabel != "" {
        label = importLabel
    }

    // Save to keystore
    stored := config.StoredInbox{
        Email:     exported.EmailAddress,
        ID:        exported.InboxHash,
        Label:     label,
        CreatedAt: exported.ExportedAt,
        ExpiresAt: exported.ExpiresAt,
        Keys: config.InboxKeys{
            KEMPrivate:  exported.Keys.KEMPrivate,
            KEMPublic:   exported.Keys.KEMPublic,
            ServerSigPK: exported.Keys.ServerSigPK,
        },
    }

    if err := keystore.AddInbox(stored); err != nil {
        return err
    }

    // Success output
    printImportSuccess(stored)

    return nil
}

func printImportSuccess(inbox config.StoredInbox) {
    remaining := time.Until(inbox.ExpiresAt).Round(time.Hour)

    boxStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(styles.Green).
        Padding(1, 2)

    content := fmt.Sprintf(`%s

Address:  %s
Label:    %s
Expires:  %s

This inbox is now your active inbox.
Run 'vsb watch' to see emails.`,
        output.Success("Import Complete"),
        inbox.Email,
        orDefault(inbox.Label, "(none)"),
        remaining.String())

    fmt.Println()
    fmt.Println(boxStyle.Render(content))
    fmt.Println()
}

func orDefault(s, def string) string {
    if s == "" {
        return def
    }
    return s
}
```

## Export File Format

```json
{
  "version": 1,
  "emailAddress": "abc123@vaultsandbox.com",
  "inboxHash": "sha256-...",
  "label": "auth-tests",
  "expiresAt": "2024-01-22T14:30:00Z",
  "exportedAt": "2024-01-15T14:30:00Z",
  "keys": {
    "kemPrivate": "base64...",
    "kemPublic": "base64...",
    "serverSigPk": "base64..."
  }
}
```

## Usage Examples

### Team Collaboration
```bash
# Developer A exports inbox
vsb export --out shared-inbox.json

# Developer B imports it
vsb import shared-inbox.json --label "shared"

# Both can now watch the same inbox
vsb watch
```

### CI/CD Persistence
```bash
# In CI setup script
vsb import $CI_INBOX_CREDENTIALS

# Run tests
./run-email-tests.sh

# Cleanup
vsb inbox delete --local-only
```

### Backup Before Expiry
```bash
# Export before it expires
vsb export important@vsb.com --out ~/backups/inbox-backup.json

# Later, check if still valid
vsb import ~/backups/inbox-backup.json
```

## Security Best Practices

1. **Never commit export files to git**
   ```bash
   echo "*.vaultsandbox.json" >> .gitignore
   ```

2. **Use environment variables in CI/CD**
   ```yaml
   # GitHub Actions
   - run: echo '${{ secrets.VSB_INBOX }}' > inbox.json
   - run: vsb import inbox.json
   - run: rm inbox.json
   ```

3. **Set restrictive file permissions**
   ```bash
   chmod 600 exported-inbox.json
   ```

## Verification

```bash
# Export active inbox
vsb export --out test-export.json

# Check file permissions
ls -la test-export.json
# Should show: -rw------- (0600)

# Import to verify format
vsb inbox delete --local-only $(vsb inbox list --active)
vsb import test-export.json

# Clean up
rm test-export.json
```

## Files Created

- `internal/cli/export.go`
- `internal/cli/import.go`

## Implementation Complete

This completes all planned commands for the VaultSandbox CLI. The implementation includes:

1. **Foundation**: Project setup, configuration, and keystore management
2. **Core Commands**: inbox create/list/use/delete
3. **Real-time TUI**: watch command with Bubble Tea
4. **CI/CD**: wait-for command with filters
5. **Developer Tools**: audit, open, view commands
6. **Portability**: export/import functionality
