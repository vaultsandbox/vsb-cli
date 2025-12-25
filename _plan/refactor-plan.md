# Refactoring Plan

**Date:** 2025-12-25
**Goal:** Clean up codebase before implementing audit, open, view commands

---

## Phase 1: Browser Package (with security fixes)

**Create:** `internal/browser/browser.go`

Move browser code from TUI to shared package, fixing security issues:

| Task | Type | Details |
|------|------|---------|
| 1.1 | New file | Create `internal/browser/browser.go` |
| 1.2 | Security fix | Use `os.CreateTemp()` instead of fixed filename |
| 1.3 | Security fix | Use mode `0600` instead of `0644` |
| 1.4 | Security fix | Whitelist URL schemes (`http`, `https`, `mailto`) |
| 1.5 | Feature | Add `CleanupPreviews()` for old temp files |
| 1.6 | Update | Change `internal/tui/watch/model.go` to import new package |
| 1.7 | Delete | Remove `internal/tui/watch/browser.go` |

**Files:**
```
internal/browser/browser.go    (NEW)
internal/tui/watch/model.go    (UPDATE imports + calls)
internal/tui/watch/browser.go  (DELETE)
```

---

## Phase 2: CLI Helpers

**Create:** `internal/cli/helpers.go`

Extract common patterns into reusable helpers:

| Task | Type | Details |
|------|------|---------|
| 2.1 | New file | Create `internal/cli/helpers.go` |
| 2.2 | Helper | `LoadKeystoreOrError()` - keystore loading |
| 2.3 | Helper | `GetInbox(keystore, emailFlag)` - inbox selection |
| 2.4 | Helper | `GetEmailByIDOrLatest(ctx, args, emailFlag)` - full email fetch |
| 2.5 | Cleanup | Remove or use `config.NewClientWithKeystore()` |

**Helpers to create:**
```go
// LoadKeystoreOrError loads keystore with consistent error message
func LoadKeystoreOrError() (*config.Keystore, error)

// GetInbox returns inbox by email flag or active inbox
func GetInbox(ks *config.Keystore, emailFlag string) (*config.StoredInbox, error)

// GetEmailByIDOrLatest fetches email - by ID if provided, otherwise latest
// This is what audit, open, view all need
func GetEmailByIDOrLatest(ctx context.Context, emailID, emailFlag string) (*vaultsandbox.Email, *vaultsandbox.Inbox, func(), error)
```

---

## Phase 3: Fix waitfor.go (with reliability fix)

**Update:** `internal/cli/waitfor.go`

| Task | Type | Details |
|------|------|---------|
| 3.1 | Bug fix | Remove `os.Exit()` calls, return errors instead |
| 3.2 | Refactor | Use `LoadKeystoreOrError()` helper |
| 3.3 | Refactor | Use `GetInbox()` helper |
| 3.4 | Verify | Ensure `defer client.Close()` runs properly |

**Current problem (line 102):**
```go
os.Exit(1)  // Bypasses defer client.Close()
```

**Fix:**
```go
return fmt.Errorf("timeout waiting for email")
```

---

## Phase 4: Refactor watch.go

**Update:** `internal/cli/watch.go`

| Task | Type | Details |
|------|------|---------|
| 4.1 | Refactor | Use `LoadKeystoreOrError()` helper |
| 4.2 | Refactor | Use `GetInbox()` helper |
| 4.3 | Verify | Browser calls use new `browser` package |

---

## Phase 5: Move Styles Package

**Rename:** `internal/tui/styles/` → `internal/styles/`

| Task | Type | Details |
|------|------|---------|
| 5.1 | Move | `internal/tui/styles/styles.go` → `internal/styles/styles.go` |
| 5.2 | Update | `internal/tui/watch/model.go` imports |
| 5.3 | Update | `internal/cli/inbox_list.go` - use shared styles |
| 5.4 | Update | `internal/cli/inbox_create.go` - use shared styles |
| 5.5 | Delete | Remove empty `internal/tui/styles/` directory |

---

## Phase 6: Verify & Test

| Task | Details |
|------|---------|
| 6.1 | `go build ./...` - ensure compilation |
| 6.2 | `vsb inbox list` - test keystore loading |
| 6.3 | `vsb inbox create` - test client + keystore |
| 6.4 | `vsb watch` - test TUI with new browser package |
| 6.5 | `vsb waitfor --timeout 5s` - test error return (no os.Exit) |

---

## Summary

### New Files
- `internal/browser/browser.go`
- `internal/cli/helpers.go`

### Moved Files
- `internal/tui/styles/` → `internal/styles/`

### Deleted Files
- `internal/tui/watch/browser.go`

### Updated Files
- `internal/cli/waitfor.go` (bug fix + refactor)
- `internal/cli/watch.go` (refactor)
- `internal/cli/inbox_list.go` (styles)
- `internal/cli/inbox_create.go` (styles)
- `internal/tui/watch/model.go` (imports)

### Security Fixes Included
- ✅ Temp file permissions (0600)
- ✅ Unique temp filenames (os.CreateTemp)
- ✅ URL scheme whitelist

### Reliability Fixes Included
- ✅ Remove os.Exit() from waitfor.go

### Deferred (not in scope)
- API key echo in config.go
- Channel reads in model.go
- Config binding mismatch

---

## After Refactoring

Ready to implement:
- `vsb audit` (Phase 3.2)
- `vsb open` (Phase 4.1)
- `vsb view` (Phase 4.1)

Each new command will use helpers and be ~50 lines shorter.
