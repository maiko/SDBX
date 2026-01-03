// Package config handles configuration loading and management for sdbx.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	// Expose modes
	ExposeModeCloudflared = "cloudflared"
	ExposeModeDirect      = "direct"
	ExposeModeLAN         = "lan"

	// Routing strategies
	RoutingStrategyPath      = "path"
	RoutingStrategySubdomain = "subdomain"
)

// Config holds the sdbx configuration
type Config struct {
	// Core settings
	Domain   string `mapstructure:"domain"`
	Timezone string `mapstructure:"timezone"`

	// Exposure configuration
	Expose ExposeConfig `mapstructure:"expose"`

	// Routing configuration
	Routing RoutingConfig `mapstructure:"routing"`

	// Paths
	ConfigPath    string `mapstructure:"config_path"`
	DataPath      string `mapstructure:"data_path"`
	DownloadsPath string `mapstructure:"downloads_path"`
	MediaPath     string `mapstructure:"media_path"`

	// Permissions
	PUID  int    `mapstructure:"puid"`
	PGID  int    `mapstructure:"pgid"`
	Umask string `mapstructure:"umask"`

	// VPN
	VPNEnabled  bool   `mapstructure:"vpn_enabled"`
	VPNProvider string `mapstructure:"vpn_provider"`
	VPNUsername string `mapstructure:"vpn_username"`
	VPNCountry  string `mapstructure:"vpn_country"`

	// Addons
	Addons []string `mapstructure:"addons"`

	// Per-service overrides
	Services map[string]ServiceOverride `mapstructure:"services"`

	// Security (Transient, not saved to config)
	AdminUser         string `mapstructure:"-"`
	AdminPasswordHash string `mapstructure:"-"`

	// Legacy field for backward compatibility (deprecated)
	ExposeMode string `mapstructure:"expose_mode"`
}

// ExposeConfig defines how services are exposed to the network
type ExposeConfig struct {
	Mode string    `mapstructure:"mode"` // "lan" | "direct" | "cloudflared"
	TLS  TLSConfig `mapstructure:"tls"`
}

// TLSConfig defines TLS/SSL settings for direct mode
type TLSConfig struct {
	Provider string `mapstructure:"provider"`  // "acme" | "custom" | "selfsigned" | "none"
	Email    string `mapstructure:"email"`     // For ACME (Let's Encrypt)
	CertFile string `mapstructure:"cert_file"` // For custom certificates
	KeyFile  string `mapstructure:"key_file"`  // For custom certificates
}

// RoutingConfig defines how services are routed (subdomain vs path)
type RoutingConfig struct {
	Strategy   string `mapstructure:"strategy"`    // "subdomain" | "path"
	BaseDomain string `mapstructure:"base_domain"` // For path mode: the subdomain to use (e.g., "sdbx" → sdbx.domain.tld)
}

