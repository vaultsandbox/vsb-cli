# CLI Unit Test Coverage Plan

**Goal**: Increase `internal/cli` test coverage with meaningful tests (not testing for coverage's sake)

**Philosophy**: Test pure functions and keystore logic. Client/SDK interactions are covered by E2E tests.

---

## Phase 1: Pure Functions (No Mocks Needed)

These have no external dependencies and real edge cases worth testing.

### 1.1 `inbox_create_test.go` - `parseTTL` tests

```go
func TestParseTTL(t *testing.T) {
    tests := []struct {
        input    string
        expected time.Duration
        wantErr  bool
    }{
        {"1h", time.Hour, false},
        {"24h", 24 * time.Hour, false},
        {"7d", 7 * 24 * time.Hour, false},
        {"30d", 30 * 24 * time.Hour, false},
        {"invalid", 0, true},
        {"d", 0, true},
        {"5x", 0, true},
    }
}
```

### 1.2 `export_test.go` - `sanitizeFilename` tests

```go
func TestSanitizeFilename(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"test@example.com", "test@example.com"},
        {"test/bad@example.com", "test_bad@example.com"},
        {"a:b:c@example.com", "a_b_c@example.com"},
    }
}
```

### 1.3 `import_test.go` - Validation logic

Test the validation that happens before any client call:

```go
func TestImportValidation(t *testing.T) {
    t.Run("rejects unsupported version", ...)
    t.Run("rejects expired inbox", ...)
    t.Run("rejects malformed JSON", ...)
}
```

---

## Phase 2: Keystore-Only Logic

Uses existing `MockKeystore` - no new abstractions needed.

### 2.1 `inbox_list_test.go` - Filtering logic

Extract and test:
```go
func filterInboxes(inboxes []config.StoredInbox, showExpired bool, now time.Time) []config.StoredInbox
```

```go
func TestFilterInboxes(t *testing.T) {
    t.Run("hides expired by default", ...)
    t.Run("shows expired with --all flag", ...)
    t.Run("returns empty for no inboxes", ...)
}
```

### 2.2 `inbox_use_test.go` - Active inbox switching

Extend `MockKeystore` with `SetActiveInbox`:
```go
func TestInboxUse(t *testing.T) {
    t.Run("sets active inbox by exact match", ...)
    t.Run("sets active inbox by partial match", ...)
    t.Run("errors on no match", ...)
    t.Run("errors on multiple matches", ...)
}
```

### 2.3 `export_test.go` - Export path logic

Test file path generation (no actual file I/O):
```go
func TestExportPathGeneration(t *testing.T) {
    t.Run("uses email as default filename", ...)
    t.Run("respects --out flag", ...)
}
```

---

## What We're NOT Testing (and why)

| Skip | Reason |
|------|--------|
| `runInboxCreate` full flow | E2E covers it, would require client mocking |
| `runInboxDelete` full flow | E2E covers it, thin wrapper around SDK |
| `runImport` server verification | E2E covers it, SDK should test its own import |
| Pretty-print output | Low value, visual changes break tests |

---

## Files to Create

| File | Tests |
|------|-------|
| `inbox_create_test.go` | `TestParseTTL` |
| `inbox_list_test.go` | `TestFilterInboxes` |
| `inbox_use_test.go` | Active inbox tests |
| `export_test.go` | `TestSanitizeFilename`, path tests |
| `import_test.go` | Validation tests |

## Files to Modify

| File | Change |
|------|--------|
| `helpers_test.go` | Add `SetActiveInbox` to `MockKeystore` |
| `inbox_list.go` | Extract `filterInboxes()` function |

---

## Implementation Order

1. `TestParseTTL` - 10 min
2. `TestSanitizeFilename` - 10 min
3. `TestFilterInboxes` - 15 min (includes extraction)
4. `TestImportValidation` - 15 min
5. `TestInboxUse` - 15 min (includes MockKeystore update)
6. Export path tests - 10 min

---

## Success Criteria

- [ ] `go test ./internal/cli/...` passes
- [ ] Pure functions have table-driven tests
- [ ] No refactoring of working command code
- [ ] No new abstractions/interfaces added
