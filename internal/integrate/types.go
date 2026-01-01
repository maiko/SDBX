package integrate

import "time"

// ServiceConfig represents a service configuration for integration
type ServiceConfig struct {
	Name    string
	URL     string // Internal Docker URL
	APIKey  string
	Enabled bool
}

// IntegrationResult represents the result of an integration attempt
type IntegrationResult struct {
	Service string
	Success bool
	Message string
	Error   error
}

// Config holds configuration for the integrator
type Config struct {
	Services       map[string]*ServiceConfig
	Timeout        time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
	DryRun         bool
	Verbose        bool
}

// DefaultConfig returns default integration configuration
func DefaultConfig() *Config {
	return &Config{
		Services:      make(map[string]*ServiceConfig),
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    5 * time.Second,
		DryRun:        false,
		Verbose:       false,
	}
}

// QBittorrentConfig represents qBittorrent configuration
type QBittorrentConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// ProwlarrApplication represents an *arr app in Prowlarr
type ProwlarrApplication struct {
	ID                int                    `json:"id,omitempty"`
	Name              string                 `json:"name"`
	SyncLevel         string                 `json:"syncLevel"` // disabled, addOnly, fullSync
	Implementation    string                 `json:"implementation"`
	ConfigContract    string                 `json:"configContract"`
	Tags              []int                  `json:"tags"`
	Fields            []ProwlarrField        `json:"fields"`
}

// ProwlarrField represents a configuration field
type ProwlarrField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// DownloadClient represents a download client configuration for *arr apps
type DownloadClient struct {
	ID                int                    `json:"id,omitempty"`
	Name              string                 `json:"name"`
	Implementation    string                 `json:"implementation"`
	ConfigContract    string                 `json:"configContract"`
	Protocol          string                 `json:"protocol"` // torrent or usenet
	Priority          int                    `json:"priority"`
	Enable            bool                   `json:"enable"`
	Fields            []DownloadClientField  `json:"fields"`
}

// DownloadClientField represents a download client field
type DownloadClientField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}
