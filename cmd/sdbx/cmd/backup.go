package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/maiko/sdbx/internal/backup"
	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/tui"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore SDBX configuration",
	Long: `Backup and restore your SDBX configuration, secrets, and service configs.

Backups include:
  - Configuration (.sdbx.yaml, .sdbx.lock)
  - Docker Compose file
  - Secrets
  - Service configurations

Backups DO NOT include:
  - Media files
  - Downloads
  - Docker volumes/data

Examples:
  sdbx backup create           # Create backup
  sdbx backup list             # List all backups
  sdbx backup restore <name>   # Restore from backup
  sdbx backup delete <name>    # Delete backup`,
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new backup",
	RunE:  runBackupCreate,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backups",
	RunE:  runBackupList,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-name>",
	Short: "Restore from backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupRestore,
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete <backup-name>",
	Short: "Delete a backup",
	Args:  cobra.ExactArgs(1),
	RunE:  runBackupDelete,
}

// Flags
var (
	backupOutput string
)

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)

	// Flags
	backupCreateCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Custom backup output directory")
}

func runBackupCreate(_ *cobra.Command, _ []string) error {
	// Get project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return fmt.Errorf("not in an SDBX project directory")
	}

	// Change to project directory
	if err := os.Chdir(projectDir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// Create backup manager
	manager := backup.NewManager(projectDir)

	ctx := context.Background()

	if !IsJSONOutput() {
		fmt.Println(tui.TitleStyle.Render("Creating Backup"))
		fmt.Println()
	}

	// Create backup
	b, err := manager.Create(ctx)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Get backup size
	size, _ := b.GetSize()

	// JSON output
	if IsJSONOutput() {
		return OutputJSON(map[string]interface{}{
			"name":      b.Name,
			"path":      b.Path,
			"size":      size,
			"timestamp": b.Metadata.Timestamp,
		})
	}

	// Human-readable output
	fmt.Println(tui.SuccessStyle.Render("✓ Backup created successfully"))
	fmt.Println()
	fmt.Printf("%s  %s\n", tui.MutedStyle.Render("Name:"), b.Name)
	fmt.Printf("%s  %s\n", tui.MutedStyle.Render("Path:"), b.Path)
	fmt.Printf("%s  %s\n", tui.MutedStyle.Render("Size:"), backup.FormatBytes(size))
	fmt.Printf("%s  %s\n", tui.MutedStyle.Render("Time:"), b.Metadata.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	return nil
}

func runBackupList(_ *cobra.Command, _ []string) error {
	// Get project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return fmt.Errorf("not in an SDBX project directory")
	}

	// Create backup manager
	manager := backup.NewManager(projectDir)

	ctx := context.Background()

	// List backups
	backups, err := manager.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// JSON output
	if IsJSONOutput() {
		result := make([]map[string]interface{}, 0, len(backups))
		for _, b := range backups {
			size, _ := b.GetSize()
			result = append(result, map[string]interface{}{
				"name":      b.Name,
				"path":      b.Path,
				"size":      size,
				"timestamp": b.Metadata.Timestamp,
				"hostname":  b.Metadata.Hostname,
			})
		}
		return OutputJSON(result)
	}

	// Human-readable output
	if len(backups) == 0 {
		fmt.Println(tui.MutedStyle.Render("No backups found"))
		return nil
	}

	fmt.Println(tui.TitleStyle.Render("Available Backups"))
	fmt.Println()

	table := tui.NewTable("Name", "Date", "Size", "Hostname")

	for _, b := range backups {
		size, _ := b.GetSize()
		age := backup.FormatAge(b.Metadata.Timestamp)

		table.AddRow(b.Name, age, backup.FormatBytes(size), b.Metadata.Hostname)
	}
	fmt.Println(table.Render())

	return nil
}

func runBackupRestore(_ *cobra.Command, args []string) error {
	backupName := args[0]

	// Get project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return fmt.Errorf("not in an SDBX project directory")
	}

	// Change to project directory
	if err := os.Chdir(projectDir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// Create backup manager
	manager := backup.NewManager(projectDir)

	ctx := context.Background()

	if !IsJSONOutput() {
		fmt.Println(tui.TitleStyle.Render("Restoring Backup"))
		fmt.Println()
		fmt.Printf("%s  %s\n", tui.MutedStyle.Render("Backup:"), backupName)
		fmt.Println()
	}

	// Restore backup
	if err := manager.Restore(ctx, backupName); err != nil {
		return fmt.Errorf("failed to restore backup: %w\n\n  Try: sdbx backup list", err)
	}

	// JSON output
	if IsJSONOutput() {
		return OutputJSON(map[string]interface{}{
			"success": true,
			"backup":  backupName,
		})
	}

	// Human-readable output
	fmt.Println(tui.SuccessStyle.Render("✓ Backup restored successfully"))
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render("Run 'sdbx up' to apply the restored configuration"))

	return nil
}

func runBackupDelete(_ *cobra.Command, args []string) error {
	backupName := args[0]

	// Get project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return fmt.Errorf("not in an SDBX project directory")
	}

	// Create backup manager
	manager := backup.NewManager(projectDir)

	ctx := context.Background()

	// Delete backup
	if err := manager.Delete(ctx, backupName); err != nil {
		return fmt.Errorf("failed to delete backup: %w\n\n  Try: sdbx backup list", err)
	}

	// JSON output
	if IsJSONOutput() {
		return OutputJSON(map[string]interface{}{
			"success": true,
			"deleted": backupName,
		})
	}

	// Human-readable output
	fmt.Println(tui.SuccessStyle.Render("✓ Backup deleted successfully"))

	return nil
}

