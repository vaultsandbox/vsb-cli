package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/output"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import inbox from export file",
	Long: `Import an inbox from a previously exported JSON file.

This adds the inbox to your local keystore and optionally verifies
it's still valid on the server.

Examples:
  vsb import backup.json          # Import and verify
  vsb import backup.json --local  # Skip server verification
  vsb import backup.json --label "shared-inbox"`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

var (
	importLocal bool
	importLabel string
	importForce bool
)

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().BoolVar(&importLocal, "local", false,
		"Skip server verification")
	importCmd.Flags().StringVar(&importLabel, "label", "",
		"Override the label for imported inbox")
	importCmd.Flags().BoolVar(&importForce, "force", false,
		"Overwrite existing inbox with same email")
}

func runImport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	filePath := args[0]

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var exported ExportedInboxFile
	if err := json.Unmarshal(data, &exported); err != nil {
		return fmt.Errorf("invalid export file format: %w", err)
	}

	// Validate version
	if exported.Version != 1 {
		return fmt.Errorf("unsupported export file version: %d", exported.Version)
	}

	// Check if expired
	if exported.ExpiresAt.Before(time.Now()) {
		warningBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Red).
			Padding(0, 1).
			Render(lipgloss.NewStyle().Foreground(styles.Red).Render("Error: This inbox has expired"))
		fmt.Println(warningBox)
		return fmt.Errorf("inbox expired on %s", exported.ExpiresAt.Format("2006-01-02"))
	}

	// Use existing helper
	keystore, err := LoadKeystoreOrError()
	if err != nil {
		return err
	}

	// Check for existing inbox
	existing, _ := keystore.GetInbox(exported.EmailAddress)
	if existing != nil && !importForce {
		return fmt.Errorf("inbox already exists: %s (use --force to overwrite)", exported.EmailAddress)
	}

	// Server verification (unless --local)
	if !importLocal {
		fmt.Println(output.PrintInfo("Verifying with server..."))

		client, err := config.NewClient()
		if err != nil {
			return err
		}
		defer client.Close()

		// Try to import into SDK to verify
		sdkExport := &vaultsandbox.ExportedInbox{
			EmailAddress: exported.EmailAddress,
			ExpiresAt:    exported.ExpiresAt,
			InboxHash:    exported.InboxHash,
			ServerSigPk:  exported.Keys.ServerSigPK,
			PublicKeyB64: exported.Keys.KEMPublic,
			SecretKeyB64: exported.Keys.KEMPrivate,
			ExportedAt:   exported.ExportedAt,
		}

		inbox, err := client.ImportInbox(ctx, sdkExport)
		if err != nil {
			return fmt.Errorf("server verification failed: %w", err)
		}

		// Check sync status
		status, err := inbox.GetSyncStatus(ctx)
		if err != nil {
			fmt.Println(output.PrintInfo("Warning: Could not verify sync status"))
		} else {
			fmt.Println(output.PrintSuccess(fmt.Sprintf("Inbox verified: %d emails", status.EmailCount)))
		}
	}

	// Determine label
	label := exported.Label
	if importLabel != "" {
		label = importLabel
	}

	// Save to keystore
	stored := config.StoredInbox{
		Email:     exported.EmailAddress,
		ID:        exported.InboxHash,
		Label:     label,
		CreatedAt: exported.ExportedAt,
		ExpiresAt: exported.ExpiresAt,
		Keys: config.InboxKeys{
			KEMPrivate:  exported.Keys.KEMPrivate,
			KEMPublic:   exported.Keys.KEMPublic,
			ServerSigPK: exported.Keys.ServerSigPK,
		},
	}

	if err := keystore.AddInbox(stored); err != nil {
		return err
	}

	// Success output
	printImportSuccess(stored)

	return nil
}

func printImportSuccess(inbox config.StoredInbox) {
	remaining := time.Until(inbox.ExpiresAt).Round(time.Hour)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Green).
		Padding(1, 2)

	successStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Green)

	content := fmt.Sprintf(`%s

Address:  %s
Label:    %s
Expires:  %s

This inbox is now your active inbox.
Run 'vsb watch' to see emails.`,
		successStyle.Render("Import Complete"),
		inbox.Email,
		orDefault(inbox.Label, "(none)"),
		remaining.String())

	fmt.Println()
	fmt.Println(boxStyle.Render(content))
	fmt.Println()
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
