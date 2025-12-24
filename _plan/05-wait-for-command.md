# Phase 3.1: Wait-For Command (CI/CD)

## Objective
Implement `vsb wait-for` - a blocking command designed for CI/CD pipelines that waits for a matching email and exits with appropriate codes.

## Command

| Command | Description |
|---------|-------------|
| `vsb wait-for` | Wait for any email on active inbox |
| `vsb wait-for --subject "..." ` | Wait for email with matching subject |
| `vsb wait-for --from "..." ` | Wait for email from specific sender |
| `vsb wait-for --timeout 60s` | Set wait timeout |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Email found matching criteria |
| 1 | Timeout - no matching email |
| 2 | Configuration error |

## Tasks

### 1. Wait-For Command

**File: `internal/cli/waitfor.go`**

```go
package cli

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "regexp"
    "time"

    "github.com/spf13/cobra"
    vaultsandbox "github.com/vaultsandbox/client-go"
    "github.com/vaultsandbox/vsb-cli/internal/config"
    "github.com/vaultsandbox/vsb-cli/internal/output"
)

var waitForCmd = &cobra.Command{
    Use:   "wait-for",
    Short: "Wait for an email matching criteria (CI/CD)",
    Long: `Block until an email matching the specified criteria arrives.

Designed for CI/CD pipelines and automated testing. Returns exit code 0
when a matching email is found, 1 on timeout.

Filter Options:
  --subject       Exact subject match
  --subject-regex Subject regex pattern
  --from          Exact sender match
  --from-regex    Sender regex pattern

Output Options:
  --json          Output matching email as JSON
  --quiet         No output, just exit code
  --extract-link  Output first link from email body

Examples:
  # Wait for any email
  vsb wait-for

  # Wait for password reset email
  vsb wait-for --subject-regex "password reset" --timeout 30s

  # Extract verification link
  LINK=$(vsb wait-for --subject "Verify" --extract-link)

  # JSON output for parsing
  vsb wait-for --from "noreply@example.com" --json | jq .subject`,
    RunE: runWaitFor,
}

var (
    waitForEmail        string
    waitForSubject      string
    waitForSubjectRegex string
    waitForFrom         string
    waitForFromRegex    string
    waitForTimeout      string
    waitForJSON         bool
    waitForQuiet        bool
    waitForExtractLink  bool
    waitForCount        int
)

func init() {
    rootCmd.AddCommand(waitForCmd)

    // Inbox selection
    waitForCmd.Flags().StringVar(&waitForEmail, "email", "",
        "Watch specific inbox (default: active)")

    // Filters
    waitForCmd.Flags().StringVar(&waitForSubject, "subject", "",
        "Exact subject match")
    waitForCmd.Flags().StringVar(&waitForSubjectRegex, "subject-regex", "",
        "Subject regex pattern")
    waitForCmd.Flags().StringVar(&waitForFrom, "from", "",
        "Exact sender match")
    waitForCmd.Flags().StringVar(&waitForFromRegex, "from-regex", "",
        "Sender regex pattern")

    // Timing
    waitForCmd.Flags().StringVar(&waitForTimeout, "timeout", "60s",
        "Maximum time to wait")
    waitForCmd.Flags().IntVar(&waitForCount, "count", 1,
        "Number of matching emails to wait for")

    // Output
    waitForCmd.Flags().BoolVar(&waitForJSON, "json", false,
        "Output email as JSON")
    waitForCmd.Flags().BoolVar(&waitForQuiet, "quiet", false,
        "No output, exit code only")
    waitForCmd.Flags().BoolVar(&waitForExtractLink, "extract-link", false,
        "Output first link from email")
}

func runWaitFor(cmd *cobra.Command, args []string) error {
    // Parse timeout
    timeout, err := time.ParseDuration(waitForTimeout)
    if err != nil {
        fmt.Fprintln(os.Stderr, output.Error("Invalid timeout format"))
        os.Exit(2)
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    // Load keystore
    keystore, err := config.LoadKeystore()
    if err != nil {
        fmt.Fprintln(os.Stderr, output.Error("Failed to load keystore"))
        os.Exit(2)
    }

    // Get inbox
    var stored *config.StoredInbox
    if waitForEmail != "" {
        stored, err = keystore.GetInbox(waitForEmail)
    } else {
        stored, err = keystore.GetActiveInbox()
    }
    if err != nil {
        fmt.Fprintln(os.Stderr, output.Error("No inbox found"))
        os.Exit(2)
    }

    // Create client
    client, err := config.NewClient()
    if err != nil {
        fmt.Fprintln(os.Stderr, output.Error(err.Error()))
        os.Exit(2)
    }
    defer client.Close()

    // Import inbox
    inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
    if err != nil {
        fmt.Fprintln(os.Stderr, output.Error("Failed to import inbox"))
        os.Exit(2)
    }

    // Build wait options
    opts := buildWaitOptions()

    // Show waiting message (unless quiet)
    if !waitForQuiet {
        fmt.Fprintf(os.Stderr, "Waiting for email on %s (timeout: %s)...\n",
            stored.Email, timeout)
    }

    // Wait for email(s)
    var emails []*vaultsandbox.Email
    if waitForCount > 1 {
        emails, err = inbox.WaitForEmailCount(ctx, waitForCount, opts...)
    } else {
        email, waitErr := inbox.WaitForEmail(ctx, opts...)
        if waitErr != nil {
            err = waitErr
        } else {
            emails = []*vaultsandbox.Email{email}
        }
    }

    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            if !waitForQuiet {
                fmt.Fprintln(os.Stderr, output.Error("Timeout waiting for email"))
            }
            os.Exit(1)
        }
        fmt.Fprintln(os.Stderr, output.Error(err.Error()))
        os.Exit(2)
    }

    // Output result
    outputEmails(emails)
    os.Exit(0)
    return nil
}

func buildWaitOptions() []vaultsandbox.WaitOption {
    var opts []vaultsandbox.WaitOption

    // Subject filters
    if waitForSubject != "" {
        opts = append(opts, vaultsandbox.WithSubject(waitForSubject))
    }
    if waitForSubjectRegex != "" {
        re := regexp.MustCompile(waitForSubjectRegex)
        opts = append(opts, vaultsandbox.WithSubjectRegex(re))
    }

    // From filters
    if waitForFrom != "" {
        opts = append(opts, vaultsandbox.WithFrom(waitForFrom))
    }
    if waitForFromRegex != "" {
        re := regexp.MustCompile(waitForFromRegex)
        opts = append(opts, vaultsandbox.WithFromRegex(re))
    }

    return opts
}

func outputEmails(emails []*vaultsandbox.Email) {
    if waitForQuiet {
        return
    }

    for _, email := range emails {
        if waitForJSON {
            // JSON output
            data, _ := json.MarshalIndent(emailToMap(email), "", "  ")
            fmt.Println(string(data))
        } else if waitForExtractLink {
            // Extract first link
            if len(email.Links) > 0 {
                fmt.Println(email.Links[0])
            }
        } else {
            // Human-readable output
            fmt.Printf("Subject: %s\n", email.Subject)
            fmt.Printf("From: %s\n", email.From)
            fmt.Printf("Received: %s\n", email.ReceivedAt.Format(time.RFC3339))
            if len(email.Links) > 0 {
                fmt.Printf("Links: %d found\n", len(email.Links))
            }
        }
    }
}

func emailToMap(email *vaultsandbox.Email) map[string]interface{} {
    return map[string]interface{}{
        "id":         email.ID,
        "subject":    email.Subject,
        "from":       email.From,
        "to":         email.To,
        "receivedAt": email.ReceivedAt.Format(time.RFC3339),
        "text":       email.Text,
        "html":       email.HTML,
        "links":      email.Links,
        "headers":    email.Headers,
    }
}
```

