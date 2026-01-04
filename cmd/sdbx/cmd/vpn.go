package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/generator"
	"github.com/maiko/sdbx/internal/tui"
)

var vpnCmd = &cobra.Command{
	Use:   "vpn",
	Short: "Manage VPN configuration",
	Long:  `Manage VPN provider and credentials configuration.`,
}

var vpnConfigureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure or reconfigure VPN credentials",
	Long: `Configure or reconfigure VPN provider and credentials.

This command allows you to update your VPN credentials after initial setup.
It will regenerate the gluetun.env file with the new credentials.

Example:
  sdbx vpn configure           # Interactive configuration
  sdbx vpn configure --provider nordvpn  # Pre-select provider`,
	RunE: runVPNConfigure,
}

var (
	vpnProviderFlag string
)

func init() {
	rootCmd.AddCommand(vpnCmd)
	vpnCmd.AddCommand(vpnConfigureCmd)

	vpnConfigureCmd.Flags().StringVar(&vpnProviderFlag, "provider", "", "VPN provider to configure")
}

func runVPNConfigure(cmd *cobra.Command, args []string) error {
	// Find project directory
	projectDir, err := config.ProjectDir()
	if err != nil {
		return fmt.Errorf("not in an sdbx project directory: %w", err)
	}

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if VPN is enabled
	if !cfg.VPNEnabled {
		fmt.Println(tui.WarningStyle.Render("VPN is not enabled in this project."))

		var enableVPN bool
		if err := huh.NewConfirm().
			Title("Would you like to enable VPN?").
			Value(&enableVPN).
			Run(); err != nil {
			return err
		}

		if !enableVPN {
			return nil
		}
		cfg.VPNEnabled = true
	}

	// Provider selection
	if vpnProviderFlag != "" {
		cfg.VPNProvider = vpnProviderFlag
	}

	// Build provider options
	var providerOpts []huh.Option[string]
	for _, id := range config.GetVPNProviderIDs() {
		provider, _ := config.GetVPNProvider(id)
		providerOpts = append(providerOpts, huh.NewOption(provider.Name, id))
	}

	// Interactive provider selection if not provided
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("VPN Provider").
				Description("Select your VPN service").
				Options(providerOpts...).
				Value(&cfg.VPNProvider),

			huh.NewInput().
				Title("VPN Server Country").
				Description("Preferred VPN exit location (e.g., Netherlands, United States)").
				Placeholder("Netherlands").
				Value(&cfg.VPNCountry),
		).Title("VPN Provider"),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Get provider info
	provider, ok := config.GetVPNProvider(cfg.VPNProvider)
	if !ok {
		return fmt.Errorf("unknown VPN provider: %s", cfg.VPNProvider)
	}

	// VPN type selection
	var vpnTypeOpts []huh.Option[string]
	if provider.SupportsWG {
		vpnTypeOpts = append(vpnTypeOpts, huh.NewOption("Wireguard (Recommended)", "wireguard"))
	}
	if provider.SupportsOpenVPN {
		vpnTypeOpts = append(vpnTypeOpts, huh.NewOption("OpenVPN", "openvpn"))
	}

	formType := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("VPN Protocol").
				Description("Wireguard is faster and more reliable, OpenVPN has wider compatibility").
				Options(vpnTypeOpts...).
				Value(&cfg.VPNType),
		).Title("VPN Protocol"),
	)

	if err := formType.Run(); err != nil {
		return err
	}

	// Show credentials link
	if provider.CredDocsURL != "" {
		fmt.Printf("\n📋 Get your credentials from: %s\n", provider.CredDocsURL)
		if provider.Notes != "" {
			fmt.Printf("   Note: %s\n\n", provider.Notes)
		}
	}

	// Collect credentials using shared function from init.go
	if err := collectVPNCredentials(cfg, provider); err != nil {
		return err
	}

	// Generate gluetun.env file
	if err := generateGluetunEnv(projectDir, cfg); err != nil {
		return fmt.Errorf("failed to generate gluetun.env: %w", err)
	}

	// Save updated config (VPN enabled, provider, type, country)
	configPath := filepath.Join(projectDir, ".sdbx.yaml")
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Print(tui.RenderSuccessBox("VPN configured successfully!",
		"Run 'sdbx down && sdbx up' to apply the new VPN settings."))
	fmt.Println()

	return nil
}

