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

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start all SDBX services",
	Long: `Start all configured SDBX services using Docker Compose.

This command will:
  • Pull latest images if needed
  • Start all enabled services
  • Wait for health checks to pass`,
	RunE: runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func runUp(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return err
	}

	compose := docker.NewCompose(projectDir)
	ctx := context.Background()

	fmt.Println(tui.InfoStyle.Render("Starting SDBX services..."))
	fmt.Println()

	// Start services
	start := time.Now()
	if err := compose.Up(ctx); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	elapsed := time.Since(start)
	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Services started in %s", elapsed.Round(time.Millisecond))))
	fmt.Println()
	fmt.Println("Run 'sdbx status' to view service health")
	fmt.Println("Run 'sdbx doctor' to verify configuration")

	return nil
}
