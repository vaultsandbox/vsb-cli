# E2E Testing Plan for vsb-cli

## Overview

End-to-end tests for vsb-cli using real VaultSandbox servers and SMTP email sending, following the same patterns established in `client-go`.

## Test Infrastructure

### Directory Structure

```
vsb-cli/
├── e2e/
│   ├── e2e_test.go          # Main test file with TestMain, helpers
│   ├── inbox_test.go        # Inbox command tests
│   ├── email_test.go        # Email command tests
│   ├── wait_test.go         # Wait command tests
│   ├── export_import_test.go # Export/import tests
│   └── config_test.go       # Config command tests
```

### Build Tag

All e2e tests use `//go:build e2e` tag to separate from unit tests.

```go
//go:build e2e

package e2e
```

### Environment Loading

Copy the godotenv approach from client-go:

```go
func TestMain(m *testing.M) {
    // Load .env from project root
    if err := godotenv.Load("../.env"); err != nil {
        fmt.Println("Warning: .env file not found, using environment variables")
    }

    // Validate required credentials
    if os.Getenv("VAULTSANDBOX_API_KEY") == "" || os.Getenv("VAULTSANDBOX_URL") == "" {
        fmt.Println("Skipping e2e tests: VAULTSANDBOX_API_KEY and VAULTSANDBOX_URL required")
        os.Exit(0)
    }

    os.Exit(m.Run())
}
```

### SMTP Utilities (Copy from client-go)

Copy these helper functions from `/home/vs/Desktop/dev/client-go/integration/readme_examples_test.go`:

```go
// getSMTPConfig - Get SMTP host/port from environment
func getSMTPConfig() (host string, port string) {
    host = os.Getenv("SMTP_HOST")
    port = os.Getenv("SMTP_PORT")
    if port == "" {
        port = "25"
    }
    return
}

// skipIfNoSMTP - Skip test if SMTP not configured
func skipIfNoSMTP(t *testing.T) {
    host, _ := getSMTPConfig()
    if host == "" {
        t.Skip("SMTP_HOST not configured, skipping test")
    }
}

// sendTestEmail - Send plain text email
func sendTestEmail(to, subject, body string) error

// sendTestHTMLEmail - Send multipart/alternative email (text + HTML)
func sendTestHTMLEmail(to, subject, textBody, htmlBody string) error

// sendTestEmailWithAttachment - Send email with attachment
func sendTestEmailWithAttachment(to, subject, body, attachmentName string, attachmentContent []byte) error
```

### CLI Execution Helper

```go
// runVSB executes the vsb CLI with given arguments and returns output
func runVSB(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
    // Build command with test config directory
    cmd := exec.Command("./vsb", args...)
    cmd.Env = append(os.Environ(),
        "VSB_API_KEY="+os.Getenv("VAULTSANDBOX_API_KEY"),
        "VSB_BASE_URL="+os.Getenv("VAULTSANDBOX_URL"),
        "VSB_CONFIG_DIR="+t.TempDir(), // Isolated config per test
    )

    var stdoutBuf, stderrBuf bytes.Buffer
    cmd.Stdout = &stdoutBuf
    cmd.Stderr = &stderrBuf

    err := cmd.Run()
    exitCode = 0
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode = exitErr.ExitCode()
    }

    return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// runVSBJSON executes vsb with --output json and parses result
func runVSBJSON[T any](t *testing.T, args ...string) T {
    args = append(args, "--output", "json")
    stdout, _, _ := runVSB(t, args...)
    var result T
    require.NoError(t, json.Unmarshal([]byte(stdout), &result))
    return result
}
```

---

## Test Categories

### 1. Inbox Commands (`inbox_test.go`)

#### TestInboxCreate
- Create inbox with default TTL
- Create inbox with custom TTL (1h, 7d)
- Verify JSON output format
- Verify inbox appears in list

#### TestInboxList
- List when no inboxes exist
- List with multiple inboxes
- List with `--all` flag (shows expired)
- JSON output format

#### TestInboxInfo
- Get info for active inbox
- Get info by email address
- Partial email matching
- JSON output format

#### TestInboxUse
- Switch active inbox
- Partial email matching
- Error on non-existent inbox

#### TestInboxDelete
- Delete from server and local
- Delete local only (`--local`)
- Partial email matching

### 2. Email Commands (`email_test.go`)

#### TestEmailList
- List emails in empty inbox
- List emails after receiving test emails
- JSON output format

#### TestEmailView
- View latest email (no ID)
- View specific email by ID
- Text-only view (`--text`)
- Raw RFC 5322 view (`--raw`)
- JSON output format

#### TestEmailAudit
- Audit email security headers
- SPF/DKIM/DMARC results
- JSON output with auth details

