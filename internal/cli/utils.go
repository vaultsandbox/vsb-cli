package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// getOutput returns the output format with priority: flag > env > config > default.
func getOutput(cmd *cobra.Command) string {
	if flag := cmd.Flag("output"); flag != nil && flag.Changed {
		return flag.Value.String()
	}
	return config.GetDefaultOutput()
}

// outputJSON marshals v to indented JSON and prints it to stdout.
func outputJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// sanitizeFilename replaces unsafe characters for use in filenames.
func sanitizeFilename(email string) string {
	var b strings.Builder
	b.Grow(len(email))
	for _, r := range email {
		if r == '@' || r == '.' {
			b.WriteByte('_')
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatDuration formats a duration as a human-readable string (e.g., "5m", "2h", "3d").
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// formatRelativeTime formats a time as a human-readable relative string (e.g., "just now", "5m ago").
func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}
