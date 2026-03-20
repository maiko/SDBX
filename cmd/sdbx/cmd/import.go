package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/tui"
)

var (
	importFile   string
	importDryRun bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import configuration from an existing Docker Compose file",
	Long: `Analyze an existing Docker Compose file and generate SDBX configuration.

This command detects known services (Sonarr, Radarr, Plex, etc.) from your
existing Docker Compose setup and creates an SDBX config that matches your
setup as closely as possible.

This is a best-effort migration helper. After import, review the generated
.sdbx.yaml and run 'sdbx init' to finalize any settings.

Examples:
  sdbx import                              # Import from docker-compose.yml
  sdbx import --file compose.yaml          # Import from a specific file
  sdbx import --dry-run                    # Preview what would be imported
  sdbx import --dry-run --json             # Machine-readable preview`,
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVar(&importFile, "file", "docker-compose.yml", "Docker Compose file to import")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Show what would be imported without writing files")
}

// composeFile represents a minimal Docker Compose file structure.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

// composeService represents a service in a Docker Compose file.
type composeService struct {
	Image       string            `yaml:"image"`
	Environment interface{}       `yaml:"environment"` // Can be map or list
	Volumes     []string          `yaml:"volumes"`
	Labels      interface{}       `yaml:"labels"` // Can be map or list
	NetworkMode string            `yaml:"network_mode"`
	Ports       []string          `yaml:"ports"`
	Deploy      map[string]interface{} `yaml:"deploy"`
}

// getEnvironment returns the environment variables as a map, handling both
// map and list formats used in Docker Compose files.
func (s composeService) getEnvironment() map[string]string {
	env := make(map[string]string)
	switch v := s.Environment.(type) {
	case map[string]interface{}:
		for key, val := range v {
			env[key] = fmt.Sprintf("%v", val)
		}
	case []interface{}:
		for _, item := range v {
			str := fmt.Sprintf("%v", item)
			if parts := strings.SplitN(str, "=", 2); len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}
	return env
}

// getLabels returns the labels as a map, handling both map and list formats.
func (s composeService) getLabels() map[string]string {
	labels := make(map[string]string)
	switch v := s.Labels.(type) {
	case map[string]interface{}:
		for key, val := range v {
			labels[key] = fmt.Sprintf("%v", val)
		}
	case []interface{}:
		for _, item := range v {
			str := fmt.Sprintf("%v", item)
			if parts := strings.SplitN(str, "=", 2); len(parts) == 2 {
				labels[parts[0]] = parts[1]
			}
		}
	}
	return labels
}

// servicePattern defines a pattern for detecting known services.
type servicePattern struct {
	name       string   // SDBX service name
	exactMatch []string // Exact image name matches (high confidence)
	contains   []string // Substring matches (medium confidence)
	isAddon    bool     // Whether this is an addon (vs core service)
	isSpecial  bool     // Special handling (VPN, traefik, authelia)
}

// detectedService represents a service found in the compose file.
type detectedService struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	ComposeAlias string `json:"compose_alias"`
	Confidence   string `json:"confidence"` // "high" or "medium"
	IsAddon      bool   `json:"is_addon"`
	IsSpecial    bool   `json:"is_special"`
}

// importResult holds the full result of an import analysis.
type importResult struct {
	Detected    []detectedService `json:"detected_services"`
	Unknown     []unknownService  `json:"unknown_services"`
	SuggestedCfg suggestedConfig  `json:"suggested_config"`
}

// unknownService represents a compose service that could not be matched.
type unknownService struct {
	ComposeAlias string `json:"compose_alias"`
	Image        string `json:"image"`
}

// suggestedConfig holds the suggested SDBX configuration values.
type suggestedConfig struct {
	Domain        string   `json:"domain,omitempty"`
	Timezone      string   `json:"timezone,omitempty"`
	MediaPath     string   `json:"media_path,omitempty"`
	DownloadsPath string   `json:"downloads_path,omitempty"`
	ConfigPath    string   `json:"config_path,omitempty"`
	VPNEnabled    bool     `json:"vpn_enabled"`
	Addons        []string `json:"addons"`
}

