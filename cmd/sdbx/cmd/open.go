package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var openCmd = &cobra.Command{
	Use:   "open [service]",
	Short: "Open SDBX service URLs in browser",
	Long: `Open one or all SDBX service URLs in your default browser.

Examples:
  sdbx open          # List all URLs
  sdbx open plex     # Open Plex in browser
  sdbx open radarr   # Open Radarr in browser`,
	RunE: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

// Service URL mappings
var serviceURLs = map[string]string{
	"plex":        "https://plex.%s",
	"radarr":      "https://radarr.%s",
	"sonarr":      "https://sonarr.%s",
	"prowlarr":    "https://prowlarr.%s",
	"lidarr":      "https://lidarr.%s",
	"readarr":     "https://readarr.%s",
	"bazarr":      "https://bazarr.%s",
	"qbittorrent": "https://qbt.%s",
	"qbt":         "https://qbt.%s",
	"overseerr":   "https://overseerr.%s",
	"wizarr":      "https://wizarr.%s",
	"tautulli":    "https://tautulli.%s",
	"authelia":    "https://auth.%s",
	"auth":        "https://auth.%s",
	"homepage":    "https://home.%s",
	"home":        "https://home.%s",
}

func runOpen(_ *cobra.Command, args []string) error {
	domain := viper.GetString("domain")
	if domain == "" {
		cfg, err := config.Load()
		if err == nil {
			domain = cfg.Domain
		}
	}
	if domain == "" {
		domain = "sdbx.example.com"
	}

	// No args - list all URLs
	if len(args) == 0 {
		fmt.Println(tui.TitleStyle.Render("SDBX Service URLs"))
		fmt.Println()

		// Core services
		fmt.Println(tui.InfoStyle.Render("Core:"))
		fmt.Printf("  %-14s %s\n", "Homepage", fmt.Sprintf("https://home.%s", domain))
		fmt.Printf("  %-14s %s\n", "Authelia", fmt.Sprintf("https://auth.%s", domain))
		fmt.Println()

		// Media
		fmt.Println(tui.InfoStyle.Render("Media:"))
		fmt.Printf("  %-14s %s\n", "Plex", fmt.Sprintf("https://plex.%s", domain))
		fmt.Printf("  %-14s %s\n", "Radarr", fmt.Sprintf("https://radarr.%s", domain))
		fmt.Printf("  %-14s %s\n", "Sonarr", fmt.Sprintf("https://sonarr.%s", domain))
		fmt.Printf("  %-14s %s\n", "Prowlarr", fmt.Sprintf("https://prowlarr.%s", domain))
		fmt.Println()

		// Downloads
		fmt.Println(tui.InfoStyle.Render("Downloads:"))
		fmt.Printf("  %-14s %s\n", "qBittorrent", fmt.Sprintf("https://qbt.%s", domain))
		fmt.Println()

		// Optional - check config for enabled addons
		cfg, _ := config.Load()
		if cfg != nil {
			enabledAddons := false
			for _, addon := range []string{"overseerr", "wizarr", "tautulli", "lidarr", "readarr", "bazarr"} {
				if cfg.IsAddonEnabled(addon) {
					if !enabledAddons {
						fmt.Println(tui.InfoStyle.Render("Addons:"))
						enabledAddons = true
					}
					if url, ok := serviceURLs[addon]; ok {
						fmt.Printf("  %-14s %s\n", addon, fmt.Sprintf(url, domain))
					}
				}
			}
			if enabledAddons {
				fmt.Println()
			}
		}

		return nil
	}

	// Open specific service
	service := args[0]
	urlPattern, ok := serviceURLs[service]
	if !ok {
		return fmt.Errorf("unknown service: %s\nRun 'sdbx open' to see available services", service)
	}

	url := fmt.Sprintf(urlPattern, domain)
	fmt.Printf("Opening %s...\n", url)

	return openBrowser(url)
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
