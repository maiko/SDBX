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

	// Embedded source should only have 7 core services (including sdbx-webui)
	if len(services) != 7 {
		t.Errorf("expected 7 core services in embedded, got %d", len(services))
	}

	t.Logf("Loaded %d services from embedded source", len(services))

	// Verify all 7 expected core services are present
	expectedCore := []string{"traefik", "authelia", "qbittorrent", "plex", "gluetun", "cloudflared", "sdbx-webui"}
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

	// Verify that addons are NOT in embedded source (they're in Git source only)
	addonExamples := []string{"sonarr", "radarr", "prowlarr", "homepage"}
	for _, name := range addonExamples {
		_, err := src.LoadService(ctx, name)
		if err == nil {
			t.Errorf("addon service %s should NOT be in embedded source", name)
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

	// Embedded source should have exactly 7 core services (including sdbx-webui)
	if len(core) != 7 {
		t.Errorf("expected 7 core services in embedded, got %d", len(core))
	}

	// Embedded source should have NO addons (they're in Git source only)
	if len(addons) != 0 {
		t.Errorf("expected 0 addon services in embedded, got %d", len(addons))
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

func TestTraefikServiceDefinition(t *testing.T) {
	src := NewEmbeddedSource()
	ctx := context.Background()

	def, err := src.LoadService(ctx, "traefik")
	if err != nil {
		t.Fatalf("failed to load traefik: %v", err)
	}

	// Verify metadata
	if def.Metadata.Name != "traefik" {
		t.Errorf("expected name traefik, got %s", def.Metadata.Name)
	}
	if def.Metadata.Category != CategoryNetworking {
		t.Errorf("expected category networking, got %s", def.Metadata.Category)
	}

	// Verify traefik is core (not addon)
	if def.Conditions.RequireAddon {
		t.Error("expected traefik to be core (always enabled)")
	}
	if !def.Conditions.Always {
		t.Error("expected traefik to have always condition set")
	}

	// Verify routing (traefik itself doesn't need routing - it IS the reverse proxy)
	if def.Routing.Enabled {
		t.Error("expected traefik routing to be disabled (it's the reverse proxy)")
	}

	// Verify watchtower integration
	if def.Integrations.Watchtower == nil || !def.Integrations.Watchtower.Enabled {
		t.Error("expected watchtower integration to be enabled")
	}
}
