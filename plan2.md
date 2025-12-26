# Code Review: Simplification Opportunities

## 1. Duplicate `formatAuthResult` Function

**Location:** `internal/cli/audit.go:199-214` and `internal/tui/watch/security.go:108-121`

Nearly identical functions with different names:

```go
// audit.go
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

// security.go
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
```

**Simpler:** Move to `internal/styles/styles.go`:

```go
func FormatAuthResult(result string, pass, fail, warn lipgloss.Style) string {
    switch strings.ToLower(result) {
    case "pass":
        return pass.Render("PASS")
    case "fail", "hardfail":
        return fail.Render("FAIL")
    case "softfail", "none", "neutral":
        return warn.Render(strings.ToUpper(result))
    default:
        return result
    }
}
```

---

## 2. Repeated Style Definitions

Every file recreates the same pass/fail/warn styles:

```go
// audit.go:78-88
passStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Green)
failStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Red)
warnStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Yellow)

// security.go:22-24
passStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Green)
failStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Red)
warnStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Yellow)
```

**Simpler:** Add to `internal/styles/styles.go`:

```go
var (
    PassStyle = lipgloss.NewStyle().Bold(true).Foreground(Green)
    FailStyle = lipgloss.NewStyle().Bold(true).Foreground(Red)
    WarnStyle = lipgloss.NewStyle().Bold(true).Foreground(Yellow)
    LabelStyle = lipgloss.NewStyle().Foreground(Gray).Width(20)
)
```

---

## 3. Duplicate Inbox Lookup Error Handling

**Location:** `inbox_use.go:40-49` and `inbox_delete.go:50-57`

```go
// inbox_use.go
inbox, matches, err := keystore.FindInbox(partial)
if err == config.ErrMultipleMatches {
    return fmt.Errorf("multiple inboxes match '%s': %v", partial, matches)
}
if errors.Is(err, config.ErrInboxNotFound) {
    return fmt.Errorf("inbox not found: %s", partial)
}

// inbox_delete.go
inbox, matches, err := keystore.FindInbox(partial)
if err == config.ErrMultipleMatches {
    return fmt.Errorf("multiple inboxes match '%s': %v", partial, matches)
}
if err != nil {
    return fmt.Errorf("inbox not found: %s", partial)
}
```

**Note:** `helpers.go:21-38` already has `GetInbox()` that does this! These files don't use it.

**Simpler:** Just use the existing helper:

```go
// inbox_use.go - use existing GetInbox
ks, err := LoadKeystoreOrError()
if err != nil {
    return err
}
stored, err := GetInbox(ks, partial)
if err != nil {
    return err
}
```

---

## 4. Inline JSON Structs Everywhere

Each command defines its own anonymous JSON struct:

```go
// list.go:57-62
type emailJSON struct {
    ID         string `json:"id"`
    Subject    string `json:"subject"`
    From       string `json:"from"`
    ReceivedAt string `json:"receivedAt"`
}

// inbox_list.go:53-58
type inboxJSON struct {
    Email     string `json:"email"`
    ExpiresAt string `json:"expiresAt"`
    IsActive  bool   `json:"isActive"`
    IsExpired bool   `json:"isExpired"`
}
```

**Current:** 12+ inline JSON struct definitions across CLI files.

**Verdict:** Acceptable for small structs used once. Leave as-is unless they need reuse.

---

## 5. Over-engineered: Cleanup Function Pattern

**Location:** `helpers.go:44-75`

```go
func LoadAndImportInbox(ctx context.Context, emailFlag string) (*vaultsandbox.Inbox, func(), error) {
    // ... setup ...
    cleanup := func() {
        client.Close()
    }
    // ... more logic, calling cleanup() on every error path ...
    return inbox, cleanup, nil
}
```

The cleanup function is returned and deferred by callers. This works but adds cognitive overhead.

**Verdict:** Not broken, but consider returning the client directly for transparency.

---

## 6. `LoadKeystoreOrError` Wrapper Adds Little Value

**Location:** `helpers.go:12-18`

```go
func LoadKeystoreOrError() (*config.Keystore, error) {
    ks, err := config.LoadKeystore()
    if err != nil {
        return nil, fmt.Errorf("failed to load keystore: %w", err)
    }
    return ks, nil
}
```

This just adds "failed to load keystore:" prefix.

**Problem:** Some files use this helper, others call `config.LoadKeystore()` directly with their own prefix (see `inbox_use.go:35`, `inbox_delete.go:46`).

**Simpler:** Either always use the helper or remove it. The inconsistency is the issue.

---

## Summary

| Issue | Impact | Action |
|-------|--------|--------|
| Duplicate `formatAuthResult` | High | Move to `internal/styles` |
| Repeated pass/fail/warn styles | Medium | Add to `styles.go` |
| `inbox_use.go` not using `GetInbox()` | Low | Use existing helper |
| Inconsistent `LoadKeystoreOrError` usage | Low | Standardize |
| Inline JSON structs | None | Leave as-is |
| Cleanup function pattern | None | Leave as-is |

The codebase is well-structured overall. The main wins are consolidating auth result formatting and common styles.
