package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/maiko/sdbx/internal/registry"
	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
)

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage service definition sources",
	Long: `Manage Git and local sources for service definitions.

SDBX loads service definitions from multiple sources, similar to Homebrew taps.
Sources are checked in priority order (highest first).

Examples:
  sdbx source list                           # List all configured sources
  sdbx source add community https://github.com/sdbx-community/services.git
  sdbx source update                         # Update all sources
  sdbx source remove community               # Remove a source`,
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured sources",
	RunE:  runSourceList,
}

var sourceAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add a new Git source",
	Long: `Add a new Git repository as a service definition source.

Examples:
  sdbx source add community https://github.com/sdbx-community/services.git
  sdbx source add mycompany git@github.com:mycompany/sdbx-services.git --priority 50
  sdbx source add internal https://internal.example.com/services.git --branch develop`,
	Args: cobra.ExactArgs(2),
	RunE: runSourceAdd,
}

var sourceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a source",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceRemove,
}

var sourceUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update sources from remote",
	Long: `Update Git sources by pulling latest changes.

Examples:
  sdbx source update          # Update all sources
  sdbx source update official # Update specific source`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSourceUpdate,
}

var sourceInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed source information",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceInfo,
}

// Flags
var (
	sourcePriority int
	sourceBranch   string
	sourceSSHKey   string
)

func init() {
	rootCmd.AddCommand(sourceCmd)
	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(sourceAddCmd)
	sourceCmd.AddCommand(sourceRemoveCmd)
	sourceCmd.AddCommand(sourceUpdateCmd)
	sourceCmd.AddCommand(sourceInfoCmd)

	// Add flags
	sourceAddCmd.Flags().IntVarP(&sourcePriority, "priority", "p", 10, "Source priority (higher = checked first)")
	sourceAddCmd.Flags().StringVarP(&sourceBranch, "branch", "b", "main", "Git branch to use")
	sourceAddCmd.Flags().StringVar(&sourceSSHKey, "ssh-key", "", "Path to SSH key for private repos")
}

