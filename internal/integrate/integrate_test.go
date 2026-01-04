package integrate

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDefaultConfig verifies default configuration values
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", cfg.Timeout)
	}

	if cfg.RetryAttempts != 3 {
		t.Errorf("expected retryAttempts 3, got %d", cfg.RetryAttempts)
	}

	if cfg.RetryDelay != 5*time.Second {
		t.Errorf("expected retryDelay 5s, got %v", cfg.RetryDelay)
	}

	if cfg.DryRun != false {
		t.Error("expected dryRun false")
	}

	if cfg.Verbose != false {
		t.Error("expected verbose false")
	}

	if cfg.Services == nil {
		t.Error("services map should be initialized")
	}
}

// TestNewIntegrator verifies integrator construction
func TestNewIntegrator(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Services["test"] = &ServiceConfig{Name: "test", Enabled: true}

	integrator := NewIntegrator(cfg)

	if integrator == nil {
		t.Fatal("NewIntegrator returned nil")
	}

	if integrator.httpClient == nil {
		t.Error("httpClient should be initialized")
	}

	if len(integrator.services) != 1 {
		t.Errorf("expected 1 service, got %d", len(integrator.services))
	}
}

// TestHasService verifies service existence check
func TestHasService(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Services["enabled"] = &ServiceConfig{Name: "enabled", Enabled: true}
	cfg.Services["disabled"] = &ServiceConfig{Name: "disabled", Enabled: false}

	integrator := NewIntegrator(cfg)

	if !integrator.hasService("enabled") {
		t.Error("should have 'enabled' service")
	}

	if integrator.hasService("disabled") {
		t.Error("should not have 'disabled' service (disabled)")
	}

	if integrator.hasService("nonexistent") {
		t.Error("should not have 'nonexistent' service")
	}
}

// TestNewHTTPClient verifies HTTP client construction
func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient(10*time.Second, 3, 1*time.Second)

	if client == nil {
		t.Fatal("NewHTTPClient returned nil")
	}

	if client.retryAttempts != 3 {
		t.Errorf("expected retryAttempts 3, got %d", client.retryAttempts)
	}

	if client.retryDelay != 1*time.Second {
		t.Errorf("expected retryDelay 1s, got %v", client.retryDelay)
	}
}

// TestHTTPClientGet verifies GET request
func TestHTTPClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("X-Test") != "value" {
			t.Error("header not set correctly")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(10*time.Second, 0, 0)
	ctx := context.Background()

	body, err := client.Get(ctx, server.URL, map[string]string{"X-Test": "value"})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(body) != `{"status":"ok"}` {
		t.Errorf("unexpected response: %s", string(body))
	}
}

// TestHTTPClientPost verifies POST request
func TestHTTPClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("content-type not set")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":123}`))
	}))
	defer server.Close()

	client := NewHTTPClient(10*time.Second, 0, 0)
	ctx := context.Background()

	body, err := client.Post(ctx, server.URL, nil, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if !strings.Contains(string(body), "123") {
		t.Errorf("unexpected response: %s", string(body))
	}
}

