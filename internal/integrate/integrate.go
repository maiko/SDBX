package integrate

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maiko/sdbx/internal/config"
)

// arrConfig represents the structure of *arr config.xml files
type arrConfig struct {
	XMLName xml.Name `xml:"Config"`
	APIKey  string   `xml:"ApiKey"`
}

// Integrator orchestrates service integrations
type Integrator struct {
	config     *Config
	httpClient *HTTPClient
	services   map[string]*ServiceConfig
}

// NewIntegrator creates a new integrator
func NewIntegrator(cfg *Config) *Integrator {
	return &Integrator{
		config:     cfg,
		httpClient: NewHTTPClient(cfg.Timeout, cfg.RetryAttempts, cfg.RetryDelay),
		services:   cfg.Services,
	}
}

// Run executes all integrations
func (i *Integrator) Run(ctx context.Context) ([]*IntegrationResult, error) {
	results := make([]*IntegrationResult, 0)

	// Wait for services to be ready
	if !i.config.DryRun {
		if err := i.waitForServices(ctx); err != nil {
			return results, fmt.Errorf("services not ready: %w", err)
		}
	}

	// Integrate Prowlarr with *arr apps
	if i.hasService("prowlarr") {
		prowlarrResults := i.integrateProwlarr(ctx)
		results = append(results, prowlarrResults...)
	}

	// Integrate qBittorrent with *arr apps
	if i.hasService("qbittorrent") {
		qbitResults := i.integrateQBittorrent(ctx)
		results = append(results, qbitResults...)
	}

	return results, nil
}

// waitForServices waits for all services to be ready
func (i *Integrator) waitForServices(ctx context.Context) error {
	const serviceTimeout = 30 * time.Second // Reduced - services should be ready quickly
	const retryInterval = 2 * time.Second

	for name, svc := range i.services {
		if !svc.Enabled {
			continue
		}

		fmt.Printf("Checking %s at %s...\n", name, svc.URL)

		// Each service gets its own timeout
		serviceDeadline := time.Now().Add(serviceTimeout)
		var lastErr error

		for time.Now().Before(serviceDeadline) {
			if err := i.checkServiceHealth(ctx, name, svc); err == nil {
				fmt.Printf("✓ %s is ready\n", name)
				break
			} else {
				lastErr = err // Save for logging
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryInterval):
				// Retry
			}
		}

		if time.Now().After(serviceDeadline) {
			// Log error but continue (non-fatal)
			fmt.Printf("⚠ Warning: %s health check failed after %v: %v\n",
				name, serviceTimeout, lastErr)
			fmt.Printf("  Continuing anyway (service may still work)...\n")
		}
	}

	return nil
}

// checkServiceHealth checks if a service is healthy
func (i *Integrator) checkServiceHealth(ctx context.Context, name string, svc *ServiceConfig) error {
	switch name {
	case "prowlarr":
		client := NewProwlarrClient(i.httpClient, svc)
		return client.CheckHealth(ctx)
	case "sonarr", "radarr", "lidarr", "readarr":
		client := NewArrClient(i.httpClient, svc)
		return client.CheckHealth(ctx)
	case "qbittorrent":
		port := svc.Port
		if port == 0 {
			port = 8080 // Default qBittorrent WebUI port
		}
		qbitCfg := &QBittorrentConfig{
			Host:     svc.URL,
			Port:     port,
			Username: "admin",
			Password: svc.APIKey,
		}
		client := NewQBittorrentClient(i.httpClient, qbitCfg)
		if err := client.Login(ctx); err != nil {
			return err
		}
		return client.CheckHealth(ctx)
	default:
		return nil
	}
}

// integrateProwlarr integrates Prowlarr with *arr apps
func (i *Integrator) integrateProwlarr(ctx context.Context) []*IntegrationResult {
	results := make([]*IntegrationResult, 0)

	prowlarrSvc := i.services["prowlarr"]
	prowlarr := NewProwlarrClient(i.httpClient, prowlarrSvc)

	// Get existing applications
	existingApps, err := prowlarr.GetApplications(ctx)
	if err != nil {
		results = append(results, &IntegrationResult{
			Service: "prowlarr",
			Success: false,
			Message: "Failed to get existing applications",
			Error:   err,
		})
		return results
	}

	existingNames := make(map[string]*ProwlarrApplication)
	for idx := range existingApps {
		existingNames[existingApps[idx].Name] = &existingApps[idx]
	}

	// Integrate with each *arr app
	arrApps := []string{"sonarr", "radarr", "lidarr", "readarr"}
	for _, appName := range arrApps {
		if !i.hasService(appName) {
			continue
		}

		svc := i.services[appName]
		result := i.addProwlarrApplication(ctx, prowlarr, appName, svc, existingNames)
		results = append(results, result)
	}

	return results
}

