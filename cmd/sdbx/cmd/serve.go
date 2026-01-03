package cmd

import (
	"os"

	"github.com/maiko/sdbx/internal/web"
	"github.com/spf13/cobra"
)

var (
	serveHost string
	servePort int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start SDBX web interface",
	Long: `Start the SDBX web UI server.

Pre-init: Serves setup wizard with token authentication
Post-init: Serves dashboard and management interface

The server binds to 0.0.0.0 by default to allow remote access.
A one-time setup token is generated for pre-init security.

Examples:
  sdbx serve                  # Start with defaults (0.0.0.0:3000)
  sdbx serve --port 8080      # Use custom port
  sdbx serve --host 127.0.0.1 # Localhost only (less secure for pre-init)`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "Host to bind to (0.0.0.0 for all interfaces)")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 3000, "Port to listen on")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Get current working directory as project dir
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create server config
	cfg := &web.ServerConfig{
		Host:       serveHost,
		Port:       servePort,
		ProjectDir: projectDir,
	}

	// Run server (handles signals internally)
	return web.Run(cfg)
}
