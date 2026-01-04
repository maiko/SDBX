package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
	"github.com/maiko/sdbx/internal/tui"
)

var addonCmd = &cobra.Command{
	Use:   "addon",
	Short: "Manage SDBX addons",
	Long: `Manage optional SDBX services (addons).

Addons are optional services that extend SDBX functionality.
Use 'sdbx addon search' to find available addons from all sources.

Examples:
  sdbx addon list                  # List enabled addons
  sdbx addon list --all            # List all available addons
  sdbx addon search media          # Search for media-related addons
  sdbx addon info overseerr        # Show addon details
  sdbx addon enable overseerr      # Enable an addon
  sdbx addon disable overseerr     # Disable an addon`,
}

var addonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and enabled addons",
	RunE:  runAddonList,
}

var addonSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for addons across all sources",
	Long: `Search for addons by name, description, or category.

Examples:
  sdbx addon search               # List all addons
  sdbx addon search media         # Search for media-related addons
  sdbx addon search --category media`,
	RunE: runAddonSearch,
}

var addonInfoCmd = &cobra.Command{
	Use:   "info <addon>",
	Short: "Show detailed addon information",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddonInfo,
}

var addonEnableCmd = &cobra.Command{
	Use:   "enable <addon>",
	Short: "Enable an addon",
	Long: `Enable an optional addon service.

After enabling, run 'sdbx up' to start the addon.`,
	Args: cobra.ExactArgs(1),
	RunE: runAddonEnable,
}

var addonDisableCmd = &cobra.Command{
	Use:   "disable <addon>",
	Short: "Disable an addon",
	Long: `Disable an optional addon service.

After disabling, run 'sdbx down && sdbx up' to apply changes.`,
	Args: cobra.ExactArgs(1),
	RunE: runAddonDisable,
}

// Flags
var (
	addonListAll  bool
	addonCategory string
)

func init() {
	rootCmd.AddCommand(addonCmd)
	addonCmd.AddCommand(addonListCmd)
	addonCmd.AddCommand(addonSearchCmd)
	addonCmd.AddCommand(addonInfoCmd)
	addonCmd.AddCommand(addonEnableCmd)
	addonCmd.AddCommand(addonDisableCmd)

	// Flags
	addonListCmd.Flags().BoolVarP(&addonListAll, "all", "a", false, "Show all available addons")
	addonSearchCmd.Flags().StringVarP(&addonCategory, "category", "c", "", "Filter by category")
}

func runAddonList(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	ctx := context.Background()

	// Get addons from registry
	reg, err := getRegistry()
	if err != nil {
		return err
	}

	services, err := reg.ListServices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	// Filter to addons only
	var addons []registry.ServiceInfo
	for _, svc := range services {
		if svc.IsAddon {
			addons = append(addons, svc)
		}
	}

	// JSON output
	if IsJSONOutput() {
		result := make([]map[string]interface{}, 0, len(addons))
		for _, addon := range addons {
			if !addonListAll && !cfg.IsAddonEnabled(addon.Name) {
				continue
			}
			result = append(result, map[string]interface{}{
				"name":        addon.Name,
				"description": addon.Description,
				"category":    addon.Category,
				"source":      addon.Source,
				"enabled":     cfg.IsAddonEnabled(addon.Name),
			})
		}
		return OutputJSON(result)
	}

	fmt.Println()
	if addonListAll {
		fmt.Println(tui.TitleStyle.Render("Available Addons"))
	} else {
		fmt.Println(tui.TitleStyle.Render("Enabled Addons"))
	}
	fmt.Println()

	// Create table
	table := tui.AddonTable()
	enabled := 0
	displayed := 0

	for _, addon := range addons {
		isEnabled := cfg.IsAddonEnabled(addon.Name)

		if !addonListAll && !isEnabled {
			continue
		}

		if isEnabled {
			enabled++
		}

		table.AddRow(
			addon.Name,
			tui.RenderCategory(string(addon.Category)),
			addon.Source,
			tui.EnabledBadge(isEnabled),
		)
		displayed++
	}

	if displayed == 0 {
		fmt.Println(tui.MutedStyle.Render("  No addons enabled"))
		fmt.Println()
		fmt.Printf("Use '%s' to see available addons\n", tui.CommandStyle.Render("sdbx addon list --all"))
	} else {
		fmt.Println(table.Render())
		fmt.Printf("%s %d enabled, %d available\n",
			tui.IconPackage,
			enabled,
			len(addons)-enabled,
		)
	}
	fmt.Println()

	return nil
}