// knownPatterns defines all the service patterns we can detect.
var knownPatterns = []servicePattern{
	{
		name:       "plex",
		exactMatch: []string{"linuxserver/plex", "plexinc/pms-docker"},
		contains:   []string{"plex"},
	},
	{
		name:       "qbittorrent",
		exactMatch: []string{"linuxserver/qbittorrent"},
		contains:   []string{"qbittorrent"},
	},
	{
		name:       "gluetun",
		exactMatch: []string{"qmcgaw/gluetun"},
		contains:   []string{"gluetun"},
		isSpecial:  true,
	},
	{
		name:       "traefik",
		exactMatch: []string{"traefik"},
		contains:   []string{"traefik"},
		isSpecial:  true,
	},
	{
		name:       "authelia",
		exactMatch: []string{"authelia/authelia"},
		contains:   []string{"authelia"},
		isSpecial:  true,
	},
	{
		name:       "sonarr",
		exactMatch: []string{"linuxserver/sonarr"},
		contains:   []string{"sonarr"},
		isAddon:    true,
	},
	{
		name:       "radarr",
		exactMatch: []string{"linuxserver/radarr"},
		contains:   []string{"radarr"},
		isAddon:    true,
	},
	{
		name:       "prowlarr",
		exactMatch: []string{"linuxserver/prowlarr"},
		contains:   []string{"prowlarr"},
		isAddon:    true,
	},
	{
		name:       "overseerr",
		exactMatch: []string{"linuxserver/overseerr", "sctx/overseerr"},
		contains:   []string{"overseerr"},
		isAddon:    true,
	},
	{
		name:       "lidarr",
		exactMatch: []string{"linuxserver/lidarr"},
		contains:   []string{"lidarr"},
		isAddon:    true,
	},
	{
		name:       "readarr",
		exactMatch: []string{"linuxserver/readarr"},
		contains:   []string{"readarr"},
		isAddon:    true,
	},
	{
		name:       "bazarr",
		exactMatch: []string{"linuxserver/bazarr"},
		contains:   []string{"bazarr"},
		isAddon:    true,
	},
	{
		name:       "tautulli",
		exactMatch: []string{"linuxserver/tautulli"},
		contains:   []string{"tautulli"},
		isAddon:    true,
	},
	{
		name:       "jellyfin",
		exactMatch: []string{"jellyfin/jellyfin"},
		contains:   []string{"jellyfin"},
		isAddon:    true,
	},
}

func runImport(_ *cobra.Command, _ []string) error {
	// Read the compose file
	data, err := os.ReadFile(importFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", importFile, err)
	}

	var compose composeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return fmt.Errorf("failed to parse %s: %w", importFile, err)
	}

	if len(compose.Services) == 0 {
		return fmt.Errorf("no services found in %s", importFile)
	}

	result := analyzeCompose(compose)

	if IsJSONOutput() {
		return OutputJSON(result)
	}

	printImportSummary(result)

	if importDryRun {
		fmt.Println()
		fmt.Printf("  %s\n", tui.MutedStyle.Render("Dry run - no files written. Remove --dry-run to generate .sdbx.yaml"))
		return nil
	}

	// Check if .sdbx.yaml already exists
	if _, err := os.Stat(".sdbx.yaml"); err == nil {
		fmt.Println()
		fmt.Printf("  %s .sdbx.yaml already exists. Remove it first or use a different directory.\n",
			tui.WarningStyle.Render(tui.IconWarning))
		return fmt.Errorf(".sdbx.yaml already exists")
	}

	// Generate and save config
	cfg := buildConfig(result.SuggestedCfg)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	cfgPath := filepath.Join(cwd, ".sdbx.yaml")
	if err := cfg.Save(cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	nextSteps := fmt.Sprintf(
		"1. Review and edit %s\n"+
			"2. Run %s to complete setup (admin credentials, VPN, etc.)\n"+
			"3. Run %s to start services",
		tui.CommandStyle.Render(".sdbx.yaml"),
		tui.CommandStyle.Render("sdbx init"),
		tui.CommandStyle.Render("sdbx up"),
	)
	fmt.Print(tui.RenderSuccessBox("Configuration imported!", nextSteps))
	fmt.Println()

	return nil
}

