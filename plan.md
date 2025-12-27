# Plan: Reorganize internal/cli into subpackages

## Final Structure

```
internal/cli/
├── root.go              # Root command, TUI launcher
├── config.go            # Config command
├── helpers.go           # Shared helpers (LoadKeystoreOrError, GetInbox, etc.)
├── utils.go             # Formatting utilities
├── json.go              # JSON formatters
├── interfaces.go        # Shared interfaces
├── helpers_test.go
├── utils_test.go
├── json_test.go
│
├── inbox/               # Inbox management commands
│   ├── inbox.go         # Parent command
│   ├── create.go
│   ├── list.go
│   ├── use.go
│   ├── delete.go
│   ├── info.go
│   ├── list_test.go
│   ├── use_test.go
│   └── info_test.go
│
├── email/               # Email commands
│   ├── email.go         # Parent command
│   ├── list.go
│   ├── view.go
│   ├── audit.go
│   ├── attachment.go
│   ├── delete.go
│   ├── url.go
│   ├── wait.go          # Wait for email command
│   ├── audit_test.go
│   ├── attachment_test.go
│   └── wait_test.go
│
└── data/                # Export/Import commands
    ├── export.go
    ├── import.go
    ├── export_test.go
    └── import_test.go
```

## Implementation Steps

### Step 1: Create directories
```bash
mkdir -p internal/cli/inbox internal/cli/email internal/cli/data
```

### Step 2: Move inbox files
```bash
git mv internal/cli/inbox.go internal/cli/inbox/inbox.go
git mv internal/cli/inbox_create.go internal/cli/inbox/create.go
git mv internal/cli/inbox_list.go internal/cli/inbox/list.go
git mv internal/cli/inbox_use.go internal/cli/inbox/use.go
git mv internal/cli/inbox_delete.go internal/cli/inbox/delete.go
git mv internal/cli/inbox_info.go internal/cli/inbox/info.go
git mv internal/cli/inbox_list_test.go internal/cli/inbox/list_test.go
git mv internal/cli/inbox_use_test.go internal/cli/inbox/use_test.go
git mv internal/cli/inbox_info_test.go internal/cli/inbox/info_test.go
```

### Step 3: Move email files
```bash
git mv internal/cli/email.go internal/cli/email/email.go
git mv internal/cli/list.go internal/cli/email/list.go
git mv internal/cli/view.go internal/cli/email/view.go
git mv internal/cli/audit.go internal/cli/email/audit.go
git mv internal/cli/attachment.go internal/cli/email/attachment.go
git mv internal/cli/email_delete.go internal/cli/email/delete.go
git mv internal/cli/url.go internal/cli/email/url.go
git mv internal/cli/wait.go internal/cli/email/wait.go
git mv internal/cli/audit_test.go internal/cli/email/audit_test.go
git mv internal/cli/attachment_test.go internal/cli/email/attachment_test.go
git mv internal/cli/wait_test.go internal/cli/email/wait_test.go
```

### Step 4: Move data files
```bash
git mv internal/cli/export.go internal/cli/data/export.go
git mv internal/cli/import.go internal/cli/data/import.go
git mv internal/cli/export_test.go internal/cli/data/export_test.go
git mv internal/cli/import_test.go internal/cli/data/import_test.go
```

### Step 5: Update package declarations
- Files in `inbox/` → `package inbox`
- Files in `email/` → `package email`
- Files in `data/` → `package data`

### Step 6: Export shared utilities
In parent `cli` package, ensure these are exported (capitalized):
- `LoadKeystoreOrError` → already exported
- `GetInbox` → already exported
- `LoadAndImportInbox` → already exported
- `GetEmailByIDOrLatest` → already exported
- Helper functions may need exporting

### Step 7: Update imports in subpackages
Each subpackage imports parent: `"github.com/vaultsandbox/vsb-cli/internal/cli"`

### Step 8: Update root.go
Import and register subcommands:
```go
import (
    "github.com/vaultsandbox/vsb-cli/internal/cli/inbox"
    "github.com/vaultsandbox/vsb-cli/internal/cli/email"
    "github.com/vaultsandbox/vsb-cli/internal/cli/data"
)

func init() {
    rootCmd.AddCommand(inbox.Cmd)
    rootCmd.AddCommand(email.Cmd)
    rootCmd.AddCommand(data.ExportCmd)
    rootCmd.AddCommand(data.ImportCmd)
}
```

### Step 9: Run tests
```bash
go build ./...
go test ./internal/cli/...
```

## Files Remaining in cli/
- `root.go` - Root command
- `config.go` - Config command
- `helpers.go` - Shared helpers
- `utils.go` - Utilities
- `json.go` - JSON formatters
- `interfaces.go` - Shared interfaces
- Test files for above
