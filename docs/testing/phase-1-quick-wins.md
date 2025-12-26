# Phase 1: Quick Wins - Pure Function Testing

**Goal**: Add unit tests for all pure functions with zero mocking needed.
**Expected Coverage Gain**: +8-10%
**Effort**: Low

## Overview

Phase 1 targets functions that are pure (no side effects, no external I/O). These are the easiest to test and provide immediate coverage gains.

---

## 1.1 Styles Package

**File to create**: `internal/styles/styles_test.go`

### Functions to Test

#### `ScoreStyle(score int) lipgloss.Style`

Returns appropriate style based on security score boundaries.

```go
func TestScoreStyle(t *testing.T) {
    tests := []struct {
        name     string
        score    int
        wantStyle string // Compare style name or color
    }{
        {"zero score", 0, "fail"},
        {"below threshold", 59, "fail"},
        {"at warn threshold", 60, "warn"},
        {"mid warn range", 70, "warn"},
        {"at pass threshold", 80, "pass"},
        {"perfect score", 100, "pass"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ScoreStyle(tt.score)
            // Assert style matches expected
        })
    }
}
```

#### `FormatAuthResult(result string) string`

Formats SPF/DKIM/DMARC results with color coding.

| Input | Expected Output |
|-------|-----------------|
| `"pass"` | Green "PASS" |
| `"fail"` | Red "FAIL" |
| `"hardfail"` | Red "HARDFAIL" |
| `"softfail"` | Yellow "SOFTFAIL" |
| `"none"` | Gray "NONE" |
| `"neutral"` | Gray "NEUTRAL" |
| `"unknown"` | Gray "UNKNOWN" |
| `"PASS"` | Green "PASS" (case insensitive) |

```go
func TestFormatAuthResult(t *testing.T) {
    tests := []struct {
        input    string
        contains string // Check output contains this
    }{
        {"pass", "PASS"},
        {"PASS", "PASS"},
        {"fail", "FAIL"},
        {"hardfail", "HARDFAIL"},
        {"softfail", "SOFTFAIL"},
        {"none", "NONE"},
        {"neutral", "NEUTRAL"},
        {"unknown", "UNKNOWN"},
        {"garbage", "GARBAGE"}, // Unknown values uppercase
    }
    // ...
}
```

#### `CalculateScore(email *vaultsandbox.Email) int`

Computes security score from auth results.

| Scenario | Expected Score |
|----------|----------------|
| Base (E2E encryption) | 50 |
| + SPF pass | 65 |
| + DKIM pass | 85 |
| + DMARC pass | 95 |
| + ReverseDNS pass | 100 |
| All fail | 50 |
| Nil auth results | 50 |

```go
func TestCalculateScore(t *testing.T) {
    tests := []struct {
        name  string
        email *vaultsandbox.Email
        want  int
    }{
        {
            name:  "nil email",
            email: nil,
            want:  50,
        },
        {
            name: "all pass",
            email: &vaultsandbox.Email{
                AuthResults: &authresults.AuthResults{
                    SPF:        &authresults.SPFResult{Result: "pass"},
                    DKIM:       []authresults.DKIMResult{{Result: "pass"}},
                    DMARC:      &authresults.DMARCResult{Result: "pass"},
                    ReverseDNS: &authresults.ReverseDNSResult{Result: "pass"},
                },
            },
            want: 100,
        },
        // ... more cases
    }
}
```

#### `RenderAuthResults(auth *authresults.AuthResults, labelStyle lipgloss.Style, verbose bool) string`

| Scenario | Test Strategy |
|----------|---------------|
| Nil auth | Returns empty or "N/A" |
| Compact mode | Check format: "SPF: PASS (details)" |
| Verbose mode | Check indented multi-line output |
| Missing fields | Graceful handling |

---

## 1.2 Files Package

**File to create**: `internal/files/download_test.go`

### Functions to Test

#### `GetUniqueFilename(dir, name string) string`

```go
func TestGetUniqueFilename(t *testing.T) {
    t.Run("no collision", func(t *testing.T) {
        dir := t.TempDir()
        got := GetUniqueFilename(dir, "test.txt")
        assert.Equal(t, filepath.Join(dir, "test.txt"), got)
    })

    t.Run("single collision", func(t *testing.T) {
        dir := t.TempDir()
        os.WriteFile(filepath.Join(dir, "test.txt"), []byte("x"), 0644)
        got := GetUniqueFilename(dir, "test.txt")
        assert.Equal(t, filepath.Join(dir, "test_1.txt"), got)
    })

    t.Run("multiple collisions", func(t *testing.T) {
        dir := t.TempDir()
        os.WriteFile(filepath.Join(dir, "test.txt"), []byte("x"), 0644)
        os.WriteFile(filepath.Join(dir, "test_1.txt"), []byte("x"), 0644)
        got := GetUniqueFilename(dir, "test.txt")
        assert.Equal(t, filepath.Join(dir, "test_2.txt"), got)
    })

    t.Run("path traversal prevention", func(t *testing.T) {
        dir := t.TempDir()
        got := GetUniqueFilename(dir, "../../../etc/passwd")
        // Should sanitize to just "passwd"
        assert.Contains(t, got, "passwd")
        assert.NotContains(t, got, "..")
    })

    t.Run("no extension", func(t *testing.T) {
        dir := t.TempDir()
        os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0644)
        got := GetUniqueFilename(dir, "README")
        assert.Equal(t, filepath.Join(dir, "README_1"), got)
    })
}
```

#### `SaveFile(dir, name string, data []byte) (string, error)`