#### TestEmailURL
- Extract URLs from HTML email
- JSON output format
- Handle email with no URLs

#### TestEmailAttachment
- List attachments
- Download single attachment (`--save 1`)
- Download all attachments (`--all`)
- Custom output directory (`--dir`)
- Handle email with no attachments

#### TestEmailDelete
- Delete email by ID
- Error on non-existent ID

### 3. Wait Command (`wait_test.go`)

#### TestWaitBasic
- Wait for any email (send email, verify receives)

#### TestWaitSubject
- Wait with `--subject` exact match
- Wait with `--subject-regex` pattern

#### TestWaitFrom
- Wait with `--from` exact match
- Wait with `--from-regex` pattern

#### TestWaitTimeout
- Verify timeout behavior (short timeout, no email)
- Exit code 1 on timeout

#### TestWaitCount
- Wait for multiple emails (`--count 2`)

#### TestWaitExtractLink
- Extract link from email (`--extract-link`)

#### TestWaitQuiet
- Quiet mode (`--quiet`)
- Exit code only, no output

### 4. Export/Import Commands (`export_import_test.go`)

#### TestExport
- Export active inbox
- Export specific inbox
- Custom output path (`--out`)
- Verify JSON format
- Verify file permissions (0600)

#### TestImport
- Import valid export file
- Import with server verification
- Import with `--local` flag
- Reject expired inboxes
- Reject duplicates (without `--force`)
- Override with `--force`

#### TestExportImportRoundTrip
- Create inbox -> Export -> Delete -> Import -> Verify emails still accessible

### 5. Config Commands (`config_test.go`)

#### TestConfigShow
- Show config with API key masked
- JSON output format

#### TestConfigSet
- Set api-key
- Set base-url
- Invalid key error

---

## Test Execution

### Running Tests

```bash
# Build the CLI first
go build -o vsb ./cmd/vsb

# Run all e2e tests
go test -tags=e2e -v ./e2e/...

# Run specific test file
go test -tags=e2e -v ./e2e/inbox_test.go ./e2e/e2e_test.go

# Run with timeout (recommended: 10m for full suite)
go test -tags=e2e -v -timeout 10m ./e2e/...
```

### Test Script

Create `scripts/test.sh`:

```bash
#!/bin/bash
set -e

# Build CLI
echo "Building vsb..."
go build -o vsb ./cmd/vsb

# Run unit tests
echo "Running unit tests..."
go test ./...

# Run e2e tests if --e2e flag provided
if [[ "$1" == "--e2e" ]]; then
    echo "Running e2e tests..."
    go test -tags=e2e -v -timeout 10m ./e2e/...
fi
```

---

## Implementation Order

### Phase 1: Test Infrastructure
1. Create `e2e/` directory
2. Create `e2e/e2e_test.go` with TestMain, helpers, SMTP utilities
3. Add `github.com/joho/godotenv` dependency

### Phase 2: Core Tests
4. `inbox_test.go` - Inbox lifecycle tests
5. `email_test.go` - Email operations tests

### Phase 3: Advanced Tests
6. `wait_test.go` - CI/CD integration tests
7. `export_import_test.go` - Backup/restore tests
8. `config_test.go` - Configuration tests

### Phase 4: Integration Workflows
9. Add cross-command workflow tests
10. Add error scenario tests

---

## Dependencies to Add

```bash
go get github.com/joho/godotenv
go get github.com/stretchr/testify  # For assertions (require/assert)
```

---

## Test Isolation

Each test should:
1. Use `t.TempDir()` for isolated config directory
2. Create its own inbox (cleanup with `defer`)
3. Not depend on state from other tests
4. Be runnable in parallel where possible

```go
func TestInboxCreate(t *testing.T) {
    // Isolated temp config dir
    configDir := t.TempDir()

    // Create inbox
    stdout, _, code := runVSBWithConfig(t, configDir, "inbox", "create")
    require.Equal(t, 0, code)

    // Extract email from output
    email := extractEmail(stdout)

    // Cleanup
    t.Cleanup(func() {
        runVSBWithConfig(t, configDir, "inbox", "delete", email)
    })

    // Verify inbox exists
    stdout, _, _ = runVSBWithConfig(t, configDir, "inbox", "list", "--output", "json")
    // ... assertions
}
```

---

## Environment Variables

Required in `.env`:
```
VAULTSANDBOX_URL=https://ctxc0a.vsx.email
VAULTSANDBOX_API_KEY=<your-api-key>
SMTP_HOST=ctxc0a.vsx.email
SMTP_PORT=25
```

---

## Notes

- Tests use real VaultSandbox servers (not mocked)
- Each test sends real emails via SMTP
- Inbox cleanup is critical to avoid quota issues
- Tests should be deterministic (no arbitrary sleeps)
- Use the SDK's wait functionality for synchronization
