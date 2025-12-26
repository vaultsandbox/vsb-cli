# VSB-CLI Testing Guide

This document provides comprehensive guidance for testing vsb-cli.

## Running Tests

### Unit Tests

Run all unit tests for internal packages:

```bash
go test ./internal/...
```

Run tests with verbose output:

```bash
go test -v ./internal/...
```

Run tests for a specific package:

```bash
go test -v ./internal/browser/...
go test -v ./internal/config/...
go test -v ./internal/cli/...
```

### E2E Tests

End-to-end tests require a running VaultSandbox Gateway and SMTP server.

#### Required Environment Variables

```bash
# Required for all e2e tests
export VAULTSANDBOX_API_KEY="your-api-key"
export VAULTSANDBOX_URL="https://api.vaultsandbox.com"

# Required for email-related tests
export SMTP_HOST="smtp.example.com"
export SMTP_PORT="25"
```

#### Running E2E Tests

```bash
# Build the binary first
go build -o vsb ./cmd/vsb

# Run e2e tests
go test -tags=e2e -v -timeout 10m ./e2e/...
```

### Race Detection

Run tests with race detection to find concurrency issues:

```bash
go test -race ./internal/...
```

## Coverage

### Quick Coverage Check

```bash
go test -cover ./internal/...
```

### Detailed Coverage Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./internal/...

# View coverage summary
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Audit Script

Use the coverage audit script for comprehensive coverage analysis:

```bash
./scripts/coverage-audit.sh
```

This script:
- Generates coverage for all internal packages
- Shows coverage by package
- Lists uncovered functions
- Compares against coverage targets
- Generates an HTML report

### Combined Coverage (Unit + E2E)

To measure combined coverage from unit and e2e tests:

```bash
# Build with coverage instrumentation
go build -cover -o vsb ./cmd/vsb

# Create coverage directory
mkdir -p coverage

# Run e2e tests with coverage collection
GOCOVERDIR=coverage go test -tags=e2e ./e2e/...

# Convert to text format
go tool covdata textfmt -i=coverage -o=combined.out

# Generate HTML report
go tool cover -html=combined.out -o combined.html
```

## Test Structure

```
vsb-cli/
├── internal/
│   ├── browser/
│   │   ├── browser.go
│   │   └── browser_test.go    # URL validation, XSS prevention tests
│   ├── cli/
│   │   ├── *.go
│   │   └── *_test.go          # Command-specific tests
│   ├── config/
│   │   ├── config.go
│   │   ├── config_test.go     # Configuration loading tests
│   │   ├── keystore.go
│   │   └── keystore_test.go   # Encrypted storage tests
│   ├── files/
│   │   └── files_test.go      # File operation tests
│   ├── styles/
│   │   └── styles_test.go     # Styling constant tests
│   └── tui/
│       └── *_test.go          # TUI component tests
├── e2e/
│   ├── e2e_test.go            # Shared test helpers
│   ├── inbox_test.go          # Inbox command tests
│   ├── email_test.go          # Email command tests
│   ├── wait_test.go           # Wait command tests
│   ├── config_test.go         # Config command tests
│   ├── export_import_test.go  # Export/import tests
│   ├── workflow_test.go       # Full workflow tests
│   └── errors_test.go         # Error handling tests
└── docs/testing/
    ├── README.md              # This file
    └── phase-*.md             # Testing implementation phases
```

## Mocking Strategy

### Interfaces for External Dependencies

The codebase uses interfaces to allow mocking:

- `vaultsandbox.Client` - SDK client for API calls
- `http.Client` - For HTTP requests
- File system operations - Use `t.TempDir()` for isolation

### Testing Pure Functions

Many functions are pure and can be tested directly:

```go
func TestParseEmail(t *testing.T) {
    result := parseEmail("test@example.com")
    assert.Equal(t, "test", result.Local)
    assert.Equal(t, "example.com", result.Domain)
}
```

### Filesystem Tests

Use `t.TempDir()` for filesystem isolation:

```go
func TestFileWrite(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "test.txt")

    err := writeFile(path, "content")
    assert.NoError(t, err)

    data, _ := os.ReadFile(path)
    assert.Equal(t, "content", string(data))
}
```

### Environment Variable Tests

Use `t.Setenv()` for environment variable tests:

```go
func TestEnvConfig(t *testing.T) {
    t.Setenv("VSB_API_KEY", "test-key")

    config := LoadConfig()
    assert.Equal(t, "test-key", config.APIKey)
}
```

## Coverage Targets by Package

| Package | Target | Notes |
|---------|--------|-------|
| `internal/styles/` | 90%+ | Pure functions, easy to test |
| `internal/files/` | 90%+ | Simple file operations |
| `internal/config/` | 85%+ | Some edge cases hard to reach |
| `internal/cli/` | 70%+ | Complex command flows |
| `internal/tui/` | 60%+ | UI code inherently harder |
| `internal/browser/` | 50%+ | System calls hard to mock |

## Known Limitations

Some code paths are difficult to test automatically:

1. **Platform-specific code** - Browser opening on Windows/macOS
2. **Interactive TUI elements** - Keyboard input, screen rendering
3. **Network error paths** - Specific timeout/connection error scenarios
4. **External system integration** - Real SMTP, browser processes

## Continuous Integration

### Pre-commit Checks

```bash
# Run all checks before committing
go test ./internal/...
go vet ./...
golangci-lint run
```

### CI Pipeline

```bash
# Full CI test suite
go build -o vsb ./cmd/vsb
go test -race -coverprofile=coverage.out ./internal/...
go test -tags=e2e -v -timeout 10m ./e2e/...
```

## Writing New Tests

### Test File Naming

- Unit tests: `*_test.go` in the same package
- E2E tests: `*_test.go` in the `e2e/` directory with `//go:build e2e` tag

### Test Function Naming

```go
// Unit tests
func TestFunctionName(t *testing.T) {}
func TestFunctionName_SpecificCase(t *testing.T) {}

// Table-driven tests
func TestFunctionName(t *testing.T) {
    tests := []struct{
        name string
        // ...
    }{}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### Using testify

We use the testify library for assertions:

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    // Use assert for non-fatal checks
    assert.Equal(t, expected, actual)
    assert.NoError(t, err)
    assert.Contains(t, output, "expected")

    // Use require for fatal checks (stops test on failure)
    require.NoError(t, err)
    require.NotNil(t, result)
}
```
