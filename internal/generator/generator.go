// Package generator handles project file generation from templates and registry.
package generator

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
	"github.com/maiko/sdbx/internal/secrets"
)

//go:embed templates/*
var TemplatesFS embed.FS

// Generator handles project generation
type Generator struct {
	Config    *config.Config
	OutputDir string
	Registry  *registry.Registry
}

// NewGenerator creates a new Generator with default registry
func NewGenerator(cfg *config.Config, outputDir string) *Generator {
	// Always create a default registry for service resolution
	reg, err := registry.NewWithDefaults()
	if err != nil {
		// Log error but continue - will retry in generateFromRegistry
		log.Printf("Warning: failed to create registry: %v (will retry during generation)", err)
		reg = nil
	}
	return &Generator{
		Config:    cfg,
		OutputDir: outputDir,
		Registry:  reg,
	}
}

// NewGeneratorWithRegistry creates a Generator with registry support
func NewGeneratorWithRegistry(cfg *config.Config, outputDir string, reg *registry.Registry) *Generator {
	return &Generator{
		Config:    cfg,
		OutputDir: outputDir,
		Registry:  reg,
	}
}

// TemplateData is passed to all templates
type TemplateData struct {
	Config  *config.Config
	Secrets map[string]string
}

// Generate creates all project files
func (g *Generator) Generate() error {
	// Create directory structure
	dirs := []string{
		"",
		"configs",
		"configs/traefik",
		"configs/traefik/dynamic",
		"configs/authelia",
		"configs/gluetun",
		"configs/homepage",
		"configs/qbittorrent",
		"configs/qbittorrent/qBittorrent",
		"configs/prowlarr",
		"configs/sonarr",
		"configs/radarr",
		"configs/lidarr",
		"configs/readarr",
		"configs/bazarr",
		"configs/recyclarr",
		"configs/unpackerr",
		"configs/plex",
		"configs/overseerr",
		"configs/wizarr",
		"configs/tautulli",
		"secrets",
	}

	// Add cloudflared config dir if using cloudflared mode
	if g.Config.Expose.Mode == config.ExposeModeCloudflared {
		dirs = append(dirs, "configs/cloudflared")
	}

	for _, dir := range dirs {
		path := filepath.Join(g.OutputDir, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate secrets
	secretsDir := filepath.Join(g.OutputDir, "secrets")
	if err := secrets.GenerateSecrets(secretsDir); err != nil {
		return fmt.Errorf("failed to generate secrets: %w", err)
	}

	// Write Cloudflare tunnel token if collected during wizard
	if g.Config.CloudflareTunnelToken != "" {
		tokenPath := filepath.Join(secretsDir, "cloudflared_tunnel_token.txt")
		if err := os.WriteFile(tokenPath, []byte(g.Config.CloudflareTunnelToken), 0600); err != nil {
			return fmt.Errorf("failed to write cloudflared token: %w", err)
		}
	}

	// Plex claim token is NOT written here - it's prompted during sdbx up

	// Read ALL generated secrets into the map
	secretsMap := make(map[string]string)
	for filename := range secrets.SecretFiles {
		val, err := secrets.ReadSecret(secretsDir, filename)
		if err == nil {
			secretsMap[filename] = val
		}
	}

	data := TemplateData{
		Config:  g.Config,
		Secrets: secretsMap,
	}

	// Use registry-based generation
	if err := g.generateFromRegistry(data); err != nil {
		return err
	}

	return nil
}

// generateFromRegistry uses the registry-based generators
func (g *Generator) generateFromRegistry(data TemplateData) error {
	ctx := context.Background()

	// Ensure we have a registry
	if g.Registry == nil {
		var err error
		g.Registry, err = registry.NewWithDefaults()
		if err != nil {
			return fmt.Errorf("failed to create registry: %w", err)
		}
	}

	// Resolve services from registry
	graph, err := g.Registry.Resolve(ctx, g.Config)
	if err != nil {
		return fmt.Errorf("failed to resolve services: %w", err)
	}

	// Generate compose.yaml using ComposeGenerator
	composeGen := NewComposeGenerator(g.Config, g.Registry, data.Secrets)
	composeFile, err := composeGen.Generate(graph)
	if err != nil {
		return fmt.Errorf("failed to generate compose file: %w", err)
	}

	composeYAML, err := composeFile.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to serialize compose file: %w", err)
	}

	composePath := filepath.Join(g.OutputDir, "compose.yaml")
	if err := os.WriteFile(composePath, composeYAML, 0o644); err != nil {
		return fmt.Errorf("failed to write compose.yaml: %w", err)
	}

	// Generate integration configs
	intGen := NewIntegrationsGenerator(g.Config, data.Secrets)

	// Homepage services
	homepageServices, err := intGen.GenerateHomepageServices(graph)
	if err != nil {
		return fmt.Errorf("failed to generate homepage services: %w", err)
	}
	if err := os.WriteFile(filepath.Join(g.OutputDir, "configs/homepage/services.yaml"), homepageServices, 0o644); err != nil {
		return fmt.Errorf("failed to write homepage services: %w", err)
	}

	// Traefik dynamic middlewares
	traefikDynamic, err := intGen.GenerateTraefikDynamic(graph)
	if err != nil {
		return fmt.Errorf("failed to generate traefik dynamic: %w", err)
	}
	if err := os.WriteFile(filepath.Join(g.OutputDir, "configs/traefik/dynamic/middlewares.yml"), traefikDynamic, 0o644); err != nil {
		return fmt.Errorf("failed to write traefik middlewares: %w", err)
	}

	// Cloudflared config (if enabled)
	if g.Config.Expose.Mode == config.ExposeModeCloudflared {
		cloudflaredConfig, err := intGen.GenerateCloudflaredConfig(graph)
		if err != nil {
			return fmt.Errorf("failed to generate cloudflared config: %w", err)
		}
		if err := os.WriteFile(filepath.Join(g.OutputDir, "configs/cloudflared/config.yml"), cloudflaredConfig, 0o644); err != nil {
			return fmt.Errorf("failed to write cloudflared config: %w", err)
		}
	}

	// .env file
	envContent, err := intGen.GenerateEnvFile(graph)
	if err != nil {
		return fmt.Errorf("failed to generate .env: %w", err)
	}
	if err := os.WriteFile(filepath.Join(g.OutputDir, ".env"), envContent, 0o644); err != nil {
		return fmt.Errorf("failed to write .env: %w", err)
	}

	// Static config files still use templates
	staticFiles := []struct {
		template string
		output   string
	}{
		{"sdbx.yaml.tmpl", ".sdbx.yaml"},
		{"gitignore.tmpl", ".gitignore"},
		{"traefik.yml.tmpl", "configs/traefik/traefik.yml"},
		{"authelia-configuration.yml.tmpl", "configs/authelia/configuration.yml"},
		{"authelia-users.yml.tmpl", "configs/authelia/users_database.yml"},
		{"homepage-settings.yaml.tmpl", "configs/homepage/settings.yaml"},
		{"homepage-docker.yaml.tmpl", "configs/homepage/docker.yaml"},
		{"gluetun.env.tmpl", "configs/gluetun/gluetun.env"},
		{"qbittorrent.conf.tmpl", "configs/qbittorrent/qBittorrent/qBittorrent.conf"},
	}

	for _, f := range staticFiles {
		if err := g.generateFile(f.template, f.output, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", f.output, err)
		}
	}

	return nil
}

// generateFile renders a template to a file
func (g *Generator) generateFile(templateName, outputPath string, data TemplateData) error {
	// Read template
	tmplContent, err := TemplatesFS.ReadFile("templates/" + templateName)
	if err != nil {
		return fmt.Errorf("template not found: %s: %w", templateName, err)
	}

	// Parse template
	tmpl, err := template.New(templateName).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output file
	outPath := filepath.Join(g.OutputDir, outputPath)
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// CreateDataDirs creates the data directory structure
func (g *Generator) CreateDataDirs() error {
	dirs := []string{
		g.Config.DownloadsPath,
		filepath.Join(g.Config.MediaPath, "movies"),
		filepath.Join(g.Config.MediaPath, "tv"),
		filepath.Join(g.Config.MediaPath, "music"),
		filepath.Join(g.Config.MediaPath, "books"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	return nil
}
