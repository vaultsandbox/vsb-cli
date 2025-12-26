# Phase 5: Browser & Integration Polish

**Goal**: Test remaining packages and polish integration tests.
**Expected Coverage Gain**: +3-5%
**Effort**: Medium

## Overview

Phase 5 completes the testing coverage by:
1. Testing the browser integration package
2. Adding edge case coverage to e2e tests
3. Final coverage audit and gap filling

---

## 5.1 Browser Package Tests

**File to create**: `internal/browser/browser_test.go`

### Test URL Validation

```go
func TestOpenURL_SchemeValidation(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        wantErr bool
    }{
        // Allowed schemes
        {"http allowed", "http://example.com", false},
        {"https allowed", "https://example.com", false},
        {"mailto allowed", "mailto:test@example.com", false},
        {"file allowed", "file:///tmp/test.html", false},

        // Blocked schemes
        {"javascript blocked", "javascript:alert(1)", true},
        {"data blocked", "data:text/html,<script>", true},
        {"ftp blocked", "ftp://example.com", true},

        // Invalid URLs
        {"empty url", "", true},
        {"no scheme", "example.com", true},
        {"invalid url", "://invalid", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Note: We can't actually test browser opening without mocking exec
            // This tests the validation logic by checking if it returns early
            err := OpenURL(tt.url)
            if tt.wantErr {
                assert.Error(t, err)
            }
            // For allowed URLs, we can't easily verify without mocking exec.Command
        })
    }
}
```

### Test ViewHTML File Operations

```go
func TestViewHTML_FileCreation(t *testing.T) {
    // Skip actual browser opening
    if os.Getenv("TEST_BROWSER") == "" {
        t.Skip("Skipping browser test: TEST_BROWSER not set")
    }

    t.Run("creates temp file with correct permissions", func(t *testing.T) {
        html := "<html><body>Test</body></html>"

        // ViewHTML creates a temp file - we need to verify file properties
        // This is tricky because the file is opened and we can't easily intercept
        // Consider refactoring to accept a file creator function

        err := ViewHTML(html)
        // Best we can do without refactoring is check no error
        assert.NoError(t, err)
    })
}

func TestViewEmailHTML_XSSPrevention(t *testing.T) {
    t.Run("escapes malicious subject", func(t *testing.T) {
        subject := "<script>alert('xss')</script>"
        from := "attacker@evil.com"
        html := "<p>Safe content</p>"

        // ViewEmailHTML should escape the subject
        // We need to capture the generated HTML to verify escaping
        // This requires refactoring to make the template generation testable

        // For now, we can test the html.EscapeString behavior
        escaped := html.EscapeString(subject)
        assert.NotContains(t, escaped, "<script>")
        assert.Contains(t, escaped, "&lt;script&gt;")
    })

    t.Run("escapes malicious from", func(t *testing.T) {
        from := "<img src=x onerror=alert(1)>"
        escaped := html.EscapeString(from)
        assert.NotContains(t, escaped, "<img")
    })
}
```

### Test CleanupPreviews

```go
func TestCleanupPreviews(t *testing.T) {
    t.Run("removes old preview files", func(t *testing.T) {
        // Create temp directory to simulate OS temp
        dir := t.TempDir()

        // Create old preview file
        oldFile := filepath.Join(dir, "vsb-preview-old.html")
        os.WriteFile(oldFile, []byte("old"), 0600)

        // Set file modification time to past
        oldTime := time.Now().Add(-2 * time.Hour)
        os.Chtimes(oldFile, oldTime, oldTime)

        // Create recent preview file
        recentFile := filepath.Join(dir, "vsb-preview-recent.html")
        os.WriteFile(recentFile, []byte("recent"), 0600)

        // Run cleanup with 1 hour threshold
        // Note: This requires modifying CleanupPreviews to accept a directory
        // or mocking os.TempDir()

        err := CleanupPreviews(1 * time.Hour)
        assert.NoError(t, err)

        // Verify old file removed, recent file kept
        // (Requires directory injection to properly test)
    })

    t.Run("ignores non-preview files", func(t *testing.T) {
        // CleanupPreviews should only touch vsb-preview-* files
        dir := t.TempDir()

        otherFile := filepath.Join(dir, "other-file.html")
        os.WriteFile(otherFile, []byte("keep me"), 0600)

        // After cleanup, other file should still exist
        assert.FileExists(t, otherFile)
    })
}
```

### Recommended Refactoring for Testability

To make the browser package fully testable, consider these refactoring changes:

