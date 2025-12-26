package styles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vaultsandbox/client-go/authresults"
)

var (
	// Colors
	Primary = lipgloss.Color("#1cc2e3")
	Green    = lipgloss.Color("#10B981")
	Red      = lipgloss.Color("#EF4444")
	Yellow   = lipgloss.Color("#F59E0B")
	Gray     = lipgloss.Color("#6B7280")
	DarkGray = lipgloss.Color("#374151")
	White    = lipgloss.Color("#FFFFFF")

	// App frame
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Gray).
			MarginTop(1)

	// Help
	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray)

	// Tabs
	TabStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Padding(0, 1)

	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Background(DarkGray).
			Padding(0, 1)

	// Active marker (for lists)
	ActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Green)

	// Expired/disabled items
	ExpiredStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Strikethrough(true)

	// Email box (for display)
	EmailBoxStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(Primary).
			Padding(0, 2)

	// Success box
	SuccessBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Success title
	SuccessTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Green)

	// Warning box
	WarningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Yellow).
			Padding(1, 2)

	// Warning title
	WarningTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Yellow)

	// Error box
	ErrorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Red).
			Padding(0, 1)

	// Error title
	ErrorTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Red)

	// Common result styles
	PassStyle = lipgloss.NewStyle().Bold(true).Foreground(Green)
	FailStyle = lipgloss.NewStyle().Bold(true).Foreground(Red)
	WarnStyle = lipgloss.NewStyle().Bold(true).Foreground(Yellow)

	// Label style for key-value displays
	LabelStyle = lipgloss.NewStyle().Foreground(Gray).Width(20)

	// Muted style for info messages
	MutedStyle = lipgloss.NewStyle().Foreground(Gray)
)

// ScoreStyle returns the appropriate style for a security score (0-100).
func ScoreStyle(score int) lipgloss.Style {
	if score < 60 {
		return FailStyle
	}
	if score < 80 {
		return WarnStyle
	}
	return PassStyle
}

// FormatAuthResult formats an authentication result (SPF/DKIM/DMARC) with appropriate styling.
func FormatAuthResult(result string) string {
	switch strings.ToLower(result) {
	case "pass":
		return PassStyle.Render("PASS")
	case "fail", "hardfail":
		return FailStyle.Render("FAIL")
	case "softfail", "none", "neutral":
		return WarnStyle.Render(strings.ToUpper(result))
	default:
		return result
	}
}

// Encryption label constant for consistent display across CLI and TUI
const EncryptionLabel = "ML-KEM-768 + AES-256-GCM"

// RenderAuthResults renders authentication results (SPF/DKIM/DMARC/ReverseDNS).
// If compact is true, details are shown in parentheses on the same line (TUI style).
// If compact is false, details are shown on separate indented lines (CLI style).
func RenderAuthResults(auth *authresults.AuthResults, labelStyle lipgloss.Style, compact bool) string {
	if auth == nil {
		return WarnStyle.Render("No authentication results available")
	}

	var lines []string

	// SPF
	if auth.SPF != nil {
		spfResult := FormatAuthResult(auth.SPF.Status)
		if compact {
			line := fmt.Sprintf("%s %s", labelStyle.Render("SPF:"), spfResult)
			if auth.SPF.Domain != "" {
				line += fmt.Sprintf(" (%s)", auth.SPF.Domain)
			}
			lines = append(lines, line)
		} else {
			lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("SPF:"), spfResult))
			if auth.SPF.Domain != "" {
				lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("  Domain:"), auth.SPF.Domain))
			}
		}
	}

	// DKIM (it's a slice)
	if len(auth.DKIM) > 0 {
		dkim := auth.DKIM[0]
		dkimResult := FormatAuthResult(dkim.Status)
		if compact {
			line := fmt.Sprintf("%s %s", labelStyle.Render("DKIM:"), dkimResult)
			if dkim.Domain != "" {
				line += fmt.Sprintf(" (%s)", dkim.Domain)
			}
			lines = append(lines, line)
		} else {
			lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("DKIM:"), dkimResult))
			if dkim.Selector != "" {
				lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("  Selector:"), dkim.Selector))
			}
			if dkim.Domain != "" {
				lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("  Domain:"), dkim.Domain))
			}
		}
	}

	// DMARC
	if auth.DMARC != nil {
		dmarcResult := FormatAuthResult(auth.DMARC.Status)
		if compact {
			line := fmt.Sprintf("%s %s", labelStyle.Render("DMARC:"), dmarcResult)
			if auth.DMARC.Policy != "" {
				line += fmt.Sprintf(" (policy: %s)", auth.DMARC.Policy)
			}
			lines = append(lines, line)
		} else {
			lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("DMARC:"), dmarcResult))
			if auth.DMARC.Policy != "" {
				lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("  Policy:"), auth.DMARC.Policy))
			}
		}
	}

	// Reverse DNS
	if auth.ReverseDNS != nil {
		rdnsResult := FormatAuthResult(auth.ReverseDNS.Status())
		if compact {
			line := fmt.Sprintf("%s %s", labelStyle.Render("Reverse DNS:"), rdnsResult)
			if auth.ReverseDNS.Hostname != "" {
				line += fmt.Sprintf(" (%s)", auth.ReverseDNS.Hostname)
			}
			lines = append(lines, line)
		} else {
			lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Reverse DNS:"), rdnsResult))
			if auth.ReverseDNS.Hostname != "" {
				lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("  Hostname:"), auth.ReverseDNS.Hostname))
			}
		}
	}

	return strings.Join(lines, "\n")
}
