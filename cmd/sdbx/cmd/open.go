package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
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

// serviceURLInfo holds URL information for a service
type serviceURLInfo struct {
	Name     string
	URL      string
	Category string
}

func runOpen(_ *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	domain := viper.GetString("domain")
	if domain == "" && cfg != nil {
		domain = cfg.Domain
	}
	if domain == "" {
		return fmt.Errorf("no domain configured. Run 'sdbx init' first")
	}

	ctx := context.Background()

	// Get enabled services from registry
	services, err := getEnabledServicesWithRouting(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}

	// Build URL map for lookup and display
	urlMap := make(map[string]serviceURLInfo)
	for _, svc := range services {
		url := cfg.GetServiceURL(svc.Name)
		if url != "" {
			urlMap[svc.Name] = serviceURLInfo{
				Name:     svc.Name,
				URL:      url,
				Category: string(svc.Category),
			}
			// Add common aliases
			switch svc.Name {
			case "qbittorrent":
				urlMap["qbt"] = urlMap[svc.Name]
			case "authelia":
				urlMap["auth"] = urlMap[svc.Name]
			case "homepage":
				urlMap["home"] = urlMap[svc.Name]
			}
		}
	}

	// No args - list all URLs
	if len(args) == 0 {
		fmt.Println(tui.TitleStyle.Render("SDBX Service URLs"))
		fmt.Println()

		// Group services by category
		categories := map[string][]serviceURLInfo{
			"auth":       {},
			"networking": {},
			"media":      {},
			"downloads":  {},
			"management": {},
			"utility":    {},
		}

		for _, svc := range services {
			if info, ok := urlMap[svc.Name]; ok {
				cat := strings.ToLower(info.Category)
				if _, exists := categories[cat]; exists {
					categories[cat] = append(categories[cat], info)
				} else {
					categories["utility"] = append(categories["utility"], info)
				}
			}
		}

		// Display in order
		categoryOrder := []string{"auth", "networking", "media", "downloads", "management", "utility"}
		categoryNames := map[string]string{
			"auth":       "Authentication",
			"networking": "Networking",
			"media":      "Media",
			"downloads":  "Downloads",
			"management": "Management",
			"utility":    "Utility",
		}

		for _, cat := range categoryOrder {
			svcs := categories[cat]
			if len(svcs) == 0 {
				continue
			}

			// Sort services in category by name
			sort.Slice(svcs, func(i, j int) bool {
				return svcs[i].Name < svcs[j].Name
			})

			fmt.Println(tui.InfoStyle.Render(categoryNames[cat] + ":"))
			for _, svc := range svcs {
				fmt.Printf("  %-14s %s\n", svc.Name, svc.URL)
			}
			fmt.Println()
		}

		return nil
	}

	// Open specific service
	service := strings.ToLower(args[0])
	info, ok := urlMap[service]
	if !ok {
		return fmt.Errorf("unknown or not enabled service: %s\nRun 'sdbx open' to see available services", service)
	}

	fmt.Printf("Opening %s...\n", info.URL)
	return openBrowser(info.URL)
}

// getEnabledServicesWithRouting returns enabled services that have routing enabled
func getEnabledServicesWithRouting(ctx context.Context, cfg *config.Config) ([]registry.ServiceInfo, error) {
	reg, err := registry.NewWithDefaults()
	if err != nil {
		return nil, err
	}

	graph, err := reg.Resolve(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var result []registry.ServiceInfo
	for _, name := range graph.Order {
		svc := graph.Services[name]
		if svc != nil && svc.Definition.Routing.Enabled {
			result = append(result, registry.ServiceInfo{
				Name:        svc.Definition.Metadata.Name,
				Description: svc.Definition.Metadata.Description,
				Category:    svc.Definition.Metadata.Category,
				IsAddon:     svc.Definition.Conditions.RequireAddon,
			})
		}
	}

	return result, nil
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
