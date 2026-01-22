package inbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var chaosSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Enable and configure chaos",
	Long: `Enable and configure chaos engineering scenarios for an inbox.

At least one chaos type must be enabled. Use boolean flags to enable chaos types
and additional flags to configure their behavior.

Examples:
  # Enable latency injection
  vsb inbox chaos set --latency --min-delay 1000 --max-delay 5000 --probability 0.5

  # Enable random errors (20% temporary errors)
  vsb inbox chaos set --random-error --error-rate 0.2 --error-types temporary

  # Enable greylisting
  vsb inbox chaos set --greylist --max-attempts 3 --retry-window 600000

  # Enable connection drops
  vsb inbox chaos set --connection-drop --drop-probability 0.3

  # Enable blackhole
  vsb inbox chaos set --blackhole --trigger-webhooks

  # Combined with expiration
  vsb inbox chaos set --latency --min-delay 500 --expires 1h

  # Disable a specific chaos type
  vsb inbox chaos set --latency=false`,
	Args: cobra.NoArgs,
	RunE: runChaosSet,
}

// Flag variables
var (
	// Global
	chaosExpires string

	// Latency flags
	chaosLatency     bool
	chaosMinDelay    int
	chaosMaxDelay    int
	chaosNoJitter    bool
	chaosProbability float64

	// Connection drop flags
	chaosConnectionDrop bool
	chaosDropProb       float64
	chaosAbrupt         bool

	// Random error flags
	chaosRandomError bool
	chaosErrorRate   float64
	chaosErrorTypes  []string

	// Greylist flags
	chaosGreylist    bool
	chaosRetryWindow int
	chaosMaxAttempts int
	chaosTrackBy     string

	// Blackhole flags
	chaosBlackhole       bool
	chaosTriggerWebhooks bool
)

func init() {
	chaosCmd.AddCommand(chaosSetCmd)

	// Global flags
	chaosSetCmd.Flags().StringVar(&chaosExpires, "expires", "",
		"Auto-disable after duration (e.g., 1h, 30m) or timestamp")

	// Latency flags
	chaosSetCmd.Flags().BoolVar(&chaosLatency, "latency", false,
		"Enable latency injection")
	chaosSetCmd.Flags().IntVar(&chaosMinDelay, "min-delay", 500,
		"Minimum delay in milliseconds")
	chaosSetCmd.Flags().IntVar(&chaosMaxDelay, "max-delay", 10000,
		"Maximum delay in milliseconds")
	chaosSetCmd.Flags().BoolVar(&chaosNoJitter, "no-jitter", false,
		"Use fixed delay (max-delay) instead of random")
	chaosSetCmd.Flags().Float64Var(&chaosProbability, "probability", 1.0,
		"Probability of applying latency (0.0-1.0)")

	// Connection drop flags
	chaosSetCmd.Flags().BoolVar(&chaosConnectionDrop, "connection-drop", false,
		"Enable connection dropping")
	chaosSetCmd.Flags().Float64Var(&chaosDropProb, "drop-probability", 1.0,
		"Probability of dropping (0.0-1.0)")
	chaosSetCmd.Flags().BoolVar(&chaosAbrupt, "abrupt", false,
		"Use abrupt close (RST) instead of graceful (FIN)")

	// Random error flags
	chaosSetCmd.Flags().BoolVar(&chaosRandomError, "random-error", false,
		"Enable random error generation")
	chaosSetCmd.Flags().Float64Var(&chaosErrorRate, "error-rate", 0.1,
		"Probability of returning error (0.0-1.0)")
	chaosSetCmd.Flags().StringSliceVar(&chaosErrorTypes, "error-types", []string{"temporary"},
		"Error types: temporary, permanent (repeatable)")

	// Greylist flags
	chaosSetCmd.Flags().BoolVar(&chaosGreylist, "greylist", false,
		"Enable greylisting simulation")
	chaosSetCmd.Flags().IntVar(&chaosRetryWindow, "retry-window", 300000,
		"Retry window in milliseconds")
	chaosSetCmd.Flags().IntVar(&chaosMaxAttempts, "max-attempts", 2,
		"Attempts before accepting")
	chaosSetCmd.Flags().StringVar(&chaosTrackBy, "track-by", "ip_sender",
		"Track by: ip, sender, ip_sender")

	// Blackhole flags
	chaosSetCmd.Flags().BoolVar(&chaosBlackhole, "blackhole", false,
		"Enable blackhole mode")
	chaosSetCmd.Flags().BoolVar(&chaosTriggerWebhooks, "trigger-webhooks", false,
		"Still trigger webhooks in blackhole mode")
}

func runChaosSet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	jsonMode := cliutil.GetOutput(cmd) == "json"

	// Check if chaos is enabled on server
	if err := checkChaosEnabled(); err != nil {
		return err
	}

	// Validate flags
	if err := validateChaosFlags(cmd); err != nil {
		return err
	}

	// Build config
	cfg, err := buildChaosConfig(cmd)
	if err != nil {
		return err
	}

	// Load inbox
	inbox, cleanup, err := cliutil.LoadAndImportInbox(ctx, ChaosInboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	if !jsonMode {
		fmt.Println(styles.MutedStyle.Render("Setting chaos configuration..."))
	}

	// Set chaos config
	result, err := inbox.SetChaosConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to set chaos config: %w", err)
	}

	// JSON output
	if jsonMode {
		return cliutil.OutputJSON(cliutil.ChaosConfigFullJSON(result))
	}

	// Pretty output
	fmt.Println(styles.PassStyle.Render("Chaos configuration updated"))
	printChaosConfig(result, inbox.EmailAddress())
	return nil
}

