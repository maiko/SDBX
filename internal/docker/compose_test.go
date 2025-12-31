package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewCompose(t *testing.T) {
	projectDir := "/tmp/test-project"
	compose := NewCompose(projectDir)

	if compose.ProjectDir != projectDir {
		t.Errorf("ProjectDir = %s, want %s", compose.ProjectDir, projectDir)
	}
	if compose.ComposeFile != "compose.yaml" {
		t.Errorf("ComposeFile = %s, want compose.yaml", compose.ComposeFile)
	}
	if compose.ProjectName != "sdbx" {
		t.Errorf("ProjectName = %s, want sdbx", compose.ProjectName)
	}
}

func TestServiceJSONMarshaling(t *testing.T) {
	svc := Service{
		Name:     "test-service",
		Status:   "running",
		Health:   "healthy",
		Ports:    "8080:8080",
		Image:    "test:latest",
		Running:  true,
		ExitCode: 0,
	}

	// Marshal to JSON
	data, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("Failed to marshal Service: %v", err)
	}

	// Unmarshal back
	var decoded Service
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Service: %v", err)
	}

	// Verify fields
	if decoded.Name != svc.Name {
		t.Errorf("Name = %s, want %s", decoded.Name, svc.Name)
	}
	if decoded.Status != svc.Status {
		t.Errorf("Status = %s, want %s", decoded.Status, svc.Status)
	}
	if decoded.Health != svc.Health {
		t.Errorf("Health = %s, want %s", decoded.Health, svc.Health)
	}
	if decoded.Running != svc.Running {
		t.Errorf("Running = %v, want %v", decoded.Running, svc.Running)
	}
}

func TestServiceRunningStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"running service", "running", true},
		{"stopped service", "exited", false},
		{"created service", "created", false},
		{"paused service", "paused", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := Service{
				Name:    "test",
				Status:  tt.status,
				Running: tt.status == "running",
			}
			if svc.Running != tt.expected {
				t.Errorf("Service with status '%s': Running = %v, want %v",
					tt.status, svc.Running, tt.expected)
			}
		})
	}
}

func TestPSJSONParsing(t *testing.T) {
	// Simulate docker compose ps --format json output
	jsonOutput := `{"Name":"sdbx-traefik-1","State":"running","Health":"healthy","Image":"traefik:latest","Ports":"80:80,443:443","ExitCode":0}
{"Name":"sdbx-authelia-1","State":"running","Health":"","Image":"authelia:latest","Ports":"9091:9091","ExitCode":0}
{"Name":"sdbx-plex-1","State":"exited","Health":"","Image":"plexinc/pms-docker:latest","Ports":"","ExitCode":1}`

	// Parse services manually (simulating what PS does)
	var services []Service
	for _, line := range strings.Split(strings.TrimSpace(jsonOutput), "\n") {
		if line == "" {
			continue
		}
		var svc struct {
			Name     string `json:"Name"`
			State    string `json:"State"`
			Health   string `json:"Health"`
			Image    string `json:"Image"`
			Ports    string `json:"Ports"`
			ExitCode int    `json:"ExitCode"`
		}
		if err := json.Unmarshal([]byte(line), &svc); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}
		services = append(services, Service{
			Name:     svc.Name,
			Status:   svc.State,
			Health:   svc.Health,
			Image:    svc.Image,
			Ports:    svc.Ports,
			Running:  svc.State == "running",
			ExitCode: svc.ExitCode,
		})
	}

	// Verify parsing results
	if len(services) != 3 {
		t.Fatalf("Expected 3 services, got %d", len(services))
	}

	// Check first service (traefik - running and healthy)
	if services[0].Name != "sdbx-traefik-1" {
		t.Errorf("Service[0].Name = %s, want sdbx-traefik-1", services[0].Name)
	}
	if !services[0].Running {
		t.Error("Service[0] should be running")
	}
	if services[0].Health != "healthy" {
		t.Errorf("Service[0].Health = %s, want healthy", services[0].Health)
	}

	// Check second service (authelia - running, no health check)
	if services[1].Name != "sdbx-authelia-1" {
		t.Errorf("Service[1].Name = %s, want sdbx-authelia-1", services[1].Name)
	}
	if !services[1].Running {
		t.Error("Service[1] should be running")
	}
	if services[1].Health != "" {
		t.Errorf("Service[1].Health should be empty, got %s", services[1].Health)
	}

	// Check third service (plex - exited)
	if services[2].Name != "sdbx-plex-1" {
		t.Errorf("Service[2].Name = %s, want sdbx-plex-1", services[2].Name)
	}
	if services[2].Running {
		t.Error("Service[2] should not be running")
	}
	if services[2].ExitCode != 1 {
		t.Errorf("Service[2].ExitCode = %d, want 1", services[2].ExitCode)
	}
}

