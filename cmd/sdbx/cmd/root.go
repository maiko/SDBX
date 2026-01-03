// Package cmd contains all CLI commands for sdbx.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	noTUI   bool
	jsonOut bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sdbx",
	Short: "Seedbox in a Box - Production-ready media automation stack",
	Long: `SDBX is a CLI tool for bootstrapping, deploying, and managing
a production-ready seedbox stack with authentication, VPN-enforced
downloads, and beautiful TUI interfaces.

Features:
  • Auth everywhere (Authelia SSO + 2FA)
  • VPN-enforced downloads with kill-switch
  • Full *arr stack automation
  • Plex integration with Overseerr & Wizarr
  • Beautiful TUI dashboard

Get started:
  sdbx init     Bootstrap a new project
  sdbx up       Start all services
  sdbx status   View live dashboard
  sdbx doctor   Run diagnostic checks`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .sdbx.yaml)")
	rootCmd.PersistentFlags().BoolVar(&noTUI, "no-tui", false, "disable TUI, use plain text output")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")

	// Bind flags to viper (panic on error as this indicates a programming bug)
	if err := viper.BindPFlag("no-tui", rootCmd.PersistentFlags().Lookup("no-tui")); err != nil {
		panic(fmt.Sprintf("failed to bind no-tui flag: %v", err))
	}
	if err := viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json")); err != nil {
		panic(fmt.Sprintf("failed to bind json flag: %v", err))
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".sdbx")
	}

	// Read environment variables with SDBX_ prefix
	viper.SetEnvPrefix("SDBX")
	viper.AutomaticEnv()

	// Read config file if it exists (errors are silently ignored)
	_ = viper.ReadInConfig()
}

// IsTUIEnabled returns true if TUI mode is enabled
func IsTUIEnabled() bool {
	// TUI is enabled by default in interactive terminals
	if noTUI || jsonOut {
		return false
	}
	// Check if stdout is a terminal
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// IsJSONOutput returns true if JSON output is requested
func IsJSONOutput() bool {
	return jsonOut
}

// OutputJSON marshals data to JSON and prints it to stdout.
// Returns an error if marshaling fails.
func OutputJSON(data interface{}) error {
	output, err := MarshalJSON(data)
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// MarshalJSON marshals data to indented JSON.
// Returns the JSON bytes or an error.
func MarshalJSON(data interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}
