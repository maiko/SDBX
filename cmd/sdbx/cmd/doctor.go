package cmd

import (
	"context"
	"fmt"
	"time"

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

	// JSON output - run all at once
	if IsJSONOutput() {
		checks := doc.RunAll(ctx)
		return OutputJSON(checks)
	}

	// Interactive output with animated progress
	fmt.Println()
	fmt.Println(tui.TitleStyle.Render("SDBX Doctor"))
	fmt.Println(tui.MutedStyle.Render("  Running diagnostic checks...\n"))

	// Run checks with live updates
	checks := doc.RunAll(ctx)

	// Display results using CheckList
	checklist := tui.NewCheckList()
	passed := 0
	failed := 0
	warnings := 0

	for _, check := range checks {
		idx := checklist.Add(check.Name)

		var status string
		var detail string

		switch check.Status {
		case doctor.StatusPassed:
			status = "success"
			passed++
		case doctor.StatusWarning:
			status = "warning"
			warnings++
		case doctor.StatusFailed:
			status = "error"
			failed++
		default:
			status = "pending"
		}

		detail = check.Message
		if check.Duration > 0 {
			detail += fmt.Sprintf(" (%s)", check.Duration.Round(time.Millisecond))
		}

		checklist.SetStatus(idx, status, detail)
	}

	fmt.Println(checklist.Render())

	// Summary box
	fmt.Println()
	if failed == 0 && warnings == 0 {
		fmt.Print(tui.RenderSuccessBox("All checks passed!",
			fmt.Sprintf("%d checks completed successfully", passed)))
	} else if failed == 0 {
		fmt.Print(tui.RenderInfoBox("Checks completed with warnings",
			fmt.Sprintf("%d passed, %d warnings", passed, warnings)))
	} else {
		fmt.Print(tui.RenderErrorBox("Some checks failed",
			fmt.Sprintf("%d passed, %d failed, %d warnings\n\nFix the issues and run 'sdbx doctor' again.", passed, failed, warnings)))
	}
	fmt.Println()

	return nil
}