// generateGluetunEnv generates the gluetun.env file from the template
func generateGluetunEnv(projectDir string, cfg *config.Config) error {
	// Ensure configs/gluetun directory exists
	gluetunDir := filepath.Join(projectDir, "configs", "gluetun")
	if err := os.MkdirAll(gluetunDir, 0o755); err != nil {
		return fmt.Errorf("failed to create gluetun config directory: %w", err)
	}

	// Load and execute template
	tmplContent, err := generator.TemplatesFS.ReadFile("templates/gluetun.env.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read gluetun.env template: %w", err)
	}

	tmpl, err := template.New("gluetun.env").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse gluetun.env template: %w", err)
	}

	// Create output file
	outputPath := filepath.Join(gluetunDir, "gluetun.env")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create gluetun.env: %w", err)
	}
	defer f.Close()

	// Execute template
	data := struct {
		Config *config.Config
	}{
		Config: cfg,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute gluetun.env template: %w", err)
	}

	return nil
}

var vpnStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show VPN configuration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Println()
		fmt.Println(tui.TitleStyle.Render("VPN Status"))
		fmt.Println()

		if !cfg.VPNEnabled {
			fmt.Println(tui.MutedStyle.Render("  VPN is not enabled"))
			fmt.Println()
			fmt.Println("  Run 'sdbx vpn configure' to enable and configure VPN")
			return nil
		}

		provider, ok := config.GetVPNProvider(cfg.VPNProvider)
		providerName := cfg.VPNProvider
		if ok {
			providerName = provider.Name
		}

		fmt.Printf("  Provider: %s\n", providerName)
		fmt.Printf("  Protocol: %s\n", cfg.VPNType)
		if cfg.VPNCountry != "" {
			fmt.Printf("  Country:  %s\n", cfg.VPNCountry)
		}

		// Check if gluetun.env exists
		projectDir, err := config.ProjectDir()
		if err == nil {
			envPath := filepath.Join(projectDir, "configs", "gluetun", "gluetun.env")
			if _, err := os.Stat(envPath); err == nil {
				fmt.Printf("  Config:   %s\n", tui.SuccessStyle.Render("✓ gluetun.env configured"))
			} else {
				fmt.Printf("  Config:   %s\n", tui.ErrorStyle.Render("✗ gluetun.env missing"))
				fmt.Println()
				fmt.Println("  Run 'sdbx vpn configure' to generate the config file")
			}
		}

		fmt.Println()
		return nil
	},
}

var vpnProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List supported VPN providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if IsJSONOutput() {
			providers := make([]map[string]interface{}, 0)
			for _, id := range config.GetVPNProviderIDs() {
				p, _ := config.GetVPNProvider(id)
				providers = append(providers, map[string]interface{}{
					"id":               id,
					"name":             p.Name,
					"auth_type":        string(p.AuthType),
					"supports_wg":      p.SupportsWG,
					"supports_openvpn": p.SupportsOpenVPN,
					"docs_url":         p.CredDocsURL,
				})
			}
			return OutputJSON(providers)
		}

		fmt.Println()
		fmt.Println(tui.TitleStyle.Render("Supported VPN Providers"))
		fmt.Println()

		for _, id := range config.GetVPNProviderIDs() {
			p, _ := config.GetVPNProvider(id)

			protocols := []string{}
			if p.SupportsWG {
				protocols = append(protocols, "WireGuard")
			}
			if p.SupportsOpenVPN {
				protocols = append(protocols, "OpenVPN")
			}

			authType := "Username/Password"
			switch p.AuthType {
			case config.VPNAuthToken:
				authType = "Token/Account ID"
			case config.VPNAuthWireguard:
				authType = "WireGuard Key"
			case config.VPNAuthConfig:
				authType = "Custom Config"
			}

			fmt.Printf("  %s (%s)\n", tui.SelectedStyle.Render(p.Name), id)
			fmt.Printf("    Auth: %s | Protocols: %s\n", authType, strings.Join(protocols, ", "))
		}

		fmt.Println()
		return nil
	},
}

func init() {
	vpnCmd.AddCommand(vpnStatusCmd)
	vpnCmd.AddCommand(vpnProvidersCmd)
}