// analyzeCompose inspects all services in a compose file and returns the analysis result.
func analyzeCompose(compose composeFile) importResult {
	var detected []detectedService
	var unknown []unknownService
	seen := make(map[string]bool)

	var timezone string
	var mediaPath, downloadsPath, configPath string
	var domain string
	vpnEnabled := false
	var addons []string

	for alias, svc := range compose.Services {
		image := normalizeImage(svc.Image)
		matched := false

		for _, pattern := range knownPatterns {
			confidence := matchService(image, pattern)
			if confidence == "" {
				continue
			}

			// Avoid duplicate detections (e.g., two sonarr-like services)
			if seen[pattern.name] {
				continue
			}
			seen[pattern.name] = true
			matched = true

			detected = append(detected, detectedService{
				Name:         pattern.name,
				Image:        svc.Image,
				ComposeAlias: alias,
				Confidence:   confidence,
				IsAddon:      pattern.isAddon,
				IsSpecial:    pattern.isSpecial,
			})

			if pattern.isAddon {
				addons = append(addons, pattern.name)
			}
			if pattern.name == "gluetun" {
				vpnEnabled = true
			}
			break
		}

		if !matched && svc.Image != "" {
			unknown = append(unknown, unknownService{
				ComposeAlias: alias,
				Image:        svc.Image,
			})
		}

		// Extract common config from environment variables
		env := svc.getEnvironment()
		if tz, ok := env["TZ"]; ok && tz != "" && timezone == "" {
			timezone = tz
		}

		// Extract domain from Traefik labels
		labels := svc.getLabels()
		for _, v := range labels {
			if d := extractDomainFromLabel(v); d != "" && domain == "" {
				domain = d
			}
		}

		// Extract paths from volumes
		for _, vol := range svc.Volumes {
			mp, dp, cp := extractPaths(vol)
			if mp != "" && mediaPath == "" {
				mediaPath = mp
			}
			if dp != "" && downloadsPath == "" {
				downloadsPath = dp
			}
			if cp != "" && configPath == "" {
				configPath = cp
			}
		}
	}

	suggested := suggestedConfig{
		Domain:        domain,
		Timezone:      timezone,
		MediaPath:     mediaPath,
		DownloadsPath: downloadsPath,
		ConfigPath:    configPath,
		VPNEnabled:    vpnEnabled,
		Addons:        addons,
	}

	return importResult{
		Detected:    detected,
		Unknown:     unknown,
		SuggestedCfg: suggested,
	}
}

// normalizeImage strips the tag and registry prefix to get a comparable image name.
// e.g. "ghcr.io/linuxserver/sonarr:latest" -> "linuxserver/sonarr"
func normalizeImage(image string) string {
	// Strip tag
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		// Make sure we're not stripping a port from a registry URL
		afterColon := image[idx+1:]
		if !strings.Contains(afterColon, "/") {
			image = image[:idx]
		}
	}
	// Strip known registry prefixes
	registries := []string{"ghcr.io/", "docker.io/", "lscr.io/", "registry.hub.docker.com/"}
	for _, reg := range registries {
		image = strings.TrimPrefix(image, reg)
	}
	return strings.ToLower(image)
}

// matchService checks if an image matches a service pattern and returns the confidence level.
func matchService(normalizedImage string, pattern servicePattern) string {
	for _, exact := range pattern.exactMatch {
		if normalizedImage == exact {
			return "high"
		}
	}
	for _, substr := range pattern.contains {
		if strings.Contains(normalizedImage, substr) {
			return "medium"
		}
	}
	return ""
}

