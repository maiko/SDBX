package integrate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// QBittorrentClient handles qBittorrent API interactions
type QBittorrentClient struct {
	client *HTTPClient
	config *QBittorrentConfig
	cookie string // Session cookie
}

// NewQBittorrentClient creates a new qBittorrent API client
func NewQBittorrentClient(httpClient *HTTPClient, config *QBittorrentConfig) *QBittorrentClient {
	return &QBittorrentClient{
		client: httpClient,
		config: config,
	}
}

// Login authenticates with qBittorrent
func (q *QBittorrentClient) Login(ctx context.Context) error {
	loginURL := fmt.Sprintf("%s:%d/api/v2/auth/login", q.config.Host, q.config.Port)

	// Create form data
	form := url.Values{}
	form.Set("username", q.config.Username)
	form.Set("password", q.config.Password)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := q.client.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	// Extract session cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "SID" {
			q.cookie = cookie.Value
			return nil
		}
	}

	return fmt.Errorf("no session cookie received")
}

// CheckHealth verifies qBittorrent is accessible
func (q *QBittorrentClient) CheckHealth(ctx context.Context) error {
	apiURL := fmt.Sprintf("%s:%d/api/v2/app/version", q.config.Host, q.config.Port)
	headers := map[string]string{}

	if q.cookie != "" {
		headers["Cookie"] = fmt.Sprintf("SID=%s", q.cookie)
	}

	_, err := q.client.Get(ctx, apiURL, headers)
	if err != nil {
		return fmt.Errorf("qbittorrent health check failed: %w", err)
	}

	return nil
}

// GetPreferences retrieves qBittorrent preferences
func (q *QBittorrentClient) GetPreferences(ctx context.Context) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s:%d/api/v2/app/preferences", q.config.Host, q.config.Port)
	headers := map[string]string{
		"Cookie": fmt.Sprintf("SID=%s", q.cookie),
	}

	body, err := q.client.Get(ctx, apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	}

	var prefs map[string]interface{}
	if err := json.Unmarshal(body, &prefs); err != nil {
		return nil, fmt.Errorf("failed to parse preferences: %w", err)
	}

	return prefs, nil
}

// SetPreferences sets qBittorrent preferences
func (q *QBittorrentClient) SetPreferences(ctx context.Context, prefs map[string]interface{}) error {
	apiURL := fmt.Sprintf("%s:%d/api/v2/app/setPreferences", q.config.Host, q.config.Port)

	// qBittorrent expects form data with json parameter
	jsonData, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	form := url.Values{}
	form.Set("json", string(jsonData))

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SID=%s", q.cookie))

	resp, err := q.client.client.Do(req)
	if err != nil {
		return fmt.Errorf("set preferences request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("set preferences failed with status %d", resp.StatusCode)
	}

	return nil
}

// CreateCategory creates a download category
func (q *QBittorrentClient) CreateCategory(ctx context.Context, category, savePath string) error {
	apiURL := fmt.Sprintf("%s:%d/api/v2/torrents/createCategory", q.config.Host, q.config.Port)

	form := url.Values{}
	form.Set("category", category)
	if savePath != "" {
		form.Set("savePath", savePath)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("SID=%s", q.cookie))

	resp, err := q.client.client.Do(req)
	if err != nil {
		return fmt.Errorf("create category request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("create category failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetCategories retrieves all categories
func (q *QBittorrentClient) GetCategories(ctx context.Context) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s:%d/api/v2/torrents/categories", q.config.Host, q.config.Port)
	headers := map[string]string{
		"Cookie": fmt.Sprintf("SID=%s", q.cookie),
	}

	body, err := q.client.Get(ctx, apiURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	var categories map[string]interface{}
	if err := json.Unmarshal(body, &categories); err != nil {
		return nil, fmt.Errorf("failed to parse categories: %w", err)
	}

	return categories, nil
}
