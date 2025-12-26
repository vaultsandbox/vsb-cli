package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import inbox from export file",
	Long: `Import an inbox from a previously exported JSON file.

This adds the inbox to your local keystore and optionally verifies
it's still valid on the server.

Examples:
  vsb import backup.json      # Import and verify
  vsb import backup.json -l   # Skip server verification
  vsb import backup.json -f   # Force overwrite existing`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

var (
	importLocal bool
	importForce bool
)

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().BoolVarP(&importLocal, "local", "l", false,
		"Skip server verification")
	importCmd.Flags().BoolVarP(&importForce, "force", "f", false,
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
	var exported config.ExportedInboxFile
	if err := json.Unmarshal(data, &exported); err != nil {
		return fmt.Errorf("invalid export file format: %w", err)
	}

	// Validate version
	if exported.Version != 1 {
		return fmt.Errorf("unsupported export file version: %d", exported.Version)
	}

	// Check if expired
	if exported.ExpiresAt.Before(time.Now()) {
		errorBox := styles.ErrorBoxStyle.Render(styles.ErrorTitleStyle.Render("Error: This inbox has expired"))
		fmt.Println(errorBox)
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
		fmt.Println(styles.MutedStyle.Render("• Verifying with server..."))

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
			fmt.Println(styles.MutedStyle.Render("• Warning: Could not verify sync status"))
		} else {
			fmt.Println(styles.PassStyle.Render(fmt.Sprintf("✓ Inbox verified: %d emails", status.EmailCount)))
		}
	}

	// Save to keystore
	stored := exported.ToStoredInbox()

	if err := keystore.AddInbox(stored); err != nil {
		return err
	}

	// Success output
	printImportSuccess(stored)

	return nil
}

func printImportSuccess(inbox config.StoredInbox) {
	remaining := time.Until(inbox.ExpiresAt).Round(time.Hour)

	content := fmt.Sprintf(`%s

Address:  %s
Expires:  %s

This inbox is now your active inbox.
Run 'vsb' to see emails.`,
		styles.SuccessTitleStyle.Render("Import Complete"),
		inbox.Email,
		remaining.String())

	fmt.Println()
	fmt.Println(styles.SuccessBoxStyle.Render(content))
	fmt.Println()
}

