package email

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/browser"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
)

var urlCmd = &cobra.Command{
	Use:   "url [email-id]",
	Short: "Extract and open URLs from an email",
	Long: `Extract HTTP/HTTPS URLs from an email.

By default, lists all URLs. Use --open to open a URL in your browser.
This is useful for quickly following verification links, password reset links,
or any other actionable URLs in emails.

Examples:
  vsb email url              # List URLs from latest email
  vsb email url abc123       # List URLs from specific email
  vsb email url --open 1     # Open first URL in browser
  vsb email url --open 2     # Open second URL in browser
  vsb email url -o json      # JSON output for CI/CD`,
	Args: cobra.MaximumNArgs(1),
	RunE: runURL,
}

var (
	urlOpen  int
	urlInbox string
)

func init() {
	Cmd.AddCommand(urlCmd)

	urlCmd.Flags().IntVarP(&urlOpen, "open", "O", 0,
		"Open the Nth URL in browser (1=first, 0=don't open)")
	urlCmd.Flags().StringVar(&urlInbox, "inbox", "",
		"Use specific inbox (default: active)")
}

func runURL(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get email ID (empty = latest)
	emailID := ""
	if len(args) > 0 {
		emailID = args[0]
	}

	// Use shared helper
	email, _, cleanup, err := cliutil.GetEmailByIDOrLatest(ctx, emailID, urlInbox)
	if err != nil {
		return err
	}
	defer cleanup()

	// Check for URLs
	if len(email.Links) == 0 {
		if cliutil.GetOutput(cmd) == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("No URLs found in email")
		}
		return nil
	}

	// If --open is specified, open the URL
	if urlOpen > 0 {
		if urlOpen > len(email.Links) {
			return fmt.Errorf("URL index %d out of range (1-%d)", urlOpen, len(email.Links))
		}
		url := email.Links[urlOpen-1]
		fmt.Printf("Opening: %s\n", url)
		return browser.OpenURL(url)
	}

	// Default: list all URLs
	if cliutil.GetOutput(cmd) == "json" {
		return cliutil.OutputJSON(email.Links)
	} else {
		for i, url := range email.Links {
			fmt.Printf("%d. %s\n", i+1, url)
		}
	}
	return nil
}
