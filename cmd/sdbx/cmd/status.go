package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/docker"
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
  • Port mappings
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

	// JSON output mode
	if IsJSONOutput() {
		data, _ := json.MarshalIndent(map[string]interface{}{
			"domain":   cfg.Domain,
			"services": services,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		MarginBottom(1)

	fmt.Println(titleStyle.Render("SDBX Status — " + cfg.Domain))
	fmt.Println()

	// Services table
	if len(services) == 0 {
		fmt.Println(tui.MutedStyle.Render("No services running. Run 'sdbx up' to start."))
		return nil
	}

	// Calculate column widths
	maxName := 20
	for _, svc := range services {
		name := extractServiceName(svc.Name)
		if len(name) > maxName {
			maxName = len(name)
		}
	}

	// Header row
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(tui.ColorMuted)
	fmt.Printf("%s  %s  %s\n",
		headerStyle.Render(padRight("SERVICE", maxName)),
		headerStyle.Render(padRight("STATUS", 12)),
		headerStyle.Render("HEALTH"),
	)
	fmt.Println(strings.Repeat("─", maxName+30))

	// Service rows
	for _, svc := range services {
		name := extractServiceName(svc.Name)

		var statusIcon, statusText string
		var statusStyle lipgloss.Style

		if svc.Running {
			statusIcon = tui.IconRunning
			statusText = "running"
			statusStyle = tui.SuccessStyle
		} else {
			statusIcon = tui.IconStopped
			statusText = "stopped"
			statusStyle = tui.MutedStyle
		}

		var healthText string
		switch svc.Health {
		case "healthy":
			healthText = tui.SuccessStyle.Render("✓ healthy")
		case "unhealthy":
			healthText = tui.ErrorStyle.Render("✗ unhealthy")
		case "starting":
			healthText = tui.WarningStyle.Render("◐ starting")
		default:
			healthText = tui.MutedStyle.Render("—")
		}

		fmt.Printf("%s  %s  %s\n",
			statusStyle.Render(statusIcon)+" "+padRight(name, maxName-2),
			padRight(statusText, 12),
			healthText,
		)
	}

	fmt.Println()
	fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("Total: %d services", len(services))))

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

// padRight pads a string to a minimum width
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
