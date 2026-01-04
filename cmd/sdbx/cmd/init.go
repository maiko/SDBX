package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/argon2"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/generator"
	"github.com/maiko/sdbx/internal/registry"
	"github.com/maiko/sdbx/internal/tui"
)

var (
	initDomain          string
	initExposeMode      string
	initRoutingStrategy string
	initTimezone        string
	initMediaPath       string
	initDownloadsPath   string
	initConfigPath      string
	initVPNEnabled      bool
	initVPNProvider     string
	initVPNCountry      string
	initSkipWizard      bool
	initAdminUser       string
	initAdminPassword   string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap a new SDBX project",
	Long: `Initialize a new SDBX seedbox project in the current directory.

This command will:
  • Guide you through configuration with an interactive wizard
  • Generate compose.yaml, .env, and config files
  • Create secrets for Authelia authentication
  • Set up directory structure for media and downloads

Use --skip-wizard with flags to run non-interactively.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&initDomain, "domain", "", "Base domain (e.g., box.sdbx.one)")
	initCmd.Flags().StringVar(&initExposeMode, "expose", "", "Exposure mode: lan, direct, or cloudflared")
	initCmd.Flags().StringVar(&initRoutingStrategy, "routing", "", "Routing strategy: subdomain or path")
	initCmd.Flags().StringVar(&initTimezone, "timezone", "", "Timezone (e.g., Europe/Paris)")
	initCmd.Flags().StringVar(&initMediaPath, "media", "", "Media storage path")
	initCmd.Flags().StringVar(&initDownloadsPath, "downloads", "", "Downloads storage path")
	initCmd.Flags().StringVar(&initConfigPath, "config", "", "Config storage path")
	initCmd.Flags().BoolVar(&initVPNEnabled, "vpn", false, "Enable VPN for downloads (requires --vpn-provider)")
	initCmd.Flags().StringVar(&initVPNProvider, "vpn-provider", "",
		"VPN provider (nordvpn, mullvad, pia, surfshark, protonvpn, expressvpn, etc.)")
	initCmd.Flags().StringVar(&initVPNCountry, "vpn-country", "France", "VPN server country")
	initCmd.Flags().BoolVar(&initSkipWizard, "skip-wizard", false, "Skip interactive wizard")
	initCmd.Flags().StringVar(&initAdminUser, "admin-user", "admin", "Admin username for Authelia")
	initCmd.Flags().StringVar(&initAdminPassword, "admin-password", "", "Admin password for Authelia (will be hashed)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check if directory is empty or confirm overwrite
	entries, err := os.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	hasExisting := false
	for _, entry := range entries {
		if entry.Name() == "compose.yaml" || entry.Name() == ".sdbx.yaml" {
			hasExisting = true
			break
		}
	}

	// Load existing config if available, otherwise default
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Initialize registry for addon selection
	reg, err := registry.NewWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// If not skipping wizard and TUI is enabled, run wizard
	if !initSkipWizard && IsTUIEnabled() {
		// Show logo with style
		fmt.Println()
		fmt.Println(tui.LogoStyled())
		fmt.Println()

		tagline := lipgloss.NewStyle().
			Foreground(tui.ColorMuted).
			Italic(true).
			Render("Seedbox in a Box — Setup Wizard")
		fmt.Println(tagline)
		fmt.Println()

		if hasExisting {
			var confirm bool
			if err := huh.NewConfirm().
				Title("Existing project detected. Overwrite?").
				Description("This will regenerate all configuration files").
				Value(&confirm).
				Run(); err != nil {
				return fmt.Errorf("confirmation prompt failed: %w", err)
			}
			if !confirm {
				fmt.Println(tui.MutedStyle.Render("Aborted."))
				return nil
			}
		}

		// Run interactive wizard
		if err := runWizard(cfg, reg); err != nil {
			return err
		}
	} else {
		// Non-interactive mode - use flags
		if initDomain != "" {
			cfg.Domain = initDomain
		}
		if initExposeMode != "" {
			cfg.Expose.Mode = initExposeMode
		}
		if initRoutingStrategy != "" {
			cfg.Routing.Strategy = initRoutingStrategy
		}
		if initTimezone != "" {
			cfg.Timezone = initTimezone
		}
		if initMediaPath != "" {
			cfg.MediaPath = initMediaPath
		}
		if initDownloadsPath != "" {
			cfg.DownloadsPath = initDownloadsPath
		}
		if initConfigPath != "" {
			cfg.ConfigPath = initConfigPath
		}
		// VPN configuration
		cfg.VPNEnabled = initVPNEnabled
		if initVPNEnabled {
			if initVPNProvider != "" {
				cfg.VPNProvider = initVPNProvider
			} else {
				cfg.VPNProvider = "custom"
			}
			cfg.VPNCountry = initVPNCountry
		}

		// Admin User Configuration
		cfg.AdminUser = initAdminUser
		password := initAdminPassword
		if password == "" {
			return fmt.Errorf("admin password is required: use --admin-password flag or run in interactive mode")
		}

		hash, err := generateArgon2Hash(password)
		if err != nil {
			return fmt.Errorf("failed to generate password hash: %w", err)
		}
		cfg.AdminPasswordHash = hash
	}

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Generate project using registry-based generator
	fmt.Println()
	fmt.Printf("  %s Generating project files...\n", tui.InfoStyle.Render(tui.IconSpinner))

	gen := generator.NewGeneratorWithRegistry(cfg, cwd, reg)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	// Create data directories if paths are relative
	if !filepath.IsAbs(cfg.MediaPath) {
		if err := gen.CreateDataDirs(); err != nil {
			return fmt.Errorf("failed to create data directories: %w", err)
		}
	}

	// Success message
	fmt.Println()
	printSuccessMessage(cfg)

	return nil
}

func runWizard(cfg *config.Config, reg *registry.Registry) error {
	// Define wizard steps for progress indicator
	wizardSteps := []string{
		"Domain & Routing",
		"Admin Credentials",
		"Storage Paths",
		"VPN Configuration",
		"System Settings",
		"Addons",
		"Confirmation",
	}
	progress := tui.NewStepProgress(wizardSteps...)

	// Helper to render step header
	renderStep := func() {
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Println()
		fmt.Println(tui.TitleStyle.Render("SDBX Setup Wizard"))
		fmt.Println()
		fmt.Println(progress.Render())
		fmt.Println()
	}

	// Step 1: Domain configuration
	renderStep()
	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Base Domain").
				Description("Your root domain for all services").
				Placeholder("box.sdbx.one").
				Value(&cfg.Domain).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("domain is required")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Exposure Mode").
				Description("How should services be accessible?").
				Options(
					huh.NewOption("Cloudflare Tunnel (recommended, zero open ports)", "cloudflared"),
					huh.NewOption("Direct HTTPS (Let's Encrypt, ports 80/443)", "direct"),
					huh.NewOption("LAN Only (HTTP, no TLS, for home lab)", "lan"),
				).
				Value(&cfg.Expose.Mode),

			huh.NewSelect[string]().
				Title("Routing Strategy").
				Description("How should services be accessed?").
				Options(
					huh.NewOption("Subdomain (radarr.domain.tld, sonarr.domain.tld)", "subdomain"),
					huh.NewOption("Path (sdbx.domain.tld/radarr, sdbx.domain.tld/sonarr)", "path"),
				).
				Value(&cfg.Routing.Strategy),
		).Title("Domain Configuration"),
	)

	if err := form1.Run(); err != nil {
		return err
	}

	// If path routing: ask for base subdomain
	if cfg.Routing.Strategy == config.RoutingStrategyPath {
		formBaseDomain := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Base Subdomain").
					Description("Subdomain for path-based access (e.g., 'sdbx' → sdbx.domain.tld/...)").
					Placeholder("sdbx").
					Value(&cfg.Routing.BaseDomain),
			).Title("Path Routing Configuration"),
		)
		if err := formBaseDomain.Run(); err != nil {
			return err
		}
	}

	// Step 2: Admin User
	progress.Next()
	renderStep()
	var adminPassword string
	formAuth := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Admin Username").
				Description("Username for Authelia SSO").
				Placeholder("admin").
				Value(&cfg.AdminUser),

			huh.NewInput().
				Title("Admin Password").
				Description("Password for Authelia (will be hashed securely)").
				Placeholder("secure_password").
				EchoMode(huh.EchoModePassword).
				Value(&adminPassword).
				Validate(func(s string) error {
					if len(s) < 4 {
						return fmt.Errorf("password must be at least 4 characters")
					}
					return nil
				}),
		).Title("Admin Configuration"),
	)

	if err := formAuth.Run(); err != nil {
		return err
	}

	// Hash password
	hash, err := generateArgon2Hash(adminPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	cfg.AdminPasswordHash = hash

	// Step 3: Storage configuration
	progress.Next()
	renderStep()
	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Media Path").
				Description("Where to store movies, TV shows, music").
				Placeholder("./data/media").
				Value(&cfg.MediaPath),

			huh.NewInput().
				Title("Downloads Path").
				Description("Where torrent client stores downloads").
				Placeholder("./data/downloads").
				Value(&cfg.DownloadsPath),

			huh.NewInput().
				Title("Config Path").
				Description("Where service configs are stored").
				Placeholder("./config").
				Value(&cfg.ConfigPath),
		).Title("Storage Configuration"),
	)

	if err := form2.Run(); err != nil {
		return err
	}

	// Step 4: VPN configuration
	progress.Next()
	renderStep()
	var wantVPN bool
	formVPN := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable VPN for downloads?").
				Description("Routes torrent traffic through VPN with kill-switch. Recommended for privacy.").
				Value(&wantVPN),
		).Title("VPN Configuration"),
	)

	if err := formVPN.Run(); err != nil {
		return err
	}

	cfg.VPNEnabled = wantVPN

	// Only ask VPN details if enabled
	if wantVPN {
		// Build provider options from config
		var providerOpts []huh.Option[string]
		for _, id := range config.GetVPNProviderIDs() {
			provider, _ := config.GetVPNProvider(id)
			providerOpts = append(providerOpts, huh.NewOption(provider.Name, id))
		}

		form3 := huh.NewForm(
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

		if err := form3.Run(); err != nil {
			return err
		}

		// Get provider info for credential form
		provider, ok := config.GetVPNProvider(cfg.VPNProvider)
		if !ok {
			return fmt.Errorf("unknown VPN provider: %s", cfg.VPNProvider)
		}

		// Build VPN type options based on provider support
		var vpnTypeOpts []huh.Option[string]
		if provider.SupportsWG {
			vpnTypeOpts = append(vpnTypeOpts, huh.NewOption("Wireguard (Recommended)", "wireguard"))
		}
		if provider.SupportsOpenVPN {
			vpnTypeOpts = append(vpnTypeOpts, huh.NewOption("OpenVPN", "openvpn"))
		}

		// VPN Type selection
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

		// Provider-specific credential forms
		if err := collectVPNCredentials(cfg, provider); err != nil {
			return err
		}
	}

	// Step 5: Timezone
	progress.Next()
	renderStep()
	form4 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Timezone").
				Description("System timezone for all services").
				Placeholder("Europe/Paris").
				Value(&cfg.Timezone),
		).Title("System Configuration"),
	)

	if err := form4.Run(); err != nil {
		return err
	}

	// Step 6: Addons - Load from registry
	progress.Next()
	renderStep()
	addonOptions, err := getAddonOptions(reg)
	if err != nil {
		return fmt.Errorf("failed to load addons: %w", err)
	}

	var selectedAddons []string
	form5 := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Optional Addons").
				Description("Select additional services to enable").
				Options(addonOptions...).
				Value(&selectedAddons),
		).Title("Addons"),
	)

	if err := form5.Run(); err != nil {
		return err
	}

	cfg.Addons = selectedAddons

	// Step 7: Confirmation
	progress.Next()
	renderStep()
	printConfigSummary(cfg)

	var confirm bool
	if err := huh.NewConfirm().
		Title("Generate project with these settings?").
		Value(&confirm).
		Run(); err != nil {
		return fmt.Errorf("confirmation prompt failed: %w", err)
	}

	if !confirm {
		return fmt.Errorf("aborted by user")
	}

	return nil
}

// getAddonOptions loads addon options from the registry
func getAddonOptions(reg *registry.Registry) ([]huh.Option[string], error) {
	ctx := context.Background()
	services, err := reg.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	var options []huh.Option[string]
	for _, svc := range services {
		if svc.IsAddon {
			label := fmt.Sprintf("%s - %s", capitalizeFirst(svc.Name), svc.Description)
			options = append(options, huh.NewOption(label, svc.Name))
		}
	}

	return options, nil
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// printConfigSummary prints a styled configuration summary
func printConfigSummary(cfg *config.Config) {
	fmt.Println()

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(tui.ColorPrimary).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(tui.ColorMuted).
		Width(14)

	valueStyle := lipgloss.NewStyle().
		Foreground(tui.ColorWhite)

	fmt.Println(headerStyle.Render("Configuration Summary"))

	printRow := func(label, value string) {
		fmt.Printf("  %s %s\n", labelStyle.Render(label+":"), valueStyle.Render(value))
	}

	printRow("Domain", cfg.Domain)
	printRow("Admin User", cfg.AdminUser)
	printRow("Expose Mode", cfg.Expose.Mode)
	printRow("Routing", cfg.Routing.Strategy)

	if cfg.Routing.Strategy == config.RoutingStrategyPath {
		printRow("Base Domain", fmt.Sprintf("%s.%s", cfg.Routing.BaseDomain, cfg.Domain))
	}

	printRow("Media Path", cfg.MediaPath)

	if cfg.VPNEnabled {
		vpnInfo := fmt.Sprintf("%s via %s", cfg.VPNProvider, cfg.VPNType)
		if cfg.VPNCountry != "" {
			vpnInfo += fmt.Sprintf(" (%s)", cfg.VPNCountry)
		}
		printRow("VPN", vpnInfo)
	} else {
		printRow("VPN", tui.MutedStyle.Render("disabled"))
	}

	printRow("Timezone", cfg.Timezone)

	if len(cfg.Addons) > 0 {
		printRow("Addons", strings.Join(cfg.Addons, ", "))
	}

	fmt.Println()
}

// printSuccessMessage prints the success message with next steps
func printSuccessMessage(cfg *config.Config) {
	var steps []string
	step := 1

	// Determine Authelia URL based on routing strategy
	var autheliaURL string
	if cfg.Routing.Strategy == config.RoutingStrategyPath {
		if cfg.Routing.BaseDomain != "" {
			autheliaURL = fmt.Sprintf("https://%s.%s/auth", cfg.Routing.BaseDomain, cfg.Domain)
		} else {
			autheliaURL = fmt.Sprintf("https://%s/auth", cfg.Domain)
		}
	} else {
		autheliaURL = fmt.Sprintf("https://auth.%s", cfg.Domain)
	}

	steps = append(steps, fmt.Sprintf("%d. Review and edit %s file", step, tui.CommandStyle.Render(".env")))
	step++

	if cfg.Expose.Mode == config.ExposeModeCloudflared {
		steps = append(steps, fmt.Sprintf("%d. Add tunnel token to %s", step, tui.CommandStyle.Render("secrets/cloudflared_tunnel_token.txt")))
		step++
	}

	steps = append(steps, fmt.Sprintf("%d. Run %s to start services", step, tui.CommandStyle.Render("sdbx up")))
	step++

	steps = append(steps, fmt.Sprintf("%d. Login at %s (User: %s)", step, tui.CommandStyle.Render(autheliaURL), cfg.AdminUser))
	step++

	steps = append(steps, fmt.Sprintf("%d. Run %s to verify setup", step, tui.CommandStyle.Render("sdbx doctor")))

	message := strings.Join(steps, "\n")
	fmt.Print(tui.RenderSuccessBox("Project initialized successfully!", message))
	fmt.Println()
}

// generateArgon2Hash generates an Argon2id hash compatible with Authelia
func generateArgon2Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Authelia defaults: time=3, memory=64MB, threads=4, keyLen=32
	time := uint32(3)
	memory := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)

	hash := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, time, threads, b64Salt, b64Hash), nil
}

// collectVPNCredentials collects VPN credentials based on provider auth type
func collectVPNCredentials(cfg *config.Config, provider config.VPNProvider) error {
	switch provider.AuthType {
	case config.VPNAuthUserPass:
		return collectUserPassCredentials(cfg, provider)
	case config.VPNAuthToken:
		return collectTokenCredentials(cfg, provider)
	case config.VPNAuthWireguard:
		return collectWireguardCredentials(cfg, provider)
	case config.VPNAuthConfig:
		// Custom config - just inform user
		fmt.Println("\n📝 Custom VPN configuration:")
		fmt.Println("   Place your .ovpn file in configs/gluetun/")
		fmt.Println("   Edit configs/gluetun/gluetun.env with your settings")
		return nil
	default:
		return fmt.Errorf("unsupported auth type: %s", provider.AuthType)
	}
}

// collectUserPassCredentials collects username/password credentials
func collectUserPassCredentials(cfg *config.Config, provider config.VPNProvider) error {
	usernameLabel := provider.UsernameLabel
	if usernameLabel == "" {
		usernameLabel = "Username"
	}
	passwordLabel := provider.PasswordLabel
	if passwordLabel == "" {
		passwordLabel = "Password"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(usernameLabel).
				Value(&cfg.VPNUsername),

			huh.NewInput().
				Title(passwordLabel).
				EchoMode(huh.EchoModePassword).
				Value(&cfg.VPNPassword),
		).Title("VPN Credentials"),
	)

	return form.Run()
}

// collectTokenCredentials collects token-based credentials (Mullvad, IVPN, AirVPN)
func collectTokenCredentials(cfg *config.Config, provider config.VPNProvider) error {
	tokenLabel := provider.TokenLabel
	if tokenLabel == "" {
		tokenLabel = "Account Token"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(tokenLabel).
				Description("This will be stored securely in your gluetun.env file").
				Value(&cfg.VPNToken),
		).Title("VPN Credentials"),
	)

	return form.Run()
}

// collectWireguardCredentials collects Wireguard credentials (e.g., NordVPN)
func collectWireguardCredentials(cfg *config.Config, provider config.VPNProvider) error {
	// For Wireguard providers, we need the private key
	// For OpenVPN fallback, we need username/password
	if cfg.VPNType == "wireguard" {
		keyLabel := provider.TokenLabel
		if keyLabel == "" {
			keyLabel = "Wireguard Private Key"
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(keyLabel).
					Description("Your Wireguard private key from the provider's setup page").
					Value(&cfg.VPNWireguardKey),
			).Title("Wireguard Credentials"),
		)

		return form.Run()
	}

	// OpenVPN fallback
	return collectUserPassCredentials(cfg, provider)
}