// extractDomainFromLabel tries to extract a domain from a Traefik Host rule label.
// e.g. "Host(`sonarr.example.com`)" -> "example.com"
func extractDomainFromLabel(value string) string {
	// Look for Host(`...`) patterns in Traefik rules
	hostPrefix := "Host(`"
	idx := strings.Index(value, hostPrefix)
	if idx == -1 {
		return ""
	}
	rest := value[idx+len(hostPrefix):]
	endIdx := strings.Index(rest, "`)")
	if endIdx == -1 {
		return ""
	}
	host := rest[:endIdx]

	// Extract the base domain (strip first subdomain)
	parts := strings.SplitN(host, ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// extractPaths inspects a volume mount string and categorizes it as media, downloads, or config.
func extractPaths(volume string) (media, downloads, configDir string) {
	parts := strings.SplitN(volume, ":", 2)
	if len(parts) < 2 {
		return
	}
	hostPath := parts[0]
	containerPath := strings.ToLower(parts[1])
	// Strip :ro, :rw suffixes from container path
	if idx := strings.LastIndex(containerPath, ":"); idx != -1 {
		containerPath = containerPath[:idx]
	}

	switch {
	case strings.Contains(containerPath, "/media") || strings.Contains(containerPath, "/movies") || strings.Contains(containerPath, "/tv"):
		media = hostPath
	case strings.Contains(containerPath, "/downloads") || strings.Contains(containerPath, "/torrents"):
		downloads = hostPath
	case strings.Contains(containerPath, "/config"):
		configDir = hostPath
	}
	return
}

// printImportSummary prints a human-readable summary of the import analysis.
func printImportSummary(result importResult) {
	fmt.Println()
	fmt.Println(tui.RenderHeader("Import Analysis", fmt.Sprintf("Source: %s", importFile)))
	fmt.Println()

	// Detected services
	if len(result.Detected) > 0 {
		fmt.Println(tui.RenderSection("  Detected Services"))
		fmt.Println()
		for _, svc := range result.Detected {
			var icon string
			var style = tui.SuccessStyle
			if svc.Confidence == "high" {
				icon = tui.IconSuccess
			} else {
				icon = tui.IconInfo
				style = tui.InfoStyle
			}

			label := svc.Name
			if svc.IsAddon {
				label += " (addon)"
			} else if svc.IsSpecial {
				label += " (infrastructure)"
			} else {
				label += " (core)"
			}

			fmt.Printf("    %s %-28s %s  %s\n",
				style.Render(icon),
				label,
				tui.MutedStyle.Render(svc.Image),
				tui.MutedStyle.Render("["+svc.Confidence+"]"),
			)
		}
	}

	// Unknown services
	if len(result.Unknown) > 0 {
		fmt.Println()
		fmt.Println(tui.RenderSection("  Unrecognized Services"))
		fmt.Println()
		for _, svc := range result.Unknown {
			fmt.Printf("    %s %-28s %s\n",
				tui.MutedStyle.Render(tui.IconDot),
				svc.ComposeAlias,
				tui.MutedStyle.Render(svc.Image),
			)
		}
		fmt.Printf("\n    %s\n", tui.MutedStyle.Render("These services are not managed by SDBX and will need manual setup."))
	}

	// Suggested configuration
	sc := result.SuggestedCfg
	hasValues := sc.Domain != "" || sc.Timezone != "" || sc.MediaPath != "" ||
		sc.DownloadsPath != "" || sc.ConfigPath != "" || sc.VPNEnabled

	if hasValues || len(sc.Addons) > 0 {
		fmt.Println()
		fmt.Println(tui.RenderSection("  Suggested Configuration"))
		fmt.Println()

		if sc.Domain != "" {
			fmt.Printf("    %s\n", tui.RenderKeyValue("Domain", sc.Domain))
		}
		if sc.Timezone != "" {
			fmt.Printf("    %s\n", tui.RenderKeyValue("Timezone", sc.Timezone))
		}
		if sc.MediaPath != "" {
			fmt.Printf("    %s\n", tui.RenderKeyValue("Media Path", sc.MediaPath))
		}
		if sc.DownloadsPath != "" {
			fmt.Printf("    %s\n", tui.RenderKeyValue("Downloads", sc.DownloadsPath))
		}
		if sc.ConfigPath != "" {
			fmt.Printf("    %s\n", tui.RenderKeyValue("Config Path", sc.ConfigPath))
		}
		if sc.VPNEnabled {
			fmt.Printf("    %s\n", tui.RenderKeyValue("VPN", tui.SuccessStyle.Render("detected (gluetun)")))
		}
		if len(sc.Addons) > 0 {
			fmt.Printf("    %s\n", tui.RenderKeyValue("Addons", strings.Join(sc.Addons, ", ")))
		}
	}
}

// buildConfig creates an SDBX Config from the suggested configuration values.
func buildConfig(sc suggestedConfig) *config.Config {
	cfg := config.DefaultConfig()

	if sc.Domain != "" {
		cfg.Domain = sc.Domain
	}
	if sc.Timezone != "" {
		cfg.Timezone = sc.Timezone
	}
	if sc.MediaPath != "" {
		cfg.MediaPath = sc.MediaPath
	}
	if sc.DownloadsPath != "" {
		cfg.DownloadsPath = sc.DownloadsPath
	}
	if sc.ConfigPath != "" {
		cfg.ConfigPath = sc.ConfigPath
	}
	cfg.VPNEnabled = sc.VPNEnabled
	if len(sc.Addons) > 0 {
		cfg.Addons = sc.Addons
	}

	return cfg
}