// ServiceOverride allows per-service routing customization
type ServiceOverride struct {
	Routing   string `mapstructure:"routing"`   // "subdomain" | "path" - override global strategy
	Subdomain string `mapstructure:"subdomain"` // Custom subdomain (e.g., "requests" for overseerr)
	Path      string `mapstructure:"path"`      // Custom path (e.g., "/movies" for radarr)
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() *Config {
	return &Config{
		Domain:   "sdbx.example.com",
		Timezone: "Europe/Paris",
		Expose: ExposeConfig{
			Mode: ExposeModeCloudflared,
			TLS: TLSConfig{
				Provider: "acme",
				Email:    "",
			},
		},
		Routing: RoutingConfig{
			Strategy:   "subdomain",
			BaseDomain: "sdbx",
		},
		ConfigPath:    "./config",
		DataPath:      "./data",
		DownloadsPath: "./data/downloads",
		MediaPath:     "./data/media",
		PUID:          1000,
		PGID:          1000,
		Umask:         "002",
		VPNEnabled:    false,
		VPNProvider:   "",
		VPNCountry:    "",
		Addons:        []string{},
		Services:      make(map[string]ServiceOverride),
	}
}

// Domain validation regex - matches valid domain names
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Required fields
	if c.Domain == "" {
		return NewValidationError("domain", "domain is required")
	}

	// Domain format validation
	if !domainRegex.MatchString(c.Domain) {
		return NewValidationError("domain", "invalid domain format")
	}

	// Timezone validation - must be a valid IANA timezone
	if c.Timezone == "" {
		return NewValidationError("timezone", "timezone is required")
	}
	if _, err := time.LoadLocation(c.Timezone); err != nil {
		return NewValidationError("timezone",
			fmt.Sprintf("invalid timezone %q - must be a valid IANA timezone (e.g., America/New_York, Europe/London)", c.Timezone))
	}

	// Expose mode validation
	validExposeModes := []string{"lan", "direct", "cloudflared"}
	if !contains(validExposeModes, c.Expose.Mode) {
		return NewValidationError("expose.mode",
			fmt.Sprintf("must be one of: %s", strings.Join(validExposeModes, ", ")))
	}

	// Routing strategy validation
	validRoutingStrategies := []string{"subdomain", "path"}
	if !contains(validRoutingStrategies, c.Routing.Strategy) {
		return NewValidationError("routing.strategy",
			fmt.Sprintf("must be one of: %s", strings.Join(validRoutingStrategies, ", ")))
	}

	// Path routing requires base domain
	if c.Routing.Strategy == RoutingStrategyPath && c.Routing.BaseDomain == "" {
		return NewValidationError("routing.base_domain",
			"base_domain is required when using path routing")
	}

	// VPN validation
	if c.VPNEnabled && c.VPNProvider == "" {
		return NewValidationError("vpn_provider",
			"vpn_provider is required when VPN is enabled")
	}

	// Path validation (basic check - non-empty)
	if c.ConfigPath == "" {
		return NewValidationError("config_path", "config_path cannot be empty")
	}
	if c.MediaPath == "" {
		return NewValidationError("media_path", "media_path cannot be empty")
	}
	if c.DownloadsPath == "" {
		return NewValidationError("downloads_path", "downloads_path cannot be empty")
	}

	// PUID/PGID validation
	if c.PUID < 0 || c.PUID > 65535 {
		return NewValidationError("puid", "must be between 0 and 65535")
	}
	if c.PGID < 0 || c.PGID > 65535 {
		return NewValidationError("pgid", "must be between 0 and 65535")
	}

	return nil
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// Load loads configuration from file and environment
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Set defaults in viper
	viper.SetDefault("domain", cfg.Domain)
	viper.SetDefault("timezone", cfg.Timezone)
	viper.SetDefault("expose.mode", cfg.Expose.Mode)
	viper.SetDefault("expose.tls.provider", cfg.Expose.TLS.Provider)
	viper.SetDefault("routing.strategy", cfg.Routing.Strategy)
	viper.SetDefault("routing.base_domain", cfg.Routing.BaseDomain)
	viper.SetDefault("config_path", cfg.ConfigPath)
	viper.SetDefault("data_path", cfg.DataPath)
	viper.SetDefault("downloads_path", cfg.DownloadsPath)
	viper.SetDefault("media_path", cfg.MediaPath)
	viper.SetDefault("puid", cfg.PUID)
	viper.SetDefault("pgid", cfg.PGID)
	viper.SetDefault("umask", cfg.Umask)
	viper.SetDefault("vpn_provider", cfg.VPNProvider)
	viper.SetDefault("vpn_country", cfg.VPNCountry)
	viper.SetDefault("addons", cfg.Addons)

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal into struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Legacy migration: if expose_mode is set but expose.mode is not, migrate
	if cfg.ExposeMode != "" && cfg.Expose.Mode == "" {
		cfg.Expose.Mode = cfg.ExposeMode
	}

	// Initialize Services map if nil
	if cfg.Services == nil {
		cfg.Services = make(map[string]ServiceOverride)
	}

	return cfg, nil
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	// Set all values in viper
	viper.Set("domain", c.Domain)
	viper.Set("timezone", c.Timezone)
	viper.Set("expose", c.Expose)
	viper.Set("routing", c.Routing)
	viper.Set("config_path", c.ConfigPath)
	viper.Set("data_path", c.DataPath)
	viper.Set("downloads_path", c.DownloadsPath)
	viper.Set("media_path", c.MediaPath)
	viper.Set("puid", c.PUID)
	viper.Set("pgid", c.PGID)
	viper.Set("umask", c.Umask)
	viper.Set("vpn_enabled", c.VPNEnabled)
	viper.Set("vpn_provider", c.VPNProvider)
	viper.Set("vpn_country", c.VPNCountry)
	viper.Set("addons", c.Addons)
	if len(c.Services) > 0 {
		viper.Set("services", c.Services)
	}

	return viper.WriteConfigAs(path)
}

