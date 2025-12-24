package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Purple   = lipgloss.Color("#7C3AED")
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
			Foreground(Purple).
			MarginBottom(1)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(Gray).
			MarginTop(1)

	// Email list item
	EmailItemStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DarkGray).
			Padding(0, 1).
			MarginBottom(1)

	EmailItemSelectedStyle = EmailItemStyle.
				BorderForeground(Purple)

	// Email fields
	SubjectStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White)

	FromStyle = lipgloss.NewStyle().
			Foreground(Gray)

	TimeStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Italic(true)

	// Labels/badges
	InboxLabelStyle = lipgloss.NewStyle().
			Background(Purple).
			Foreground(White).
			Padding(0, 1).
			MarginRight(1)

	UnreadBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(Green)

	// Preview pane
	PreviewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(1, 2)

	// Help
	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray)
)
