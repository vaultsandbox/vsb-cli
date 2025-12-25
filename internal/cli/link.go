package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/browser"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

var linkCmd = &cobra.Command{
	Use:   "link [email-id]",
	Short: "Extract and optionally open links from an email",
	Long: `Extract HTTP/HTTPS links from an email.

By default, lists all links. Use --open to open a link in your browser.
This is useful for quickly following verification links, password reset links,
or any other actionable URLs in emails.

For interactive use, run 'vsb' and press 'o' to open links.

Examples:
  vsb link              # List links from latest email
  vsb link abc123       # List links from specific email
  vsb link --open       # Open first link in browser
  vsb link --open 2     # Open second link in browser
  vsb link -o json      # JSON output for CI/CD`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLink,
}

var (
	linkOpen  int
	linkEmail string
)

func init() {
	rootCmd.AddCommand(linkCmd)

	linkCmd.Flags().IntVarP(&linkOpen, "open", "O", 0,
		"Open the Nth link in browser (1=first, 0=don't open)")
	linkCmd.Flags().StringVar(&linkEmail, "email", "",
		"Use specific inbox (default: active)")
}

func runLink(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get email ID (empty = latest)
	emailID := ""
	if len(args) > 0 {
		emailID = args[0]
	}

	// Use shared helper
	email, _, cleanup, err := GetEmailByIDOrLatest(ctx, emailID, linkEmail)
	if err != nil {
		return err
	}
	defer cleanup()

	// Check for links
	if len(email.Links) == 0 {
		if config.GetOutput() == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("No links found in email")
		}
		return nil
	}

	// If --open is specified, open the link
	if linkOpen > 0 {
		if linkOpen > len(email.Links) {
			return fmt.Errorf("link index %d out of range (1-%d)", linkOpen, len(email.Links))
		}
		link := email.Links[linkOpen-1]
		fmt.Printf("Opening: %s\n", link)
		return browser.OpenURL(link)
	}

	// Default: list all links
	if config.GetOutput() == "json" {
		data, _ := json.MarshalIndent(email.Links, "", "  ")
		fmt.Println(string(data))
	} else {
		for i, link := range email.Links {
			fmt.Printf("%d. %s\n", i+1, link)
		}
	}
	return nil
}
