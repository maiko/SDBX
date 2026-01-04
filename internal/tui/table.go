package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table renders data in a formatted table
type Table struct {
	Headers     []string
	Rows        [][]string
	HeaderStyle lipgloss.Style
	CellStyle   lipgloss.Style
	BorderStyle lipgloss.Style
	columnWidths []int
}

// NewTable creates a new table with headers
func NewTable(headers ...string) *Table {
	return &Table{
		Headers:     headers,
		Rows:        [][]string{},
		HeaderStyle: TableHeaderStyle,
		CellStyle:   TableCellStyle,
		BorderStyle: lipgloss.NewStyle().Foreground(ColorMuted),
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	// Pad row if needed
	for len(cells) < len(t.Headers) {
		cells = append(cells, "")
	}
	t.Rows = append(t.Rows, cells)
}

// calculateWidths determines column widths
func (t *Table) calculateWidths() {
	t.columnWidths = make([]int, len(t.Headers))

	// Start with header widths
	for i, h := range t.Headers {
		t.columnWidths[i] = len(h)
	}

	// Check all rows
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(t.columnWidths) {
				// Strip ANSI codes for width calculation
				plainCell := stripAnsi(cell)
				if len(plainCell) > t.columnWidths[i] {
					t.columnWidths[i] = len(plainCell)
				}
			}
		}
	}

	// Add padding
	for i := range t.columnWidths {
		t.columnWidths[i] += 2
	}
}

// Render renders the table as a string
func (t *Table) Render() string {
	if len(t.Headers) == 0 {
		return ""
	}

	t.calculateWidths()

	var sb strings.Builder

	// Render header
	for i, h := range t.Headers {
		width := t.columnWidths[i]
		sb.WriteString(t.HeaderStyle.Width(width).Render(h))
	}
	sb.WriteString("\n")

	// Render separator
	for _, w := range t.columnWidths {
		sb.WriteString(t.BorderStyle.Render(strings.Repeat("─", w)))
	}
	sb.WriteString("\n")

	// Render rows
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < len(t.columnWidths) {
				width := t.columnWidths[i]
				// Calculate padding needed for styled cells
				plainLen := len(stripAnsi(cell))
				padding := width - plainLen
				if padding > 0 {
					sb.WriteString(cell + strings.Repeat(" ", padding))
				} else {
					sb.WriteString(cell)
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// stripAnsi removes ANSI escape codes for width calculation
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}

// ServiceTable creates a pre-styled table for services
func ServiceTable() *Table {
	t := NewTable("Service", "Status", "Health", "URL")
	return t
}

// AddonTable creates a pre-styled table for addons
func AddonTable() *Table {
	t := NewTable("Addon", "Category", "Status", "Description")
	return t
}

// SourceTable creates a pre-styled table for sources
func SourceTable() *Table {
	t := NewTable("Name", "Type", "Priority", "Services")
	return t
}

// StatusBadge returns a styled status badge
func StatusBadge(running bool) string {
	if running {
		return SuccessStyle.Render(IconRunning + " Running")
	}
	return MutedStyle.Render(IconStopped + " Stopped")
}

// HealthBadge returns a styled health badge
func HealthBadge(health string) string {
	switch health {
	case "healthy":
		return SuccessStyle.Render(IconSuccess + " Healthy")
	case "unhealthy":
		return ErrorStyle.Render(IconError + " Unhealthy")
	case "starting":
		return WarningStyle.Render(IconSpinner + " Starting")
	default:
		return MutedStyle.Render("─")
	}
}

// EnabledBadge returns a styled enabled/disabled badge
func EnabledBadge(enabled bool) string {
	if enabled {
		return SuccessStyle.Render(IconCheck + " Enabled")
	}
	return MutedStyle.Render(IconCross + " Disabled")
}