func runAddonSearch(_ *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	ctx := context.Background()

	reg, err := getRegistry()
	if err != nil {
		return err
	}

	var category registry.ServiceCategory
	if addonCategory != "" {
		category = registry.ServiceCategory(addonCategory)
	}

	results, err := reg.SearchServices(ctx, query, category)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Filter to addons only
	var addons []registry.ServiceInfo
	for _, svc := range results {
		if svc.IsAddon {
			addons = append(addons, svc)
		}
	}

	// JSON output
	if IsJSONOutput() {
		return OutputJSON(addons)
	}

	if len(addons) == 0 {
		fmt.Println(tui.MutedStyle.Render("No addons found matching your query"))
		return nil
	}

	fmt.Println()
	if query != "" {
		fmt.Println(tui.TitleStyle.Render(fmt.Sprintf("Search Results for '%s'", query)))
	} else {
		fmt.Println(tui.TitleStyle.Render("Available Addons"))
	}
	fmt.Println()

	// Create table with description column
	table := tui.NewTable("Name", "Category", "Source", "Description")

	for _, addon := range addons {
		table.AddRow(
			addon.Name,
			tui.RenderCategory(string(addon.Category)),
			addon.Source,
			truncateDesc(addon.Description, 40),
		)
	}

	fmt.Println(table.Render())
	fmt.Printf("%s %d addons found. Use '%s' for details.\n",
		tui.IconPackage,
		len(addons),
		tui.CommandStyle.Render("sdbx addon info <name>"),
	)
	fmt.Println()

	return nil
}

