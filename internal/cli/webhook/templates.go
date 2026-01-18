package webhook

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available built-in templates",
	Long:  `Display all available built-in webhook templates.`,
	RunE:  runTemplates,
}

func init() {
	Cmd.AddCommand(templatesCmd)
}

func runTemplates(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Get templates
	templates, err := client.GetWebhookTemplates(ctx)
	if err != nil {
		return fmt.Errorf("failed to get templates: %w", err)
	}

	// JSON output
	if cliutil.GetOutput(cmd) == "json" {
		var result []map[string]interface{}
		for _, t := range templates {
			result = append(result, cliutil.WebhookTemplateJSON(t))
		}
		return cliutil.OutputJSON(result)
	}

	// Pretty output
	fmt.Println()
	fmt.Println(styles.TitleStyle.Render("Available Webhook Templates"))
	fmt.Println()

	table := cliutil.NewTable(
		cliutil.Column{Header: "TEMPLATE", Width: 12},
		cliutil.Column{Header: "DESCRIPTION"},
	).WithIndent("   ")

	table.PrintHeader()

	for _, t := range templates {
		table.PrintRow(t.Value, t.Label)
	}

	fmt.Println()
	return nil
}
