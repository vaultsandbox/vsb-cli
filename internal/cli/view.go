package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/browser"
	"github.com/vaultsandbox/vsb-cli/internal/config"
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
	if config.GetOutput() == "json" {
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
		output, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(output))
		return nil
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

	// Wrap HTML with proper document structure
	wrappedHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .header {
            background: #7C3AED;
            color: white;
            padding: 20px;
            border-radius: 8px 8px 0 0;
        }
        .header h1 {
            margin: 0 0 10px 0;
            font-size: 1.2em;
        }
        .header .meta {
            font-size: 0.9em;
            opacity: 0.9;
        }
        .content {
            background: white;
            padding: 20px;
            border-radius: 0 0 8px 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        .vsb-badge {
            background: #10B981;
            color: white;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 0.8em;
            margin-left: 10px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>%s <span class="vsb-badge">VaultSandbox</span></h1>
        <div class="meta">
            <strong>From:</strong> %s<br>
            <strong>Date:</strong> %s
        </div>
    </div>
    <div class="content">
        %s
    </div>
</body>
</html>`,
		html.EscapeString(email.Subject),
		html.EscapeString(email.Subject),
		html.EscapeString(email.From),
		email.ReceivedAt.Format("January 2, 2006 at 3:04 PM"),
		email.HTML,
	)

	fmt.Println("Opening email in browser...")

	// Cleanup old previews (older than 1 hour)
	browser.CleanupPreviews(time.Hour)

	return browser.ViewHTML(wrappedHTML)
}
