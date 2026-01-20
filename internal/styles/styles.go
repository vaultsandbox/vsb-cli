package styles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	vaultsandbox "github.com/vaultsandbox/client-go"
	"github.com/vaultsandbox/client-go/authresults"
	"github.com/vaultsandbox/client-go/spamanalysis"
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

	// Section header for CLI reports
	SectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginTop(1)

	// Generic bordered box (neutral color)
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Common list/table styles
	IDStyle      = lipgloss.NewStyle().Foreground(Gray)
	SubjectStyle = lipgloss.NewStyle().Bold(true)
	FromStyle    = lipgloss.NewStyle().Foreground(Primary)
	TimeStyle    = lipgloss.NewStyle().Foreground(Gray)

	// Title style for sections
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	// Audit report title
	AuditTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(Primary).
			Padding(0, 2)

	// Base style for bold primary text (used by detail labels, list headers, selection markers)
	PrimaryBoldStyle = lipgloss.NewStyle().Bold(true).Foreground(Primary)

	// TUI detail view styles
	DetailLabelStyle   = PrimaryBoldStyle // Labels like "From:", "To:", "Subject:"
	DetailValueStyle   = lipgloss.NewStyle().Foreground(White)
	DetailSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(White).MarginTop(1)

	// TUI list item styles (links, attachments)
	ListLabelStyle    = PrimaryBoldStyle // Section headers like "Found X items:"
	ListSelectedStyle = PrimaryBoldStyle // Selection marker ">"
	ListSizeStyle     = lipgloss.NewStyle().Foreground(Gray)

	// Badge style for status indicators
	BadgeStyle = lipgloss.NewStyle().
			Foreground(White).
			Padding(0, 1)
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
	case "skipped":
		return MutedStyle.Render("SKIPPED")
	default:
		return result
	}
}

// Table column widths for consistent formatting across list commands
const (
	ColWidthID      = 8
	ColWidthEmail   = 38
	ColWidthSubject = 30
	ColWidthFrom    = 25
)

// authField represents a single authentication result with its details.
type authField struct {
	label   string
	status  string
	details []authDetail
}

type authDetail struct {
	label string
	value string
}


// buildAuthFields extracts authentication fields from AuthResults.
func buildAuthFields(auth *authresults.AuthResults) []authField {
	if auth == nil {
		return nil
	}

	var fields []authField

	if auth.SPF != nil {
		f := authField{label: "SPF:", status: auth.SPF.Result}
		if auth.SPF.Domain != "" {
			f.details = append(f.details, authDetail{"Domain:", auth.SPF.Domain})
		}
		fields = append(fields, f)
	}

	if len(auth.DKIM) > 0 {
		dkim := auth.DKIM[0]
		f := authField{label: "DKIM:", status: dkim.Result}
		if dkim.Selector != "" {
			f.details = append(f.details, authDetail{"Selector:", dkim.Selector})
		}
		if dkim.Domain != "" {
			f.details = append(f.details, authDetail{"Domain:", dkim.Domain})
		}
		fields = append(fields, f)
	}

	if auth.DMARC != nil {
		f := authField{label: "DMARC:", status: auth.DMARC.Result}
		if auth.DMARC.Policy != "" {
			f.details = append(f.details, authDetail{"Policy:", auth.DMARC.Policy})
		}
		fields = append(fields, f)
	}

	if auth.ReverseDNS != nil {
		f := authField{label: "Reverse DNS:", status: auth.ReverseDNS.Result}
		if auth.ReverseDNS.Hostname != "" {
			f.details = append(f.details, authDetail{"Hostname:", auth.ReverseDNS.Hostname})
		}
		fields = append(fields, f)
	}

	return fields
}

