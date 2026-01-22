package inbox

import (
	"fmt"

	"github.com/spf13/cobra"
)

// chaosCmd is the inbox chaos parent command
var chaosCmd = &cobra.Command{
	Use:   "chaos",
	Short: "Manage chaos engineering configuration",
	Long: `Configure chaos engineering scenarios for a specific inbox.

Chaos commands allow you to inject various failure scenarios for testing
email delivery resilience. Available chaos types:
  - Latency injection (--latency)
  - Connection drops (--connection-drop)
  - Random errors (--random-error)
  - Greylisting simulation (--greylist)
  - Blackhole mode (--blackhole)

Note: Chaos must be enabled on the server for these commands to work.`,
	RunE: runChaos,
}

// ChaosInboxFlag is used by chaos subcommands to specify which inbox to operate on
var ChaosInboxFlag string

func init() {
	Cmd.AddCommand(chaosCmd)

	// Persistent flag for all chaos subcommands
	chaosCmd.PersistentFlags().StringVar(&ChaosInboxFlag, "inbox", "",
		"Use specific inbox (default: active)")
}

func runChaos(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	return cmd.Help()
}
