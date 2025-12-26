# Code Simplification Plan

## 1. Duplicated Code

### 1.1 Authentication Rendering (High Impact)

The SPF/DKIM/DMARC rendering logic is nearly identical in two places:

**`internal/cli/audit.go:101-145`**
```go
if auth.SPF != nil {
    spfResult := styles.FormatAuthResult(auth.SPF.Status)
    fmt.Printf("%s %s\n", labelStyle.Render("SPF:"), spfResult)
    if auth.SPF.Domain != "" {
        fmt.Printf("%s %s\n", labelStyle.Render("  Domain:"), auth.SPF.Domain)
    }
}
// ... identical pattern for DKIM, DMARC, ReverseDNS
```

**`internal/tui/watch/security.go:32-74`**
```go
if auth.SPF != nil {
    spfResult := styles.FormatAuthResult(auth.SPF.Status)
    sb.WriteString(fmt.Sprintf("%s %s", labelStyle.Render("SPF:"), spfResult))
    if auth.SPF.Domain != "" {
        sb.WriteString(fmt.Sprintf(" (%s)", auth.SPF.Domain))
    }
    sb.WriteString("\n")
}
// ... identical pattern for DKIM, DMARC, ReverseDNS
```

**Suggestion:** Add a `RenderAuthResults(auth *vaultsandbox.AuthResults) string` function to the `styles` package:

```go
func RenderAuthResults(auth *vaultsandbox.AuthResults, labelWidth int) string {
    if auth == nil {
        return WarnStyle.Render("No authentication results")
    }
    var lines []string
    label := lipgloss.NewStyle().Bold(true).Width(labelWidth)

    if auth.SPF != nil {
        lines = append(lines, fmt.Sprintf("%s %s", label.Render("SPF:"),
            FormatAuthResult(auth.SPF.Status)))
    }
    // ... etc
    return strings.Join(lines, "\n")
}
```

---

### 1.2 Size Formatting (Medium Impact)

Two different implementations for the same thing:

**`internal/tui/watch/attachments.go:65-73`** - Custom implementation:
```go
func formatSize(bytes int) string {
    if bytes < 1024 {
        return fmt.Sprintf("%d B", bytes)
    }
    if bytes < 1024*1024 {
        return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
    }
    return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
```

**`internal/cli/attachment.go:120`** - Uses library:
```go
fmt.Printf("     Size: %s\n", humanize.Bytes(uint64(att.Size)))
```

**Suggestion:** Just use `humanize.Bytes()` everywhere (already a dependency). Delete `formatSize()` in attachments.go.

---

### 1.3 Transport Security Hardcoded Constants (Low Impact)

The same encryption strings appear in 4 places:

| File | Line | String |
|------|------|--------|
| `audit.go` | 164 | `"ML-KEM-768 + AES-256-GCM"` |
| `security.go` | 85 | `"ML-KEM-768 + AES-256-GCM"` |
| `inbox_info.go` | 156 | `"ML-KEM-768 (Quantum-Safe)"` |
| `inbox_create.go` | 117 | `"ML-KEM-768 (Quantum-Safe)"` |

**Suggestion:** Add constant to `styles/styles.go`:
```go
const EncryptionLabel = "ML-KEM-768 (Quantum-Safe)"
```

---

### 1.4 Empty Results Pattern (Low Impact)

This pattern appears twice identically:

**`internal/cli/url.go:61-68` and `attachment.go:71-78`**
```go
if len(email.Links) == 0 {
    if getOutput(cmd) == "json" {
        fmt.Println("[]")
    } else {
        fmt.Println("No URLs found in email")
    }
    return nil
}
```

**Suggestion:** This is minor. Could add a helper but probably not worth it for 2 occurrences.

---

### 1.5 StoredInbox <-> ExportedInboxFile Conversion

**`internal/cli/export.go:90-101`** manually constructs `ExportedInboxFile`:
```go
exportData := config.ExportedInboxFile{
    Version:      1,
    EmailAddress: stored.Email,
    InboxHash:    stored.ID,
    ExpiresAt:    stored.ExpiresAt,
    // ... more fields
}
```

**`internal/cli/import.go:122-132`** manually constructs `StoredInbox`:
```go
stored := config.StoredInbox{
    Email:     exported.EmailAddress,
    ID:        exported.InboxHash,
    // ... more fields
}
```

**Suggestion:** Add conversion methods in `config/keystore.go`:
```go
func (s *StoredInbox) ToExportFile() ExportedInboxFile {
    return ExportedInboxFile{
        Version:      1,
        EmailAddress: s.Email,
        // ...
    }
}

func (e *ExportedInboxFile) ToStoredInbox() StoredInbox {
    return StoredInbox{
        Email: e.EmailAddress,
        // ...
    }
}
```

