package output

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	Primary = lipgloss.Color("#1cc2e3")
	Success = lipgloss.Color("#10B981") // Green
	Warning = lipgloss.Color("#F59E0B") // Amber
	Error   = lipgloss.Color("#EF4444") // Red
	Muted   = lipgloss.Color("#6B7280") // Gray

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Success)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Error)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	// Box for important info
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Email address highlight
	EmailStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(Primary).
			Padding(0, 1)
)

// PrintSuccess prints a success message with checkmark
func PrintSuccess(msg string) string {
	return SuccessStyle.Render("✓ " + msg)
}

// PrintError prints an error message
func PrintError(msg string) string {
	return ErrorStyle.Render("✗ " + msg)
}

// PrintInfo prints an info message
func PrintInfo(msg string) string {
	return MutedStyle.Render("• " + msg)
}
