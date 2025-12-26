# Code Review: Simplification Opportunities

## 1. Consolidate Configuration Logic

**Issue:** Logic for determining output format (Flag > Env > Config > Default) is split between `internal/cli/utils.go` and `internal/config/config.go`.

**Current flow:**
```go
// internal/cli/utils.go
func getOutput(cmd *cobra.Command) string {
    if f := cmd.Flag("output"); f != nil && f.Changed {
        return f.Value.String()
    }
    if env := os.Getenv("VSB_OUTPUT"); env != "" {
        return env
    }
    return config.GetDefaultOutput()
}

// internal/config/config.go
func GetDefaultOutput() string {
    cfg := Load()
    if cfg.DefaultOutput != "" {
        return cfg.DefaultOutput
    }
    return "pretty"
}
```

**Simpler:** Move env var check into config:

```go
// internal/config/config.go
func GetDefaultOutput() string {
    if env := os.Getenv("VSB_OUTPUT"); env != "" {
        return env
    }
    cfg := Load()
    if cfg.DefaultOutput != "" {
        return cfg.DefaultOutput
    }
    return "pretty"
}

// internal/cli/utils.go
func getOutput(cmd *cobra.Command) string {
    if f := cmd.Flag("output"); f != nil && f.Changed {
        return f.Value.String()
    }
    return config.GetDefaultOutput()
}
```

---

## 2. Duplicate `formatAuthResult` Function (HIGH IMPACT)

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

## 3. Repeated Style Definitions (MEDIUM IMPACT)

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

## 4. Duplicate Inbox Lookup Error Handling

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

## 5. Inline JSON Structs Everywhere (SKIP)

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

## 6. Cleanup Function Pattern (SKIP)

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

## 7. `LoadKeystoreOrError` Wrapper Inconsistency

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

| # | Issue | Impact | Action |
|---|-------|--------|--------|
| 1 | Config logic split across files | Medium | Move env check into `config.GetDefaultOutput()` |
| 2 | Duplicate `formatAuthResult` | **High** | Move to `internal/styles` |
| 3 | Repeated pass/fail/warn styles | Medium | Add to `styles.go` |
| 4 | `inbox_use.go` not using `GetInbox()` | Low | Use existing helper |
| 5 | Inline JSON structs | Skip | Leave as-is |
| 6 | Cleanup function pattern | Skip | Leave as-is |
| 7 | Inconsistent `LoadKeystoreOrError` usage | Low | Standardize |

## Execution Order

1. **styles.go changes** (items 2 & 3): Add `PassStyle`, `FailStyle`, `WarnStyle` and `FormatAuthResult()` function
2. **Update consumers**: Change `audit.go` and `security.go` to use shared styles/function
3. **Config consolidation** (item 1): Move env var check into config package
4. **Helper cleanup** (items 4 & 7): Use `GetInbox()` consistently in inbox commands
