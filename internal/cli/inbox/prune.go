package inbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove inboxes that no longer exist on the server",
	Long: `Remove inboxes from the local keystore that no longer exist on the server.

This is useful when inboxes have been deleted server-side but still exist locally,
causing errors like "API error 404: Not Found" when starting the dashboard.

Examples:
  vsb inbox prune`,
	RunE: runPrune,
}

var (
	pruneDryRun bool
)

func init() {
	Cmd.AddCommand(pruneCmd)

	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false,
		"Show what would be pruned without removing anything")
}

func runPrune(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ks, err := cliutil.LoadKeystoreOrError()
	if err != nil {
		return err
	}

	storedInboxes := ks.ListInboxes()
	if len(storedInboxes) == 0 {
		fmt.Println(styles.MutedStyle.Render("No inboxes to prune"))
		return nil
	}

	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	var toPrune []string

	// Check each inbox against the server
	for _, stored := range storedInboxes {
		exported := stored.ToExportedInbox()
		_, err := client.ImportInbox(ctx, exported)
		if err != nil {
			// Check if it's a 404 error (inbox not found on server)
			if strings.Contains(err.Error(), "404") {
				toPrune = append(toPrune, stored.Email)
				if pruneDryRun {
					fmt.Println(styles.WarnStyle.Render(fmt.Sprintf("Would prune: %s", stored.Email)))
				}
			} else {
				// Other error - warn but don't prune
				fmt.Println(styles.FailStyle.Render(fmt.Sprintf("✗ Error checking %s: %v", stored.Email, err)))
			}
		}
	}

	if len(toPrune) == 0 {
		fmt.Println(styles.PassStyle.Render("✓ All inboxes are valid"))
		return nil
	}

	if pruneDryRun {
		fmt.Println(styles.MutedStyle.Render(fmt.Sprintf("\n%d inbox(es) would be pruned", len(toPrune))))
		return nil
	}

	// Remove invalid inboxes
	for _, email := range toPrune {
		if err := ks.RemoveInbox(email); err != nil {
			fmt.Println(styles.FailStyle.Render(fmt.Sprintf("✗ Failed to remove %s: %v", email, err)))
			continue
		}
		fmt.Println(styles.PassStyle.Render(fmt.Sprintf("✓ Pruned: %s", email)))
	}

	fmt.Println(styles.MutedStyle.Render(fmt.Sprintf("\n%d inbox(es) pruned", len(toPrune))))
	return nil
}
