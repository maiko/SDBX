package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/maiko/sdbx/internal/backup"
	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
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
  sdbx backup                  # Create backup
  sdbx backup list             # List all backups
  sdbx backup restore <name>   # Restore from backup
  sdbx backup delete <name>    # Delete backup`,
	RunE: runBackupCreate,
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
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)

	// Flags
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Custom backup output directory")
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
	fmt.Printf("%s  %s\n", tui.MutedStyle.Render("Size:"), formatBytes(size))
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	headers := tui.TableHeaderStyle.Render("NAME") + "\t" +
		tui.TableHeaderStyle.Render("DATE") + "\t" +
		tui.TableHeaderStyle.Render("SIZE") + "\t" +
		tui.TableHeaderStyle.Render("HOSTNAME")
	fmt.Fprintln(w, headers)

	for _, b := range backups {
		size, _ := b.GetSize()
		age := formatAge(b.Metadata.Timestamp)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			b.Name,
			age,
			formatBytes(size),
			b.Metadata.Hostname,
		)
	}
	w.Flush()

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
		return fmt.Errorf("failed to restore backup: %w", err)
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
		return fmt.Errorf("failed to delete backup: %w", err)
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

// formatBytes formats bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatAge formats a timestamp as a relative time
func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}

	// For older backups, show full date
	return t.Format("2006-01-02 15:04")
}
