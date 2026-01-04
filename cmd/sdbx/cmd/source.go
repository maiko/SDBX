package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	cfg := loadSourceConfig()

	// JSON output
	if IsJSONOutput() {
		return OutputJSON(cfg.Sources)
	}

	fmt.Println()
	fmt.Println(tui.TitleStyle.Render("Service Sources"))
	fmt.Println()

	// Create table
	table := tui.SourceTable()

	for _, src := range cfg.Sources {
		url := src.URL
		if src.Type == "local" {
			url = src.Path
		}

		table.AddRow(
			src.Name,
			src.Type,
			truncate(url, 50),
			tui.EnabledBadge(src.Enabled),
		)
	}

	fmt.Println(table.Render())
	fmt.Printf("%s %d sources configured. Use '%s' to add a new source.\n",
		tui.IconNetwork,
		len(cfg.Sources),
		tui.CommandStyle.Render("sdbx source add <name> <url>"),
	)
	fmt.Println()

	return nil
}

func runSourceAdd(_ *cobra.Command, args []string) error {
	name := args[0]
	url := args[1]

	cfg := loadSourceConfig()

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

	cfg := loadSourceConfig()

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

		fmt.Printf("%s Updating source %s...\n", tui.IconRefresh, name)
		if err := src.Update(ctx); err != nil {
			return fmt.Errorf("failed to update %s: %w", name, err)
		}

		fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("%s Updated: %s", tui.IconSuccess, name)))
	} else {
		// Update all sources using checklist
		fmt.Println()
		fmt.Println(tui.TitleStyle.Render("Updating Sources"))
		fmt.Println()

		checklist := tui.NewCheckList()
		sources := reg.Sources()

		// Add updatable sources to checklist
		for _, src := range sources {
			if src.Type() == "local" || src.Type() == "embedded" {
				continue
			}
			checklist.Add(src.Name())
		}

		updated := 0
		failed := 0
		idx := 0
		for _, src := range sources {
			if src.Type() == "local" || src.Type() == "embedded" {
				continue
			}

			if err := src.Update(ctx); err != nil {
				checklist.SetStatus(idx, "error", err.Error())
				failed++
			} else {
				checklist.SetStatus(idx, "success", "updated")
				updated++
			}
			idx++
		}

		fmt.Println(checklist.Render())

		if failed == 0 {
			fmt.Print(tui.RenderSuccessBox("All sources updated",
				fmt.Sprintf("%d sources updated successfully", updated)))
		} else {
			fmt.Print(tui.RenderWarningBox("Update completed with errors",
				fmt.Sprintf("%d updated, %d failed", updated, failed)))
		}
		fmt.Println()
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

	fmt.Println()
	fmt.Println(tui.TitleStyle.Render(tui.IconNetwork + " " + name))
	fmt.Println()

	// Status badge
	fmt.Printf("  %s  %s\n",
		tui.EnabledBadge(src.IsEnabled()),
		tui.MutedStyle.Render(src.Type()),
	)
	fmt.Println()

	// Details section
	fmt.Println(tui.RenderSection("  Details"))
	fmt.Printf("  %s\n", tui.RenderKeyValue("Type", src.Type()))
	fmt.Printf("  %s\n", tui.RenderKeyValue("Priority", fmt.Sprintf("%d", src.Priority())))

	if gitSrc, ok := src.(*registry.GitSource); ok {
		fmt.Printf("  %s\n", tui.RenderKeyValue("URL", gitSrc.GetURL()))
		fmt.Printf("  %s\n", tui.RenderKeyValue("Branch", gitSrc.GetBranch()))
		if commit := gitSrc.GetCommit(); commit != "" {
			fmt.Printf("  %s\n", tui.RenderKeyValue("Commit", truncate(commit, 12)))
		}
		if !gitSrc.GetLastUpdated().IsZero() {
			fmt.Printf("  %s\n", tui.RenderKeyValue("Updated", gitSrc.GetLastUpdated().Format("2006-01-02 15:04:05")))
		}
	}
	fmt.Println()

	// List services from this source
	services, err := src.ListServices(ctx)
	if err != nil {
		return err
	}

	fmt.Println(tui.RenderSection(fmt.Sprintf("  Services (%d)", len(services))))
	if len(services) == 0 {
		fmt.Println(tui.MutedStyle.Render("  No services found"))
	} else if len(services) <= 20 {
		for _, svcName := range services {
			def, _ := src.LoadService(ctx, svcName)
			if def != nil {
				addonTag := ""
				if def.Conditions.RequireAddon {
					addonTag = tui.MutedStyle.Render(" (addon)")
				} else {
					addonTag = tui.SuccessStyle.Render(" (core)")
				}
				fmt.Printf("  %s %s%s\n", tui.IconPackage, svcName, addonTag)
			}
		}
	} else {
		// Show count and first few
		for i, svcName := range services[:10] {
			def, _ := src.LoadService(ctx, svcName)
			if def != nil {
				addonTag := ""
				if def.Conditions.RequireAddon {
					addonTag = tui.MutedStyle.Render(" (addon)")
				} else {
					addonTag = tui.SuccessStyle.Render(" (core)")
				}
				fmt.Printf("  %s %s%s\n", tui.IconPackage, svcName, addonTag)
			}
			if i == 9 {
				fmt.Printf("  %s\n", tui.MutedStyle.Render(fmt.Sprintf("  ... and %d more", len(services)-10)))
			}
		}
	}
	fmt.Println()

	return nil
}

// loadSourceConfig loads the source configuration
func loadSourceConfig() *registry.SourceConfig {
	configPath := getSourceConfigPath()

	loader := registry.NewLoader()
	cfg, err := loader.LoadSourceConfig(configPath)
	if err != nil {
		return registry.DefaultSourceConfig()
	}

	return cfg
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
