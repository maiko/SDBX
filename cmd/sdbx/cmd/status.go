package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/registry"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all SDBX services",
	Long: `Display the current status of all SDBX services.

Shows:
  • Service name and health status
  • Container state (running/stopped)
  • Service URLs
  • VPN connection status`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return err
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	compose := docker.NewCompose(projectDir)
	ctx := context.Background()

	// Get service status
	services, err := compose.PS(ctx)
	if err != nil {
		return fmt.Errorf("failed to get service status: %w", err)
	}

	// Get registry for service info
	reg, _ := registry.NewWithDefaults()
	serviceInfo := make(map[string]registry.ServiceInfo)
	if reg != nil {
		if svcList, err := reg.ListServices(ctx); err == nil {
			for _, svc := range svcList {
				serviceInfo[svc.Name] = svc
			}
		}
	}

	// JSON output mode
	if IsJSONOutput() {
		return OutputJSON(map[string]interface{}{
			"domain":   cfg.Domain,
			"services": services,
		})
	}

	// Header with summary
	running := 0
	for _, svc := range services {
		if svc.Running {
			running++
		}
	}

	fmt.Println()
	fmt.Println(tui.TitleStyle.Render("SDBX Status"))
	fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Domain:"), cfg.Domain)
	fmt.Printf("  %s %s\n", tui.MutedStyle.Render("Mode:"), cfg.Expose.Mode)
	if cfg.VPNEnabled {
		fmt.Printf("  %s %s\n", tui.MutedStyle.Render("VPN:"), tui.SuccessStyle.Render(cfg.VPNProvider+" (enabled)"))
	}
	fmt.Println()

	// Services table
	if len(services) == 0 {
		fmt.Println(tui.MutedStyle.Render("  No services running. Run 'sdbx up' to start."))
		return nil
	}

	// Create table
	table := tui.NewTable("Service", "Status", "Health", "URL")

	for _, svc := range services {
		name := extractServiceName(svc.Name)

		// Status badge
		status := tui.StatusBadge(svc.Running)

		// Health badge
		health := tui.HealthBadge(svc.Health)

		// URL
		url := tui.MutedStyle.Render("—")
		if info, ok := serviceInfo[name]; ok && info.HasWebUI && svc.Running {
			url = tui.InfoStyle.Render(cfg.GetServiceURL(name))
		}

		table.AddRow(name, status, health, url)
	}

	fmt.Println(table.Render())

	// Summary
	summaryStyle := tui.MutedStyle
	if running == len(services) {
		msg := summaryStyle.Render(fmt.Sprintf("All %d services running", running))
		fmt.Printf("%s %s\n", tui.SuccessStyle.Render(tui.IconSuccess), msg)
	} else {
		msg := summaryStyle.Render(fmt.Sprintf("%d/%d services running", running, len(services)))
		fmt.Printf("%s %s\n", tui.WarningStyle.Render(tui.IconWarning), msg)
	}
	fmt.Println()

	return nil
}

// extractServiceName gets the service name from container name (removes project prefix)
func extractServiceName(containerName string) string {
	parts := strings.Split(containerName, "-")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "-")
	}
	return containerName
}
