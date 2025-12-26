# Refactoring Plan

Technical debt and improvements identified for better KISS/DRY adherence.

## High Priority

### 1. JSON Serialization Boilerplate
**Location:** 8+ CLI commands (`list.go`, `view.go`, `audit.go`, `inbox_list.go`, `inbox_info.go`, `wait.go`, etc.)

**Problem:** Each command defines its own `emailJSON`/`inboxJSON` structs with identical field mappings.

**Solution:** Create `internal/cli/json.go` with reusable serializers:
```go
func emailToJSON(email *vaultsandbox.Email) map[string]interface{}
func inboxToJSON(inbox *config.StoredInbox) map[string]interface{}
```

### 2. TUI Update Function Complexity
**Location:** `internal/tui/emails/update.go` (286 lines)

**Problem:** Massive nested switch statements handling detail view, list view, tabs, and operations.

**Solution:** Extract sub-handlers:
```go
func (m Model) handleDetailViewUpdate(msg tea.Msg) (Model, tea.Cmd)
func (m Model) handleListViewUpdate(msg tea.Msg) (Model, tea.Cmd)
```

### 3. Duplicate Auth Results Rendering
**Location:**
- `internal/cli/audit.go:printAuthResults()` (lines 220-258)
- `internal/styles/styles.go:RenderAuthResults()` (lines 154-197)

**Problem:** Two functions iterate over SPF/DKIM/DMARC/ReverseDNS with similar logic but different output styles.

**Solution:** Unify into single function in `styles.go`, parameterize output format if needed.

---

## Medium Priority

### 4. Email-to-Map Conversion Duplication
**Location:** `view.go`, `wait.go`, `audit.go`

**Problem:** Near-identical `map[string]interface{}` construction for email data.

**Solution:** create shared version in `helpers.go`.

### 5. Inline Styles Scattered Across Commands
**Location:** `list.go`, `inbox_info.go`, `view.go`, TUI files

**Problem:** Commands define styles inline:
```go
idStyle := lipgloss.NewStyle().Foreground(styles.Gray)
subjectStyle := lipgloss.NewStyle().Bold(true)
```

**Solution:** Add to `internal/styles/styles.go`:
```go
var IDStyle = lipgloss.NewStyle().Foreground(Gray)
var SubjectStyle = lipgloss.NewStyle().Bold(true)
```

### 6. TUI Model Struct Size
**Location:** `internal/tui/emails/model.go`

**Problem:** Model mixes list view state, detail view state, connection state, and error state.

**Solution:** Consider splitting into sub-models for better testability:
- `ListModel` - list navigation state
- `DetailModel` - email detail view state
- `ConnectionModel` - SSE connection state

---

## Low Priority

### 7. API Key Masking Duplication
**Location:** `internal/cli/config.go` (lines 81-86 and 125-131)

**Problem:** Same masking logic appears twice.

**Solution:** Extract helper:
```go
func maskAPIKey(key string) string {
    if len(key) >= 11 {
        return key[:7] + "..." + key[len(key)-4:]
    }
    return "****"
}
```

### 8. String Concatenation in sanitizeFilename
**Location:** `internal/cli/utils.go` (lines 30-43)

**Problem:** Uses string concatenation in loop.

**Solution:** Use `strings.Builder` for better performance.

### 9. Output Format Check Pattern
**Location:** Nearly every CLI command

**Problem:** Repeated `if getOutput(cmd) == "json"` checks.

**Solution:** Consider `OutputMode` enum:
```go
type OutputMode int
const (
    Pretty OutputMode = iota
    JSON
)
```

---

## Other Notes

- **Global config state** (`internal/config/config.go` line 20) - consider dependency injection for testability
- **Silent error in initConfig** (`internal/cli/root.go` lines 43-55) - config load errors are swallowed