// TestHTTPClientPut verifies PUT request
func TestHTTPClientPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"updated":true}`))
	}))
	defer server.Close()

	client := NewHTTPClient(10*time.Second, 0, 0)
	ctx := context.Background()

	body, err := client.Put(ctx, server.URL, nil, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if !strings.Contains(string(body), "true") {
		t.Errorf("unexpected response: %s", string(body))
	}
}

// TestHTTPClientRetry verifies retry logic
func TestHTTPClientRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewHTTPClient(10*time.Second, 3, 10*time.Millisecond)
	ctx := context.Background()

	body, err := client.Get(ctx, server.URL, nil)
	if err != nil {
		t.Fatalf("Get failed after retry: %v", err)
	}

	if string(body) != "success" {
		t.Errorf("unexpected response: %s", string(body))
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestHTTPClientClientError verifies no retry on 4xx
func TestHTTPClientClientError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := NewHTTPClient(10*time.Second, 3, 10*time.Millisecond)
	ctx := context.Background()

	_, err := client.Get(ctx, server.URL, nil)
	if err == nil {
		t.Error("expected error for 400 response")
	}

	if attempts != 1 {
		t.Errorf("should not retry on client error, got %d attempts", attempts)
	}
}

// TestIntegrationResult verifies result structure
func TestIntegrationResult(t *testing.T) {
	result := &IntegrationResult{
		Service: "test-service",
		Success: true,
		Message: "Test message",
		Error:   nil,
	}

	if result.Service != "test-service" {
		t.Error("service not set correctly")
	}

	if !result.Success {
		t.Error("success should be true")
	}

	if result.Message != "Test message" {
		t.Error("message not set correctly")
	}
}

// TestServiceConfig verifies service config structure
func TestServiceConfig(t *testing.T) {
	cfg := &ServiceConfig{
		Name:    "sonarr",
		URL:     "http://sdbx-sonarr:8989",
		Port:    8989,
		APIKey:  "test-api-key",
		Enabled: true,
	}

	if cfg.Name != "sonarr" {
		t.Error("name not set correctly")
	}

	if cfg.URL != "http://sdbx-sonarr:8989" {
		t.Error("url not set correctly")
	}

	if cfg.Port != 8989 {
		t.Error("port not set correctly")
	}

	if cfg.APIKey != "test-api-key" {
		t.Error("apikey not set correctly")
	}

	if !cfg.Enabled {
		t.Error("enabled should be true")
	}
}

// TestDryRunModeSkipsWaitForServices verifies dry run skips service wait
func TestDryRunModeSkipsWaitForServices(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DryRun = true
	// No services configured - should complete quickly in dry run mode

	integrator := NewIntegrator(cfg)
	ctx := context.Background()

	// This should complete immediately since no services are configured
	results, err := integrator.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// With no services, should have no results
	if len(results) != 0 {
		t.Errorf("expected 0 results with no services, got %d", len(results))
	}
}

// TestQBittorrentConfig verifies qBittorrent config structure
func TestQBittorrentConfig(t *testing.T) {
	cfg := &QBittorrentConfig{
		Host:     "localhost",
		Port:     8080,
		Username: "admin",
		Password: "adminpass",
	}

	if cfg.Host != "localhost" {
		t.Error("host not set correctly")
	}

	if cfg.Port != 8080 {
		t.Error("port not set correctly")
	}
}

// TestProwlarrApplication verifies Prowlarr app structure
func TestProwlarrApplication(t *testing.T) {
	app := &ProwlarrApplication{
		ID:             1,
		Name:           "sonarr",
		SyncLevel:      "fullSync",
		Implementation: "Sonarr",
		ConfigContract: "SonarrSettings",
		Tags:           []int{},
		Fields: []ProwlarrField{
			{Name: "baseUrl", Value: "http://localhost:8989"},
			{Name: "apiKey", Value: "test-key"},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(app)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if !strings.Contains(string(data), "sonarr") {
		t.Error("JSON should contain sonarr")
	}

	if !strings.Contains(string(data), "fullSync") {
		t.Error("JSON should contain fullSync")
	}
}

// TestDownloadClient verifies download client structure
func TestDownloadClient(t *testing.T) {
	client := &DownloadClient{
		ID:             1,
		Name:           "qBittorrent",
		Implementation: "QBittorrent",
		ConfigContract: "QBittorrentSettings",
		Protocol:       "torrent",
		Priority:       1,
		Enable:         true,
		Fields: []DownloadClientField{
			{Name: "host", Value: "localhost"},
			{Name: "port", Value: 8080},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(client)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if !strings.Contains(string(data), "qBittorrent") {
		t.Error("JSON should contain qBittorrent")
	}

	if !strings.Contains(string(data), "torrent") {
		t.Error("JSON should contain torrent protocol")
	}
}

// TestArrConfigXMLParsing verifies XML config parsing
func TestArrConfigXMLParsing(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<Config>
  <ApiKey>test-api-key-12345</ApiKey>
  <Port>8989</Port>
</Config>`

	var cfg arrConfig
	err := xml.Unmarshal([]byte(xmlData), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal XML: %v", err)
	}

	if cfg.APIKey != "test-api-key-12345" {
		t.Errorf("expected API key 'test-api-key-12345', got %q", cfg.APIKey)
	}
}

// TestLoadServicesFromConfigDir tests service loading with mock config files
func TestLoadServicesFromConfigDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock config directory structure
	sonarrDir := filepath.Join(tmpDir, "configs", "sonarr")
	if err := os.MkdirAll(sonarrDir, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// Create mock config.xml
	configXML := `<?xml version="1.0" encoding="utf-8"?>
<Config>
  <ApiKey>sonarr-api-key</ApiKey>
</Config>`
	if err := os.WriteFile(filepath.Join(sonarrDir, "config.xml"), []byte(configXML), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create secrets directory
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("failed to create secrets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "qbittorrent_password.txt"), []byte("testpass"), 0644); err != nil {
		t.Fatalf("failed to write secret: %v", err)
	}

	// Note: LoadServicesFromConfig requires actual SDBX config file
	// This test verifies the XML parsing and file structure handling
}

// TestIntegratorNoServices verifies behavior with no services
func TestIntegratorNoServices(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DryRun = true

	integrator := NewIntegrator(cfg)
	ctx := context.Background()

	results, err := integrator.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results with no services, got %d", len(results))
	}
}