---

## 2. Over-Engineered / Unnecessary

### 2.1 Tiny Package: `internal/security/`

The entire `internal/security/` package is one 38-line file with a single function:

**`internal/security/score.go`**
```go
func CalculateScore(email *vaultsandbox.Email) int {
    score := 50
    if email.AuthResults == nil { return score }
    if auth.SPF != nil && strings.EqualFold(auth.SPF.Status, "pass") { score += 15 }
    // ... 4 more checks
    return score
}
```

**Used in only 2 places:**
- `cli/audit.go:175`
- `tui/watch/security.go:91`

**Suggestion:** Move to `styles/styles.go` alongside `ScoreStyle()` which depends on it:
```go
// styles/styles.go
func CalculateScore(email *vaultsandbox.Email) int { ... }
func ScoreStyle(score int) lipgloss.Style { ... }
```

Then delete the `internal/security/` package entirely.

---

### 2.2 Separate `config/export.go` File

**`internal/config/export.go`** is only 20 lines:
```go
type ExportedInboxFile struct { ... }
type ExportedKeys struct { ... }
```

These types are tightly coupled to `StoredInbox` in keystore.go.

**Suggestion:** Merge into `keystore.go`. Having a separate file for 2 type definitions adds navigation overhead with no benefit.

---

### 2.3 LoadKeystoreOrError Helper (Borderline)

**`internal/cli/helpers.go:12-18`**
```go
func LoadKeystoreOrError() (*config.Keystore, error) {
    ks, err := config.LoadKeystore()
    if err != nil {
        return nil, fmt.Errorf("failed to load keystore: %w", err)
    }
    return ks, nil
}
```

This wraps one function call to add an error message prefix. It's used 6 times.

**Verdict:** Keep it - the consistent error message is worth the small wrapper.

---

## 3. Security & Reliability Fixes

### 3.1 Path Traversal in File Downloads (Critical)

**`internal/files/download.go:11-12`** is vulnerable to path traversal:
```go
func GetUniqueFilename(dir, name string) string {
    path := filepath.Join(dir, name)  // VULNERABLE
```

A malicious email attachment named `../../.bashrc` could overwrite files outside the target directory.

**Fix:** Sanitize filename with `filepath.Base()`:
```go
func GetUniqueFilename(dir, name string) string {
    cleanName := filepath.Base(name)  // strips "../" and directory components
    path := filepath.Join(dir, cleanName)
```

---

### 3.2 TUI Email Ordering Race Condition (Low)

**`internal/tui/watch/model.go`** runs `LoadExistingEmails()` and `WatchEmails()` concurrently. If `LoadExistingEmails` is slow, older emails might arrive after newer ones from `WatchEmails`, causing incorrect display order.

**Current code in `Init()`:**
```go
func (m Model) Init() tea.Cmd {
    return tea.Batch(
        m.startWatching(),  // both run in parallel
    )
}
```

**Fix:** Load existing emails first, then start watching:
```go
func (m Model) Init() tea.Cmd {
    return m.loadExistingThenWatch()
}

func (m *Model) loadExistingThenWatch() tea.Cmd {
    return func() tea.Msg {
        // Load existing emails synchronously first
        for _, inbox := range m.inboxes {
            emails, err := inbox.GetEmails(m.ctx)
            if err != nil {
                continue
            }
            for _, email := range emails {
                // Add directly to m.emails (sorted by ReceivedAt)
            }
        }
        return connectedMsg{}  // then start watching
    }
}
```

This ensures existing emails are loaded before the SSE watcher starts, avoiding ordering issues.

---

## 4. Summary

| Priority | Change | Impact |
|----------|--------|--------|
| **Critical** | Fix path traversal in `files/download.go` | Security fix |
| High | Extract `RenderAuthResults()` to styles | ~40 lines saved |
| Medium | Use `humanize.Bytes()` in TUI | ~10 lines saved |
| Medium | Add `ToExportFile()`/`ToStoredInbox()` methods | ~20 lines saved |
| Low | Fix TUI email ordering (sequential load) | Reliability |
| Low | Move `security.CalculateScore` to styles | File delete |
| Low | Merge `config/export.go` into `keystore.go` | File delete |
| Low | Add encryption constant | ~3 lines saved |

---

## 5. What's Already Good (Don't Change)

- **`helpers.go`** - Well-designed abstractions (`LoadAndImportInbox`, `GetEmailByIDOrLatest`)
- **Thread-safe keystore** - Proper mutex usage
- **`styles/` package** - Clean separation of presentation
- **Config priority chain** - Correctly implemented (flag > env > file > default)
