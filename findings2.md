# Refactoring Findings - Pre-Feature Analysis

**Date:** 2025-12-25
**Context:** Before implementing Phase 3.2 (audit) and Phase 4.1 (open/view) commands

## Summary

Analysis of the codebase reveals significant code duplication across CLI commands. Refactoring before adding new features will reduce code by ~150 lines and improve maintainability.

---

## 1. Critical Duplications

### 1.1 Client Creation Pattern (4 locations)

**Files affected:**
- `internal/cli/inbox_create.go:61`
- `internal/cli/inbox_delete.go:48`
- `internal/cli/watch.go:75`
- `internal/cli/waitfor.go:125`

**Duplicated code:**
```go
client, err := config.NewClient()
if err != nil {
    return err
}
defer client.Close()
```

### 1.2 Keystore Loading Pattern (6 locations)

**Files affected:**
- `internal/cli/inbox_list.go:33`
- `internal/cli/inbox_create.go:79`
- `internal/cli/inbox_delete.go:41`
- `internal/cli/inbox_use.go:30`
- `internal/cli/watch.go:47`
- `internal/cli/waitfor.go:106`

**Duplicated code:**
```go
keystore, err := config.LoadKeystore()
if err != nil {
    return fmt.Errorf("failed to load keystore: %w", err)
}
```

### 1.3 Inbox Selection Pattern (2+ locations)

**Files affected:**
- `internal/cli/watch.go:55-72`
- `internal/cli/waitfor.go:114-118`

**Duplicated code:**
```go
var stored *config.StoredInbox
if emailFlag != "" {
    stored, err = keystore.GetInbox(emailFlag)
} else {
    stored, err = keystore.GetActiveInbox()
}
if err != nil {
    return fmt.Errorf("no inbox found: %w", err)
}
```

### 1.4 Inbox Import Pattern (2 locations)

**Files affected:**
- `internal/cli/watch.go:83-90`
- `internal/cli/waitfor.go:137`

**Duplicated code:**
```go
inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
if err != nil {
    return err
}
```

---

## 2. Unused Code

### 2.1 `NewClientWithKeystore()` - Never Used

**Location:** `internal/config/client.go:28`

```go
func NewClientWithKeystore() (*vaultsandbox.Client, *Keystore, error) {
    // This helper exists but no command uses it
}
```

**Action:** Either use it or remove it.

---

## 3. Browser Code Location Issue

### Current State
- `internal/tui/watch/browser.go` - 38 lines
- Contains `openBrowser()` and `viewInBrowser()`
- Only accessible to TUI code

### Problem
- New CLI commands (`open`, `view`) need browser operations
- Can't import from TUI package without circular dependencies

### Solution
- Move to `internal/browser/browser.go` (as specified in plan)
- Both TUI and CLI can import from shared location

---

## 4. Style Duplication

### CLI Inline Styles
**Location:** `internal/cli/inbox_list.go:45-55`
```go
headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
expiredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
```

**Location:** `internal/cli/inbox_create.go:97-134`
```go
title := lipgloss.NewStyle().Bold(true)...
emailBox := lipgloss.NewStyle()...
```

### TUI Centralized Styles
**Location:** `internal/tui/styles/styles.go`
```go
var (
    Purple = lipgloss.Color("99")
    Green  = lipgloss.Color("42")
    // ... centralized colors
)
```

### Problem
- CLI defines styles inline
- TUI uses centralized styles
- Colors may diverge

### Solution
- Extend `internal/tui/styles/` to be `internal/styles/`
- Both CLI and TUI import from same location

---

## 5. New Commands Will Need These Patterns

| Command | Keystore | Client | Inbox Select | Email Fetch | Browser |
|---------|----------|--------|--------------|-------------|---------|
| `audit` | ✓ | ✓ | ✓ | ✓ | ✗ |
| `open` | ✓ | ✓ | ✓ | ✓ | ✓ |
| `view` | ✓ | ✓ | ✓ | ✓ | ✓ |

Without refactoring, each new command adds ~40-50 lines of duplicated boilerplate.

---

## 6. Proposed Refactoring

### Phase 1: Create CLI Helpers

**New file:** `internal/cli/helpers.go`

```go
package cli

// CommandContext holds common resources for CLI commands
type CommandContext struct {
    Client   *vaultsandbox.Client
    Keystore *config.Keystore
    Ctx      context.Context
    cancel   context.CancelFunc
}

// NewCommandContext creates client and loads keystore
func NewCommandContext(timeout time.Duration) (*CommandContext, error)

// Close cleans up resources
func (c *CommandContext) Close()

// GetActiveInbox returns the active inbox or specified by email
func (c *CommandContext) GetActiveInbox(emailFlag string) (*config.StoredInbox, error)

// GetLatestEmail returns the most recent email from active inbox
func (c *CommandContext) GetLatestEmail(emailFlag string) (*vaultsandbox.Email, error)

// GetEmail returns a specific email by ID
func (c *CommandContext) GetEmail(emailFlag, emailID string) (*vaultsandbox.Email, error)
```

### Phase 2: Extract Browser Package

**Move:** `internal/tui/watch/browser.go` → `internal/browser/browser.go`

```go
package browser

func Open(url string) error
func OpenHTML(html string) error
func CleanupPreviews() error
```

### Phase 3: Consolidate Styles

**Rename:** `internal/tui/styles/` → `internal/styles/`

Update imports in:
- `internal/tui/watch/model.go`
- `internal/cli/inbox_list.go`
- `internal/cli/inbox_create.go`

---

## 7. Impact Assessment

### Before Refactoring
- 4 commands × ~40 lines boilerplate = 160 lines duplicated
- New command requires copying same patterns
- Bug fixes need updating in multiple places

### After Refactoring
- Common patterns in 1 location (~60 lines)
- New command needs ~10 lines to use helpers
- Bug fixes in one place

### Files to Modify
1. `internal/cli/helpers.go` (NEW)
2. `internal/browser/browser.go` (NEW - moved from TUI)
3. `internal/styles/styles.go` (RENAMED from tui/styles)
4. `internal/cli/watch.go` (UPDATE - use helpers)
5. `internal/cli/waitfor.go` (UPDATE - use helpers)
6. `internal/cli/inbox_*.go` (UPDATE - use shared styles)
7. `internal/tui/watch/model.go` (UPDATE - import paths)
8. `internal/tui/watch/browser.go` (DELETE - moved)

---

## 8. Recommended Order

1. **Create `internal/browser/browser.go`** - No breaking changes
2. **Update TUI to use new browser package** - Simple import change
3. **Create `internal/cli/helpers.go`** - No breaking changes
4. **Update `watch.go` to use helpers** - First migration
5. **Update `waitfor.go` to use helpers** - Second migration
6. **Move styles package** - Update all imports
7. **Implement new features** - audit, open, view

---

## 9. Decision

**Recommendation:** Refactor first, then implement features.

**Rationale:**
- Each new command will be ~50 lines shorter
- Consistent error handling across all commands
- Browser code available to both CLI and TUI
- Single source of truth for styles
- Easier code review for new features