```go
func TestSaveFile(t *testing.T) {
    t.Run("creates directory", func(t *testing.T) {
        base := t.TempDir()
        dir := filepath.Join(base, "new", "nested")
        path, err := SaveFile(dir, "test.txt", []byte("content"))
        require.NoError(t, err)
        assert.FileExists(t, path)
    })

    t.Run("writes content correctly", func(t *testing.T) {
        dir := t.TempDir()
        content := []byte("test content 123")
        path, err := SaveFile(dir, "test.txt", content)
        require.NoError(t, err)
        got, _ := os.ReadFile(path)
        assert.Equal(t, content, got)
    })

    t.Run("handles collision", func(t *testing.T) {
        dir := t.TempDir()
        SaveFile(dir, "test.txt", []byte("first"))
        path, err := SaveFile(dir, "test.txt", []byte("second"))
        require.NoError(t, err)
        assert.Contains(t, path, "test_1.txt")
    })
}
```

---

## 1.3 CLI Utils Package

**File to create**: `internal/cli/utils_test.go`

### Functions to Test

#### `parseTTL(s string) (time.Duration, error)`

```go
func TestParseTTL(t *testing.T) {
    tests := []struct {
        input   string
        want    time.Duration
        wantErr bool
    }{
        // Valid inputs
        {"1h", time.Hour, false},
        {"24h", 24 * time.Hour, false},
        {"7d", 7 * 24 * time.Hour, false},
        {"30m", 30 * time.Minute, false},
        {"1d", 24 * time.Hour, false},

        // Invalid inputs
        {"", 0, true},
        {"x", 0, true},
        {"1y", 0, true},      // Year not supported
        {"-1h", 0, true},     // Negative
        {"1.5h", 0, true},    // Decimal
        {"h", 0, true},       // No number
    }
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            got, err := parseTTL(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.want, got)
            }
        })
    }
}
```

#### `formatDuration(d time.Duration) string`

```go
func TestFormatDuration(t *testing.T) {
    tests := []struct {
        input time.Duration
        want  string
    }{
        {30 * time.Minute, "30m"},
        {59 * time.Minute, "59m"},
        {60 * time.Minute, "1h"},
        {90 * time.Minute, "1h"},
        {23 * time.Hour, "23h"},
        {24 * time.Hour, "1d"},
        {48 * time.Hour, "2d"},
        {7 * 24 * time.Hour, "7d"},
    }
    // ...
}
```

#### `formatRelativeTime(t time.Time) string`

```go
func TestFormatRelativeTime(t *testing.T) {
    now := time.Now()
    tests := []struct {
        name  string
        input time.Time
        want  string
    }{
        {"just now", now.Add(-30 * time.Second), "just now"},
        {"minutes ago", now.Add(-5 * time.Minute), "5m ago"},
        {"hours ago", now.Add(-3 * time.Hour), "3h ago"},
        {"days ago", now.Add(-2 * 24 * time.Hour), "2d ago"},
        {"old date", now.Add(-30 * 24 * time.Hour), ""}, // Falls back to date
    }
    // ...
}
```

#### `truncate(s string, max int) string`

```go
func TestTruncate(t *testing.T) {
    tests := []struct {
        input string
        max   int
        want  string
    }{
        {"short", 10, "short"},
        {"exactly10!", 10, "exactly10!"},
        {"this is too long", 10, "this is..."},
        {"", 5, ""},
        {"abc", 3, "abc"},
        {"abcd", 3, "..."},
    }
    // ...
}
```

#### `sanitizeFilename(email string) string`

```go
func TestSanitizeFilename(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"test@example.com", "test_example_com"},
        {"user.name@domain.co.uk", "user_name_domain_co_uk"},
        {"simple", "simple"},
    }
    // ...
}
```

#### `maskAPIKey(key string) string`

```go
func TestMaskAPIKey(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"sk-ant-1234567890abcdef", "sk-ant-...cdef"},
        {"short", "****"},
        {"12345678901", "1234567...8901"}, // Exactly 11 chars
        {"", "****"},
    }
    // ...
}
```

---

## 1.4 CLI JSON Package

**File to create**: `internal/cli/json_test.go`

### Functions to Test

#### `EmailSummaryJSON(email *vaultsandbox.Email) map[string]interface{}`

```go
func TestEmailSummaryJSON(t *testing.T) {
    email := &vaultsandbox.Email{
        ID:         "msg-123",
        From:       "sender@example.com",
        To:         []string{"recipient@test.com"},
        Subject:    "Test Subject",
        ReceivedAt: time.Now(),
    }

    result := EmailSummaryJSON(email)

    assert.Equal(t, "msg-123", result["id"])
    assert.Equal(t, "sender@example.com", result["from"])
    assert.Equal(t, "Test Subject", result["subject"])
    assert.Contains(t, result, "received_at")
}
```

#### `EmailFullJSON(email *vaultsandbox.Email) map[string]interface{}`

Test includes links and attachments arrays.

#### `InboxSummaryJSON` / `InboxFullJSON`

Test active flag, time formatting, error field inclusion.

---

## Checklist

- [ ] Create `internal/styles/styles_test.go`
- [ ] Create `internal/files/download_test.go`
- [ ] Create `internal/cli/utils_test.go`
- [ ] Create `internal/cli/json_test.go`
- [ ] Run `go test ./internal/styles/... ./internal/files/... ./internal/cli/...`
- [ ] Verify coverage increase with `go test -cover`

## Commands

```bash
# Run Phase 1 tests
go test -v ./internal/styles/... ./internal/files/...

# Check coverage for these packages
go test -coverprofile=phase1.out ./internal/styles/... ./internal/files/...
go tool cover -func=phase1.out
```
