package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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
)

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