// addProwlarrApplication adds or updates a *arr app in Prowlarr
func (i *Integrator) addProwlarrApplication(
	ctx context.Context, prowlarr *ProwlarrClient, appName string,
	svc *ServiceConfig, existing map[string]*ProwlarrApplication,
) *IntegrationResult {
	// Check if already exists
	if existingApp, exists := existing[appName]; exists {
		return &IntegrationResult{
			Service: fmt.Sprintf("prowlarr → %s", appName),
			Success: true,
			Message: fmt.Sprintf("Already configured (ID: %d)", existingApp.ID),
			Error:   nil,
		}
	}

	// Create application config
	var app *ProwlarrApplication
	syncLevel := "fullSync"

	switch appName {
	case "sonarr":
		app = CreateSonarrApplication(appName, svc.URL, svc.APIKey, syncLevel)
	case "radarr":
		app = CreateRadarrApplication(appName, svc.URL, svc.APIKey, syncLevel)
	case "lidarr":
		app = CreateLidarrApplication(appName, svc.URL, svc.APIKey, syncLevel)
	case "readarr":
		app = CreateReadarrApplication(appName, svc.URL, svc.APIKey, syncLevel)
	default:
		return &IntegrationResult{
			Service: fmt.Sprintf("prowlarr → %s", appName),
			Success: false,
			Message: "Unsupported application type",
			Error:   fmt.Errorf("unknown app: %s", appName),
		}
	}

	// Add application
	if i.config.DryRun {
		return &IntegrationResult{
			Service: fmt.Sprintf("prowlarr → %s", appName),
			Success: true,
			Message: "[DRY RUN] Would add application",
			Error:   nil,
		}
	}

	result, err := prowlarr.AddApplication(ctx, app)
	if err != nil {
		return &IntegrationResult{
			Service: fmt.Sprintf("prowlarr → %s", appName),
			Success: false,
			Message: "Failed to add application",
			Error:   err,
		}
	}

	return &IntegrationResult{
		Service: fmt.Sprintf("prowlarr → %s", appName),
		Success: true,
		Message: fmt.Sprintf("Added successfully (ID: %d)", result.ID),
		Error:   nil,
	}
}

// integrateQBittorrent integrates qBittorrent with *arr apps
func (i *Integrator) integrateQBittorrent(ctx context.Context) []*IntegrationResult {
	results := make([]*IntegrationResult, 0)

	qbitSvc := i.services["qbittorrent"]
	port := qbitSvc.Port
	if port == 0 {
		port = 8080 // Default qBittorrent WebUI port
	}
	qbitCfg := &QBittorrentConfig{
		Host:     qbitSvc.URL,
		Port:     port,
		Username: "admin",
		Password: qbitSvc.APIKey,
	}
	qbit := NewQBittorrentClient(i.httpClient, qbitCfg)

	// Login to qBittorrent
	if !i.config.DryRun {
		if err := qbit.Login(ctx); err != nil {
			results = append(results, &IntegrationResult{
				Service: "qbittorrent",
				Success: false,
				Message: "Failed to login",
				Error:   err,
			})
			return results
		}
	}

	// Integrate with each *arr app
	arrApps := []string{"sonarr", "radarr", "lidarr", "readarr"}
	for _, appName := range arrApps {
		if !i.hasService(appName) {
			continue
		}

		svc := i.services[appName]
		result := i.addQBittorrentToArr(ctx, qbit, appName, svc, qbitCfg)
		results = append(results, result)
	}

	return results
}

