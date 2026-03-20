package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/tui"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop all SDBX services",
	Long:  `Stop all running SDBX services.`,
	RunE:  runDown,
}

var downDryRun bool

func init() {
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().BoolVar(&downDryRun, "dry-run", false, "Show what would be done without stopping services")
}

func runDown(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return err
	}

	// Dry-run: show what would happen
	if downDryRun {
		fmt.Println(tui.TitleStyle.Render("Dry Run: sdbx down"))
		fmt.Println()
		fmt.Printf("  %s Stop all services via docker compose down\n", tui.IconArrow)
		fmt.Printf("  %s Project directory: %s\n", tui.IconArrow, projectDir)
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("No changes made (dry run)."))
		return nil
	}

	compose := docker.NewCompose(projectDir)
	ctx := context.Background()

	if IsTUIEnabled() {
		err = tui.RunWithSpinner("Stopping SDBX services...", func() error {
			return compose.Down(ctx)
		})
		if err != nil {
			return fmt.Errorf("failed to stop services: %w\n\n  Try: sdbx doctor", err)
		}
	} else {
		fmt.Println(tui.InfoStyle.Render("Stopping SDBX services..."))
		if err := compose.Down(ctx); err != nil {
			return fmt.Errorf("failed to stop services: %w\n\n  Try: sdbx doctor", err)
		}
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("✓ All services stopped"))

	return nil
}
