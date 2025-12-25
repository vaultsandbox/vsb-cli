package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/browser"
)

var openCmd = &cobra.Command{
	Use:   "open [email-id]",
	Short: "Extract and open the first link from an email",
	Long: `Extract the first HTTP/HTTPS link from an email and open it in your browser.

This is useful for quickly following verification links, password reset links,
or any other actionable URLs in emails.

For interactive use, prefer 'vsb watch' and press 'o' to open links.

Examples:
  vsb open              # Open first link from latest email
  vsb open abc123       # Open first link from specific email
  vsb open --list       # List all links (for scripting)
  vsb open --nth 2      # Open the second link
  vsb open --json       # JSON output for CI/CD`,
	Args: cobra.MaximumNArgs(1),
	RunE: runOpen,
}

var (
	openList  bool
	openNth   int
	openEmail string
	openJSON  bool
)

func init() {
	rootCmd.AddCommand(openCmd)

	openCmd.Flags().BoolVar(&openList, "list", false,
		"List all links without opening")
	openCmd.Flags().IntVar(&openNth, "nth", 1,
		"Open the Nth link (1-indexed)")
	openCmd.Flags().StringVar(&openEmail, "email", "",
		"Use specific inbox (default: active)")
	openCmd.Flags().BoolVar(&openJSON, "json", false,
		"Output as JSON")
}

func runOpen(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get email ID (empty = latest)
	emailID := ""
	if len(args) > 0 {
		emailID = args[0]
	}

	// Use shared helper
	email, _, cleanup, err := GetEmailByIDOrLatest(ctx, emailID, openEmail)
	if err != nil {
		return err
	}
	defer cleanup()

	// Check for links
	if len(email.Links) == 0 {
		if openJSON {
			fmt.Println("[]")
		} else {
			fmt.Println("No links found in email")
		}
		return nil
	}

	// List mode
	if openList {
		if openJSON {
			data, _ := json.MarshalIndent(email.Links, "", "  ")
			fmt.Println(string(data))
		} else {
			for i, link := range email.Links {
				fmt.Printf("%d. %s\n", i+1, link)
			}
		}
		return nil
	}

	// Get the requested link
	if openNth < 1 || openNth > len(email.Links) {
		return fmt.Errorf("link index %d out of range (1-%d)", openNth, len(email.Links))
	}
	link := email.Links[openNth-1]

	if openJSON {
		data, _ := json.Marshal(map[string]string{"url": link})
		fmt.Println(string(data))
	} else {
		fmt.Printf("Opening: %s\n", link)
	}

	return browser.OpenURL(link)
}
