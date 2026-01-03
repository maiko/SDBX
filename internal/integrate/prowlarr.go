package integrate

import (
	"context"
	"encoding/json"
	"fmt"
)

// ProwlarrClient handles Prowlarr API interactions
type ProwlarrClient struct {
	client *HTTPClient
	config *ServiceConfig
}

// NewProwlarrClient creates a new Prowlarr API client
func NewProwlarrClient(httpClient *HTTPClient, config *ServiceConfig) *ProwlarrClient {
	return &ProwlarrClient{
		client: httpClient,
		config: config,
	}
}

// CheckHealth verifies Prowlarr is accessible
func (p *ProwlarrClient) CheckHealth(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/system/status", p.config.URL)
	headers := map[string]string{
		"X-Api-Key": p.config.APIKey,
	}

	_, err := p.client.Get(ctx, url, headers)
	if err != nil {
		return fmt.Errorf("prowlarr health check failed: %w", err)
	}

	return nil
}

// GetApplications retrieves all configured applications
func (p *ProwlarrClient) GetApplications(ctx context.Context) ([]ProwlarrApplication, error) {
	url := fmt.Sprintf("%s/api/v1/applications", p.config.URL)
	headers := map[string]string{
		"X-Api-Key": p.config.APIKey,
	}

	body, err := p.client.Get(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}

	var apps []ProwlarrApplication
	if err := json.Unmarshal(body, &apps); err != nil {
		return nil, fmt.Errorf("failed to parse applications: %w", err)
	}

	return apps, nil
}

// AddApplication adds a new *arr application to Prowlarr
func (p *ProwlarrClient) AddApplication(ctx context.Context, app *ProwlarrApplication) (*ProwlarrApplication, error) {
	url := fmt.Sprintf("%s/api/v1/applications", p.config.URL)
	headers := map[string]string{
		"X-Api-Key": p.config.APIKey,
	}

	body, err := p.client.Post(ctx, url, headers, app)
	if err != nil {
		return nil, fmt.Errorf("failed to add application: %w", err)
	}

	var result ProwlarrApplication
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse application response: %w", err)
	}

	return &result, nil
}

// UpdateApplication updates an existing *arr application
func (p *ProwlarrClient) UpdateApplication(ctx context.Context, app *ProwlarrApplication) error {
	url := fmt.Sprintf("%s/api/v1/applications/%d", p.config.URL, app.ID)
	headers := map[string]string{
		"X-Api-Key": p.config.APIKey,
	}

	_, err := p.client.Put(ctx, url, headers, app)
	if err != nil {
		return fmt.Errorf("failed to update application: %w", err)
	}

	return nil
}

// CreateSonarrApplication creates a Sonarr application config
func CreateSonarrApplication(name, baseURL, apiKey string, syncLevel string) *ProwlarrApplication {
	return &ProwlarrApplication{
		Name:              name,
		SyncLevel:         syncLevel,
		Implementation:    "Sonarr",
		ConfigContract:    "SonarrSettings",
		Tags:              []int{},
		Fields: []ProwlarrField{
			{Name: "baseUrl", Value: baseURL},
			{Name: "apiKey", Value: apiKey},
			{Name: "syncCategories", Value: []int{5000, 5030, 5040}}, // TV categories
		},
	}
}

// CreateRadarrApplication creates a Radarr application config
func CreateRadarrApplication(name, baseURL, apiKey string, syncLevel string) *ProwlarrApplication {
	return &ProwlarrApplication{
		Name:              name,
		SyncLevel:         syncLevel,
		Implementation:    "Radarr",
		ConfigContract:    "RadarrSettings",
		Tags:              []int{},
		Fields: []ProwlarrField{
			{Name: "baseUrl", Value: baseURL},
			{Name: "apiKey", Value: apiKey},
			{Name: "syncCategories", Value: []int{2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060}}, // Movie categories
		},
	}
}

// CreateLidarrApplication creates a Lidarr application config
func CreateLidarrApplication(name, baseURL, apiKey string, syncLevel string) *ProwlarrApplication {
	return &ProwlarrApplication{
		Name:              name,
		SyncLevel:         syncLevel,
		Implementation:    "Lidarr",
		ConfigContract:    "LidarrSettings",
		Tags:              []int{},
		Fields: []ProwlarrField{
			{Name: "baseUrl", Value: baseURL},
			{Name: "apiKey", Value: apiKey},
			{Name: "syncCategories", Value: []int{3000, 3010, 3020, 3030, 3040}}, // Music categories
		},
	}
}

// CreateReadarrApplication creates a Readarr application config
func CreateReadarrApplication(name, baseURL, apiKey string, syncLevel string) *ProwlarrApplication {
	return &ProwlarrApplication{
		Name:              name,
		SyncLevel:         syncLevel,
		Implementation:    "Readarr",
		ConfigContract:    "ReadarrSettings",
		Tags:              []int{},
		Fields: []ProwlarrField{
			{Name: "baseUrl", Value: baseURL},
			{Name: "apiKey", Value: apiKey},
			{Name: "syncCategories", Value: []int{3030, 7020, 8010}}, // Books/Ebooks categories
		},
	}
}