func validateChaosFlags(cmd *cobra.Command) error {
	// Check at least one chaos type is set (changed implies the flag was provided)
	latencySet := cmd.Flags().Changed("latency")
	connDropSet := cmd.Flags().Changed("connection-drop")
	randomErrorSet := cmd.Flags().Changed("random-error")
	greylistSet := cmd.Flags().Changed("greylist")
	blackholeSet := cmd.Flags().Changed("blackhole")

	if !latencySet && !connDropSet && !randomErrorSet && !greylistSet && !blackholeSet {
		return fmt.Errorf("at least one chaos type must be enabled")
	}

	// Validate latency
	if chaosLatency {
		if chaosMinDelay < 0 {
			return fmt.Errorf("--min-delay must be non-negative")
		}
		if chaosMaxDelay < 0 {
			return fmt.Errorf("--max-delay must be non-negative")
		}
		if chaosMinDelay > chaosMaxDelay {
			return fmt.Errorf("--min-delay must not be greater than --max-delay")
		}
		if chaosProbability < 0 || chaosProbability > 1 {
			return fmt.Errorf("--probability must be between 0.0 and 1.0")
		}
	}

	// Validate connection drop
	if chaosConnectionDrop {
		if chaosDropProb < 0 || chaosDropProb > 1 {
			return fmt.Errorf("--drop-probability must be between 0.0 and 1.0")
		}
	}

	// Validate random error
	if chaosRandomError {
		if chaosErrorRate < 0 || chaosErrorRate > 1 {
			return fmt.Errorf("--error-rate must be between 0.0 and 1.0")
		}
		validTypes := map[string]bool{"temporary": true, "permanent": true}
		for _, t := range chaosErrorTypes {
			if !validTypes[t] {
				return fmt.Errorf("--error-types must be 'temporary' or 'permanent'")
			}
		}
	}

	// Validate greylist
	if chaosGreylist {
		validTrackBy := map[string]bool{"ip": true, "sender": true, "ip_sender": true}
		if !validTrackBy[chaosTrackBy] {
			return fmt.Errorf("--track-by must be 'ip', 'sender', or 'ip_sender'")
		}
		if chaosMaxAttempts < 1 {
			return fmt.Errorf("--max-attempts must be at least 1")
		}
		if chaosRetryWindow < 0 {
			return fmt.Errorf("--retry-window must be non-negative")
		}
	}

	return nil
}

func buildChaosConfig(cmd *cobra.Command) (*vaultsandbox.ChaosConfig, error) {
	cfg := &vaultsandbox.ChaosConfig{
		Enabled: true,
	}

	// Parse expiry
	if chaosExpires != "" {
		expiresAt, err := parseExpiry(chaosExpires)
		if err != nil {
			return nil, fmt.Errorf("invalid --expires value: %w", err)
		}
		cfg.ExpiresAt = &expiresAt
	}

	// Latency
	if cmd.Flags().Changed("latency") {
		cfg.Latency = &vaultsandbox.LatencyConfig{
			Enabled:     chaosLatency,
			MinDelayMs:  chaosMinDelay,
			MaxDelayMs:  chaosMaxDelay,
			Jitter:      !chaosNoJitter,
			Probability: chaosProbability,
		}
	}

	// Connection Drop
	if cmd.Flags().Changed("connection-drop") {
		cfg.ConnectionDrop = &vaultsandbox.ConnectionDropConfig{
			Enabled:     chaosConnectionDrop,
			Probability: chaosDropProb,
			Graceful:    !chaosAbrupt,
		}
	}

	// Random Error
	if cmd.Flags().Changed("random-error") {
		errorTypes := make([]vaultsandbox.RandomErrorType, len(chaosErrorTypes))
		for i, t := range chaosErrorTypes {
			switch strings.ToLower(t) {
			case "temporary":
				errorTypes[i] = vaultsandbox.RandomErrorTypeTemporary
			case "permanent":
				errorTypes[i] = vaultsandbox.RandomErrorTypePermanent
			}
		}
		cfg.RandomError = &vaultsandbox.RandomErrorConfig{
			Enabled:    chaosRandomError,
			ErrorRate:  chaosErrorRate,
			ErrorTypes: errorTypes,
		}
	}

	// Greylist
	if cmd.Flags().Changed("greylist") {
		trackBy := vaultsandbox.GreylistTrackByIPSender
		switch chaosTrackBy {
		case "ip":
			trackBy = vaultsandbox.GreylistTrackByIP
		case "sender":
			trackBy = vaultsandbox.GreylistTrackBySender
		}
		cfg.Greylist = &vaultsandbox.GreylistConfig{
			Enabled:       chaosGreylist,
			RetryWindowMs: chaosRetryWindow,
			MaxAttempts:   chaosMaxAttempts,
			TrackBy:       trackBy,
		}
	}

	// Blackhole
	if cmd.Flags().Changed("blackhole") {
		cfg.Blackhole = &vaultsandbox.BlackholeConfig{
			Enabled:         chaosBlackhole,
			TriggerWebhooks: chaosTriggerWebhooks,
		}
	}

	return cfg, nil
}

// parseExpiry parses a duration string (e.g., "1h", "30m") or timestamp.
func parseExpiry(s string) (time.Time, error) {
	// Try parsing as duration first
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(d), nil
	}

	// Try parsing as RFC3339 timestamp
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try parsing as a simpler format
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("must be a duration (e.g., 1h, 30m) or ISO timestamp")
}
