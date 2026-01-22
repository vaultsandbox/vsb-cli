package inbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
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

	emailArg := cliutil.GetArg(args, 0, "")

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
	emailCount, syncErr := getInboxEmailCount(ctx, stored)

	// Get chaos config if server supports it
	chaosConfig, _ := getInboxChaosConfig(ctx, stored)

	isExpired := cliutil.IsExpired(stored.ExpiresAt)
	isActive := stored.Email == ks.ActiveInbox

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		return cliutil.OutputJSON(cliutil.InboxFullJSON(stored, isActive, emailCount, syncErr, chaosConfig, time.Now()))
	}

	// Pretty output
	content := formatInboxInfoContent(stored, isActive, isExpired, emailCount, syncErr, chaosConfig)

	fmt.Println()
	fmt.Println(styles.BoxStyle.Render(content))
	fmt.Println()

	return nil
}

// formatInboxInfoContent builds the formatted content string for inbox info display.
func formatInboxInfoContent(stored *config.StoredInbox, isActive, isExpired bool, emailCount int, syncErr error, chaosConfig *vaultsandbox.ChaosConfig) string {
	labelStyle := styles.LabelStyle.Width(14)

	var content string

	// Title with active badge
	title := styles.TitleStyle.Render(stored.Email)
	if isActive {
		title = title + "  " + styles.BadgeStyle.Background(styles.Green).Render("ACTIVE")
	}
	content += title + "\n\n"

	// Details
	content += fmt.Sprintf("%s %s\n", labelStyle.Render("ID:"), stored.ID)
	content += fmt.Sprintf("%s %s\n", labelStyle.Render("Created:"), stored.CreatedAt.Format(cliutil.TimeFormatShort))

	// Expiry with color
	var expiryStr string
	if isExpired {
		expiryStr = styles.FailStyle.Render("EXPIRED")
	} else {
		expiryStr = fmt.Sprintf("%s (%s)", stored.ExpiresAt.Format(cliutil.TimeFormatShort), cliutil.FormatExpiry(stored.ExpiresAt))
	}
	content += fmt.Sprintf("%s %s\n", labelStyle.Render("Expires:"), expiryStr)

	// Email count
	if syncErr != nil {
		content += fmt.Sprintf("%s %s\n", labelStyle.Render("Emails:"), styles.WarnStyle.Render("(sync error)"))
	} else {
		content += fmt.Sprintf("%s %d\n", labelStyle.Render("Emails:"), emailCount)
	}

	// Chaos status
	if chaosConfig != nil {
		var chaosStr string
		if chaosConfig.Enabled {
			types := cliutil.ChaosEnabledTypes(chaosConfig)
			if len(types) > 0 {
				chaosStr = styles.WarnStyle.Render(fmt.Sprintf("Enabled (%s)", strings.Join(types, ", ")))
			} else {
				chaosStr = styles.WarnStyle.Render("Enabled")
			}
		} else {
			chaosStr = styles.MutedStyle.Render("Disabled")
		}
		content += fmt.Sprintf("%s %s\n", labelStyle.Render("Chaos:"), chaosStr)
	}

	return content
}

// getInboxEmailCount fetches the email count for an inbox from the server.
func getInboxEmailCount(ctx context.Context, stored *config.StoredInbox) (int, error) {
	client, err := config.NewClient()
	if err != nil {
		return 0, err
	}
	defer client.Close()

	inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
	if err != nil {
		return 0, err
	}

	status, err := inbox.GetSyncStatus(ctx)
	if err != nil {
		return 0, err
	}

	return status.EmailCount, nil
}

// getInboxChaosConfig fetches the chaos config for an inbox from the server.
// Returns nil if chaos is not enabled on the server or if there's an error.
func getInboxChaosConfig(ctx context.Context, stored *config.StoredInbox) (*vaultsandbox.ChaosConfig, error) {
	client, err := config.NewClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Check if server supports chaos
	if !client.ServerInfo().ChaosEnabled {
		return nil, nil
	}

	inbox, err := client.ImportInbox(ctx, stored.ToExportedInbox())
	if err != nil {
		return nil, err
	}

	return inbox.GetChaosConfig(ctx)
}
