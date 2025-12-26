package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/browser"
)

var viewCmd = &cobra.Command{
	Use:   "view [email-id]",
	Short: "Preview email content",
	Long: `View email content in various formats.

Examples:
  vsb email view              # View latest email HTML in browser
  vsb email view abc123       # View specific email
  vsb email view -t           # Print plain text to terminal
  vsb email view -r           # Print raw email source (RFC 5322)
  vsb email view -o json      # JSON output`,
	Args: cobra.MaximumNArgs(1),
	RunE: runView,
}

var (
	viewText  bool
	viewRaw   bool
	viewInbox string
)

func init() {
	emailCmd.AddCommand(viewCmd)

	viewCmd.Flags().BoolVarP(&viewText, "text", "t", false,
		"Show plain text version in terminal")
	viewCmd.Flags().BoolVarP(&viewRaw, "raw", "r", false,
		"Show raw email source (RFC 5322)")
	viewCmd.Flags().StringVar(&viewInbox, "inbox", "",
		"Use specific inbox (default: active)")
}

func runView(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get email ID (empty = latest)
	emailID := ""
	if len(args) > 0 {
		emailID = args[0]
	}

	// Use shared helper (returns email, inbox, cleanup, error)
	email, inbox, cleanup, err := GetEmailByIDOrLatest(ctx, emailID, viewInbox)
	if err != nil {
		return err
	}
	defer cleanup()

	// JSON output
	if getOutput(cmd) == "json" {
		data := map[string]interface{}{
			"id":         email.ID,
			"subject":    email.Subject,
			"from":       email.From,
			"to":         strings.Join(email.To, ", "),
			"receivedAt": email.ReceivedAt.Format(time.RFC3339),
			"text":       email.Text,
			"html":       email.HTML,
			"links":      email.Links,
		}
		return outputJSON(data)
	}

	// Raw mode - show RFC 5322 source
	if viewRaw {
		raw, err := inbox.GetRawEmail(ctx, email.ID)
		if err != nil {
			return err
		}
		fmt.Println(raw)
		return nil
	}

	// Text mode - print to terminal
	if viewText {
		if email.Text == "" {
			fmt.Println("No plain text version available")
			return nil
		}
		fmt.Printf("Subject: %s\n", email.Subject)
		fmt.Printf("From: %s\n", email.From)
		fmt.Printf("Date: %s\n\n", email.ReceivedAt.Format("2006-01-02 15:04:05"))
		fmt.Println(email.Text)
		return nil
	}

	// HTML mode - open in browser
	if email.HTML == "" {
		fmt.Println("No HTML version, showing text:")
		fmt.Println(email.Text)
		return nil
	}

	fmt.Println("Opening email in browser...")

	// Cleanup old previews (older than 1 hour)
	browser.CleanupPreviews(time.Hour)

	return browser.ViewEmailHTML(email.Subject, email.From, email.ReceivedAt, email.HTML)
}