func runAddonInfo(_ *cobra.Command, args []string) error {
	addonName := args[0]

	ctx := context.Background()

	reg, err := getRegistry()
	if err != nil {
		return err
	}

	def, source, err := reg.GetService(ctx, addonName)
	if err != nil {
		return fmt.Errorf("addon not found: %s", addonName)
	}

	if !def.Conditions.RequireAddon {
		return fmt.Errorf("%s is a core service, not an addon", addonName)
	}

	cfg, _ := config.Load()
	isEnabled := cfg != nil && cfg.IsAddonEnabled(addonName)

	// JSON output
	if IsJSONOutput() {
		return OutputJSON(map[string]interface{}{
			"name":        def.Metadata.Name,
			"version":     def.Metadata.Version,
			"description": def.Metadata.Description,
			"category":    def.Metadata.Category,
			"source":      source,
			"homepage":    def.Metadata.Homepage,
			"image":       def.Spec.Image.Repository + ":" + def.Spec.Image.Tag,
			"port":        def.Routing.Port,
			"enabled":     isEnabled,
		})
	}

	fmt.Println(tui.TitleStyle.Render(tui.IconPackage + " " + def.Metadata.Name))
	fmt.Println()

	// Status badge
	if isEnabled {
		fmt.Printf("  %s  %s  %s\n",
			tui.SuccessStyle.Render(tui.IconRunning+" enabled"),
			tui.MutedStyle.Render("|"),
			tui.RenderCategory(string(def.Metadata.Category)),
		)
	} else {
		fmt.Printf("  %s  %s  %s\n",
			tui.MutedStyle.Render(tui.IconStopped+" not enabled"),
			tui.MutedStyle.Render("|"),
			tui.RenderCategory(string(def.Metadata.Category)),
		)
	}
	fmt.Println()

	// Description
	fmt.Println(tui.MutedStyle.Render("  " + def.Metadata.Description))
	fmt.Println()

	// Details section
	fmt.Println(tui.RenderSection("  Details"))
	fmt.Printf("  %s\n", tui.RenderKeyValue("Version", def.Metadata.Version))
	fmt.Printf("  %s\n", tui.RenderKeyValue("Source", source))
	fmt.Printf("  %s\n", tui.RenderKeyValue("Image", def.Spec.Image.Repository+":"+def.Spec.Image.Tag))
	if def.Routing.Enabled {
		fmt.Printf("  %s\n", tui.RenderKeyValue("Port", fmt.Sprintf("%d", def.Routing.Port)))
	}
	fmt.Println()

	if def.Routing.Enabled {
		fmt.Println(tui.RenderSection("  " + tui.IconNetwork + " Routing"))
		fmt.Printf("  %s\n", tui.RenderKeyValue("Subdomain", def.Routing.Subdomain))
		fmt.Printf("  %s\n", tui.RenderKeyValue("Path", def.Routing.Path))
		if def.Routing.Auth.Required {
			fmt.Printf("  %s\n", tui.RenderKeyValue("Auth", tui.IconLock+" required"))
		} else {
			fmt.Printf("  %s\n", tui.RenderKeyValue("Auth", "not required"))
		}
		fmt.Println()
	}

	if def.Metadata.Homepage != "" {
		fmt.Println(tui.RenderSection("  Links"))
		fmt.Printf("  %s\n", tui.RenderKeyValue("Homepage", def.Metadata.Homepage))
		if def.Metadata.Documentation != "" {
			fmt.Printf("  %s\n", tui.RenderKeyValue("Docs", def.Metadata.Documentation))
		}
		fmt.Println()
	}

	fmt.Println(tui.RenderDivider(50))
	if !isEnabled {
		fmt.Printf("  %s Enable with: %s\n", tui.IconArrow, tui.CommandStyle.Render("sdbx addon enable "+addonName))
	} else {
		fmt.Printf("  %s Disable with: %s\n", tui.IconArrow, tui.CommandStyle.Render("sdbx addon disable "+addonName))
	}

	return nil
}

func runAddonEnable(_ *cobra.Command, args []string) error {
	addonName := args[0]

	ctx := context.Background()

	// Validate addon exists in registry
	reg, err := getRegistry()
	if err != nil {
		return err
	}

	def, _, err := reg.GetService(ctx, addonName)
	if err != nil {
		return fmt.Errorf("addon not found: %s\nRun 'sdbx addon search' to see available addons", addonName)
	}

	if !def.Conditions.RequireAddon {
		return fmt.Errorf("%s is a core service, not an addon", addonName)
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if cfg.IsAddonEnabled(addonName) {
		fmt.Printf("%s Addon '%s' is already enabled\n", tui.IconInfo, addonName)
		return nil
	}

	cfg.EnableAddon(addonName)

	// Save config
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("%s Enabled: %s", tui.IconSuccess, addonName)))
	fmt.Println()
	fmt.Printf("  %s Run %s to start the service\n",
		tui.IconArrow,
		tui.CommandStyle.Render("sdbx up"))

	return nil
}

func runAddonDisable(_ *cobra.Command, args []string) error {
	addonName := args[0]

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	if !cfg.IsAddonEnabled(addonName) {
		fmt.Printf("%s Addon '%s' is not enabled\n", tui.IconInfo, addonName)
		return nil
	}

	cfg.DisableAddon(addonName)

	// Save config
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("%s Disabled: %s", tui.IconSuccess, addonName)))
	fmt.Println()
	fmt.Printf("  %s Run %s to apply changes\n",
		tui.IconArrow,
		tui.CommandStyle.Render("sdbx down && sdbx up"))

	return nil
}

// getRegistry returns a registry instance
func getRegistry() (*registry.Registry, error) {
	// Use registry with default sources (embedded + configured sources)
	return registry.NewWithDefaults()
}

// truncateDesc truncates a description string
func truncateDesc(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
