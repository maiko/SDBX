// Package tui provides TUI components and styles for sdbx.
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette - matches PRD specification
var (
	ColorPrimary = lipgloss.Color("#7C3AED") // Violet
	ColorSuccess = lipgloss.Color("#10B981") // Emerald
	ColorWarning = lipgloss.Color("#F59E0B") // Amber
	ColorError   = lipgloss.Color("#EF4444") // Red
	ColorInfo    = lipgloss.Color("#3B82F6") // Blue
	ColorMuted   = lipgloss.Color("#6B7280") // Gray
	ColorWhite   = lipgloss.Color("#FFFFFF")
	ColorBlack   = lipgloss.Color("#000000")
)

// Base styles
var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// Status indicators
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// CommandStyle for rendering CLI commands in help text
	CommandStyle = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	// Interactive elements
	SelectedStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	FocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Padding(0, 1)

	TableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)
)

// Status icons
const (
	IconSuccess = "✓"
	IconError   = "✗"
	IconWarning = "⚠"
	IconInfo    = "ℹ"
	IconRunning = "●"
	IconStopped = "○"
	IconSpinner = "◐"
	IconArrow   = "→"
	IconCheck   = "✓"
	IconCross   = "✗"
	IconLock    = "🔒"
	IconUnlock  = "🔓"
	IconStar    = "★"
	IconDot     = "•"
	IconDash    = "─"
	IconBox     = "▪"
	IconFolder  = "📁"
	IconGear    = "⚙"
	IconNetwork = "🌐"
	IconDocker  = "🐳"
	IconKey     = "🔑"
	IconRocket  = "🚀"
	IconPackage = "📦"
)

// Logo is the ASCII art logo for SDBX
const Logo = `
███████╗██████╗ ██████╗ ██╗  ██╗
██╔════╝██╔══██╗██╔══██╗╚██╗██╔╝
███████╗██║  ██║██████╔╝ ╚███╔╝ 
╚════██║██║  ██║██╔══██╗ ██╔██╗ 
███████║██████╔╝██████╔╝██╔╝ ██╗
╚══════╝╚═════╝ ╚═════╝ ╚═╝  ╚═╝`

// LogoStyled returns the logo with styling applied
func LogoStyled() string {
	return lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Render(Logo)
}

// RenderStatus returns a styled status indicator
func RenderStatus(healthy bool) string {
	if healthy {
		return SuccessStyle.Render(IconSuccess + " Healthy")
	}
	return ErrorStyle.Render(IconError + " Unhealthy")
}

// RenderServiceStatus returns a styled service status line
func RenderServiceStatus(name string, running bool, healthy bool) string {
	var icon string
	var style lipgloss.Style

	if !running {
		icon = IconStopped
		style = MutedStyle
	} else if healthy {
		icon = IconRunning
		style = SuccessStyle
	} else {
		icon = IconRunning
		style = WarningStyle
	}

	return style.Render(icon) + " " + name
}

// ProgressBar renders a simple progress bar
func ProgressBar(percent float64, width int) string {
	filled := int(float64(width) * percent)
	empty := width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}

	return lipgloss.NewStyle().Foreground(ColorPrimary).Render(bar)
}

// RenderSuccessBox returns a polished success message box
func RenderSuccessBox(title, message string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSuccess).
		Padding(1, 2).
		Margin(1, 0).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				SuccessStyle.Copy().Foreground(ColorSuccess).Bold(true).Render(IconSuccess+" "+title),
				"",
				lipgloss.NewStyle().Foreground(ColorWhite).Render(message),
			),
		)
}

// RenderInfoBox returns a styled information box
func RenderInfoBox(title, message string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorInfo).
		Padding(1, 2).
		Margin(1, 0).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				InfoStyle.Copy().Bold(true).Render(IconInfo+" "+title),
				"",
				lipgloss.NewStyle().Foreground(ColorWhite).Render(message),
			),
		)
}

// RenderErrorBox returns a styled error box
func RenderErrorBox(title, message string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorError).
		Padding(1, 2).
		Margin(1, 0).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				ErrorStyle.Copy().Bold(true).Render(IconError+" "+title),
				"",
				lipgloss.NewStyle().Foreground(ColorWhite).Render(message),
			),
		)
}

// RenderWarningBox returns a styled warning box
func RenderWarningBox(title, message string) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorWarning).
		Padding(1, 2).
		Margin(1, 0).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				WarningStyle.Copy().Bold(true).Render(IconWarning+" "+title),
				"",
				lipgloss.NewStyle().Foreground(ColorWhite).Render(message),
			),
		)
}

// RenderSection renders a styled section header
func RenderSection(title string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginTop(1).
		Render(title)
}

// RenderKeyValue renders a styled key-value pair
func RenderKeyValue(key, value string) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorMuted).Width(14)
	return keyStyle.Render(key+":") + " " + value
}

// RenderBullet renders a bulleted list item
func RenderBullet(text string) string {
	return MutedStyle.Render("  "+IconDot+" ") + text
}

// RenderCommand renders a styled command hint
func RenderCommand(cmd string) string {
	return CommandStyle.Render(cmd)
}

// RenderDivider renders a horizontal divider line
func RenderDivider(width int) string {
	line := ""
	for i := 0; i < width; i++ {
		line += IconDash
	}
	return MutedStyle.Render(line)
}

// RenderHeader renders a styled header with optional subtitle
func RenderHeader(title string, subtitle string) string {
	result := TitleStyle.Render(title)
	if subtitle != "" {
		result += "\n" + SubtitleStyle.Render(subtitle)
	}
	return result
}

// RenderStats renders statistics in a formatted way
func RenderStats(stats map[string]string) string {
	var parts []string
	for key, value := range stats {
		parts = append(parts, MutedStyle.Render(key+": ")+SuccessStyle.Render(value))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// Spinner characters for animation
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// CategoryColors maps service categories to colors
var CategoryColors = map[string]lipgloss.Color{
	"media":       lipgloss.Color("#E879F9"), // Fuchsia
	"downloads":   lipgloss.Color("#38BDF8"), // Sky
	"management":  lipgloss.Color("#A3E635"), // Lime
	"utility":     lipgloss.Color("#FBBF24"), // Amber
	"networking":  lipgloss.Color("#2DD4BF"), // Teal
	"auth":        lipgloss.Color("#F87171"), // Red
}

// RenderCategory renders a styled category tag
func RenderCategory(category string) string {
	color, ok := CategoryColors[category]
	if !ok {
		color = ColorMuted
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render(category)
}
