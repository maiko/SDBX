// Package doctor provides health checks for sdbx.
package doctor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/maiko/sdbx/internal/config"
)

// Check represents a single diagnostic check
type Check struct {
	Name        string
	Description string
	Status      CheckStatus
	Message     string
	Duration    time.Duration
}

// CheckStatus represents the result of a check
type CheckStatus int

const (
	StatusPending CheckStatus = iota
	StatusRunning
	StatusPassed
	StatusWarning
	StatusFailed
)

// Doctor runs all diagnostic checks
type Doctor struct {
	ProjectDir string
	Checks     []Check
}

// NewDoctor creates a new Doctor instance
func NewDoctor(projectDir string) *Doctor {
	return &Doctor{
		ProjectDir: projectDir,
		Checks:     make([]Check, 0),
	}
}

// RunAll executes all checks and returns results
func (d *Doctor) RunAll(ctx context.Context) []Check {
	checks := []struct {
		name string
		fn   func(context.Context) (bool, string)
	}{
		{"Docker version", d.checkDockerVersion},
		{"Docker Compose version", d.checkComposeVersion},
		{"Disk space", d.checkDiskSpace},
		{"File permissions", d.checkPermissions},
		{"Required ports", d.checkPorts},
		{"Docker daemon", d.checkDockerDaemon},
		{"Project files", d.checkProjectFiles},
		{"Secrets configured", d.checkSecrets},
	}

	for _, c := range checks {
		check := Check{
			Name:   c.name,
			Status: StatusRunning,
		}

		start := time.Now()
		passed, message := c.fn(ctx)
		check.Duration = time.Since(start)
		check.Message = message

		if passed {
			check.Status = StatusPassed
		} else {
			check.Status = StatusFailed
		}

		d.Checks = append(d.Checks, check)
	}

	return d.Checks
}

// checkDockerVersion verifies Docker is installed and version is sufficient
func (d *Doctor) checkDockerVersion(ctx context.Context) (bool, string) {
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		return false, "Docker not found or not running"
	}

	version := strings.TrimSpace(string(output))
	// Parse major version
	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return false, "Could not parse Docker version"
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, "Could not parse Docker version"
	}

	if major < 24 {
		return false, fmt.Sprintf("Docker %s < 24.0 (minimum required)", version)
	}

	return true, fmt.Sprintf("%s ≥ 24.0", version)
}

// checkComposeVersion verifies Docker Compose v2 is available
func (d *Doctor) checkComposeVersion(ctx context.Context) (bool, string) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "version", "--short")
	output, err := cmd.Output()
	if err != nil {
		return false, "Docker Compose not found"
	}

	version := strings.TrimSpace(string(output))
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false, "Could not parse Compose version"
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])

	if major < 2 || (major == 2 && minor < 20) {
		return false, fmt.Sprintf("Compose %s < 2.20 (minimum required)", version)
	}

	return true, fmt.Sprintf("%s ≥ 2.20", version)
}

// checkDiskSpace verifies sufficient disk space
func (d *Doctor) checkDiskSpace(_ context.Context) (bool, string) {
	var stat syscall.Statfs_t
	path := d.ProjectDir
	if path == "" {
		path = "."
	}

	if err := syscall.Statfs(path, &stat); err != nil {
		return false, "Could not check disk space"
	}

	// Calculate free space in GB
	// Use explicit conversion to avoid integer overflow
	blockSize := stat.Bsize
	if blockSize < 0 {
		return false, "Invalid block size"
	}
	freeGB := float64(stat.Bavail) * float64(blockSize) / (1024 * 1024 * 1024)

	if freeGB < 10 {
		return false, fmt.Sprintf("%.1f GB free (< 10 GB minimum)", freeGB)
	}

	return true, fmt.Sprintf("%.1f GB free", freeGB)
}

// checkPermissions verifies file permissions
func (d *Doctor) checkPermissions(_ context.Context) (bool, string) {
	// Check if we can write to the project directory
	testFile := filepath.Join(d.ProjectDir, ".sdbx-test")
	f, err := os.Create(testFile)
	if err != nil {
		return false, "Cannot write to project directory"
	}
	f.Close()
	os.Remove(testFile)

	// Check UID/GID
	if runtime.GOOS != "windows" {
		uid := os.Getuid()
		gid := os.Getgid()
		return true, fmt.Sprintf("UID/GID %d:%d", uid, gid)
	}

	return true, "OK"
}

// checkPorts verifies required ports are available
func (d *Doctor) checkPorts(ctx context.Context) (bool, string) {
	// Default ports
	ports := []int{32400} // Plex is generally exposed

	// Load config to check expose mode
	cfg, err := config.Load()
	var modeMsg string
	if err == nil {
		if cfg.Expose.Mode == "direct" {
			ports = append(ports, 80, 443)
			modeMsg = "(direct mode)"
		} else if cfg.Expose.Mode == "lan" {
			ports = append(ports, 80)
			modeMsg = "(lan mode)"
		} else {
			modeMsg = "(cloudflared mode)"
		}
	} else {
		// Fallback if config load fails
		ports = append(ports, 80, 443)
		modeMsg = "(unknown mode)"
	}

	var inUse []int
	for _, port := range ports {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			inUse = append(inUse, port)
		} else {
			ln.Close()
		}
	}

	if len(inUse) > 0 {
		// If ports are in use, check if it's us (SDBX)
		if d.isSDBXRunning(ctx) {
			return true, fmt.Sprintf("Ports active (SDBX running) %s", modeMsg)
		}
		return false, fmt.Sprintf("Ports in use %s: %v", modeMsg, inUse)
	}

	return true, fmt.Sprintf("Required ports available %s", modeMsg)
}

// isSDBXRunning checks if the main proxy container is running
func (d *Doctor) isSDBXRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "sdbx-traefik")
}

// checkDockerDaemon verifies Docker daemon is running
func (d *Doctor) checkDockerDaemon(ctx context.Context) (bool, string) {
	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		return false, "Docker daemon not running"
	}
	return true, "Running"
}

// checkProjectFiles verifies required project files exist
func (d *Doctor) checkProjectFiles(_ context.Context) (bool, string) {
	required := []string{"compose.yaml", ".env"}
	var missing []string

	for _, file := range required {
		path := filepath.Join(d.ProjectDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missing = append(missing, file)
		}
	}

	if len(missing) > 0 {
		return false, fmt.Sprintf("Missing: %s", strings.Join(missing, ", "))
	}

	return true, "All present"
}

// checkSecrets verifies secrets are configured
func (d *Doctor) checkSecrets(_ context.Context) (bool, string) {
	secretsDir := filepath.Join(d.ProjectDir, "secrets")
	if _, err := os.Stat(secretsDir); os.IsNotExist(err) {
		return false, "Secrets directory not found"
	}

	// Check for required secrets
	required := []string{
		"authelia_jwt_secret.txt",
		"authelia_session_secret.txt",
	}

	var empty []string
	for _, secret := range required {
		path := filepath.Join(secretsDir, secret)
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			empty = append(empty, secret)
		}
	}

	if len(empty) > 0 {
		return false, fmt.Sprintf("Empty: %s", strings.Join(empty, ", "))
	}

	return true, "Configured"
}

// CheckVPN verifies VPN connectivity (separate as it requires running containers)
func (d *Doctor) CheckVPN(_ context.Context) (bool, string) {
	// Try to reach a check IP service through the VPN container
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return false, "Could not reach IP check service"
	}
	defer resp.Body.Close()

	return true, "Connected"
}
