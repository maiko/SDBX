// Package docker provides Docker Compose operations for sdbx.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	stateRunning  = "running"
	healthHealthy = "healthy"
)

// Service represents a Docker Compose service
type Service struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Health   string `json:"health,omitempty"`
	Ports    string `json:"ports,omitempty"`
	Image    string `json:"image"`
	Running  bool   `json:"running"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// Compose handles Docker Compose operations
type Compose struct {
	ProjectDir  string
	ComposeFile string
	ProjectName string
}

// NewCompose creates a new Compose instance
func NewCompose(projectDir string) *Compose {
	return &Compose{
		ProjectDir:  projectDir,
		ComposeFile: "compose.yaml",
		ProjectName: "sdbx",
	}
}

// run executes a docker compose command
func (c *Compose) run(ctx context.Context, args ...string) (string, error) {
	cmdArgs := []string{"compose", "-f", c.ComposeFile, "-p", c.ProjectName}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Dir = c.ProjectDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// Up starts all services
func (c *Compose) Up(ctx context.Context) error {
	_, err := c.run(ctx, "up", "-d", "--remove-orphans")
	return err
}

// Down stops all services
func (c *Compose) Down(ctx context.Context) error {
	_, err := c.run(ctx, "down")
	return err
}

// Start starts a specific service or all services
func (c *Compose) Start(ctx context.Context, service string) error {
	if service == "" {
		_, err := c.run(ctx, "start")
		return err
	}
	_, err := c.run(ctx, "start", service)
	return err
}

// Stop stops a specific service or all services
func (c *Compose) Stop(ctx context.Context, service string) error {
	if service == "" {
		_, err := c.run(ctx, "stop")
		return err
	}
	_, err := c.run(ctx, "stop", service)
	return err
}

// Restart restarts a specific service or all services
func (c *Compose) Restart(ctx context.Context, service string) error {
	if service == "" {
		_, err := c.run(ctx, "restart")
		return err
	}
	_, err := c.run(ctx, "restart", service)
	return err
}

// Pull pulls images for all services
func (c *Compose) Pull(ctx context.Context) error {
	_, err := c.run(ctx, "pull")
	return err
}

// Logs returns logs for a service
func (c *Compose) Logs(ctx context.Context, service string, lines int, follow bool) (string, error) {
	args := []string{"logs"}
	if lines > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", lines))
	}
	if follow {
		args = append(args, "-f")
	}
	if service != "" {
		args = append(args, service)
	}
	return c.run(ctx, args...)
}

// LogsStream returns a streaming reader for service logs
func (c *Compose) LogsStream(ctx context.Context, service string, lines int) (*exec.Cmd, error) {
	cmdArgs := []string{"compose", "-f", c.ComposeFile, "-p", c.ProjectName, "logs"}
	if lines > 0 {
		cmdArgs = append(cmdArgs, "--tail", fmt.Sprintf("%d", lines))
	}
	cmdArgs = append(cmdArgs, "-f", service)

	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	cmd.Dir = c.ProjectDir

	return cmd, nil
}

// PS returns the status of all services
func (c *Compose) PS(ctx context.Context) ([]Service, error) {
	output, err := c.run(ctx, "ps", "--format", "json")
	if err != nil {
		return nil, err
	}

	var services []Service
	// docker compose ps --format json outputs one JSON object per line
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
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
			continue
		}
		services = append(services, Service{
			Name:     svc.Name,
			Status:   svc.State,
			Health:   svc.Health,
			Image:    svc.Image,
			Ports:    svc.Ports,
			Running:  svc.State == stateRunning,
			ExitCode: svc.ExitCode,
		})
	}

	return services, nil
}

// Exec executes a command in a running container
func (c *Compose) Exec(ctx context.Context, service string, cmd ...string) (string, error) {
	args := []string{"exec", "-T", service}
	args = append(args, cmd...)
	return c.run(ctx, args...)
}

// IsHealthy checks if a service is healthy
func (c *Compose) IsHealthy(ctx context.Context, service string) (bool, error) {
	services, err := c.PS(ctx)
	if err != nil {
		return false, err
	}

	for _, svc := range services {
		if strings.Contains(svc.Name, service) {
			return svc.Running && (svc.Health == "" || svc.Health == healthHealthy), nil
		}
	}

	return false, fmt.Errorf("service %s not found", service)
}

// WaitHealthy waits for a service to become healthy
func (c *Compose) WaitHealthy(ctx context.Context, service string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		healthy, err := c.IsHealthy(ctx, service)
		if err == nil && healthy {
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout waiting for %s to become healthy", service)
}