## CI/CD Usage Examples

### GitHub Actions

```yaml
jobs:
  test-email-flow:
    runs-on: ubuntu-latest
    steps:
      - name: Setup VaultSandbox
        run: |
          curl -sSL https://vaultsandbox.com/install.sh | bash
          vsb inbox create ci-test
          echo "TEST_EMAIL=$(vsb inbox list --json | jq -r '.active')" >> $GITHUB_ENV

      - name: Trigger email send
        run: |
          curl -X POST https://myapp.com/forgot-password \
            -d "email=$TEST_EMAIL"

      - name: Wait for reset email
        run: |
          LINK=$(vsb wait-for \
            --subject-regex "password reset" \
            --timeout 30s \
            --extract-link)
          echo "RESET_LINK=$LINK" >> $GITHUB_ENV

      - name: Complete reset flow
        run: |
          curl -X POST "$RESET_LINK" -d "password=newpass123"
```

### GitLab CI

```yaml
test-verification:
  script:
    - vsb inbox create --label gitlab-test
    - |
      # Register user
      curl -X POST $APP_URL/register \
        -d "email=$(vsb inbox list --active)"
    - |
      # Wait for verification and extract link
      VERIFY_LINK=$(vsb wait-for \
        --subject "Verify your email" \
        --timeout 60s \
        --extract-link)
    - curl "$VERIFY_LINK"
```

### Shell Script

```bash
#!/bin/bash
set -e

# Create inbox
vsb inbox create

# Get email address
EMAIL=$(vsb inbox list --json | jq -r '.active')

# Trigger your app's email
your-app send-welcome --to "$EMAIL"

# Wait and validate
if vsb wait-for --subject "Welcome" --timeout 30s --quiet; then
    echo "Welcome email received!"
else
    echo "Failed to receive welcome email"
    exit 1
fi

# Extract OTP code from email
OTP=$(vsb wait-for --subject "Your code" --json | jq -r '.text' | grep -oE '[0-9]{6}')
echo "OTP: $OTP"
```

## Verification

```bash
# Basic wait
vsb wait-for --timeout 10s

# With subject filter
vsb wait-for --subject-regex "verify|confirm"

# JSON output
vsb wait-for --json | jq .subject

# Extract link
LINK=$(vsb wait-for --extract-link)
echo $LINK

# Quiet mode for scripting
if vsb wait-for --quiet --timeout 5s; then
  echo "Got it!"
fi
```

## Files Created

- `internal/cli/waitfor.go`

## Next Steps

Proceed to [06-audit-command.md](06-audit-command.md) to implement the deep-dive audit command.
