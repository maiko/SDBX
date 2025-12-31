package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/secrets"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage SDBX secrets",
	Long: `Generate and rotate secrets used by SDBX services.

Secrets are stored in the secrets/ directory and include:
  • Authelia JWT, session, and storage encryption keys
  • VPN credentials (user-provided)
  • Cloudflare tunnel token (user-provided)`,
}

var secretsGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate missing secrets",
	Long: `Generate any missing secret files.

This command will create new random secrets for Authelia and other services.
Existing secrets are preserved.`,
	RunE: runSecretsGenerate,
}

var secretsRotateCmd = &cobra.Command{
	Use:   "rotate [name]",
	Short: "Rotate a secret",
	Long: `Rotate (regenerate) a specific secret.

Warning: Rotating secrets may require restarting services.

Example:
  sdbx secrets rotate authelia_jwt_secret.txt`,
	Args: cobra.ExactArgs(1),
	RunE: runSecretsRotate,
}

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets and their status",
	Long:  `Display all managed secrets and whether they are configured.`,
	RunE:  runSecretsList,
}

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.AddCommand(secretsGenerateCmd)
	secretsCmd.AddCommand(secretsRotateCmd)
	secretsCmd.AddCommand(secretsListCmd)
}

func runSecretsGenerate(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		projectDir = "."
	}

	secretsDir := filepath.Join(projectDir, "secrets")

	fmt.Println(tui.InfoStyle.Render("Generating secrets..."))
	fmt.Println()

	if err := secrets.GenerateSecrets(secretsDir); err != nil {
		return fmt.Errorf("failed to generate secrets: %w", err)
	}

	// List what was created
	status, _ := secrets.ListSecrets(secretsDir)
	for name, configured := range status {
		if configured {
			fmt.Printf("  %s %s\n", tui.SuccessStyle.Render(tui.IconSuccess), name)
		} else {
			fmt.Printf("  %s %s %s\n", tui.WarningStyle.Render(tui.IconWarning), name, tui.MutedStyle.Render("(needs manual config)"))
		}
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("✓ Secrets generated"))

	return nil
}

func runSecretsRotate(_ *cobra.Command, args []string) error {
	name := args[0]

	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		projectDir = "."
	}

	secretsDir := filepath.Join(projectDir, "secrets")

	fmt.Printf("Rotating %s...\n", name)

	newSecret, err := secrets.RotateSecret(secretsDir, name)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Rotated %s", name)))
	fmt.Println(tui.MutedStyle.Render(fmt.Sprintf("  New value: %s...", newSecret[:16])))
	fmt.Println()
	fmt.Println(tui.WarningStyle.Render("⚠ You may need to restart services: sdbx restart"))

	return nil
}

func runSecretsList(_ *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		projectDir = "."
	}

	secretsDir := filepath.Join(projectDir, "secrets")

	status, err := secrets.ListSecrets(secretsDir)
	if err != nil {
		return err
	}

	// JSON output
	if IsJSONOutput() {
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(tui.TitleStyle.Render("SDBX Secrets"))
	fmt.Println()

	configured := 0
	missing := 0

	for name, isConfigured := range status {
		if isConfigured {
			fmt.Printf("  %s %s\n", tui.SuccessStyle.Render(tui.IconSuccess), name)
			configured++
		} else {
			fmt.Printf("  %s %s %s\n", tui.ErrorStyle.Render(tui.IconError), name, tui.MutedStyle.Render("(empty)"))
			missing++
		}
	}

	fmt.Println()
	if missing > 0 {
		fmt.Println(tui.WarningStyle.Render(fmt.Sprintf("%d of %d secrets need configuration", missing, configured+missing)))
	} else {
		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ All %d secrets configured", configured)))
	}

	return nil
}
