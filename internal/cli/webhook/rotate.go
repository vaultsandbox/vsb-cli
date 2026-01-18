package webhook

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var rotateCmd = &cobra.Command{
	Use:   "rotate <webhook-id>",
	Short: "Rotate webhook signing secret",
	Long: `Rotate the signing secret for a webhook.

The previous secret will remain valid for a grace period (24 hours) to allow
you to update your endpoint without downtime.`,
	Args: cobra.ExactArgs(1),
	RunE: runRotate,
}

var rotateForce bool

func init() {
	Cmd.AddCommand(rotateCmd)

	rotateCmd.Flags().BoolVarP(&rotateForce, "force", "f", false,
		"Skip confirmation prompt")
}

func runRotate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	jsonMode := cliutil.GetOutput(cmd) == "json"
	webhookID := args[0]

	// Confirmation
	if !rotateForce && !jsonMode {
		fmt.Printf("Rotate secret for webhook %s? [y/N]: ", webhookID)
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	// Show progress
	if !jsonMode {
		fmt.Println(styles.MutedStyle.Render("• Rotating secret..."))
	}

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Rotate secret
	resp, err := client.Admin().RotateWebhookSecret(ctx, webhookID)
	if err != nil {
		return fmt.Errorf("failed to rotate secret: %w", err)
	}

	// JSON output
	if jsonMode {
		return cliutil.OutputJSON(cliutil.RotateSecretJSON(resp))
	}

	// Pretty output
	printRotateResult(resp)
	return nil
}

func printRotateResult(resp *vaultsandbox.RotateSecretResponse) {
	title := styles.SuccessTitleStyle.Render("Secret Rotated Successfully")

	details := fmt.Sprintf(`

  Webhook ID:      %s
  New Secret:      %s
`,
		resp.ID,
		resp.Secret,
	)

	var graceInfo string
	if resp.PreviousSecretValidUntil != nil {
		graceInfo = fmt.Sprintf(`
  Previous secret valid until: %s (24h grace period)
`,
			resp.PreviousSecretValidUntil.Format(cliutil.TimeFormatWithZone),
		)
	}

	warning := styles.WarnStyle.Render("  Update your endpoint with the new secret before the grace period expires.")

	box := styles.SuccessBoxStyle.Render(title + details + graceInfo + "\n" + warning)
	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}
