package cliutil

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaultsandbox/vsb-cli/internal/config"
)

// TLS extraction regexes for parsing Received headers
var (
	tlsVersionRegex = regexp.MustCompile(`version=(TLSv[\d.]+)`)
	tlsCipherRegex  = regexp.MustCompile(`cipher=(\S+)\)`)
)

// Time format constants for consistent display
const (
	TimeFormatShort    = "2006-01-02 15:04"
	TimeFormatFull     = "2006-01-02 15:04:05"
	TimeFormatWithZone = "2006-01-02 15:04:05 MST"
	TimeFormatTimeOnly = "15:04:05"
)

// NoSubject is the fallback text for emails without a subject.
const NoSubject = "(no subject)"

// SubjectOrDefault returns the subject, or NoSubject if empty.
func SubjectOrDefault(subject string) string {
	if subject == "" {
		return NoSubject
	}
	return subject
}

// GetOutput returns the output format with priority: flag > env > config > default.
func GetOutput(cmd *cobra.Command) string {
	if flag := cmd.Flag("output"); flag != nil && flag.Changed {
		return flag.Value.String()
	}
	return config.GetDefaultOutput()
}

// OutputJSON marshals v to indented JSON and prints it to stdout.
func OutputJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// SanitizeFilename replaces unsafe characters for use in filenames.
func SanitizeFilename(email string) string {
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

// FormatDuration formats a duration as a human-readable string (e.g., "5m", "2h", "3d").
func FormatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// FormatRelativeTime formats a time as a human-readable relative string (e.g., "just now", "5m ago").
func FormatRelativeTime(t time.Time) string {
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

// ExtractTLSVersion parses TLS version from a Received header.
// Returns empty string if not found.
func ExtractTLSVersion(received string) string {
	if match := tlsVersionRegex.FindStringSubmatch(received); len(match) > 1 {
		return match[1]
	}
	return ""
}

// ExtractTLSCipher parses TLS cipher suite from a Received header.
// Returns empty string if not found.
func ExtractTLSCipher(received string) string {
	if match := tlsCipherRegex.FindStringSubmatch(received); len(match) > 1 {
		return match[1]
	}
	return ""
}

// IsExpired checks if a time is in the past.
func IsExpired(expiresAt time.Time) bool {
	return expiresAt.Before(time.Now())
}

// FormatExpiry returns remaining time as a formatted string, or "expired" if past.
func FormatExpiry(expiresAt time.Time) string {
	if IsExpired(expiresAt) {
		return "expired"
	}
	remaining := time.Until(expiresAt).Round(time.Minute)
	return FormatDuration(remaining)
}
