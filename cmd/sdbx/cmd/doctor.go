package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/doctor"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run diagnostic checks on your SDBX installation",
	Long: `Run a series of diagnostic checks to verify your SDBX installation.

Checks include:
  • Docker and Docker Compose versions
  • Disk space availability
  • File permissions
  • Port availability
  • Project file integrity
  • Secrets configuration
  • VPN connectivity (if services running)`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		// Still run checks even without a project
		projectDir = "."
	}

	ctx := context.Background()
	doc := doctor.NewDoctor(projectDir)

	// Header
	if !IsJSONOutput() {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(tui.ColorPrimary)
		fmt.Println(titleStyle.Render("SDBX Doctor"))
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("Running diagnostics..."))
		fmt.Println()
	}

	// Run checks
	checks := doc.RunAll(ctx)

	// JSON output
	if IsJSONOutput() {
		data, _ := json.MarshalIndent(checks, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display results
	passed := 0
	failed := 0

	for _, check := range checks {
		var icon string
		var style lipgloss.Style

		switch check.Status {
		case doctor.StatusPassed:
			icon = tui.IconSuccess
			style = tui.SuccessStyle
			passed++
		case doctor.StatusWarning:
			icon = tui.IconWarning
			style = tui.WarningStyle
		case doctor.StatusFailed:
			icon = tui.IconError
			style = tui.ErrorStyle
			failed++
		default:
			icon = "○"
			style = tui.MutedStyle
		}

		// Format: ✓ Check name          message (duration)
		nameWidth := 25
		name := check.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}
		for len(name) < nameWidth {
			name += " "
		}

		durationStr := ""
		if check.Duration > 0 {
			durationStr = fmt.Sprintf(" (%s)", check.Duration.Round(time.Millisecond))
		}

		fmt.Printf("  %s %s %s%s\n",
			style.Render(icon),
			name,
			check.Message,
			tui.MutedStyle.Render(durationStr),
		)
	}

	// Summary
	fmt.Println()
	if failed == 0 {
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ All %d checks passed", passed)))
	} else {
		fmt.Println(tui.ErrorStyle.Render(fmt.Sprintf("✗ %d of %d checks failed", failed, passed+failed)))
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("Fix the failed checks and run 'sdbx doctor' again."))
	}

	return nil
}