func runSourceList(_ *cobra.Command, _ []string) error {
	cfg, err := loadSourceConfig()
	if err != nil {
		return err
	}

	// JSON output
	if IsJSONOutput() {
		return OutputJSON(cfg.Sources)
	}

	fmt.Println(tui.TitleStyle.Render("Service Sources"))
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tPRIORITY\tURL\tSTATUS")

	for _, src := range cfg.Sources {
		status := tui.SuccessStyle.Render("active")
		if !src.Enabled {
			status = tui.MutedStyle.Render("disabled")
		}

		url := src.URL
		if src.Type == "local" {
			url = src.Path
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			src.Name,
			src.Type,
			src.Priority,
			truncate(url, 50),
			status,
		)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("Use '%s' to add a new source\n", tui.CommandStyle.Render("sdbx source add <name> <url>"))

	return nil
}

func runSourceAdd(_ *cobra.Command, args []string) error {
	name := args[0]
	url := args[1]

	cfg, err := loadSourceConfig()
	if err != nil {
		cfg = registry.DefaultSourceConfig()
	}

	// Check for duplicate
	for _, src := range cfg.Sources {
		if src.Name == name {
			return fmt.Errorf("source %s already exists", name)
		}
	}

	// Add new source
	newSource := registry.Source{
		Name:     name,
		Type:     "git",
		URL:      url,
		Branch:   sourceBranch,
		SSHKey:   sourceSSHKey,
		Priority: sourcePriority,
		Enabled:  true,
	}

	cfg.Sources = append(cfg.Sources, newSource)

	// Save config
	if err := saveSourceConfig(cfg); err != nil {
		return err
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Added source: %s", name)))
	fmt.Println()
	fmt.Printf("Run '%s' to fetch service definitions\n", tui.CommandStyle.Render("sdbx source update "+name))

	return nil
}

func runSourceRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	// Prevent removing official or embedded sources
	if name == "official" || name == "embedded" {
		return fmt.Errorf("cannot remove built-in source: %s", name)
	}

	cfg, err := loadSourceConfig()
	if err != nil {
		return err
	}

	// Find and remove
	found := false
	newSources := make([]registry.Source, 0, len(cfg.Sources))
	for _, src := range cfg.Sources {
		if src.Name == name {
			found = true
			continue
		}
		newSources = append(newSources, src)
	}

	if !found {
		return fmt.Errorf("source %s not found", name)
	}

	cfg.Sources = newSources

	// Save config
	if err := saveSourceConfig(cfg); err != nil {
		return err
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Removed source: %s", name)))

	return nil
}

func runSourceUpdate(_ *cobra.Command, args []string) error {
	reg, err := registry.NewWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	ctx := context.Background()

	if len(args) == 1 {
		// Update specific source
		name := args[0]
		src, err := reg.GetSource(name)
		if err != nil {
			return err
		}

		fmt.Printf("Updating source %s...\n", name)
		if err := src.Update(ctx); err != nil {
			return fmt.Errorf("failed to update %s: %w", name, err)
		}

		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Updated source: %s", name)))
	} else {
		// Update all sources
		fmt.Println("Updating all sources...")

		for _, src := range reg.Sources() {
			if src.Type() == "local" || src.Type() == "embedded" {
				continue
			}

			fmt.Printf("  Updating %s...", src.Name())
			if err := src.Update(ctx); err != nil {
				fmt.Printf(" %s\n", tui.ErrorStyle.Render("failed"))
				continue
			}
			fmt.Printf(" %s\n", tui.SuccessStyle.Render("done"))
		}

		fmt.Println()
		fmt.Println(tui.SuccessStyle.Render("✓ All sources updated"))
	}

	return nil
}

func runSourceInfo(_ *cobra.Command, args []string) error {
	name := args[0]

	reg, err := registry.NewWithDefaults()
	if err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	src, err := reg.GetSource(name)
	if err != nil {
		return err
	}

	ctx := context.Background()

	fmt.Println(tui.TitleStyle.Render("Source: " + name))
	fmt.Println()

	fmt.Printf("Type:     %s\n", src.Type())
	fmt.Printf("Priority: %d\n", src.Priority())
	fmt.Printf("Enabled:  %t\n", src.IsEnabled())

	if gitSrc, ok := src.(*registry.GitSource); ok {
		fmt.Printf("URL:      %s\n", gitSrc.GetURL())
		fmt.Printf("Branch:   %s\n", gitSrc.GetBranch())
		fmt.Printf("Commit:   %s\n", truncate(gitSrc.GetCommit(), 12))
		fmt.Printf("Updated:  %s\n", gitSrc.GetLastUpdated().Format("2006-01-02 15:04:05"))
	}

	fmt.Println()

	// List services from this source
	services, err := src.ListServices(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Services: %d\n", len(services))
	if len(services) > 0 && len(services) <= 20 {
		for _, svcName := range services {
			def, _ := src.LoadService(ctx, svcName)
			if def != nil {
				addonTag := ""
				if def.Conditions.RequireAddon {
					addonTag = tui.MutedStyle.Render(" (addon)")
				}
				fmt.Printf("  - %s%s\n", svcName, addonTag)
			}
		}
	}

	return nil
}

// loadSourceConfig loads the source configuration
func loadSourceConfig() (*registry.SourceConfig, error) {
	configPath := getSourceConfigPath()

	loader := registry.NewLoader()
	cfg, err := loader.LoadSourceConfig(configPath)
	if err != nil {
		return registry.DefaultSourceConfig(), nil
	}

	return cfg, nil
}

// saveSourceConfig saves the source configuration
func saveSourceConfig(cfg *registry.SourceConfig) error {
	configPath := getSourceConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	loader := registry.NewLoader()
	return loader.SaveSourceConfig(configPath, cfg)
}

// getSourceConfigPath returns the path to sources.yaml
func getSourceConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "sdbx", "sources.yaml")
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
