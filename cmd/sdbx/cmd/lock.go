package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Manage service version lock file",
	Long: `Manage the .sdbx.lock file that pins service versions and image digests.

Lock files ensure reproducible deployments by recording:
- Source commit hashes
- Service definition versions
- Container image digests

Examples:
  sdbx lock                    # Generate/update lock file
  sdbx lock verify             # Verify lock file integrity
  sdbx lock diff               # Show differences from lock
  sdbx lock update [service]   # Update specific service in lock`,
	RunE: runLockGenerate,
}

var lockVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify lock file integrity",
	Long: `Verify that the lock file matches current sources and services.

Returns exit code 0 if everything is in sync, 1 if there are differences.`,
	RunE: runLockVerify,
}

var lockDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences from lock file",
	Long: `Show what would change if the lock file was regenerated.

Useful to see if source updates introduce new versions.`,
	RunE: runLockDiff,
}

var lockUpdateCmd = &cobra.Command{
	Use:   "update [service...]",
	Short: "Update services in lock file",
	Long: `Update specific services in the lock file, or all if none specified.

Examples:
  sdbx lock update             # Update all services
  sdbx lock update sonarr      # Update only sonarr
  sdbx lock update radarr plex # Update multiple services`,
	RunE: runLockUpdate,
}

func init() {
	rootCmd.AddCommand(lockCmd)
	lockCmd.AddCommand(lockVerifyCmd)
	lockCmd.AddCommand(lockDiffCmd)
	lockCmd.AddCommand(lockUpdateCmd)
}

func runLockGenerate(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	reg, err := getRegistry()
	if err != nil {
		return err
	}

	// Generate lock file
	lockFile, err := reg.GenerateLockFile(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to generate lock file: %w", err)
	}

	// Save lock file
	loader := registry.NewLoader()
	if err := loader.SaveLockFile(".sdbx.lock", lockFile); err != nil {
		return fmt.Errorf("failed to save lock file: %w", err)
	}

	// JSON output
	if IsJSONOutput() {
		data, _ := json.MarshalIndent(lockFile, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(tui.SuccessStyle.Render("✓ Generated .sdbx.lock"))
	fmt.Println()
	fmt.Printf("Locked %d sources, %d services\n",
		len(lockFile.Sources),
		len(lockFile.Services),
	)

	return nil
}

func runLockVerify(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	reg, err := getRegistry()
	if err != nil {
		return err
	}

	// Load existing lock file
	loader := registry.NewLoader()
	existing, err := loader.LoadLockFile(".sdbx.lock")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(tui.WarningStyle.Render("No lock file found"))
			fmt.Println()
			fmt.Printf("Run '%s' to generate one\n", tui.CommandStyle.Render("sdbx lock"))
			return nil
		}
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Generate current lock file
	current, err := reg.GenerateLockFile(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to generate current state: %w", err)
	}

	// Compare
	diffs := reg.DiffLockFiles(existing, current)

	// JSON output
	if IsJSONOutput() {
		result := map[string]interface{}{
			"valid":       len(diffs) == 0,
			"differences": diffs,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		if len(diffs) > 0 {
			os.Exit(1)
		}
		return nil
	}

	if len(diffs) == 0 {
		fmt.Println(tui.SuccessStyle.Render("✓ Lock file is valid and up-to-date"))
		return nil
	}

	fmt.Println(tui.WarningStyle.Render("Lock file has differences:"))
	fmt.Println()
	for _, diff := range diffs {
		fmt.Printf("  %s: %s\n", tui.InfoStyle.Render(diff.Type), diff.Description)
	}
	fmt.Println()
	fmt.Printf("Run '%s' to update the lock file\n", tui.CommandStyle.Render("sdbx lock"))

	os.Exit(1)
	return nil
}

func runLockDiff(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	reg, err := getRegistry()
	if err != nil {
		return err
	}

	// Load existing lock file
	loader := registry.NewLoader()
	existing, err := loader.LoadLockFile(".sdbx.lock")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(tui.MutedStyle.Render("No lock file found - showing what would be generated"))
			fmt.Println()

			// Generate and display what would be created
			current, err := reg.GenerateLockFile(ctx, cfg)
			if err != nil {
				return fmt.Errorf("failed to generate lock file: %w", err)
			}

			for sourceName := range current.Sources {
				fmt.Printf("  %s source: %s\n", tui.SuccessStyle.Render("+"), sourceName)
			}
			for serviceName := range current.Services {
				fmt.Printf("  %s service: %s\n", tui.SuccessStyle.Render("+"), serviceName)
			}
			return nil
		}
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	// Generate current lock file
	current, err := reg.GenerateLockFile(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to generate current state: %w", err)
	}

	// Compare
	diffs := reg.DiffLockFiles(existing, current)

	// JSON output
	if IsJSONOutput() {
		data, _ := json.MarshalIndent(diffs, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(diffs) == 0 {
		fmt.Println(tui.MutedStyle.Render("No differences found"))
		return nil
	}

	fmt.Println(tui.TitleStyle.Render("Lock File Differences"))
	fmt.Println()

	for _, diff := range diffs {
		var icon string
		switch diff.Type {
		case "added":
			icon = tui.SuccessStyle.Render("+")
		case "removed":
			icon = tui.ErrorStyle.Render("-")
		case "changed":
			icon = tui.WarningStyle.Render("~")
		default:
			icon = " "
		}
		fmt.Printf("  %s %s\n", icon, diff.Description)
	}

	return nil
}

func runLockUpdate(_ *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	reg, err := getRegistry()
	if err != nil {
		return err
	}

	// Load existing lock file
	loader := registry.NewLoader()
	existing, err := loader.LoadLockFile(".sdbx.lock")
	if err != nil {
		if os.IsNotExist(err) {
			// No existing lock file, generate new one
			return runLockGenerate(nil, nil)
		}
		return fmt.Errorf("failed to load lock file: %w", err)
	}

	var servicesToUpdate []string
	if len(args) > 0 {
		servicesToUpdate = args
	}

	// Update lock file
	updated, err := reg.UpdateLockFile(ctx, cfg, existing, servicesToUpdate)
	if err != nil {
		return fmt.Errorf("failed to update lock file: %w", err)
	}

	// Save updated lock file
	if err := loader.SaveLockFile(".sdbx.lock", updated); err != nil {
		return fmt.Errorf("failed to save lock file: %w", err)
	}

	// JSON output
	if IsJSONOutput() {
		data, _ := json.MarshalIndent(updated, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(servicesToUpdate) > 0 {
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Updated %d services in .sdbx.lock", len(servicesToUpdate))))
	} else {
		fmt.Println(tui.SuccessStyle.Render("✓ Updated .sdbx.lock"))
	}

	return nil
}
