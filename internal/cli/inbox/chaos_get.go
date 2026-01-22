package inbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var chaosGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current chaos configuration",
	Long: `Display the current chaos configuration for an inbox.

Shows the status and settings for all configured chaos types including
latency injection, connection drops, random errors, greylisting, and blackhole mode.`,
	Args: cobra.NoArgs,
	RunE: runChaosGet,
}

func init() {
	chaosCmd.AddCommand(chaosGetCmd)
}

func runChaosGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check if chaos is enabled on server
	if err := checkChaosEnabled(); err != nil {
		return err
	}

	// Load inbox
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, ChaosInboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// Get chaos config
	cfg, err := inbox.GetChaosConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chaos config: %w", err)
	}

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		return cliutil.OutputJSON(cliutil.ChaosConfigFullJSON(cfg))
	}

	// Pretty output
	printChaosConfig(cfg, inbox.EmailAddress())
	return nil
}

// checkChaosEnabled verifies the server has chaos enabled.
func checkChaosEnabled() error {
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	if !client.ServerInfo().ChaosEnabled {
		return fmt.Errorf("chaos is not enabled on this server\n\nChaos engineering features must be enabled server-side.\nContact your administrator or check server configuration")
	}
	return nil
}

// printChaosConfig displays the chaos configuration in pretty format.
func printChaosConfig(cfg *vaultsandbox.ChaosConfig, email string) {
	labelStyle := styles.LabelStyle

	fmt.Println()
	fmt.Printf("%s %s\n", styles.TitleStyle.Render("Chaos Configuration for"), email)
	fmt.Println()

	// Status
	if cfg.Enabled {
		fmt.Printf("  %s %s\n", labelStyle.Render("Status:"), styles.PassStyle.Render("Enabled"))
	} else {
		fmt.Printf("  %s %s\n", labelStyle.Render("Status:"), styles.MutedStyle.Render("Disabled"))
		fmt.Println()
		return
	}

	// Expiry
	if cfg.ExpiresAt != nil {
		expiryStr := fmt.Sprintf("%s (%s)", cfg.ExpiresAt.Format(cliutil.TimeFormatWithZone), cliutil.FormatExpiry(*cfg.ExpiresAt))
		fmt.Printf("  %s %s\n", labelStyle.Render("Expires:"), expiryStr)
	}

	// Latency
	if cfg.Latency != nil && cfg.Latency.Enabled {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("Latency:"))
		jitterStr := "jitter enabled"
		if !cfg.Latency.Jitter {
			jitterStr = "fixed"
		}
		fmt.Printf("  %s %d-%dms (%s)\n", labelStyle.Render("Delay:"), cfg.Latency.MinDelayMs, cfg.Latency.MaxDelayMs, jitterStr)
		fmt.Printf("  %s %.0f%%\n", labelStyle.Render("Probability:"), cfg.Latency.Probability*100)
	}

	// Connection Drop
	if cfg.ConnectionDrop != nil && cfg.ConnectionDrop.Enabled {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("Connection Drop:"))
		fmt.Printf("  %s %.0f%%\n", labelStyle.Render("Probability:"), cfg.ConnectionDrop.Probability*100)
		if !cfg.ConnectionDrop.Graceful {
			fmt.Printf("  %s %s\n", labelStyle.Render("Mode:"), "Abrupt (RST)")
		}
	}

	// Random Error
	if cfg.RandomError != nil && cfg.RandomError.Enabled {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("Random Error:"))
		fmt.Printf("  %s %.0f%%\n", labelStyle.Render("Error Rate:"), cfg.RandomError.ErrorRate*100)
		types := formatErrorTypes(cfg.RandomError.ErrorTypes)
		fmt.Printf("  %s %s\n", labelStyle.Render("Types:"), types)
	}

	// Greylist
	if cfg.Greylist != nil && cfg.Greylist.Enabled {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("Greylist:"))
		fmt.Printf("  %s %d\n", labelStyle.Render("Max Attempts:"), cfg.Greylist.MaxAttempts)
		fmt.Printf("  %s %dms\n", labelStyle.Render("Retry Window:"), cfg.Greylist.RetryWindowMs)
		trackByStr := formatTrackBy(cfg.Greylist.TrackBy)
		fmt.Printf("  %s %s\n", labelStyle.Render("Track By:"), trackByStr)
	}

	// Blackhole
	if cfg.Blackhole != nil && cfg.Blackhole.Enabled {
		fmt.Println()
		fmt.Println(styles.SectionStyle.Render("Blackhole:"))
		webhooksStr := "No"
		if cfg.Blackhole.TriggerWebhooks {
			webhooksStr = "Yes"
		}
		fmt.Printf("  %s %s\n", labelStyle.Render("Trigger Webhooks:"), webhooksStr)
	}

	fmt.Println()
}

func formatErrorTypes(types []vaultsandbox.RandomErrorType) string {
	strs := make([]string, len(types))
	for i, t := range types {
		strs[i] = string(t)
	}
	return strings.Join(strs, ", ")
}

func formatTrackBy(trackBy vaultsandbox.GreylistTrackBy) string {
	switch trackBy {
	case vaultsandbox.GreylistTrackByIP:
		return "IP"
	case vaultsandbox.GreylistTrackBySender:
		return "Sender"
	case vaultsandbox.GreylistTrackByIPSender:
		return "IP + Sender"
	default:
		return string(trackBy)
	}
}
