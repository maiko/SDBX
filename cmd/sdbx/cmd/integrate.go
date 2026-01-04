package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/integrate"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var integrateCmd = &cobra.Command{
	Use:   "integrate",
	Short: "Auto-configure service integrations",
	Long: `Automatically configure integrations between services.

This command will:
  - Configure Prowlarr to sync indexers with *arr apps (Sonarr, Radarr, Lidarr, Readarr)
  - Configure qBittorrent as download client in *arr apps
  - Create download categories in qBittorrent

Services must be running before integration. Run 'sdbx up' first if needed.

Examples:
  sdbx integrate              # Configure all integrations
  sdbx integrate --dry-run    # Preview changes without applying
  sdbx integrate --verbose    # Show detailed progress`,
	RunE: runIntegrate,
}

// Flags
var (
	integrateDryRun bool
	integrateVerbose bool
)

func init() {
	rootCmd.AddCommand(integrateCmd)

	integrateCmd.Flags().BoolVar(&integrateDryRun, "dry-run", false, "Preview integrations without applying changes")
	integrateCmd.Flags().BoolVar(&integrateVerbose, "verbose", false, "Show detailed progress")
}

func runIntegrate(_ *cobra.Command, _ []string) error {
	// Get project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return fmt.Errorf("not in an SDBX project directory")
	}

	// Change to project directory
	if err := os.Chdir(projectDir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// Load service configurations
	services, err := integrate.LoadServicesFromConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load service configurations: %w", err)
	}

	if len(services) == 0 {
		if IsJSONOutput() {
			return OutputJSON(map[string]interface{}{
				"success": false,
				"message": "No services found to integrate",
			})
		}

		fmt.Println(tui.WarningStyle.Render("⚠ No services found to integrate"))
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("Make sure services are enabled and running."))
		fmt.Println(tui.MutedStyle.Render("Run 'sdbx up' to start services."))
		return nil
	}

	// Create integration config
	cfg := integrate.DefaultConfig()
	cfg.Services = services
	cfg.DryRun = integrateDryRun
	cfg.Verbose = integrateVerbose || !IsJSONOutput()

	// Create integrator
	integrator := integrate.NewIntegrator(cfg)

	ctx := context.Background()

	if !IsJSONOutput() {
		fmt.Println(tui.TitleStyle.Render("Service Integration"))
		fmt.Println()

		if integrateDryRun {
			fmt.Println(tui.WarningStyle.Render("🔍 DRY RUN MODE - No changes will be made"))
			fmt.Println()
		}

		// Show services to integrate
		fmt.Println(tui.MutedStyle.Render("Services detected:"))
		for name := range services {
			fmt.Printf("  • %s\n", name)
		}
		fmt.Println()

		if !integrateDryRun {
			fmt.Println(tui.MutedStyle.Render("Waiting for services to be ready..."))
			fmt.Println()
		}
	}

	// Run integrations
	results, err := integrator.Run(ctx)
	if err != nil {
		if IsJSONOutput() {
			if jsonErr := OutputJSON(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}); jsonErr != nil {
				return jsonErr
			}
			return err
		}

		return fmt.Errorf("integration failed: %w", err)
	}

	// JSON output
	if IsJSONOutput() {
		output := make([]map[string]interface{}, 0, len(results))
		successCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			}
			entry := map[string]interface{}{
				"service": r.Service,
				"success": r.Success,
				"message": r.Message,
			}
			if r.Error != nil {
				entry["error"] = r.Error.Error()
			}
			output = append(output, entry)
		}

		return OutputJSON(map[string]interface{}{
			"success":      successCount == len(results),
			"total":        len(results),
			"successful":   successCount,
			"failed":       len(results) - successCount,
			"integrations": output,
		})
	}

	// Human-readable output
	fmt.Println(tui.TitleStyle.Render("Integration Results"))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, tui.TableHeaderStyle.Render("INTEGRATION")+"\t"+tui.TableHeaderStyle.Render("STATUS")+"\t"+tui.TableHeaderStyle.Render("MESSAGE"))

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}

		status := "✓"
		statusStyle := tui.SuccessStyle
		if !result.Success {
			status = "✗"
			statusStyle = tui.ErrorStyle
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			result.Service,
			statusStyle.Render(status),
			result.Message,
		)
	}
	w.Flush()

	fmt.Println()

	// Summary
	if successCount == len(results) {
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ All %d integrations completed successfully", len(results))))
	} else {
		failCount := len(results) - successCount
		fmt.Println(tui.WarningStyle.Render(fmt.Sprintf("⚠ %d succeeded, %d failed", successCount, failCount)))
	}

	if integrateDryRun {
		fmt.Println()
		fmt.Println(tui.MutedStyle.Render("This was a dry run. Run without --dry-run to apply changes."))
	}

	return nil
}