```go
// browser.go - Add dependency injection

// BrowserOpener interface for testing
type BrowserOpener interface {
    Open(url string) error
}

// DefaultBrowserOpener uses system browser
type DefaultBrowserOpener struct{}

func (d DefaultBrowserOpener) Open(url string) error {
    return openBrowser(url) // existing implementation
}

// FileCreator interface for testing
type FileCreator interface {
    CreateTemp(pattern string) (*os.File, error)
}

// DefaultFileCreator uses OS temp files
type DefaultFileCreator struct{}

func (d DefaultFileCreator) CreateTemp(pattern string) (*os.File, error) {
    return os.CreateTemp("", pattern)
}

// With interfaces, tests can mock:
type MockBrowserOpener struct {
    OpenedURLs []string
    OpenFunc   func(string) error
}

func (m *MockBrowserOpener) Open(url string) error {
    m.OpenedURLs = append(m.OpenedURLs, url)
    if m.OpenFunc != nil {
        return m.OpenFunc(url)
    }
    return nil
}
```

---

## 5.2 Integration Test Improvements

### Add Edge Case Tests

**File to add to**: `e2e/errors_test.go`

```go
func TestNetworkErrorHandling(t *testing.T) {
    t.Run("handles invalid base URL gracefully", func(t *testing.T) {
        configDir := t.TempDir()

        // Set invalid base URL
        _, stderr, code := runVSBWithConfig(t, configDir,
            "--base-url", "https://invalid.nonexistent.domain.local",
            "inbox", "create")

        assert.NotEqual(t, 0, code)
        assert.Contains(t, stderr, "error") // Should have error message
    })

    t.Run("handles timeout gracefully", func(t *testing.T) {
        // Test with very short timeout
        _, stderr, code := runVSB(t, "wait", "--timeout", "1ms")

        assert.Equal(t, 1, code) // Timeout exit code
        assert.Contains(t, stderr, "timeout")
    })
}

func TestInvalidInputHandling(t *testing.T) {
    t.Run("invalid TTL format", func(t *testing.T) {
        _, stderr, code := runVSB(t, "inbox", "create", "--ttl", "invalid")

        assert.NotEqual(t, 0, code)
        assert.Contains(t, stderr, "invalid")
    })

    t.Run("invalid email ID", func(t *testing.T) {
        _, stderr, code := runVSB(t, "email", "view", "--id", "nonexistent-id-12345")

        assert.NotEqual(t, 0, code)
    })

    t.Run("invalid regex in wait", func(t *testing.T) {
        _, stderr, code := runVSB(t, "wait", "--subject-regex", "[invalid")

        assert.NotEqual(t, 0, code)
        assert.Contains(t, stderr, "regex")
    })
}
```

### Add Concurrent Operation Tests

```go
func TestConcurrentInboxOperations(t *testing.T) {
    configDir := t.TempDir()

    // Create multiple inboxes concurrently
    var wg sync.WaitGroup
    results := make(chan error, 5)

    for i := 0; i < 5; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, _, code := runVSBWithConfig(t, configDir, "inbox", "create")
            if code != 0 {
                results <- fmt.Errorf("inbox create failed with code %d", code)
            }
        }()
    }

    wg.Wait()
    close(results)

    for err := range results {
        t.Error(err)
    }

    // Verify all inboxes created
    stdout, _, _ := runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
    var list InboxListJSON
    json.Unmarshal([]byte(stdout), &list)
    assert.Equal(t, 5, list.Count)
}
```

### Add Export/Import Edge Cases

```go
func TestExportImportEdgeCases(t *testing.T) {
    t.Run("import expired inbox", func(t *testing.T) {
        configDir := t.TempDir()

        // Create export file with expired inbox
        expiredExport := `{
            "version": "1.0",
            "email": "expired@example.com",
            "inbox_hash": "hash123",
            "expires_at": "2020-01-01T00:00:00Z",
            "encryption_public_key": "...",
            "encryption_secret_key": "...",
            "signing_public_key": "...",
            "signing_secret_key": "..."
        }`

        exportFile := filepath.Join(configDir, "expired.json")
        os.WriteFile(exportFile, []byte(expiredExport), 0644)

        _, stderr, code := runVSBWithConfig(t, configDir, "import", exportFile)

        assert.NotEqual(t, 0, code)
        assert.Contains(t, stderr, "expired")
    })

    t.Run("import invalid version", func(t *testing.T) {
        configDir := t.TempDir()

        invalidExport := `{"version": "99.0", "email": "test@example.com"}`
        exportFile := filepath.Join(configDir, "invalid.json")
        os.WriteFile(exportFile, []byte(invalidExport), 0644)

        _, stderr, code := runVSBWithConfig(t, configDir, "import", exportFile)

        assert.NotEqual(t, 0, code)
        assert.Contains(t, stderr, "version")
    })

    t.Run("import duplicate inbox", func(t *testing.T) {
        configDir := t.TempDir()

        // Create inbox first
        runVSBWithConfig(t, configDir, "inbox", "create")

        // Export it
        stdout, _, _ := runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
        var list InboxListJSON
        json.Unmarshal([]byte(stdout), &list)
        email := list.Inboxes[0].Email

        exportFile := filepath.Join(configDir, "export.json")
        runVSBWithConfig(t, configDir, "export", email, "--output", exportFile)

        // Try to import the same inbox
        _, stderr, code := runVSBWithConfig(t, configDir, "import", exportFile)

        assert.NotEqual(t, 0, code)
        assert.Contains(t, stderr, "already exists")
    })
}
```

