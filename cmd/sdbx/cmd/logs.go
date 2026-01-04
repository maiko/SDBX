package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/docker"
)

var (
	logsTail   int
	logsFollow bool
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "View logs from SDBX services",
	Long: `View logs from one or all SDBX services.

Examples:
  sdbx logs              # All services
  sdbx logs plex         # Specific service
  sdbx logs -f radarr    # Follow logs
  sdbx logs -n 50 sonarr # Last 50 lines`,
	RunE: runLogs,
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().IntVarP(&logsTail, "tail", "n", 100, "Number of lines to show")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
}

func runLogs(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return err
	}

	service := ""
	if len(args) > 0 {
		service = args[0]
	}

	// For follow mode, use exec directly for better UX
	if logsFollow {
		cmdArgs := []string{"compose", "-f", "compose.yaml", "-p", "sdbx", "logs", "-f"}
		if logsTail > 0 {
			cmdArgs = append(cmdArgs, "--tail", fmt.Sprintf("%d", logsTail))
		}
		if service != "" {
			cmdArgs = append(cmdArgs, service)
		}

		execCmd := exec.Command("docker", cmdArgs...)
		execCmd.Dir = projectDir
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		return execCmd.Run()
	}

	// Non-follow mode
	compose := docker.NewCompose(projectDir)
	ctx := context.Background()

	output, err := compose.Logs(ctx, service, logsTail, false)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	fmt.Print(output)
	return nil
}
