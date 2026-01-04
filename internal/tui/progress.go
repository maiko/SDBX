package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressStyle defines styles for progress bars
var (
	ProgressFilledStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
	ProgressEmptyStyle  = lipgloss.NewStyle().Foreground(ColorMuted)
	ProgressLabelStyle  = lipgloss.NewStyle().Foreground(ColorWhite)
)

// ProgressBarConfig configures progress bar appearance
type ProgressBarConfig struct {
	Width       int
	FilledChar  string
	EmptyChar   string
	ShowPercent bool
}

// DefaultProgressConfig returns default progress bar configuration
func DefaultProgressConfig() ProgressBarConfig {
	return ProgressBarConfig{
		Width:       40,
		FilledChar:  "█",
		EmptyChar:   "░",
		ShowPercent: true,
	}
}

// RenderProgressBar renders a progress bar with the given percentage
func RenderProgressBar(percent float64, config ProgressBarConfig) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100 * float64(config.Width))
	empty := config.Width - filled

	bar := ProgressFilledStyle.Render(strings.Repeat(config.FilledChar, filled)) +
		ProgressEmptyStyle.Render(strings.Repeat(config.EmptyChar, empty))

	if config.ShowPercent {
		return fmt.Sprintf("%s %3.0f%%", bar, percent)
	}
	return bar
}

// StepProgress tracks progress through a series of steps
type StepProgress struct {
	steps       []string
	current     int
	total       int
	titleStyle  lipgloss.Style
	activeStyle lipgloss.Style
	doneStyle   lipgloss.Style
	pendStyle   lipgloss.Style
}

// NewStepProgress creates a new step progress tracker
func NewStepProgress(steps ...string) *StepProgress {
	return &StepProgress{
		steps:       steps,
		current:     0,
		total:       len(steps),
		titleStyle:  TitleStyle,
		activeStyle: lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true),
		doneStyle:   lipgloss.NewStyle().Foreground(ColorSuccess),
		pendStyle:   lipgloss.NewStyle().Foreground(ColorMuted),
	}
}

// Render renders the current step progress
func (p *StepProgress) Render() string {
	var sb strings.Builder

	// Progress indicator (1/5)
	progress := fmt.Sprintf("Step %d of %d", p.current+1, p.total)
	sb.WriteString(MutedStyle.Render(progress))
	sb.WriteString("\n\n")

	// Visual step indicator
	for i, step := range p.steps {
		var icon, label string
		if i < p.current {
			icon = p.doneStyle.Render(IconSuccess)
			label = p.doneStyle.Render(step)
		} else if i == p.current {
			icon = p.activeStyle.Render(IconArrow)
			label = p.activeStyle.Render(step)
		} else {
			icon = p.pendStyle.Render(IconDot)
			label = p.pendStyle.Render(step)
		}
		sb.WriteString(fmt.Sprintf("  %s %s\n", icon, label))
	}

	return sb.String()
}

// RenderCompact renders a compact single-line progress indicator
func (p *StepProgress) RenderCompact() string {
	var parts []string
	for i := range p.steps {
		if i < p.current {
			parts = append(parts, p.doneStyle.Render("●"))
		} else if i == p.current {
			parts = append(parts, p.activeStyle.Render("●"))
		} else {
			parts = append(parts, p.pendStyle.Render("○"))
		}
	}
	return strings.Join(parts, " ")
}

// Next advances to the next step
func (p *StepProgress) Next() {
	if p.current < p.total-1 {
		p.current++
	}
}

// SetStep sets the current step (0-indexed)
func (p *StepProgress) SetStep(step int) {
	if step >= 0 && step < p.total {
		p.current = step
	}
}

// CurrentStep returns the current step name
func (p *StepProgress) CurrentStep() string {
	if p.current < len(p.steps) {
		return p.steps[p.current]
	}
	return ""
}

// IsComplete returns true if all steps are complete
func (p *StepProgress) IsComplete() bool {
	return p.current >= p.total-1
}

// CheckList renders a checklist-style progress display
type CheckList struct {
	items []checkItem
}

type checkItem struct {
	label  string
	status string // "pending", "running", "success", "error"
	detail string
}

// NewCheckList creates a new checklist
func NewCheckList() *CheckList {
	return &CheckList{
		items: []checkItem{},
	}
}

// Add adds an item to the checklist
func (c *CheckList) Add(label string) int {
	c.items = append(c.items, checkItem{
		label:  label,
		status: "pending",
	})
	return len(c.items) - 1
}

// SetStatus updates an item's status
func (c *CheckList) SetStatus(index int, status string, detail string) {
	if index >= 0 && index < len(c.items) {
		c.items[index].status = status
		c.items[index].detail = detail
	}
}

// Render renders the checklist
func (c *CheckList) Render() string {
	var sb strings.Builder

	for _, item := range c.items {
		var icon string
		var labelStyle lipgloss.Style

		switch item.status {
		case "success":
			icon = SuccessStyle.Render(IconSuccess)
			labelStyle = SuccessStyle
		case "error":
			icon = ErrorStyle.Render(IconError)
			labelStyle = ErrorStyle
		case "running":
			icon = InfoStyle.Render(IconSpinner)
			labelStyle = InfoStyle
		case "warning":
			icon = WarningStyle.Render(IconWarning)
			labelStyle = WarningStyle
		default:
			icon = MutedStyle.Render(IconDot)
			labelStyle = MutedStyle
		}

		sb.WriteString(fmt.Sprintf("  %s %s", icon, labelStyle.Render(item.label)))
		if item.detail != "" {
			sb.WriteString(MutedStyle.Render(fmt.Sprintf(" (%s)", item.detail)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
