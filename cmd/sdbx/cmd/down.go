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

func init() {
	rootCmd.AddCommand(downCmd)
}

func runDown(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return err
	}

	compose := docker.NewCompose(projectDir)
	ctx := context.Background()

	fmt.Println(tui.InfoStyle.Render("Stopping SDBX services..."))

	if err := compose.Down(ctx); err != nil {
		return fmt.Errorf("failed to stop services: %w", err)
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("✓ All services stopped"))

	return nil
}
