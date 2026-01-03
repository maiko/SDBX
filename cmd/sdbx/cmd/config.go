package cmd

import (
	"fmt"
	"strings"

	"github.com/maiko/sdbx/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage SDBX configuration",
	Long: `View and modify SDBX configuration values.

Use 'sdbx config get' to view current settings.
Use 'sdbx config set' to modify settings.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration value(s)",
	Long: `Get one or all configuration values.

If no key is specified, displays all configuration.

Available keys:
  domain, expose_mode, timezone, config_path, data_path,
  downloads_path, media_path, puid, pgid, umask,
  vpn_provider, vpn_country, addons`,
	RunE: runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Example:
  sdbx config set domain sdbx.example.com
  sdbx config set expose_mode cloudflared
  sdbx config set timezone America/New_York`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigGet(_ *cobra.Command, args []string) error {
	// Load all settings
	allSettings := viper.AllSettings()

	// JSON output
	if IsJSONOutput() {
		if len(args) == 1 {
			value := viper.Get(args[0])
			return OutputJSON(map[string]interface{}{args[0]: value})
		}
		return OutputJSON(allSettings)
	}

	// Single key
	if len(args) == 1 {
		key := args[0]
		value := viper.Get(key)
		if value == nil {
			return fmt.Errorf("unknown configuration key: %s", key)
		}
		fmt.Printf("%s = %v\n", key, value)
		return nil
	}

	// All keys
	fmt.Println(tui.TitleStyle.Render("SDBX Configuration"))
	fmt.Println()

	// Group settings for display
	groups := map[string][]string{
		"Core":        {"domain", "expose_mode", "timezone"},
		"Paths":       {"config_path", "data_path", "downloads_path", "media_path"},
		"Permissions": {"puid", "pgid", "umask"},
		"VPN":         {"vpn_provider", "vpn_country", "vpn_username"},
		"Addons":      {"addons"},
	}

	for groupName, keys := range groups {
		fmt.Println(tui.InfoStyle.Render(groupName + ":"))
		for _, key := range keys {
			value := viper.Get(key)
			if value != nil {
				valueStr := fmt.Sprintf("%v", value)
				if valueStr == "" {
					valueStr = tui.MutedStyle.Render("(not set)")
				}
				fmt.Printf("  %-20s %s\n", key, valueStr)
			}
		}
		fmt.Println()
	}

	return nil
}

func runConfigSet(_ *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Validate key exists
	validKeys := []string{
		"domain", "expose_mode", "timezone",
		"config_path", "data_path", "downloads_path", "media_path",
		"puid", "pgid", "umask",
		"vpn_provider", "vpn_country", "vpn_username",
	}

	isValid := false
	for _, k := range validKeys {
		if k == key {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("invalid configuration key: %s\nValid keys: %s", key, strings.Join(validKeys, ", "))
	}

	// Set value
	viper.Set(key, value)

	// Write config
	if err := viper.WriteConfig(); err != nil {
		// Try to write as new file
		if err := viper.SafeWriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("✓ Set %s = %s", key, value)))
	return nil
}
