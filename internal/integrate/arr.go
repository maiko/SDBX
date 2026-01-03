package integrate

import (
	"context"
	"encoding/json"
	"fmt"
)

// ArrClient handles *arr apps (Sonarr, Radarr, Lidarr, Readarr) API interactions
type ArrClient struct {
	client *HTTPClient
	config *ServiceConfig
}

// NewArrClient creates a new *arr app API client
func NewArrClient(httpClient *HTTPClient, config *ServiceConfig) *ArrClient {
	return &ArrClient{
		client: httpClient,
		config: config,
	}
}

// CheckHealth verifies *arr app is accessible
func (a *ArrClient) CheckHealth(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v3/system/status", a.config.URL)
	headers := map[string]string{
		"X-Api-Key": a.config.APIKey,
	}

	_, err := a.client.Get(ctx, url, headers)
	if err != nil {
		return fmt.Errorf("%s health check failed: %w", a.config.Name, err)
	}

	return nil
}

// GetDownloadClients retrieves all configured download clients
func (a *ArrClient) GetDownloadClients(ctx context.Context) ([]DownloadClient, error) {
	url := fmt.Sprintf("%s/api/v3/downloadclient", a.config.URL)
	headers := map[string]string{
		"X-Api-Key": a.config.APIKey,
	}

	body, err := a.client.Get(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get download clients: %w", err)
	}

	var clients []DownloadClient
	if err := json.Unmarshal(body, &clients); err != nil {
		return nil, fmt.Errorf("failed to parse download clients: %w", err)
	}

	return clients, nil
}

// AddDownloadClient adds a new download client
func (a *ArrClient) AddDownloadClient(ctx context.Context, client *DownloadClient) (*DownloadClient, error) {
	url := fmt.Sprintf("%s/api/v3/downloadclient", a.config.URL)
	headers := map[string]string{
		"X-Api-Key": a.config.APIKey,
	}

	body, err := a.client.Post(ctx, url, headers, client)
	if err != nil {
		return nil, fmt.Errorf("failed to add download client: %w", err)
	}

	var result DownloadClient
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse download client response: %w", err)
	}

	return &result, nil
}

// UpdateDownloadClient updates an existing download client
func (a *ArrClient) UpdateDownloadClient(ctx context.Context, client *DownloadClient) error {
	url := fmt.Sprintf("%s/api/v3/downloadclient/%d", a.config.URL, client.ID)
	headers := map[string]string{
		"X-Api-Key": a.config.APIKey,
	}

	_, err := a.client.Put(ctx, url, headers, client)
	if err != nil {
		return fmt.Errorf("failed to update download client: %w", err)
	}

	return nil
}

// CreateQBittorrentClient creates a qBittorrent download client config
func CreateQBittorrentClient(name, host string, port int, username, password string) *DownloadClient {
	return &DownloadClient{
		Name:           name,
		Implementation: "QBittorrent",
		ConfigContract: "QBittorrentSettings",
		Protocol:       "torrent",
		Priority:       1,
		Enable:         true,
		Fields: []DownloadClientField{
			{Name: "host", Value: host},
			{Name: "port", Value: port},
			{Name: "username", Value: username},
			{Name: "password", Value: password},
			{Name: "category", Value: name}, // Use service name as category
			{Name: "recentTvPriority", Value: 0},
			{Name: "olderTvPriority", Value: 0},
			{Name: "initialState", Value: 0},
			{Name: "sequentialOrder", Value: false},
			{Name: "firstAndLast", Value: false},
		},
	}
}

// CreateSABnzbdClient creates a SABnzbd download client config
func CreateSABnzbdClient(name, host string, port int, apiKey string) *DownloadClient {
	return &DownloadClient{
		Name:           name,
		Implementation: "Sabnzbd",
		ConfigContract: "SabnzbdSettings",
		Protocol:       "usenet",
		Priority:       1,
		Enable:         true,
		Fields: []DownloadClientField{
			{Name: "host", Value: host},
			{Name: "port", Value: port},
			{Name: "apiKey", Value: apiKey},
			{Name: "category", Value: name},
			{Name: "recentTvPriority", Value: 0},
			{Name: "olderTvPriority", Value: 0},
		},
	}
}
