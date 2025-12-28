# Refactoring Plan: DRY & KISS Improvements

This document outlines specific refactoring tasks to improve code quality by applying DRY (Don't Repeat Yourself) and KISS (Keep It Simple, Stupid) principles.

---

## 1. Flatten Nested Error Handling in `inbox/info.go`

**File:** `internal/cli/inbox/info.go:59-76`

**Problem:** Pyramid of doom pattern with nested if-else chains makes the code hard to follow and maintain.

**Principle:** KISS - Simplify control flow with early returns.

### Before

```go
client, err := config.NewClient()
if err == nil {
    defer client.Close()
    inbox, importErr := client.ImportInbox(ctx, stored.ToExportedInbox())
    if importErr == nil {
        status, statusErr := inbox.GetSyncStatus(ctx)
        if statusErr == nil {
            emailCount = status.EmailCount
        } else {
            syncErr = statusErr
        }
    } else {
        syncErr = importErr
    }
} else {
    syncErr = err
}
```

### After

```go
emailCount, syncErr := getInboxEmailCount(ctx, stored)

// In a new helper function:
func getInboxEmailCount(ctx context.Context, stored *config.StoredInbox) (int, error) {
    client, err := config.NewClient()
    if err != nil {
        return 0, err
    }
    defer client.Close()

    inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
    if err != nil {
        return 0, err
    }

    status, err := inbox.GetSyncStatus(ctx)
    if err != nil {
        return 0, err
    }

    return status.EmailCount, nil
}
```

**Why:**
- Reduces nesting from 4 levels to 1
- Each error is handled immediately with early return
- Logic is extracted to a reusable, testable function
- Main function stays focused on its purpose

---

## 2. Extract Table Rendering Helper

**Files:**
- `internal/cli/inbox/list.go:74-92`
- `internal/cli/email/list.go:73-92`

**Problem:** Both files implement nearly identical table rendering logic with headers, separators, and styled rows.

**Principle:** DRY - Extract common table formatting into a reusable component.

### Before (inbox/list.go)

```go
fmt.Printf("   %s  %s\n",
    headerStyle.Render(fmt.Sprintf("%-38s", "EMAIL")),
    headerStyle.Render("EXPIRES"))
fmt.Println(strings.Repeat("-", 55))

for _, inbox := range inboxes {
    // ... format each row
    fmt.Printf("   %s  %s\n", emailStr, expiresStr)
}
```

### Before (email/list.go)

```go
fmt.Printf("  %s  %s  %s  %s\n",
    headerStyle.Render(fmt.Sprintf("%-8s", "ID")),
    headerStyle.Render(fmt.Sprintf("%-30s", "SUBJECT")),
    headerStyle.Render(fmt.Sprintf("%-25s", "FROM")),
    headerStyle.Render("RECEIVED"))
fmt.Println(strings.Repeat("-", 80))

for _, email := range emails {
    // ... format each row
}
```

### After

Create `internal/cliutil/table.go`:

```go
package cliutil

import (
    "fmt"
    "strings"

    "github.com/vaultsandbox/vsb-cli/internal/styles"
)

type Column struct {
    Header string
    Width  int  // 0 means no padding
}

type Table struct {
    Columns []Column
    Indent  string
}

func NewTable(columns ...Column) *Table {
    return &Table{Columns: columns, Indent: "  "}
}

func (t *Table) PrintHeader() {
    headers := make([]string, len(t.Columns))
    totalWidth := 0

    for i, col := range t.Columns {
        if col.Width > 0 {
            headers[i] = styles.HeaderStyle.Render(fmt.Sprintf("%-*s", col.Width, col.Header))
            totalWidth += col.Width + 2
        } else {
            headers[i] = styles.HeaderStyle.Render(col.Header)
            totalWidth += len(col.Header) + 2
        }
    }

    fmt.Printf("%s%s\n", t.Indent, strings.Join(headers, "  "))
    fmt.Println(strings.Repeat("-", totalWidth))
}

func (t *Table) PrintRow(values ...string) {
    cells := make([]string, len(values))
    for i, val := range values {
        if i < len(t.Columns) && t.Columns[i].Width > 0 {
            cells[i] = fmt.Sprintf("%-*s", t.Columns[i].Width, Truncate(val, t.Columns[i].Width))
        } else {
            cells[i] = val
        }
    }
    fmt.Printf("%s%s\n", t.Indent, strings.Join(cells, "  "))
}
```

Usage in `inbox/list.go`:

```go
table := cliutil.NewTable(
    cliutil.Column{Header: "EMAIL", Width: 38},
    cliutil.Column{Header: "EXPIRES"},
)
table.PrintHeader()

for _, inbox := range inboxes {
    table.PrintRow(emailStr, expiresStr)
}
```

**Why:**
- Eliminates duplicate table rendering logic
- Consistent formatting across all list commands
- Easy to add new table-based outputs
- Column widths defined in one place

---

## 3. Extract Optional Argument Parser

**Files:**
- `internal/cli/inbox/info.go:38-41`
- `internal/cli/email/view.go:48-51`
- `internal/cli/email/audit.go:46-49`
- `internal/cli/email/attachment.go:58-61`
- `internal/cli/email/url.go:48-51`
- `internal/cli/data/export.go:54-56`

**Problem:** Same 4-line pattern repeated in 6+ files for parsing optional positional arguments.

**Principle:** DRY - Create a simple helper function.

### Before (repeated in 6 files)

```go
emailArg := ""
if len(args) > 0 {
    emailArg = args[0]
}
```

### After

Add to `internal/cliutil/helpers.go`:

```go
// GetArg returns args[index] if it exists, otherwise returns defaultValue.
func GetArg(args []string, index int, defaultValue string) string {
    if index < len(args) {
        return args[index]
    }
    return defaultValue
}
```

Usage:

```go
emailArg := cliutil.GetArg(args, 0, "")
```

**Why:**
- Reduces 4 lines to 1 line
- Self-documenting function name
- Handles edge cases consistently
- Supports any argument index, not just 0

---

## 4. Simplify Email Deletion Logic in TUI

**File:** `internal/tui/emails/commands.go:42-62`

**Problem:** Unnecessary branching based on inbox count with duplicated delete logic in both branches.

**Principle:** KISS - Remove redundant conditional, find inbox first.

### Before

```go
func (m Model) deleteEmail() tea.Cmd {
    return func() tea.Msg {
        filtered := m.filteredEmails()
        if i := m.list.Index(); i >= 0 && i < len(filtered) {
            emailItem := filtered[i]
            for _, inbox := range m.inboxes {
                if len(m.inboxes) > 1 {
                    if inbox.EmailAddress() == emailItem.InboxLabel {
                        err := inbox.DeleteEmail(m.ctx, emailItem.Email.ID)
                        return emailDeletedMsg{emailID: emailItem.Email.ID, err: err}
                    }
                } else {
                    err := inbox.DeleteEmail(m.ctx, emailItem.Email.ID)
                    return emailDeletedMsg{emailID: emailItem.Email.ID, err: err}
                }
            }
        }
        return nil
    }
}
```

### After

```go
func (m Model) deleteEmail() tea.Cmd {
    return func() tea.Msg {
        filtered := m.filteredEmails()
        i := m.list.Index()
        if i < 0 || i >= len(filtered) {
            return nil
        }

        emailItem := filtered[i]
        inbox := m.findInboxForEmail(emailItem)
        if inbox == nil {
            return nil
        }

        err := inbox.DeleteEmail(m.ctx, emailItem.Email.ID)
        return emailDeletedMsg{emailID: emailItem.Email.ID, err: err}
    }
}

func (m Model) findInboxForEmail(item EmailItem) *vaultsandbox.Inbox {
    for _, inbox := range m.inboxes {
        if len(m.inboxes) == 1 || inbox.EmailAddress() == item.InboxLabel {
            return inbox
        }
    }
    return nil
}
```

**Why:**
- Removes duplicated `inbox.DeleteEmail()` call
- Early return for invalid index (clearer flow)
- Inbox lookup extracted to descriptive helper
- Delete logic appears exactly once

---

## 5. Define Column Width Constants

**Files:**
- `internal/cli/inbox/list.go:77`
- `internal/cli/email/list.go:74-76`

**Problem:** Magic numbers for column widths scattered across files.

**Principle:** DRY - Define constants in one place.

### Before

```go
// inbox/list.go
fmt.Sprintf("%-38s", "EMAIL")
strings.Repeat("-", 55)

// email/list.go
fmt.Sprintf("%-8s", "ID")
fmt.Sprintf("%-30s", "SUBJECT")
fmt.Sprintf("%-25s", "FROM")
strings.Repeat("-", 80)
```

### After

Add to `internal/styles/styles.go`:

```go
// Table column widths
const (
    ColWidthID      = 8
    ColWidthEmail   = 38
    ColWidthSubject = 30
    ColWidthFrom    = 25
)
```

Or better, use the Table helper from refactoring #2 which encapsulates widths.

**Why:**
- Single source of truth for column widths
- Easy to adjust formatting globally
- Self-documenting constants
- Prevents width mismatches between header and data

---

## 6. Extract TUI View Rendering Template

**Files:**
- `internal/tui/emails/view.go`
- `internal/tui/emails/links.go`
- `internal/tui/emails/attachments.go`
- `internal/tui/emails/security.go`
- `internal/tui/emails/raw.go`

**Problem:** All 5 view renderers follow identical pattern:
1. Check if `m.viewedEmail == nil`
2. Get `email := m.viewedEmail.Email`
3. Create `strings.Builder`
4. Write tabs via `m.renderTabs()`
5. Format content
6. Return string

**Principle:** DRY - Use a template pattern to eliminate repetition.

### Before (repeated 5 times)

```go
func (m Model) renderLinksView() string {
    if m.viewedEmail == nil {
        return m.renderTabs() + "\n\n  No email selected"
    }

    email := m.viewedEmail.Email
    var b strings.Builder
    b.WriteString(m.renderTabs())
    b.WriteString("\n\n")

    // ... content-specific logic ...

    return b.String()
}
```

### After

Add to `internal/tui/emails/render.go`:

```go
// renderDetailView handles common view rendering with a content function.
func (m Model) renderDetailView(emptyMsg string, renderContent func(*vaultsandbox.Email, *strings.Builder)) string {
    var b strings.Builder
    b.WriteString(m.renderTabs())
    b.WriteString("\n\n")

    if m.viewedEmail == nil {
        b.WriteString("  ")
        b.WriteString(emptyMsg)
        return b.String()
    }

    renderContent(m.viewedEmail.Email, &b)
    return b.String()
}
```

Usage in `links.go`:

```go
func (m Model) renderLinksView() string {
    return m.renderDetailView("No email selected", func(email *vaultsandbox.Email, b *strings.Builder) {
        if len(email.Links) == 0 {
            b.WriteString("  No links found in this email")
            return
        }
        for i, link := range email.Links {
            fmt.Fprintf(b, "  %d. %s\n", i+1, link)
        }
    })
}
```

**Why:**
- Eliminates 5x repeated null check and setup code
- Each view only defines its unique content logic
- Consistent structure across all views
- Easy to add new views following the pattern

---

## 7. Consolidate JSON Output Helpers

**File:** `internal/cliutil/json.go`

**Problem:** Four similar functions building map structures with overlapping fields:
- `EmailSummaryJSON`
- `EmailFullJSON`
- `InboxSummaryJSON`
- `InboxFullJSON`

**Principle:** DRY - Use builder pattern or optional fields.

### Before

```go
func EmailSummaryJSON(email *vaultsandbox.Email) map[string]interface{} {
    return map[string]interface{}{
        "id":      email.ID,
        "subject": email.Subject,
        "from":    email.From,
    }
}

func EmailFullJSON(email *vaultsandbox.Email) map[string]interface{} {
    return map[string]interface{}{
        "id":          email.ID,
        "subject":     email.Subject,
        "from":        email.From,
        "body":        email.Body,
        "received_at": email.ReceivedAt,
        "links":       email.Links,
        // ... more fields
    }
}
```

### After

```go
type EmailJSONOptions struct {
    IncludeBody    bool
    IncludeLinks   bool
    IncludeRaw     bool
}

func EmailJSON(email *vaultsandbox.Email, opts EmailJSONOptions) map[string]interface{} {
    m := map[string]interface{}{
        "id":          email.ID,
        "subject":     email.Subject,
        "from":        email.From,
        "received_at": email.ReceivedAt,
    }

    if opts.IncludeBody {
        m["body"] = email.Body
    }
    if opts.IncludeLinks {
        m["links"] = email.Links
    }
    if opts.IncludeRaw {
        m["raw"] = email.Raw
    }

    return m
}

// Convenience wrappers
func EmailSummaryJSON(email *vaultsandbox.Email) map[string]interface{} {
    return EmailJSON(email, EmailJSONOptions{})
}

func EmailFullJSON(email *vaultsandbox.Email) map[string]interface{} {
    return EmailJSON(email, EmailJSONOptions{
        IncludeBody:  true,
        IncludeLinks: true,
        IncludeRaw:   true,
    })
}
```

**Why:**
- Core fields defined once
- Options control what's included
- Easy to add new output variants
- Existing callers unchanged via wrappers

---

## Implementation Priority

| Priority | Task | Impact | Effort |
|----------|------|--------|--------|
| 1 | Flatten `info.go` error handling | High (readability) | Low |
| 2 | Extract optional arg parser | Medium (DRY) | Low |
| 3 | Simplify TUI delete logic | Medium (KISS) | Low |
| 4 | Define column constants | Low (DRY) | Low |
| 5 | Extract table renderer | High (DRY) | Medium |
| 6 | TUI view template pattern | Medium (DRY) | Medium |
| 7 | Consolidate JSON helpers | Low (DRY) | Medium |

---

## Testing Strategy

After each refactoring:

1. **Run existing tests:** `go test ./...`
2. **Manual smoke test:**
   - `./vsb inbox list`
   - `./vsb inbox info`
   - `./vsb email list`
   - `./vsb` (TUI mode)
3. **Verify JSON output:** `./vsb inbox list -o json | jq .`

---

## Notes

- Refactorings 1-4 are quick wins with immediate impact
- Refactorings 5-7 require more careful testing but provide better long-term maintainability
- All changes maintain backward compatibility with existing CLI behavior
- No public API changes required
