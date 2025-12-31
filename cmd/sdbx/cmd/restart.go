package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart SDBX services",
	Long: `Restart one or all SDBX services.

Examples:
  sdbx restart          # Restart all services
  sdbx restart plex     # Restart only Plex
  sdbx restart radarr sonarr  # Restart multiple services`,
	RunE: runRestart,
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return err
	}

	compose := docker.NewCompose(projectDir)
	ctx := context.Background()

	if len(args) == 0 {
		fmt.Println(tui.InfoStyle.Render("Restarting all services..."))
		start := time.Now()

		if err := compose.Restart(ctx, ""); err != nil {
			return fmt.Errorf("failed to restart services: %w", err)
		}

		fmt.Println()
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ All services restarted in %s", time.Since(start).Round(time.Millisecond))))
	} else {
		for _, service := range args {
			fmt.Printf("Restarting %s...\n", service)
			if err := compose.Restart(ctx, service); err != nil {
				fmt.Println(tui.ErrorStyle.Render(fmt.Sprintf("  ✗ Failed to restart %s: %v", service, err)))
			} else {
				fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("  ✓ %s restarted", service)))
			}
		}
	}

	return nil
}
