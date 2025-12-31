package registry

import (
	"context"
	"testing"
)

func TestEmbeddedSourceLoad(t *testing.T) {
	src := NewEmbeddedSource()

	ctx := context.Background()
	services, err := src.ListServices(ctx)
	if err != nil {
		t.Fatalf("failed to list services: %v", err)
	}

	if len(services) == 0 {
		t.Fatal("no services loaded from embedded source")
	}

	t.Logf("Loaded %d services from embedded source", len(services))

	// Verify expected core services
	expectedCore := []string{"traefik", "authelia", "sonarr", "radarr", "prowlarr", "qbittorrent", "plex", "homepage", "watchtower"}
	for _, name := range expectedCore {
		def, err := src.LoadService(ctx, name)
		if err != nil {
			t.Errorf("failed to load core service %s: %v", name, err)
			continue
		}
		if def.Conditions.RequireAddon {
			t.Errorf("core service %s should not require addon", name)
		}
	}

	// Verify expected addon services
	expectedAddons := []string{"overseerr", "wizarr", "tautulli", "lidarr", "readarr", "bazarr", "flaresolverr"}
	for _, name := range expectedAddons {
		def, err := src.LoadService(ctx, name)
		if err != nil {
			t.Errorf("failed to load addon service %s: %v", name, err)
			continue
		}
		if !def.Conditions.RequireAddon {
			t.Errorf("addon service %s should require addon", name)
		}
	}
}

func TestEmbeddedSourceCategories(t *testing.T) {
	src := NewEmbeddedSource()

	categories, err := src.GetServiceCategories()
	if err != nil {
		t.Fatalf("failed to get categories: %v", err)
	}

	if len(categories) == 0 {
		t.Fatal("no categories found")
	}

	t.Logf("Found %d categories: %v", len(categories), categories)
}

func TestEmbeddedSourceCoreAddons(t *testing.T) {
	src := NewEmbeddedSource()

	core, err := src.GetCoreServices()
	if err != nil {
		t.Fatalf("failed to get core services: %v", err)
	}

	addons, err := src.GetAddonServices()
	if err != nil {
		t.Fatalf("failed to get addon services: %v", err)
	}

	t.Logf("Core services: %d, Addon services: %d", len(core), len(addons))

	if len(core) == 0 {
		t.Error("expected some core services")
	}
	if len(addons) == 0 {
		t.Error("expected some addon services")
	}
}

func TestServiceDefinitionValidation(t *testing.T) {
	src := NewEmbeddedSource()
	validator := NewValidator()

	ctx := context.Background()
	services, err := src.Load(ctx)
	if err != nil {
		t.Fatalf("failed to load services: %v", err)
	}

	for _, def := range services {
		errors := validator.Validate(def)
		errorCount := 0
		for _, e := range errors {
			if e.Severity == "error" {
				errorCount++
				t.Errorf("validation error in %s: %s - %s", def.Metadata.Name, e.Field, e.Message)
			}
		}
	}
}

func TestSonarrServiceDefinition(t *testing.T) {
	src := NewEmbeddedSource()
	ctx := context.Background()

	def, err := src.LoadService(ctx, "sonarr")
	if err != nil {
		t.Fatalf("failed to load sonarr: %v", err)
	}

	// Verify metadata
	if def.Metadata.Name != "sonarr" {
		t.Errorf("expected name sonarr, got %s", def.Metadata.Name)
	}
	if def.Metadata.Category != CategoryMedia {
		t.Errorf("expected category media, got %s", def.Metadata.Category)
	}

	// Verify routing
	if !def.Routing.Enabled {
		t.Error("expected routing to be enabled")
	}
	if def.Routing.Port != 8989 {
		t.Errorf("expected port 8989, got %d", def.Routing.Port)
	}
	if !def.Routing.Auth.Required {
		t.Error("expected auth to be required")
	}

	// Verify secrets
	if len(def.Secrets) == 0 {
		t.Error("expected sonarr to have secrets defined")
	}

	// Verify integrations
	if def.Integrations.Homepage == nil || !def.Integrations.Homepage.Enabled {
		t.Error("expected homepage integration to be enabled")
	}
}