// RenderAuthResults renders authentication results.
// When verbose is false (compact mode), details are shown in parentheses on the same line.
// When verbose is true, details are shown on separate indented lines.
func RenderAuthResults(auth *authresults.AuthResults, labelStyle lipgloss.Style, verbose bool) string {
	fields := buildAuthFields(auth)
	if fields == nil {
		return WarnStyle.Render("No authentication results available")
	}

	var lines []string
	for _, f := range fields {
		line := fmt.Sprintf("%s %s", labelStyle.Render(f.label), FormatAuthResult(f.status))

		if verbose {
			lines = append(lines, line)
			for _, d := range f.details {
				lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("  "+d.label), d.value))
			}
		} else {
			// Compact: show first detail in parentheses
			if len(f.details) > 0 {
				d := f.details[0]
				line += fmt.Sprintf(" (%s %s)", strings.TrimSuffix(strings.ToLower(d.label), ":"), d.value)
			}
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// CalculateScore computes a security score (0-100) based on auth results.
// Base score of 50 assumes E2E encryption.
func CalculateScore(email *vaultsandbox.Email) int {
	score := 50
	if email.AuthResults == nil {
		return score
	}
	auth := email.AuthResults
	if auth.SPF != nil && strings.EqualFold(auth.SPF.Result, "pass") {
		score += 15
	}
	if len(auth.DKIM) > 0 && strings.EqualFold(auth.DKIM[0].Result, "pass") {
		score += 20
	}
	if auth.DMARC != nil && strings.EqualFold(auth.DMARC.Result, "pass") {
		score += 10
	}
	if auth.ReverseDNS != nil && strings.EqualFold(auth.ReverseDNS.Result, "pass") {
		score += 5
	}
	return score
}

// FormatSpamAction formats a spam action with appropriate styling.
func FormatSpamAction(action spamanalysis.SpamAction) string {
	switch action {
	case spamanalysis.ActionNoAction:
		return PassStyle.Render("no action")
	case spamanalysis.ActionGreylist:
		return WarnStyle.Render("greylist")
	case spamanalysis.ActionAddHeader:
		return WarnStyle.Render("add header")
	case spamanalysis.ActionRewriteSubject:
		return WarnStyle.Render("rewrite subject")
	case spamanalysis.ActionSoftReject:
		return FailStyle.Render("soft reject")
	case spamanalysis.ActionReject:
		return FailStyle.Render("reject")
	default:
		return string(action)
	}
}

// FormatSpamStatus formats a spam status with appropriate styling.
func FormatSpamStatus(status spamanalysis.SpamStatus) string {
	switch status {
	case spamanalysis.StatusAnalyzed:
		return PassStyle.Render("analyzed")
	case spamanalysis.StatusSkipped:
		return MutedStyle.Render("skipped")
	case spamanalysis.StatusError:
		return FailStyle.Render("error")
	default:
		return string(status)
	}
}

// FormatSpamVerdict formats the is-spam verdict with appropriate styling.
func FormatSpamVerdict(isSpam *bool) string {
	if isSpam == nil {
		return MutedStyle.Render("unknown")
	}
	if *isSpam {
		return FailStyle.Render("YES")
	}
	return PassStyle.Render("NO")
}

// FormatSpamScore formats a spam score with color based on threshold.
func FormatSpamScore(score, requiredScore *float64) string {
	if score == nil {
		return MutedStyle.Render("N/A")
	}
	scoreStr := fmt.Sprintf("%.1f", *score)
	if requiredScore != nil {
		scoreStr += fmt.Sprintf(" / %.1f", *requiredScore)
	}
	if requiredScore != nil && *score >= *requiredScore {
		return FailStyle.Render(scoreStr)
	}
	return PassStyle.Render(scoreStr)
}

// RenderSpamAnalysis renders spam analysis results.
// When verbose is false (compact mode), shows only key info.
// When verbose is true, shows full details including symbols.
func RenderSpamAnalysis(sa *spamanalysis.SpamAnalysis, labelStyle lipgloss.Style, verbose bool) string {
	if sa == nil {
		return WarnStyle.Render("No spam analysis results available")
	}

	var lines []string

	// Status
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Status:"), FormatSpamStatus(sa.Status)))

	// If not analyzed, show info and return
	if sa.Status != spamanalysis.StatusAnalyzed {
		if sa.Info != "" {
			lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Info:"), sa.Info))
		}
		return strings.Join(lines, "\n")
	}

	// Score
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Score:"), FormatSpamScore(sa.Score, sa.RequiredScore)))

	// Verdict
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Is Spam:"), FormatSpamVerdict(sa.IsSpam)))

	// Action
	lines = append(lines, fmt.Sprintf("%s %s", labelStyle.Render("Action:"), FormatSpamAction(sa.Action)))

	// Processing time
	if sa.ProcessingTimeMs != nil {
		lines = append(lines, fmt.Sprintf("%s %dms", labelStyle.Render("Processing:"), *sa.ProcessingTimeMs))
	}

	// Symbols (only in verbose mode)
	if verbose && len(sa.Symbols) > 0 {
		lines = append(lines, "")
		lines = append(lines, labelStyle.Render("Triggered Rules:"))

		positive, negative, _ := spamanalysis.CategorizeSymbols(sa.Symbols)

		// Show spam indicators (positive scores)
		if len(positive) > 0 {
			for _, sym := range positive {
				scoreStr := fmt.Sprintf("+%.1f", sym.Score)
				line := fmt.Sprintf("  %s %s", FailStyle.Render(scoreStr), sym.Name)
				if sym.Description != "" {
					line += fmt.Sprintf(" - %s", MutedStyle.Render(sym.Description))
				}
				lines = append(lines, line)
			}
		}

		// Show ham indicators (negative scores)
		if len(negative) > 0 {
			for _, sym := range negative {
				scoreStr := fmt.Sprintf("%.1f", sym.Score)
				line := fmt.Sprintf("  %s %s", PassStyle.Render(scoreStr), sym.Name)
				if sym.Description != "" {
					line += fmt.Sprintf(" - %s", MutedStyle.Render(sym.Description))
				}
				lines = append(lines, line)
			}
		}
	}

	return strings.Join(lines, "\n")
}
