package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/generator"
	"github.com/maiko/sdbx/internal/tui"
)

var regenerateCmd = &cobra.Command{
	Use:     "regenerate",
	Aliases: []string{"regen"},
	Short:   "Regenerate compose.yaml and config files from current configuration",
	Long: `Regenerate all project files (compose.yaml, integration configs, etc.)
from the current .sdbx.yaml configuration.

This is useful after:
  • Enabling or disabling addons
  • Changing configuration values (domain, routing, VPN, etc.)
  • Updating service definitions from sources

The command reads your existing .sdbx.yaml, validates it, resolves
services from the registry, and regenerates all output files.

Note: This does NOT restart services. Run 'sdbx up' after regenerating
to apply changes.`,
	RunE: runRegenerate,
}

func init() {
	rootCmd.AddCommand(regenerateCmd)
}

func runRegenerate(_ *cobra.Command, _ []string) error {
	// Load existing configuration
	cfg, err := config.Load()
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(
				"no .sdbx.yaml found in current directory\n\n" +
					"Hint: Run 'sdbx init' first to create a project",
			)
		}
		return fmt.Errorf(
			"failed to load configuration: %w\n\nHint: Check .sdbx.yaml for syntax errors",
			err,
		)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w\n\nHint: Run 'sdbx config get' to inspect current values", err)
	}

	outputDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// JSON output mode
	if IsJSONOutput() {
		gen := generator.NewGenerator(cfg, outputDir)
		if err := gen.Generate(); err != nil {
			return OutputJSON(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}
		return OutputJSON(map[string]interface{}{
			"success": true,
			"message": "Project files regenerated successfully",
		})
	}

	// TUI mode with spinner
	if IsTUIEnabled() {
		genErr := tui.RunWithSpinner("Regenerating project files...", func() error {
			gen := generator.NewGenerator(cfg, outputDir)
			return gen.Generate()
		})

		if genErr != nil {
			fmt.Println(tui.IconError + " Regeneration failed: " + genErr.Error())
			return genErr
		}

		fmt.Println(tui.IconSuccess + " Project files regenerated successfully")
		fmt.Println()
		fmt.Println(tui.IconInfo + " Run 'sdbx up' to apply changes")
		return nil
	}

	// Plain text mode
	fmt.Println("Regenerating project files...")
	gen := generator.NewGenerator(cfg, outputDir)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("regeneration failed: %w", err)
	}

	fmt.Println("Project files regenerated successfully.")
	fmt.Println("Run 'sdbx up' to apply changes.")
	return nil
}
