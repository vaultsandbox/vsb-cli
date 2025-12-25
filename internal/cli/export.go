package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/output"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var exportCmd = &cobra.Command{
	Use:   "export [email-address]",
	Short: "Export inbox with private keys",
	Long: `Export an inbox's keys and metadata to a JSON file for backup or sharing.

WARNING: The exported file contains your PRIVATE KEY. Anyone with this file
can read emails sent to your inbox. Handle it securely!

Use cases:
- Backup inbox before it expires
- Share inbox with CI/CD systems
- Transfer inbox to another machine/team member

Examples:
  vsb export                     # Export active inbox
  vsb export abc@vsb.com         # Export specific inbox
  vsb export --out ~/backup.json # Specify output file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runExport,
}

var (
	exportOut string
)

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportOut, "out", "",
		"Output file path (default: <email>.json)")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Use existing helpers
	ks, err := LoadKeystoreOrError()
	if err != nil {
		return err
	}

	// Get inbox (by arg or active)
	emailArg := ""
	if len(args) > 0 {
		emailArg = args[0]
	}
	stored, err := GetInbox(ks, emailArg)
	if err != nil {
		return err
	}

	// Check if expired
	if stored.ExpiresAt.Before(time.Now()) {
		warningBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Yellow).
			Padding(0, 1).
			Render(lipgloss.NewStyle().Foreground(styles.Yellow).Render("Warning: This inbox has expired"))
		fmt.Println(warningBox)
	}

	// Determine output file
	outPath := exportOut
	if outPath == "" {
		// Default to email.json in current directory
		safeEmail := sanitizeFilename(stored.Email)
		outPath = safeEmail + ".json"
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(outPath)
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(absPath); err == nil {
		return fmt.Errorf("file already exists: %s (use --out to specify different path)", absPath)
	}

	// Create export data
	exportData := ExportedInboxFile{
		Version:      1,
		EmailAddress: stored.Email,
		InboxHash:    stored.ID,
		Label:        stored.Label,
		ExpiresAt:    stored.ExpiresAt,
		ExportedAt:   time.Now(),
		Keys: ExportedKeys{
			KEMPrivate:  stored.Keys.KEMPrivate,
			KEMPublic:   stored.Keys.KEMPublic,
			ServerSigPK: stored.Keys.ServerSigPK,
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return err
	}

	// Write with secure permissions
	if err := os.WriteFile(absPath, data, 0600); err != nil {
		return err
	}

	// Security warning
	printExportWarning(absPath, stored.Email)

	return nil
}

// ExportedInboxFile is the file format for exported inboxes
type ExportedInboxFile struct {
	Version      int          `json:"version"`
	EmailAddress string       `json:"emailAddress"`
	InboxHash    string       `json:"inboxHash"`
	Label        string       `json:"label,omitempty"`
	ExpiresAt    time.Time    `json:"expiresAt"`
	ExportedAt   time.Time    `json:"exportedAt"`
	Keys         ExportedKeys `json:"keys"`
}

type ExportedKeys struct {
	KEMPrivate  string `json:"kemPrivate"`
	KEMPublic   string `json:"kemPublic"`
	ServerSigPK string `json:"serverSigPk"`
}

func sanitizeFilename(email string) string {
	// Replace @ and . with underscores for safe filename
	result := ""
	for _, r := range email {
		if r == '@' || r == '.' {
			result += "_"
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			result += string(r)
		}
	}
	return result
}

func printExportWarning(path, email string) {
	warningStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Yellow)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Yellow).
		Padding(1, 2)

	warning := fmt.Sprintf(`%s

This file contains your PRIVATE KEY for:
  %s

Anyone with this file can read emails sent to this inbox.
Keep it secure and do not commit it to version control!

File: %s`,
		warningStyle.Render("SECURITY WARNING"),
		email,
		path)

	fmt.Println()
	fmt.Println(boxStyle.Render(warning))
	fmt.Println()
	fmt.Println(output.PrintSuccess("Export complete"))
}