// ProjectDir returns the base project directory
func ProjectDir() (string, error) {
	// Look for .sdbx.yaml or compose.yaml in current or parent dirs
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	startPath := dir

	for {
		if _, err := os.Stat(filepath.Join(dir, ".sdbx.yaml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "compose.yaml")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", &ProjectNotFoundError{StartPath: startPath}
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

// IsAddonEnabled checks if an addon is enabled
func (c *Config) IsAddonEnabled(addon string) bool {
	for _, a := range c.Addons {
		if a == addon {
			return true
		}
	}
	return false
}

// EnableAddon adds an addon to the enabled list
func (c *Config) EnableAddon(addon string) {
	if !c.IsAddonEnabled(addon) {
		c.Addons = append(c.Addons, addon)
	}
}

// DisableAddon removes an addon from the enabled list
func (c *Config) DisableAddon(addon string) {
	newAddons := make([]string, 0, len(c.Addons))
	for _, a := range c.Addons {
		if a != addon {
			newAddons = append(newAddons, a)
		}
	}
	c.Addons = newAddons
}

// GetServiceRoutingStrategy returns the effective routing strategy for a service
// It checks for per-service overrides first, then falls back to global routing strategy
func (c *Config) GetServiceRoutingStrategy(service string) string {
	if override, ok := c.Services[service]; ok && override.Routing != "" {
		return override.Routing
	}
	return c.Routing.Strategy
}

// GetServiceSubdomain returns the subdomain for a service
// For subdomain routing: returns the custom subdomain or the service name
// For path routing with subdomain exception: returns the custom subdomain
func (c *Config) GetServiceSubdomain(service string) string {
	if override, ok := c.Services[service]; ok && override.Subdomain != "" {
		return override.Subdomain
	}
	return service
}

// GetServicePath returns the path prefix for a service
// For path routing: returns the custom path or /service-name
func (c *Config) GetServicePath(service string) string {
	if override, ok := c.Services[service]; ok && override.Path != "" {
		return override.Path
	}
	return "/" + service
}

// GetServiceURL returns the full URL for a service based on routing configuration
func (c *Config) GetServiceURL(service string) string {
	strategy := c.GetServiceRoutingStrategy(service)

	if strategy == "subdomain" {
		subdomain := c.GetServiceSubdomain(service)
		return fmt.Sprintf("https://%s.%s", subdomain, c.Domain)
	}

	// Path-based routing
	path := c.GetServicePath(service)
	if c.Routing.BaseDomain != "" {
		return fmt.Sprintf("https://%s.%s%s", c.Routing.BaseDomain, c.Domain, path)
	}
	return fmt.Sprintf("https://%s%s", c.Domain, path)
}

// IsPathRouting returns true if the service uses path-based routing
func (c *Config) IsPathRouting(service string) bool {
	return c.GetServiceRoutingStrategy(service) == "path"
}

// NeedsTLS returns true if the exposure mode requires TLS configuration
func (c *Config) NeedsTLS() bool {
	return c.Expose.Mode == ExposeModeDirect
}

// IsCloudflared returns true if using Cloudflare Tunnel mode
func (c *Config) IsCloudflared() bool {
	return c.Expose.Mode == ExposeModeCloudflared
}

// IsLANMode returns true if in LAN (no-TLS) mode
func (c *Config) IsLANMode() bool {
	return c.Expose.Mode == ExposeModeLAN
}
