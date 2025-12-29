package email

import (
	"context"
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/vsb-cli/internal/cliutil"
	"github.com/vaultsandbox/vsb-cli/internal/files"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

var attachmentCmd = &cobra.Command{
	Use:   "attachment [email-id]",
	Short: "List and download email attachments",
	Long: `List and download attachments from an email.

By default, lists all attachments with their index, filename, type, and size.
Use --save to download a specific attachment by its index number.
Use --all to download all attachments at once.

Examples:
  vsb email attachment              # List attachments from latest email
  vsb email attachment abc123       # List attachments from specific email
  vsb email attachment --save 1     # Download first attachment
  vsb email attachment --all        # Download all attachments
  vsb email attachment --all -d ./downloads  # Download to specific directory
  vsb email attachment -o json      # JSON output for scripting`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAttachment,
}

var (
	attachmentSave int
	attachmentAll  bool
	attachmentDir  string
)

func init() {
	Cmd.AddCommand(attachmentCmd)

	attachmentCmd.Flags().IntVarP(&attachmentSave, "save", "s", 0,
		"Download the Nth attachment (1=first, 0=don't download)")
	attachmentCmd.Flags().BoolVarP(&attachmentAll, "all", "a", false,
		"Download all attachments")
	attachmentCmd.Flags().StringVarP(&attachmentDir, "dir", "d", ".",
		"Directory to save attachments (default: current directory)")
}

func runAttachment(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	emailID := cliutil.GetArg(args, 0, "")

	// Use shared helper
	email, _, cleanup, err := cliutil.GetEmailByIDOrLatest(ctx, emailID, InboxFlag)
	if err != nil {
		return err
	}
	defer cleanup()

	// Check for attachments
	if len(email.Attachments) == 0 {
		if cliutil.GetOutput(cmd) == "json" {
			return cliutil.OutputJSON([]struct{}{})
		}
		fmt.Println("No attachments found in email")
		return nil
	}

	// Download all attachments
	if attachmentAll {
		return downloadAllAttachments(email.Attachments)
	}

	// Download specific attachment
	if attachmentSave > 0 {
		if attachmentSave > len(email.Attachments) {
			return fmt.Errorf("attachment index %d out of range (1-%d)", attachmentSave, len(email.Attachments))
		}
		att := email.Attachments[attachmentSave-1]
		return downloadAttachment(att.Filename, att.Content)
	}

	// Default: list all attachments
	if cliutil.GetOutput(cmd) == "json" {
		// Build JSON-friendly output (without binary content)
		type attachmentInfo struct {
			Index       int    `json:"index"`
			Filename    string `json:"filename"`
			ContentType string `json:"contentType"`
			Size        int    `json:"size"`
			Checksum    string `json:"checksum,omitempty"`
		}
		infos := make([]attachmentInfo, len(email.Attachments))
		for i, att := range email.Attachments {
			infos[i] = attachmentInfo{
				Index:       i + 1,
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Size:        att.Size,
				Checksum:    att.Checksum,
			}
		}
		return cliutil.OutputJSON(infos)
	}

	fmt.Printf("Attachments (%d):\n\n", len(email.Attachments))
	for i, att := range email.Attachments {
		fmt.Printf("  %d. %s\n", i+1, att.Filename)
		fmt.Printf("     Type: %s\n", att.ContentType)
		fmt.Printf("     Size: %s\n", humanize.Bytes(uint64(att.Size)))
		if i < len(email.Attachments)-1 {
			fmt.Println()
		}
	}
	fmt.Printf("\nUse --save N to download an attachment, or --all to download all\n")
	return nil
}

func downloadAttachment(filename string, content []byte) error {
	path, err := files.SaveFile(attachmentDir, filename, content)
	if err != nil {
		return err
	}

	fmt.Println(styles.PassStyle.Render(fmt.Sprintf("✓ Saved: %s (%s)", path, humanize.Bytes(uint64(len(content))))))
	return nil
}

func downloadAllAttachments(attachments []vaultsandbox.Attachment) error {
	saved := 0
	for _, att := range attachments {
		if err := downloadAttachment(att.Filename, att.Content); err != nil {
			fmt.Println(styles.FailStyle.Render(fmt.Sprintf("✗ Failed to save %s: %v", att.Filename, err)))
		} else {
			saved++
		}
	}

	if saved == len(attachments) {
		fmt.Printf("\n%s\n", styles.PassStyle.Render(fmt.Sprintf("✓ Downloaded all %d attachments", saved)))
	} else {
		fmt.Printf("\nDownloaded %d of %d attachments\n", saved, len(attachments))
	}
	return nil
}
