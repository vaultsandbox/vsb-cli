package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/output"
)

var inboxCreateCmd = &cobra.Command{
	Use:   "create [label]",
	Short: "Create a new temporary inbox",
	Long: `Create a new temporary email inbox with quantum-safe encryption.

The inbox uses ML-KEM-768 for key encapsulation and ML-DSA-65 for signatures.
Your private key never leaves your machine - all decryption happens locally.

Examples:
  vsb inbox create
  vsb inbox create auth-tests
  vsb inbox create --ttl 1h`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInboxCreate,
}

var (
	createTTL string
)

func init() {
	inboxCmd.AddCommand(inboxCreateCmd)

	inboxCreateCmd.Flags().StringVar(&createTTL, "ttl", "24h",
		"Inbox lifetime (e.g., 1h, 24h, 7d)")
}

func runInboxCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get optional label
	label := ""
	if len(args) > 0 {
		label = args[0]
	}

	// Parse TTL
	ttl, err := time.ParseDuration(createTTL)
	if err != nil {
		return fmt.Errorf("invalid TTL format: %w", err)
	}

	// Show spinner
	fmt.Println(output.PrintInfo("Generating quantum-safe keys..."))

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Create inbox with SDK
	fmt.Println(output.PrintInfo("Registering with VaultSandbox..."))

	inbox, err := client.CreateInbox(ctx, vaultsandbox.WithTTL(ttl))
	if err != nil {
		return fmt.Errorf("failed to create inbox: %w", err)
	}

	// Export inbox data for keystore
	exported := inbox.Export()

	// Save to keystore
	keystore, err := config.LoadKeystore()
	if err != nil {
		return fmt.Errorf("failed to load keystore: %w", err)
	}

	stored := config.StoredInboxFromExport(exported, label)
	if err := keystore.AddInbox(stored); err != nil {
		return fmt.Errorf("failed to save inbox: %w", err)
	}

	// Pretty output
	printInboxCreated(stored)

	return nil
}

func printInboxCreated(inbox config.StoredInbox) {
	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#10B981")).
		Render("Inbox Ready!")

	// Email address box
	emailBox := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C3AED")).
		Padding(0, 2).
		Render(inbox.Email)

	// Details
	expiry := inbox.ExpiresAt.Sub(time.Now()).Round(time.Hour)
	expiryStr := fmt.Sprintf("%v", expiry)

	labelStr := inbox.Label
	if labelStr == "" {
		labelStr = "(none)"
	}

	details := fmt.Sprintf(`

  Address:  %s
  Label:    %s
  Security: ML-KEM-768 (Quantum-Safe)
  Expires:  %s

Run 'vsb watch' to see emails arrive live.`,
		emailBox, labelStr, expiryStr)

	// Box it all
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1, 2).
		Render(title + details)

	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}