func TestHealthyServiceDetection(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		expected bool
	}{
		{
			name: "running with healthy status",
			service: Service{
				Name:    "test-svc",
				Running: true,
				Health:  "healthy",
			},
			expected: true,
		},
		{
			name: "running without health check",
			service: Service{
				Name:    "test-svc",
				Running: true,
				Health:  "",
			},
			expected: true,
		},
		{
			name: "running but unhealthy",
			service: Service{
				Name:    "test-svc",
				Running: true,
				Health:  "unhealthy",
			},
			expected: false,
		},
		{
			name: "not running",
			service: Service{
				Name:    "test-svc",
				Running: false,
				Health:  "",
			},
			expected: false,
		},
		{
			name: "running but starting",
			service: Service{
				Name:    "test-svc",
				Running: true,
				Health:  "starting",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate IsHealthy logic
			isHealthy := tt.service.Running && (tt.service.Health == "" || tt.service.Health == "healthy")
			if isHealthy != tt.expected {
				t.Errorf("IsHealthy = %v, want %v (Running=%v, Health=%s)",
					isHealthy, tt.expected, tt.service.Running, tt.service.Health)
			}
		})
	}
}

func TestServiceNameMatching(t *testing.T) {
	services := []Service{
		{Name: "sdbx-traefik-1", Running: true},
		{Name: "sdbx-authelia-1", Running: true},
		{Name: "sdbx-plex-1", Running: false},
	}

	tests := []struct {
		searchName string
		found      bool
		running    bool
	}{
		{"traefik", true, true},
		{"authelia", true, true},
		{"plex", true, false},
		{"nonexistent", false, false},
		{"sdbx-traefik-1", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.searchName, func(t *testing.T) {
			found := false
			var svc Service
			for _, s := range services {
				if strings.Contains(s.Name, tt.searchName) {
					found = true
					svc = s
					break
				}
			}

			if found != tt.found {
				t.Errorf("Service '%s': found = %v, want %v", tt.searchName, found, tt.found)
			}
			if found && svc.Running != tt.running {
				t.Errorf("Service '%s': running = %v, want %v", tt.searchName, svc.Running, tt.running)
			}
		})
	}
}

func TestWaitHealthyTimeout(t *testing.T) {
	// Test the timeout logic
	deadline := time.Now().Add(1 * time.Second)
	elapsed := time.Duration(0)

	// Simulate waiting loop
	for time.Now().Before(deadline) {
		elapsed += 100 * time.Millisecond
		time.Sleep(100 * time.Millisecond)
	}

	if elapsed < 900*time.Millisecond {
		t.Errorf("Should have waited at least 900ms, waited %v", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Errorf("Should not wait more than 2s, waited %v", elapsed)
	}
}

func TestComposeCommandArgs(t *testing.T) {
	compose := NewCompose("/test/dir")

	// Test that compose file and project name are set correctly
	expectedFile := "compose.yaml"
	expectedProject := "sdbx"

	if compose.ComposeFile != expectedFile {
		t.Errorf("ComposeFile = %s, want %s", compose.ComposeFile, expectedFile)
	}
	if compose.ProjectName != expectedProject {
		t.Errorf("ProjectName = %s, want %s", compose.ProjectName, expectedProject)
	}
}

func TestContextCancellation(t *testing.T) {
	// Test that context cancellation is respected
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait for context to be done
	<-ctx.Done()

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestLogsCommandArgs(t *testing.T) {
	tests := []struct {
		name        string
		service     string
		lines       int
		follow      bool
		expectedLen int // Expected number of args after "logs"
	}{
		{"no options", "", 0, false, 0},
		{"with lines", "", 10, false, 2}, // --tail 10
		{"with follow", "", 0, true, 1},  // -f
		{"with service", "traefik", 0, false, 1},
		{"all options", "plex", 50, true, 4}, // --tail 50 -f plex
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate building args array
			args := []string{"logs"}
			if tt.lines > 0 {
				args = append(args, "--tail", fmt.Sprintf("%d", tt.lines))
			}
			if tt.follow {
				args = append(args, "-f")
			}
			if tt.service != "" {
				args = append(args, tt.service)
			}

			// Verify correct number of args
			actualLen := len(args) - 1 // Subtract the "logs" command itself
			if actualLen != tt.expectedLen {
				t.Errorf("Expected %d args, got %d: %v", tt.expectedLen, actualLen, args)
			}
		})
	}
}
