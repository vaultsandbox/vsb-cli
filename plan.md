# Refactoring Plan

## High Priority

### 1. JSON Output Boilerplate

**Problem:** Same JSON marshaling pattern repeated 50+ times across 11+ CLI files.

**Files affected:**
- `internal/cli/inbox_create.go`
- `internal/cli/inbox_list.go`
- `internal/cli/inbox_info.go`
- `internal/cli/list.go`
- `internal/cli/view.go`
- `internal/cli/audit.go`
- `internal/cli/wait.go`
- `internal/cli/url.go`
- `internal/cli/attachment.go`
- `internal/cli/export.go`
- `internal/cli/import.go`

**Fix:** Create helper in `internal/cli/utils.go`:
```go
func outputJSON(v interface{}) error {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}
```

---

### 2. Duplicated Security Score Calculation

**Problem:** Identical security score logic in two places with different function names.

**Files affected:**
- `internal/cli/audit.go:273-301` - `calculateSecurityScore()`
- `internal/tui/watch/security.go:123-142` - `calculateScore()`

**Fix:** Create shared package `internal/security/score.go`:
```go
package security

func CalculateScore(spf, dkim, dmarc, reverseDNS string) int {
    score := 50
    if spf == "pass" { score += 15 }
    if dkim == "pass" { score += 20 }
    if dmarc == "pass" { score += 10 }
    if reverseDNS == "pass" { score += 5 }
    return score
}

func FormatAuthResult(result string) string {
    // shared formatting logic
}
```

---

### 3. Duplicate Attachment Download Logic

**Problem:** File download with collision detection implemented twice, with inconsistent naming (`name_1.ext` vs `name (1).ext`).

**Files affected:**
- `internal/cli/attachment.go:134-186`
- `internal/tui/watch/attachments.go:77-106`

**Fix:** Create shared `internal/files/download.go`:
```go
package files

func GetUniqueFilename(dir, name string) string { ... }
func SaveFile(dir, name string, data []byte) (string, error) { ... }
```

---

## Medium Priority

### 4. Config Pointer Pattern

**Problem:** Global mutable state via `SetFlagPointers()` creates tight coupling and makes testing difficult.

**File:** `internal/config/config.go:104-106`

**Fix:** Remove `SetFlagPointers()`. Pass output format directly to functions that need it, or use a context-based config.

---

### 5. Unnecessary noop Abstraction

**Problem:** `noop := func() {}` pattern obscures simple cleanup logic.

**File:** `internal/cli/helpers.go:44-76`

**Fix:** Return cleanup function directly or restructure to use `defer`. Example:
```go
// Before
noop := func() {}
if err != nil {
    return nil, noop, err
}

// After
if err != nil {
    return nil, func() {}, err
}
```

---

### 6. Redundant Keystore Lookups

**Problem:** `FindInbox()` loops twice (exact then partial). `SetActiveInbox()` duplicates existence check.

**File:** `internal/config/keystore.go`

**Fix:** Single-pass lookup:
```go
func (k *Keystore) FindInbox(query string) (*StoredInbox, error) {
    var partial *StoredInbox
    for _, inbox := range k.Inboxes {
        if inbox.Email == query {
            return &inbox, nil // exact match, return immediately
        }
        if strings.Contains(inbox.Email, query) && partial == nil {
            partial = &inbox
        }
    }
    if partial != nil {
        return partial, nil
    }
    return nil, fmt.Errorf("inbox not found: %s", query)
}
```

---

## Low Priority

### 7. Unused Styles

**Problem:** Multiple exported styles that are never used.

**File:** `internal/styles/styles.go`

**Unused styles to remove:**
- `EmailItemStyle` / `EmailItemSelectedStyle`
- `SubjectStyle`
- `FromStyle`
- `TimeStyle`
- `InboxLabelStyle`
- `UnreadBadge`
- `PreviewStyle`

---

### 8. Scattered Time Formatting

**Problem:** Time/duration formatting functions scattered across files.

**Files affected:**
- `internal/cli/utils.go:31-40` - `formatDuration()`
- `internal/cli/list.go:128-147` - `formatRelativeTime()`
- `internal/cli/inbox_info.go:114` - inline formatting

**Fix:** Consolidate in `internal/cli/utils.go` or create `internal/timeutil/format.go`.

---

### 9. Overly Defensive Browser Validation

**Problem:** URL scheme whitelist validation is redundant with SDK validation.

**File:** `internal/browser/browser.go:27-36`

**Fix:** Remove or simplify. Trust SDK-provided URLs.

---

## Execution Order

1. JSON output helper (biggest impact, lowest risk)
2. Security score deduplication
3. Attachment download consolidation
4. Remove unused styles
5. Consolidate time formatting
6. Simplify keystore lookups
7. Remove config pointer pattern
8. Remove noop abstraction
9. Simplify browser validation
