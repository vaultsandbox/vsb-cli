package inbox

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/config"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new temporary inbox",
	Long: `Create a new temporary encrypted email inbox.

The inbox uses ML-KEM-768 for key encapsulation and ML-DSA-65 for signatures.
Your private key never leaves your machine - all decryption happens locally.

Examples:
  vsb inbox create
  vsb inbox create --ttl 1h
  vsb inbox create --ttl 7d`,
	RunE: runCreate,
}

var (
	createTTL string
)

func init() {
	Cmd.AddCommand(createCmd)

	createCmd.Flags().StringVar(&createTTL, "ttl", "24h",
		"Inbox lifetime (e.g., 1h, 24h, 7d)")
}

func runCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	jsonMode := cliutil.GetOutput(cmd) == "json"

	// Parse TTL
	ttl, err := parseTTL(createTTL)
	if err != nil {
		return fmt.Errorf("invalid TTL format: %w", err)
	}

	// Show progress (not in JSON mode)
	if !jsonMode {
		fmt.Println(styles.MutedStyle.Render("• Generating keys..."))
	}

	// Create client
	client, err := config.NewClient()
	if err != nil {
		return err
	}
	defer client.Close()

	// Create inbox with SDK
	if !jsonMode {
		fmt.Println(styles.MutedStyle.Render("• Registering with VaultSandbox..."))
	}

	inbox, err := client.CreateInbox(ctx, vaultsandbox.WithTTL(ttl))
	if err != nil {
		return fmt.Errorf("failed to create inbox: %w", err)
	}

	// Export inbox data for keystore
	exported := inbox.Export()

	// Save to keystore
	keystore, err := cliutil.LoadKeystoreOrError()
	if err != nil {
		return err
	}

	stored := config.StoredInboxFromExport(exported)
	if err := keystore.AddInbox(stored); err != nil {
		return fmt.Errorf("failed to save inbox: %w", err)
	}

	// Output
	if jsonMode {
		data := map[string]interface{}{
			"email":     stored.Email,
			"expiresAt": stored.ExpiresAt.Format(time.RFC3339),
			"createdAt": stored.CreatedAt.Format(time.RFC3339),
		}
		return cliutil.OutputJSON(data)
	} else {
		printInboxCreated(stored)
	}

	return nil
}

func printInboxCreated(inbox config.StoredInbox) {
	// Title
	title := styles.SuccessTitleStyle.Render("Inbox Ready!")

	// Email address box
	emailBox := styles.EmailBoxStyle.Render(inbox.Email)

	// Details
	expiry := time.Until(inbox.ExpiresAt).Round(time.Hour)
	expiryStr := fmt.Sprintf("%v", expiry)

	details := fmt.Sprintf(`

  Address:  %s
  Expires:  %s

Run 'vsb' to see emails arrive live.`,
		emailBox, expiryStr)

	// Box it all
	box := styles.SuccessBoxStyle.Render(title + details)

	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}

func parseTTL(s string) (time.Duration, error) {
	// Handle days suffix (not supported by time.ParseDuration)
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		n, err := strconv.Atoi(days)
		if err != nil {
			return 0, fmt.Errorf("invalid day value: %s", days)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
