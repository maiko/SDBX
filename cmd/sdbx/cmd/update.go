package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/registry"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var (
	updateSafe bool
	updateAll  bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update SDBX services to latest versions",
	Long: `Pull latest Docker images and restart services.

By default, services are updated one at a time with health checks.
Use --all to update all services simultaneously (faster but riskier).

Examples:
  sdbx update          # Safe update (one at a time)
  sdbx update --all    # Update all at once
  sdbx update --safe   # Extra safe mode (with rollback)`,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateSafe, "safe", false, "Extra safe mode with automatic rollback on failure")
	updateCmd.Flags().BoolVar(&updateAll, "all", false, "Update all services at once (faster)")
}

func runUpdate(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return err
	}

	compose := docker.NewCompose(projectDir)
	ctx := context.Background()

	fmt.Println(tui.TitleStyle.Render("SDBX Update"))
	fmt.Println()

	// Step 1: Pull images
	fmt.Println(tui.InfoStyle.Render("Pulling latest images..."))
	start := time.Now()

	if err := compose.Pull(ctx); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("  ✓ Images pulled in %s", time.Since(start).Round(time.Millisecond))))
	fmt.Println()

	// Step 2: Restart services
	if updateAll {
		fmt.Println(tui.InfoStyle.Render("Restarting all services..."))
		if err := compose.Down(ctx); err != nil {
			return fmt.Errorf("failed to stop services: %w", err)
		}
		if err := compose.Up(ctx); err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}
		fmt.Println(tui.SuccessStyle.Render("  ✓ All services restarted"))
	} else {
		fmt.Println(tui.InfoStyle.Render("Restarting services (ordered)..."))

		// Get enabled services from registry in dependency order
		services, err := getEnabledServicesOrdered(ctx, projectDir)
		if err != nil {
			return fmt.Errorf("failed to get enabled services: %w", err)
		}

		for _, svc := range services {
			fmt.Printf("  %s %s...", tui.IconRunning, svc)

			if err := compose.Restart(ctx, svc); err != nil {
				fmt.Printf("%s\n", tui.WarningStyle.Render(" skipped"))
				fmt.Fprintf(os.Stderr, "  Failed to restart %s: %v\n", svc, err)
				continue
			}

			// Wait for health if safe mode
			if updateSafe {
				healthy, err := compose.IsHealthy(ctx, svc)
				if err != nil || !healthy {
					// Wait up to 30 seconds for service to become healthy
					if err := compose.WaitHealthy(ctx, svc, 30*time.Second); err != nil {
						fmt.Fprintf(os.Stderr, "\nWarning: %s may not be fully healthy: %v\n", svc, err)
					}
				}
			}

			fmt.Println(tui.SuccessStyle.Render(" done"))
		}
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("✓ Update complete"))
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render("Run 'sdbx status' to verify all services are healthy"))

	return nil
}

// getEnabledServicesOrdered returns enabled services in dependency order
func getEnabledServicesOrdered(ctx context.Context, projectDir string) ([]string, error) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Get registry
	reg, err := registry.NewWithDefaults()
	if err != nil {
		return nil, fmt.Errorf("failed to create registry: %w", err)
	}

	// Resolve services based on config
	graph, err := reg.Resolve(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve services: %w", err)
	}

	return graph.Order, nil
}
