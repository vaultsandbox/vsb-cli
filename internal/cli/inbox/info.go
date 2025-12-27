package inbox

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var infoCmd = &cobra.Command{
	Use:   "info [email]",
	Short: "Show inbox details",
	Long: `Display detailed information about an inbox.

Shows email address, creation date, expiry, email count, and sync status.

Examples:
  vsb inbox info           # Info for active inbox
  vsb inbox info abc       # Info for inbox matching 'abc'
  vsb inbox info -o json   # JSON output`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInfo,
}

func init() {
	Cmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get email arg
	emailArg := ""
	if len(args) > 0 {
		emailArg = args[0]
	}

	// Load keystore
	ks, err := cliutil.LoadKeystoreOrError()
	if err != nil {
		return err
	}

	// Get inbox
	stored, err := cliutil.GetInbox(ks, emailArg)
	if err != nil {
		return err
	}

	// Get email count from server
	var emailCount int
	var syncErr error

	client, err := config.NewClient()
	if err == nil {
		defer client.Close()

		inbox, importErr := client.ImportInbox(ctx, stored.ToExportedInbox())
		if importErr == nil {
			status, statusErr := inbox.GetSyncStatus(ctx)
			if statusErr == nil {
				emailCount = status.EmailCount
			} else {
				syncErr = statusErr
			}
		} else {
			syncErr = importErr
		}
	} else {
		syncErr = err
	}

	// Calculate time values
	now := time.Now()
	isExpired := stored.ExpiresAt.Before(now)
	var remaining time.Duration
	if !isExpired {
		remaining = stored.ExpiresAt.Sub(now).Round(time.Minute)
	}

	isActive := stored.Email == ks.ActiveInbox

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		return cliutil.OutputJSON(cliutil.InboxFullJSON(stored, isActive, emailCount, syncErr))
	}

	// Pretty output
	content := formatInboxInfoContent(stored, isActive, isExpired, remaining, emailCount, syncErr)

	fmt.Println()
	fmt.Println(styles.BoxStyle.Render(content))
	fmt.Println()

	return nil
}

// formatInboxInfoContent builds the formatted content string for inbox info display.
func formatInboxInfoContent(stored *config.StoredInbox, isActive, isExpired bool, remaining time.Duration, emailCount int, syncErr error) string {
	labelStyle := styles.LabelStyle.Width(14)

	var content string

	// Title with active badge
	title := styles.TitleStyle.Render(stored.Email)
	if isActive {
		badge := lipgloss.NewStyle().
			Background(styles.Green).
			Foreground(styles.White).
			Padding(0, 1).
			Render("ACTIVE")
		title = title + "  " + badge
	}
	content += title + "\n\n"

	// Details
	content += fmt.Sprintf("%s %s\n", labelStyle.Render("ID:"), stored.ID)
	content += fmt.Sprintf("%s %s\n", labelStyle.Render("Created:"), stored.CreatedAt.Format("2006-01-02 15:04"))

	// Expiry with color
	var expiryStr string
	if isExpired {
		expiryStr = lipgloss.NewStyle().Foreground(styles.Red).Render("EXPIRED")
	} else {
		expiryStr = fmt.Sprintf("%s (%s)", stored.ExpiresAt.Format("2006-01-02 15:04"), cliutil.FormatDuration(remaining))
	}
	content += fmt.Sprintf("%s %s\n", labelStyle.Render("Expires:"), expiryStr)

	// Email count
	if syncErr != nil {
		content += fmt.Sprintf("%s %s\n", labelStyle.Render("Emails:"), lipgloss.NewStyle().Foreground(styles.Yellow).Render("(sync error)"))
	} else {
		content += fmt.Sprintf("%s %d\n", labelStyle.Render("Emails:"), emailCount)
	}

	// Security info
	content += fmt.Sprintf("%s %s", labelStyle.Render("Encryption:"), "ML-KEM-768")

	return content
}
