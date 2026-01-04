package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/tui"
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

	// Load config to check for Plex
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	compose := docker.NewCompose(projectDir)
	ctx := context.Background()

	// Prompt for Plex claim token if needed (before starting containers)
	if err := promptPlexClaimToken(cfg, projectDir); err != nil {
		return fmt.Errorf("failed to handle Plex claim token: %w", err)
	}

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

// promptPlexClaimToken checks if Plex addon is enabled and prompts for claim token
func promptPlexClaimToken(cfg *config.Config, projectDir string) error {
	// Check if Plex addon is enabled
	plexEnabled := false
	for _, addon := range cfg.Addons {
		if addon == "plex" {
			plexEnabled = true
			break
		}
	}

	if !plexEnabled {
		return nil // Plex not enabled, skip
	}

	// IMPORTANT: Check if claim token already exists - avoid prompting every time
	secretsDir := filepath.Join(projectDir, "secrets")
	tokenPath := filepath.Join(secretsDir, "plex_claim_token.txt")

	existingToken, err := os.ReadFile(tokenPath)
	if err == nil && len(bytes.TrimSpace(existingToken)) > 0 {
		// Token already exists and is not empty - skip prompting entirely
		// This ensures we only ask ONCE, not on every 'sdbx up' command
		return nil
	}

	// TUI mode only - in non-interactive mode, skip with warning
	if !IsTUIEnabled() {
		fmt.Println(tui.WarningStyle.Render("⚠ Warning: Plex claim token not set. Server will be unclaimed."))
		fmt.Println("To claim later, visit http://SERVER_IP:32400/web from local network")
		fmt.Println()
		return nil
	}

	// Show prompt with instructions
	instructions := fmt.Sprintf(
		"Your Plex server needs to be claimed to link it to your account.\n\n"+
			"Two options:\n"+
			"1. Provide claim token NOW (expires in 4 minutes)\n"+
			"   - Visit https://plex.tv/claim\n"+
			"   - Sign in and copy the token\n"+
			"2. Skip and claim via LOCAL NETWORK after containers start\n"+
			"   - Access http://YOUR_SERVER_IP:32400/web\n"+
			"   - Sign in to claim automatically\n\n"+
			"Which would you like to do?",
	)

	var choice string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("Plex Server Claiming").
				Description(instructions),
			huh.NewSelect[string]().
				Title("Choose claiming method").
				Options(
					huh.NewOption("Provide claim token now", "token"),
					huh.NewOption("Skip - claim via local network later", "skip"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if choice == "skip" {
		fmt.Println()
		fmt.Println(tui.WarningStyle.Render("⚠ Skipped Plex claiming"))
		fmt.Println("After containers start, access http://YOUR_SERVER_IP:32400/web to claim")
		fmt.Println()
		return nil
	}

	// User chose to provide token - collect it now
	var token string
	form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Plex Claim Token").
				Description("Paste your claim token from https://plex.tv/claim").
				Value(&token).
				Placeholder("claim-XXXXXXXXXXXXXXXXXXXX"),
		).Title("Plex Credentials"),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Validate token is not empty
	token = strings.TrimSpace(token)
	if token == "" {
		fmt.Println(tui.WarningStyle.Render("⚠ No token provided - server will be unclaimed"))
		return nil
	}

	// Write token to secrets file
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to write Plex claim token: %w", err)
	}

	fmt.Println(tui.SuccessStyle.Render("✓ Plex claim token saved"))
	fmt.Println()
	return nil
}
