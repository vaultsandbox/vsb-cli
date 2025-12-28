package cliutil

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vaultsandbox/vsb-cli/internal/styles"
)

// Column defines a table column with a header and optional width.
type Column struct {
	Header string
	Width  int           // 0 means no padding
	Style  lipgloss.Style // optional style for cell values
}

// Table renders formatted table output.
type Table struct {
	Columns []Column
	Indent  string
}

// NewTable creates a new table with the given columns.
func NewTable(columns ...Column) *Table {
	return &Table{Columns: columns, Indent: "  "}
}

// WithIndent sets a custom indent string and returns the table for chaining.
func (t *Table) WithIndent(indent string) *Table {
	t.Indent = indent
	return t
}

// PrintHeader prints the styled header row and separator line.
func (t *Table) PrintHeader() {
	headerStyle := styles.HeaderStyle.MarginBottom(0)
	headers := make([]string, len(t.Columns))
	totalWidth := len(t.Indent)

	for i, col := range t.Columns {
		if col.Width > 0 {
			headers[i] = headerStyle.Render(fmt.Sprintf("%-*s", col.Width, col.Header))
			totalWidth += col.Width + 2
		} else {
			headers[i] = headerStyle.Render(col.Header)
			totalWidth += len(col.Header) + 2
		}
	}

	fmt.Println()
	fmt.Printf("%s%s\n", t.Indent, strings.Join(headers, "  "))
	fmt.Println(strings.Repeat("-", totalWidth))
}

// PrintRow prints a data row with values aligned to column widths.
// If a column has a Style set, it will be applied to the cell value.
func (t *Table) PrintRow(values ...string) {
	cells := make([]string, len(values))
	for i, val := range values {
		cell := val
		if i < len(t.Columns) {
			col := t.Columns[i]
			if col.Width > 0 {
				cell = fmt.Sprintf("%-*s", col.Width, Truncate(val, col.Width))
			}
			if col.Style.Value() != "" {
				cell = col.Style.Render(cell)
			}
		}
		cells[i] = cell
	}
	fmt.Printf("%s%s\n", t.Indent, strings.Join(cells, "  "))
}

// Truncate shortens a string to max length with ellipsis.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