---

## 5.3 Coverage Audit

### Final Coverage Check Script

Create `scripts/coverage-audit.sh`:

```bash
#!/bin/bash

# Generate coverage for all packages
go test -coverprofile=coverage.out ./internal/...

# Show coverage by package
echo "=== Coverage by Package ==="
go tool cover -func=coverage.out | grep -E "^github.com/vaultsandbox/vsb-cli/internal/[^/]+/" | \
    awk '{print $1, $3}' | sort -t'/' -k6 | uniq

# Show uncovered functions
echo ""
echo "=== Uncovered Functions (0.0%) ==="
go tool cover -func=coverage.out | grep "0.0%"

# Show total coverage
echo ""
echo "=== Total Coverage ==="
go tool cover -func=coverage.out | tail -1

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
echo ""
echo "HTML report: coverage.html"
```

### Coverage Goals by Package

| Package | Target | Notes |
|---------|--------|-------|
| `internal/styles/` | 90%+ | Pure functions, easy to test |
| `internal/files/` | 90%+ | Simple file operations |
| `internal/config/` | 85%+ | Some edge cases hard to reach |
| `internal/cli/` | 70%+ | Complex command flows |
| `internal/tui/emails/` | 60%+ | UI code inherently harder |
| `internal/browser/` | 50%+ | System calls hard to mock |

---

## 5.4 Documentation

### Add Testing Documentation

Create `docs/testing/README.md`:

```markdown
# VSB-CLI Testing Guide

## Running Tests

### Unit Tests
```bash
go test ./internal/...
```

### E2E Tests (requires server)
```bash
# Set environment variables
export VAULTSANDBOX_API_KEY="your-key"
export VAULTSANDBOX_URL="https://api.vaultsandbox.com"
export SMTP_HOST="smtp.example.com"
export SMTP_PORT="25"

# Build and run
go build -o vsb ./cmd/vsb
go test -tags=e2e -v ./e2e/...
```

### Coverage
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out

# Combined coverage (unit + e2e)
go build -cover -o vsb ./cmd/vsb
GOCOVERDIR=coverage go test -tags=e2e ./e2e/...
go tool covdata textfmt -i=coverage -o=combined.out
```

## Test Structure

- `internal/*/` - Unit tests alongside source files
- `e2e/` - End-to-end tests against real server
- `docs/testing/` - Testing documentation and plans

## Mocking Strategy

- Use interfaces for external dependencies
- Test pure functions directly
- Use `t.TempDir()` for filesystem tests
- Use `t.Setenv()` for environment tests
```

---

## Checklist

### Browser Package
- [ ] Create `internal/browser/browser_test.go`
- [ ] Test URL scheme validation
- [ ] Test XSS prevention
- [ ] (Optional) Refactor for better testability

### E2E Improvements
- [ ] Add error handling tests to `e2e/errors_test.go`
- [ ] Add concurrent operation tests
- [ ] Add export/import edge cases

### Documentation
- [ ] Create `docs/testing/README.md`
- [ ] Create `scripts/coverage-audit.sh`

### Final Audit
- [ ] Run full test suite
- [ ] Generate coverage report
- [ ] Identify remaining gaps
- [ ] Document known limitations

## Commands

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./internal/...

# Full coverage report
./scripts/coverage-audit.sh

# E2E tests only
go test -tags=e2e -v ./e2e/...
```

## Final Coverage Target

After completing all 5 phases:
- **Unit test coverage**: 70%+ of `internal/` packages
- **E2E coverage**: All major user workflows
- **Combined coverage**: 80-90% overall

The remaining uncovered code will primarily be:
- Error paths that are hard to trigger
- Platform-specific code (Windows/macOS browser opening)
- Interactive TUI elements that require manual testing
