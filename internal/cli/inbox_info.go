package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var inboxInfoCmd = &cobra.Command{
	Use:   "info [email]",
	Short: "Show inbox details",
	Long: `Display detailed information about an inbox.

Shows email address, creation date, expiry, email count, and sync status.

Examples:
  vsb inbox info           # Info for active inbox
  vsb inbox info abc       # Info for inbox matching 'abc'
  vsb inbox info -o json   # JSON output`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInboxInfo,
}

func init() {
	inboxCmd.AddCommand(inboxInfoCmd)
}

func runInboxInfo(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get email arg
	emailArg := ""
	if len(args) > 0 {
		emailArg = args[0]
	}

	// Load keystore
	ks, err := LoadKeystoreOrError()
	if err != nil {
		return err
	}

	// Get inbox
	stored, err := GetInbox(ks, emailArg)
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
	if getOutput(cmd) == "json" {
		data := map[string]interface{}{
			"email":      stored.Email,
			"id":         stored.ID,
			"createdAt":  stored.CreatedAt.Format(time.RFC3339),
			"expiresAt":  stored.ExpiresAt.Format(time.RFC3339),
			"isExpired":  isExpired,
			"isActive":   isActive,
			"emailCount": emailCount,
		}
		if syncErr != nil {
			data["syncError"] = syncErr.Error()
		}
		return outputJSON(data)
	}

	// Pretty output
	labelStyle := lipgloss.NewStyle().
		Foreground(styles.Gray).
		Width(14)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Primary)

	// Build content
	var content string

	// Title with active badge
	title := titleStyle.Render(stored.Email)
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
		expiryStr = fmt.Sprintf("%s (%s)", stored.ExpiresAt.Format("2006-01-02 15:04"), formatDuration(remaining))
	}
	content += fmt.Sprintf("%s %s\n", labelStyle.Render("Expires:"), expiryStr)

	// Email count
	if syncErr != nil {
		content += fmt.Sprintf("%s %s\n", labelStyle.Render("Emails:"), lipgloss.NewStyle().Foreground(styles.Yellow).Render("(sync error)"))
	} else {
		content += fmt.Sprintf("%s %d\n", labelStyle.Render("Emails:"), emailCount)
	}

	// Security info
	content += fmt.Sprintf("%s %s", labelStyle.Render("Encryption:"), "ML-KEM-768 (Quantum-Safe)")

	fmt.Println()
	fmt.Println(styles.BoxStyle.Render(content))
	fmt.Println()

	return nil
}