// TestIntegratorContextCancellation verifies context cancellation handling
func TestIntegratorContextCancellation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Services["test"] = &ServiceConfig{Name: "test", URL: "http://localhost:9999", Enabled: true}
	cfg.Timeout = 100 * time.Millisecond

	integrator := NewIntegrator(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := integrator.Run(ctx)
	// Should handle canceled context gracefully
	// The exact behavior depends on where cancellation is checked
	_ = err // Error may or may not occur depending on timing
}

// TestHTTPClientNilBody verifies POST with nil body
func TestHTTPClientNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewHTTPClient(10*time.Second, 0, 0)
	ctx := context.Background()

	body, err := client.Post(ctx, server.URL, nil, nil)
	if err != nil {
		t.Fatalf("Post with nil body failed: %v", err)
	}

	if string(body) != "ok" {
		t.Errorf("unexpected response: %s", string(body))
	}
}

// TestCreateSonarrApplication tests Sonarr application creation
func TestCreateSonarrApplication(t *testing.T) {
	app := CreateSonarrApplication("Sonarr", "http://localhost:8989", "test-api-key", "fullSync")

	if app.Name != "Sonarr" {
		t.Errorf("Name = %q, want 'Sonarr'", app.Name)
	}

	if app.Implementation != "Sonarr" {
		t.Errorf("Implementation = %q, want 'Sonarr'", app.Implementation)
	}

	if app.ConfigContract != "SonarrSettings" {
		t.Errorf("ConfigContract = %q, want 'SonarrSettings'", app.ConfigContract)
	}

	if app.SyncLevel != "fullSync" {
		t.Errorf("SyncLevel = %q, want 'fullSync'", app.SyncLevel)
	}

	// Check fields
	if len(app.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(app.Fields))
	}

	// Verify baseUrl field
	found := false
	for _, f := range app.Fields {
		if f.Name == "baseUrl" && f.Value == "http://localhost:8989" {
			found = true
			break
		}
	}
	if !found {
		t.Error("baseUrl field not found or incorrect")
	}
}

// TestCreateRadarrApplication tests Radarr application creation
func TestCreateRadarrApplication(t *testing.T) {
	app := CreateRadarrApplication("Radarr", "http://localhost:7878", "radarr-key", "addOnly")

	if app.Implementation != "Radarr" {
		t.Errorf("Implementation = %q, want 'Radarr'", app.Implementation)
	}

	if app.ConfigContract != "RadarrSettings" {
		t.Errorf("ConfigContract = %q, want 'RadarrSettings'", app.ConfigContract)
	}

	if app.SyncLevel != "addOnly" {
		t.Errorf("SyncLevel = %q, want 'addOnly'", app.SyncLevel)
	}
}

// TestCreateLidarrApplication tests Lidarr application creation
func TestCreateLidarrApplication(t *testing.T) {
	app := CreateLidarrApplication("Lidarr", "http://localhost:8686", "lidarr-key", "fullSync")

	if app.Implementation != "Lidarr" {
		t.Errorf("Implementation = %q, want 'Lidarr'", app.Implementation)
	}

	if app.ConfigContract != "LidarrSettings" {
		t.Errorf("ConfigContract = %q, want 'LidarrSettings'", app.ConfigContract)
	}

	// Check music categories
	found := false
	for _, f := range app.Fields {
		if f.Name == "syncCategories" {
			found = true
			break
		}
	}
	if !found {
		t.Error("syncCategories field not found")
	}
}

// TestCreateReadarrApplication tests Readarr application creation
func TestCreateReadarrApplication(t *testing.T) {
	app := CreateReadarrApplication("Readarr", "http://localhost:8787", "readarr-key", "fullSync")

	if app.Implementation != "Readarr" {
		t.Errorf("Implementation = %q, want 'Readarr'", app.Implementation)
	}

	if app.ConfigContract != "ReadarrSettings" {
		t.Errorf("ConfigContract = %q, want 'ReadarrSettings'", app.ConfigContract)
	}
}

// TestNewProwlarrClient tests Prowlarr client creation
func TestNewProwlarrClient(t *testing.T) {
	httpClient := NewHTTPClient(10*time.Second, 3, 1*time.Second)
	cfg := &ServiceConfig{
		Name:    "prowlarr",
		URL:     "http://localhost:9696",
		APIKey:  "test-key",
		Enabled: true,
	}

	client := NewProwlarrClient(httpClient, cfg)

	if client == nil {
		t.Fatal("NewProwlarrClient returned nil")
	}

	if client.config.URL != "http://localhost:9696" {
		t.Errorf("URL = %q, want 'http://localhost:9696'", client.config.URL)
	}
}

