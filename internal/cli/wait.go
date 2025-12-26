package cli

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
)

var waitCmd = &cobra.Command{
	Use:   "wait",
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
  --quiet         No output, just exit code
  --extract-link  Output first link from email body

Examples:
  # Wait for any email
  vsb wait

  # Wait for password reset email
  vsb wait --subject-regex "password reset" --timeout 30s

  # Extract verification link
  LINK=$(vsb wait --subject "Verify" --extract-link)

  # JSON output for parsing
  vsb wait --from "noreply@example.com" -o json | jq .subject`,
	RunE: runWait,
}

var (
	waitForInbox        string
	waitForSubject      string
	waitForSubjectRegex string
	waitForFrom         string
	waitForFromRegex    string
	waitForTimeout      string
	waitForQuiet        bool
	waitForExtractLink  bool
	waitForCount        int
)

func init() {
	rootCmd.AddCommand(waitCmd)

	// Inbox selection
	waitCmd.Flags().StringVar(&waitForInbox, "inbox", "",
		"Use specific inbox (default: active)")

	// Filters
	waitCmd.Flags().StringVar(&waitForSubject, "subject", "",
		"Exact subject match")
	waitCmd.Flags().StringVar(&waitForSubjectRegex, "subject-regex", "",
		"Subject regex pattern")
	waitCmd.Flags().StringVar(&waitForFrom, "from", "",
		"Exact sender match")
	waitCmd.Flags().StringVar(&waitForFromRegex, "from-regex", "",
		"Sender regex pattern")

	// Timing
	waitCmd.Flags().StringVar(&waitForTimeout, "timeout", "60s",
		"Maximum time to wait")
	waitCmd.Flags().IntVar(&waitForCount, "count", 1,
		"Number of matching emails to wait for")

	// Output
	waitCmd.Flags().BoolVarP(&waitForQuiet, "quiet", "q", false,
		"No output, exit code only")
	waitCmd.Flags().BoolVar(&waitForExtractLink, "extract-link", false,
		"Output first link from email")
}

func runWait(cmd *cobra.Command, args []string) error {
	// Parse timeout
	timeout, err := time.ParseDuration(waitForTimeout)
	if err != nil {
		return fmt.Errorf("invalid timeout format: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use shared helper
	inbox, cleanup, err := LoadAndImportInbox(ctx, waitForInbox)
	if err != nil {
		return err
	}
	defer cleanup()

	// Build wait options
	opts, err := buildWaitOptions(timeout)
	if err != nil {
		return err
	}

	// Show waiting message (unless quiet)
	if !waitForQuiet {
		fmt.Fprintf(os.Stderr, "Waiting for email on %s (timeout: %s)...\n",
			inbox.Export().EmailAddress, timeout)
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
			return fmt.Errorf("timeout waiting for email")
		}
		return err
	}

	// Output result
	outputEmails(cmd, emails)
	return nil
}

func buildWaitOptions(timeout time.Duration) ([]vaultsandbox.WaitOption, error) {
	var opts []vaultsandbox.WaitOption

	// Set timeout
	opts = append(opts, vaultsandbox.WithWaitTimeout(timeout))

	// Subject filters
	if waitForSubject != "" {
		opts = append(opts, vaultsandbox.WithSubject(waitForSubject))
	}
	if waitForSubjectRegex != "" {
		re, err := regexp.Compile(waitForSubjectRegex)
		if err != nil {
			return nil, fmt.Errorf("invalid subject regex: %w", err)
		}
		opts = append(opts, vaultsandbox.WithSubjectRegex(re))
	}

	// From filters
	if waitForFrom != "" {
		opts = append(opts, vaultsandbox.WithFrom(waitForFrom))
	}
	if waitForFromRegex != "" {
		re, err := regexp.Compile(waitForFromRegex)
		if err != nil {
			return nil, fmt.Errorf("invalid from regex: %w", err)
		}
		opts = append(opts, vaultsandbox.WithFromRegex(re))
	}

	return opts, nil
}

func outputEmails(cmd *cobra.Command, emails []*vaultsandbox.Email) {
	if waitForQuiet {
		return
	}

	for _, email := range emails {
		if getOutput(cmd) == "json" {
			// JSON output
			_ = outputJSON(emailToMap(email))
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
	// Format To field - join array if present
	var to interface{}
	if len(email.To) > 0 {
		to = strings.Join(email.To, ", ")
	} else {
		to = ""
	}

	return map[string]interface{}{
		"id":         email.ID,
		"subject":    email.Subject,
		"from":       email.From,
		"to":         to,
		"receivedAt": email.ReceivedAt.Format(time.RFC3339),
		"text":       email.Text,
		"html":       email.HTML,
		"links":      email.Links,
		"headers":    email.Headers,
	}
}
