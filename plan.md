# Code Simplification Plan

## Overview

This plan addresses code duplication and unnecessary abstractions identified in the vsb-cli codebase, applying the KISS principle.

---

## 1. Delete `internal/output/` Package (HIGH)

**Problem:** The `output/printer.go` package (59 lines) duplicates color definitions from `styles/styles.go` and provides only 3 trivial functions used in 4 places.

**Files affected:**
- `internal/output/printer.go` (DELETE)
- `internal/cli/inbox_create.go`
- `internal/cli/export.go`
- `internal/cli/import.go`

**Changes:**

1. Add `MutedStyle` to `internal/styles/styles.go`:
```go
MutedStyle = lipgloss.NewStyle().Foreground(Gray)
```

2. Replace usages:
```go
// Before
import "github.com/vaultsandbox/vsb-cli/internal/output"
fmt.Println(output.PrintSuccess("msg"))
fmt.Println(output.PrintInfo("msg"))

// After
import "github.com/vaultsandbox/vsb-cli/internal/styles"
fmt.Println(styles.PassStyle.Render("✓ msg"))
fmt.Println(styles.MutedStyle.Render("• msg"))
```

3. Delete `internal/output/` directory

**Impact:** -59 lines, eliminates duplicate color definitions

---

## 2. Consolidate Security Score Styling (MEDIUM)

**Problem:** Identical score-to-color threshold logic duplicated in two files.

**Files affected:**
- `internal/cli/audit.go` (lines 170-176)
- `internal/tui/watch/security.go` (lines 92-98)
- `internal/styles/styles.go` (add function)

**Changes:**

1. Add to `internal/styles/styles.go`:
```go
// ScoreStyle returns the appropriate style for a security score (0-100).
func ScoreStyle(score int) lipgloss.Style {
    if score < 60 {
        return FailStyle
    }
    if score < 80 {
        return WarnStyle
    }
    return PassStyle
}
```

2. Update `audit.go`:
```go
// Before
scoreColor := styles.PassStyle
if score < 80 {
    scoreColor = styles.WarnStyle
}
if score < 60 {
    scoreColor = styles.FailStyle
}
summary := fmt.Sprintf("Security Score: %s", scoreColor.Render(...))

// After
summary := fmt.Sprintf("Security Score: %s", styles.ScoreStyle(score).Render(...))
```

3. Update `security.go`:
```go
// Before
scoreStyle := styles.PassStyle
if score < 80 {
    scoreStyle = styles.WarnStyle
}
if score < 60 {
    scoreStyle = styles.FailStyle
}
sb.WriteString(scoreStyle.Render(...))

// After
sb.WriteString(styles.ScoreStyle(score).Render(...))
```

**Impact:** -10 lines, single source of truth for score thresholds

---

## 3. Inline `extractHeader` Function (LOW)

**Problem:** 6-line helper function used exactly once.

**File:** `internal/cli/audit.go`

**Changes:**

```go
// Before (lines 153-154 + 186-191)
tlsVersion := extractHeader(email.Headers, "X-TLS-Version", "TLS 1.3")
cipherSuite := extractHeader(email.Headers, "X-TLS-Cipher", "ECDHE-RSA-AES256-GCM-SHA384")

func extractHeader(headers map[string]string, key, defaultVal string) string {
    if val, ok := headers[key]; ok && val != "" {
        return val
    }
    return defaultVal
}

// After (inline at usage site)
tlsVersion := "TLS 1.3"
if v := email.Headers["X-TLS-Version"]; v != "" {
    tlsVersion = v
}
cipherSuite := "ECDHE-RSA-AES256-GCM-SHA384"
if v := email.Headers["X-TLS-Cipher"]; v != "" {
    cipherSuite = v
}
```

**Impact:** -6 lines, removes unnecessary abstraction

---

## 4. Merge `convert.go` into `keystore.go` (LOW)

**Problem:** `internal/config/convert.go` (34 lines) contains 2 functions only used by keystore operations.

**Files affected:**
- `internal/config/convert.go` (DELETE)
- `internal/config/keystore.go` (add functions)

**Changes:**

Move these functions to end of `keystore.go`:
```go
// StoredInboxFromExport converts SDK ExportedInbox to StoredInbox for keystore storage.
func StoredInboxFromExport(e *vaultsandbox.ExportedInbox) StoredInbox { ... }

// ToExportedInbox converts StoredInbox back to SDK ExportedInbox format.
func (s *StoredInbox) ToExportedInbox() *vaultsandbox.ExportedInbox { ... }
```

Delete `convert.go`.

**Impact:** -1 file, keeps related code together

---

## Not Changing

### Auth Results Rendering (audit.go vs security.go)

The SPF/DKIM/DMARC rendering in `audit.go` and `security.go` looks similar but has intentional formatting differences:
- CLI audit: Shows domain on separate indented line
- TUI security: Shows domain inline in parentheses

Extracting to shared code would require format parameters, adding complexity. Leave as-is.

### `orDefault` in utils.go

Simple 5-line function, but used pattern. Keep for readability.

---

## Execution Order

1. **Delete output package** - Highest impact, no dependencies
2. **Add ScoreStyle** - Quick win
3. **Inline extractHeader** - Minor cleanup
4. **Merge convert.go** - Optional, lowest priority

---

## Verification

After each change:
```bash
go build -o vsb ./cmd/vsb
./vsb inbox list        # Test CLI output styling
./vsb                   # Test TUI renders correctly
```