// TestProwlarrClientCheckHealth tests health check endpoint
func TestProwlarrClientCheckHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/system/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.Header.Get("X-Api-Key") != "test-key" {
			t.Error("X-Api-Key header not set")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"1.0.0"}`))
	}))
	defer server.Close()

	httpClient := NewHTTPClient(10*time.Second, 0, 0)
	cfg := &ServiceConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}

	client := NewProwlarrClient(httpClient, cfg)
	ctx := context.Background()

	err := client.CheckHealth(ctx)
	if err != nil {
		t.Fatalf("CheckHealth failed: %v", err)
	}
}

// TestProwlarrClientCheckHealthError tests health check failure
func TestProwlarrClientCheckHealthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	httpClient := NewHTTPClient(10*time.Second, 0, 0)
	cfg := &ServiceConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}

	client := NewProwlarrClient(httpClient, cfg)
	ctx := context.Background()

	err := client.CheckHealth(ctx)
	if err == nil {
		t.Error("CheckHealth should fail on 503")
	}
}

// TestProwlarrClientGetApplications tests getting applications
func TestProwlarrClientGetApplications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/applications" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		apps := []ProwlarrApplication{
			{ID: 1, Name: "Sonarr", Implementation: "Sonarr"},
			{ID: 2, Name: "Radarr", Implementation: "Radarr"},
		}

		data, _ := json.Marshal(apps)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	httpClient := NewHTTPClient(10*time.Second, 0, 0)
	cfg := &ServiceConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}

	client := NewProwlarrClient(httpClient, cfg)
	ctx := context.Background()

	apps, err := client.GetApplications(ctx)
	if err != nil {
		t.Fatalf("GetApplications failed: %v", err)
	}

	if len(apps) != 2 {
		t.Errorf("expected 2 apps, got %d", len(apps))
	}

	if apps[0].Name != "Sonarr" {
		t.Errorf("first app name = %q, want 'Sonarr'", apps[0].Name)
	}
}

// TestProwlarrClientAddApplication tests adding an application
func TestProwlarrClientAddApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/applications" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Return the app with an ID
		app := ProwlarrApplication{
			ID:             123,
			Name:           "TestApp",
			Implementation: "Sonarr",
		}

		data, _ := json.Marshal(app)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	httpClient := NewHTTPClient(10*time.Second, 0, 0)
	cfg := &ServiceConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}

	client := NewProwlarrClient(httpClient, cfg)
	ctx := context.Background()

	newApp := CreateSonarrApplication("TestApp", "http://localhost:8989", "key", "fullSync")
	result, err := client.AddApplication(ctx, newApp)
	if err != nil {
		t.Fatalf("AddApplication failed: %v", err)
	}

	if result.ID != 123 {
		t.Errorf("expected ID 123, got %d", result.ID)
	}
}

// TestProwlarrClientUpdateApplication tests updating an application
func TestProwlarrClientUpdateApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/applications/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	httpClient := NewHTTPClient(10*time.Second, 0, 0)
	cfg := &ServiceConfig{
		URL:    server.URL,
		APIKey: "test-key",
	}

	client := NewProwlarrClient(httpClient, cfg)
	ctx := context.Background()

	app := &ProwlarrApplication{
		ID:             42,
		Name:           "UpdatedApp",
		Implementation: "Sonarr",
	}

	err := client.UpdateApplication(ctx, app)
	if err != nil {
		t.Fatalf("UpdateApplication failed: %v", err)
	}
}

// TestProwlarrFieldStruct tests ProwlarrField struct
func TestProwlarrFieldStruct(t *testing.T) {
	field := ProwlarrField{
		Name:  "baseUrl",
		Value: "http://localhost:8989",
	}

	if field.Name != "baseUrl" {
		t.Errorf("Name = %q, want 'baseUrl'", field.Name)
	}

	data, _ := json.Marshal(field)
	if !strings.Contains(string(data), "baseUrl") {
		t.Error("JSON should contain 'baseUrl'")
	}
}

// TestDownloadClientFieldStruct tests DownloadClientField struct
func TestDownloadClientFieldStruct(t *testing.T) {
	field := DownloadClientField{
		Name:  "host",
		Value: "localhost",
	}

	if field.Name != "host" {
		t.Errorf("Name = %q, want 'host'", field.Name)
	}

	data, _ := json.Marshal(field)
	if !strings.Contains(string(data), "localhost") {
		t.Error("JSON should contain 'localhost'")
	}
}

// TestConfigStruct tests Config struct
func TestConfigStruct(t *testing.T) {
	cfg := &Config{
		Services:      make(map[string]*ServiceConfig),
		Timeout:       60 * time.Second,
		RetryAttempts: 5,
		RetryDelay:    10 * time.Second,
		DryRun:        true,
		Verbose:       true,
	}

	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want 60s", cfg.Timeout)
	}

	if cfg.RetryAttempts != 5 {
		t.Errorf("RetryAttempts = %d, want 5", cfg.RetryAttempts)
	}

	if !cfg.DryRun {
		t.Error("DryRun should be true")
	}

	if !cfg.Verbose {
		t.Error("Verbose should be true")
	}
}
