package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var (
	backupName string
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore SDBX data",
	Long: `Create and restore backups of SDBX configuration and data.

Backups include:
  • Configuration files (configs/, .env, .sdbx.yaml)
  • Secrets (secrets/)
  • Service data (config/)

Media files are NOT included in backups.`,
}

var backupRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Create a backup",
	Long: `Create a backup of SDBX configuration and data.

The backup is stored as a timestamped tarball in the backups/ directory.`,
	RunE: runBackupRun,
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	RunE:  runBackupList,
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup-file>",
	Short: "Restore from a backup",
	Long: `Restore SDBX configuration and data from a backup.

Warning: This will overwrite existing configuration!`,
	Args: cobra.ExactArgs(1),
	RunE: runBackupRestore,
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupRunCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)

	backupRunCmd.Flags().StringVar(&backupName, "name", "", "Custom backup name (default: timestamp)")
}

func runBackupRun(_ *cobra.Command, args []string) error {
	projectDir, err := config.ProjectDir()
	if err != nil {
		return err
	}

	// Create backups directory
	backupsDir := filepath.Join(projectDir, "backups")
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create backups directory: %w", err)
	}

	// Generate backup filename
	if backupName == "" {
		backupName = time.Now().Format("20060102-150405")
	}
	backupFile := filepath.Join(backupsDir, fmt.Sprintf("sdbx-backup-%s.tar.gz", backupName))

	fmt.Println(tui.InfoStyle.Render("Creating backup..."))
	fmt.Println()

	// Directories to backup
	dirsToBackup := []string{
		"configs",
		"secrets",
		"config",
		".env",
		".sdbx.yaml",
		"compose.yaml",
	}

	// Filter existing directories
	var existingPaths []string
	for _, dir := range dirsToBackup {
		path := filepath.Join(projectDir, dir)
		if _, err := os.Stat(path); err == nil {
			existingPaths = append(existingPaths, dir)
		}
	}

	if len(existingPaths) == 0 {
		return fmt.Errorf("no files to backup")
	}

	// Create tar command
	tarArgs := append([]string{"-czf", backupFile}, existingPaths...)
	tarCmd := exec.CommandContext(context.Background(), "tar", tarArgs...)
	tarCmd.Dir = projectDir

	if output, err := tarCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("backup failed: %w\n%s", err, output)
	}

	// Get backup size
	info, err := os.Stat(backupFile)
	size := 0.0
	if err == nil {
		size = float64(info.Size()) / 1024 / 1024 // MB
	}

	fmt.Printf("  %s configs/\n", tui.SuccessStyle.Render(tui.IconSuccess))
	fmt.Printf("  %s secrets/\n", tui.SuccessStyle.Render(tui.IconSuccess))
	fmt.Printf("  %s config/\n", tui.SuccessStyle.Render(tui.IconSuccess))
	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Backup created: %s (%.2f MB)", filepath.Base(backupFile), size)))

	return nil
}

func runBackupList(_ *cobra.Command, args []string) error {
	projectDir, err := config.ProjectDir()
	if err != nil {
		projectDir = "."
	}

	backupsDir := filepath.Join(projectDir, "backups")
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(tui.MutedStyle.Render("No backups found. Run 'sdbx backup run' to create one."))
			return nil
		}
		return err
	}

	fmt.Println(tui.TitleStyle.Render("SDBX Backups"))
	fmt.Println()

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".gz" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip entries where we can't get info
		}
		size := float64(info.Size()) / 1024 / 1024
		modTime := info.ModTime().Format("2006-01-02 15:04")

		fmt.Printf("  %s  %s  %.2f MB\n", modTime, entry.Name(), size)
		count++
	}

	if count == 0 {
		fmt.Println(tui.MutedStyle.Render("No backups found."))
	} else {
		fmt.Println()
		fmt.Printf("%d backup(s) found\n", count)
	}

	return nil
}

func runBackupRestore(_ *cobra.Command, args []string) error {
	backupFile := args[0]

	projectDir, err := config.ProjectDir()
	if err != nil {
		projectDir = "."
	}

	// Check if backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		// Try in backups directory
		backupFile = filepath.Join(projectDir, "backups", backupFile)
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			return fmt.Errorf("backup file not found: %s", args[0])
		}
	}

	fmt.Println(tui.WarningStyle.Render("⚠ This will overwrite existing configuration!"))
	fmt.Println()
	fmt.Println(tui.InfoStyle.Render("Restoring backup..."))

	// Extract backup
	tarCmd := exec.CommandContext(context.Background(), "tar", "-xzf", backupFile)
	tarCmd.Dir = projectDir

	if output, err := tarCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restore failed: %w\n%s", err, output)
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("✓ Backup restored successfully"))
	fmt.Println()
	fmt.Println(tui.MutedStyle.Render("Run 'sdbx up' to start services with restored configuration"))

	return nil
}