// addQBittorrentToArr adds qBittorrent as download client to *arr app
func (i *Integrator) addQBittorrentToArr(
	ctx context.Context, qbit *QBittorrentClient, appName string,
	arrSvc *ServiceConfig, qbitCfg *QBittorrentConfig,
) *IntegrationResult {
	arr := NewArrClient(i.httpClient, arrSvc)

	// Get existing download clients
	existingClients, err := arr.GetDownloadClients(ctx)
	if err != nil {
		return &IntegrationResult{
			Service: fmt.Sprintf("%s → qbittorrent", appName),
			Success: false,
			Message: "Failed to get download clients",
			Error:   err,
		}
	}

	// Check if qBittorrent already configured
	for _, client := range existingClients {
		if strings.EqualFold(client.Implementation, "qbittorrent") {
			return &IntegrationResult{
				Service: fmt.Sprintf("%s → qbittorrent", appName),
				Success: true,
				Message: fmt.Sprintf("Already configured (ID: %d)", client.ID),
				Error:   nil,
			}
		}
	}

	// Create category in qBittorrent
	if !i.config.DryRun {
		if err := qbit.CreateCategory(ctx, appName, ""); err != nil {
			// Non-fatal, continue anyway
			if i.config.Verbose {
				fmt.Printf("Warning: Failed to create category %s: %v\n", appName, err)
			}
		}
	}

	// Create download client config
	client := CreateQBittorrentClient(
		"qBittorrent",
		strings.TrimPrefix(qbitCfg.Host, "http://"),
		qbitCfg.Port,
		qbitCfg.Username,
		qbitCfg.Password,
	)

	// Set category to app name
	for idx, field := range client.Fields {
		if field.Name == "category" {
			client.Fields[idx].Value = appName
			break
		}
	}

	// Add download client
	if i.config.DryRun {
		return &IntegrationResult{
			Service: fmt.Sprintf("%s → qbittorrent", appName),
			Success: true,
			Message: "[DRY RUN] Would add download client",
			Error:   nil,
		}
	}

	result, err := arr.AddDownloadClient(ctx, client)
	if err != nil {
		return &IntegrationResult{
			Service: fmt.Sprintf("%s → qbittorrent", appName),
			Success: false,
			Message: "Failed to add download client",
			Error:   err,
		}
	}

	return &IntegrationResult{
		Service: fmt.Sprintf("%s → qbittorrent", appName),
		Success: true,
		Message: fmt.Sprintf("Added successfully (ID: %d)", result.ID),
		Error:   nil,
	}
}

// hasService checks if a service is enabled
func (i *Integrator) hasService(name string) bool {
	svc, exists := i.services[name]
	return exists && svc.Enabled
}

// LoadServicesFromConfig loads service configurations from SDBX config
func LoadServicesFromConfig(projectDir string) (map[string]*ServiceConfig, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	services := make(map[string]*ServiceConfig)

	// Helper to read API key from config file
	readAPIKey := func(serviceName string) (string, error) {
		configPath := filepath.Join(projectDir, "configs", serviceName, "config.xml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return "", err
		}

		// Parse XML properly
		var cfg arrConfig
		if err := xml.Unmarshal(data, &cfg); err != nil {
			return "", fmt.Errorf("failed to parse config.xml: %w", err)
		}

		if cfg.APIKey == "" {
			return "", fmt.Errorf("api key not found in config")
		}

		return cfg.APIKey, nil
	}

	// Prowlarr
	if cfg.IsAddonEnabled("prowlarr") {
		apiKey, err := readAPIKey("prowlarr")
		if err == nil {
			services["prowlarr"] = &ServiceConfig{
				Name:    "prowlarr",
				URL:     "http://sdbx-prowlarr:9696",
				APIKey:  apiKey,
				Enabled: true,
			}
		}
	}

	// Sonarr
	if cfg.IsAddonEnabled("sonarr") {
		apiKey, err := readAPIKey("sonarr")
		if err == nil {
			services["sonarr"] = &ServiceConfig{
				Name:    "sonarr",
				URL:     "http://sdbx-sonarr:8989",
				APIKey:  apiKey,
				Enabled: true,
			}
		}
	}

	// Radarr
	if cfg.IsAddonEnabled("radarr") {
		apiKey, err := readAPIKey("radarr")
		if err == nil {
			services["radarr"] = &ServiceConfig{
				Name:    "radarr",
				URL:     "http://sdbx-radarr:7878",
				APIKey:  apiKey,
				Enabled: true,
			}
		}
	}

	// Lidarr
	if cfg.IsAddonEnabled("lidarr") {
		apiKey, err := readAPIKey("lidarr")
		if err == nil {
			services["lidarr"] = &ServiceConfig{
				Name:    "lidarr",
				URL:     "http://sdbx-lidarr:8686",
				APIKey:  apiKey,
				Enabled: true,
			}
		}
	}

	// Readarr
	if cfg.IsAddonEnabled("readarr") {
		apiKey, err := readAPIKey("readarr")
		if err == nil {
			services["readarr"] = &ServiceConfig{
				Name:    "readarr",
				URL:     "http://sdbx-readarr:8787",
				APIKey:  apiKey,
				Enabled: true,
			}
		}
	}

	// qBittorrent - password is stored in secrets
	secretsPath := filepath.Join(projectDir, "secrets", "qbittorrent_password.txt")
	if password, err := os.ReadFile(secretsPath); err == nil {
		services["qbittorrent"] = &ServiceConfig{
			Name:    "qbittorrent",
			URL:     "http://sdbx-qbittorrent",
			Port:    8080, // Default qBittorrent WebUI port
			APIKey:  strings.TrimSpace(string(password)),
			Enabled: true,
		}
	}

	return services, nil
}
