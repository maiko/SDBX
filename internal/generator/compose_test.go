package generator

import (
	"strings"
	"testing"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
)

// TestLabelTransferWithVPN verifies that Traefik labels are transferred from
// qbittorrent to gluetun when VPN is enabled
func TestLabelTransferWithVPN(t *testing.T) {
	// Setup config with VPN enabled
	cfg := &config.Config{
		VPNEnabled: true,
		Domain:     "example.com",
		Routing: config.RoutingConfig{
			Strategy:   config.RoutingStrategySubdomain,
			BaseDomain: "sdbx",
		},
		Timezone: "UTC",
		PUID:     1000,
		PGID:     1000,
		Umask:    "2",
		Expose: config.ExposeConfig{
			Mode: config.ExposeModeCloudflared,
		},
	}

	// Create compose file with qbittorrent and gluetun services
	compose := &ComposeFile{
		Name: "sdbx",
		Services: map[string]ComposeService{
			"gluetun": {
				Image:         "qmcgaw/gluetun:latest",
				ContainerName: "sdbx-gluetun",
				Networks:      []string{"proxy"},
				Labels: []string{
					"com.centurylinklabs.watchtower.enable=true",
				},
			},
			"qbittorrent": {
				Image:         "linuxserver/qbittorrent:latest",
				ContainerName: "sdbx-qbittorrent",
				NetworkMode:   "service:gluetun",
				Labels: []string{
					"com.centurylinklabs.watchtower.enable=true",
					"traefik.enable=true",
					"traefik.http.routers.qbittorrent.rule=Host(`qbt.example.com`)",
					"traefik.http.routers.qbittorrent.entrypoints=web",
					"traefik.http.services.qbittorrent.loadbalancer.server.port=8080",
				},
			},
		},
	}

	// Create generator and transfer labels
	gen := NewComposeGenerator(cfg, nil, nil)
	gen.transferLabelsForNetworkSharing(compose)

	// Verify qBittorrent has network_mode: service:gluetun
	qbt := compose.Services["qbittorrent"]
	if qbt.NetworkMode != "service:gluetun" {
		t.Errorf("Expected network_mode 'service:gluetun', got '%s'", qbt.NetworkMode)
	}

	// Verify gluetun has qBittorrent's Traefik labels
	gluetun := compose.Services["gluetun"]
	hasTraefikEnable := false
	hasQbtRouter := false
	hasQbtService := false

	for _, label := range gluetun.Labels {
		if label == "traefik.enable=true" {
			hasTraefikEnable = true
		}
		if strings.Contains(label, "traefik.http.routers.qbittorrent.rule") {
			hasQbtRouter = true
		}
		if strings.Contains(label, "traefik.http.services.qbittorrent.loadbalancer.server.port") {
			hasQbtService = true
		}
	}

	if !hasTraefikEnable {
		t.Error("gluetun should have traefik.enable=true label")
	}
	if !hasQbtRouter {
		t.Error("gluetun should have qbittorrent router label")
	}
	if !hasQbtService {
		t.Error("gluetun should have qbittorrent service label")
	}

	// Verify qBittorrent does NOT have Traefik labels anymore
	for _, label := range qbt.Labels {
		if strings.HasPrefix(label, "traefik.") {
			t.Errorf("qbittorrent should not have traefik label: %s", label)
		}
	}

	// Verify qBittorrent still has watchtower label
	hasWatchtower := false
	for _, label := range qbt.Labels {
		if label == "com.centurylinklabs.watchtower.enable=true" {
			hasWatchtower = true
			break
		}
	}
	if !hasWatchtower {
		t.Error("qbittorrent should still have watchtower label")
	}
}

// TestNoLabelTransferWithoutVPN verifies that labels stay on qbittorrent when
// VPN is disabled (no network_mode: service:X)
func TestNoLabelTransferWithoutVPN(t *testing.T) {
	cfg := &config.Config{
		VPNEnabled: false,
		Domain:     "example.com",
		Routing: config.RoutingConfig{
			Strategy:   config.RoutingStrategySubdomain,
			BaseDomain: "sdbx",
		},
	}

	// Create compose file with qbittorrent using normal networking
	compose := &ComposeFile{
		Name: "sdbx",
		Services: map[string]ComposeService{
			"qbittorrent": {
				Image:         "linuxserver/qbittorrent:latest",
				ContainerName: "sdbx-qbittorrent",
				Networks:      []string{"proxy"},
				Labels: []string{
					"com.centurylinklabs.watchtower.enable=true",
					"traefik.enable=true",
					"traefik.http.routers.qbittorrent.rule=Host(`qbt.example.com`)",
					"traefik.http.routers.qbittorrent.entrypoints=web",
					"traefik.http.services.qbittorrent.loadbalancer.server.port=8080",
				},
			},
		},
	}

	// Create generator and attempt label transfer
	gen := NewComposeGenerator(cfg, nil, nil)
	gen.transferLabelsForNetworkSharing(compose)

	// Verify qBittorrent still has its Traefik labels
	qbt := compose.Services["qbittorrent"]
	hasTraefikEnable := false
	hasQbtRouter := false

	for _, label := range qbt.Labels {
		if label == "traefik.enable=true" {
			hasTraefikEnable = true
		}
		if strings.Contains(label, "traefik.http.routers.qbittorrent.rule") {
			hasQbtRouter = true
		}
	}

	if !hasTraefikEnable {
		t.Error("qbittorrent should have traefik.enable=true label")
	}
	if !hasQbtRouter {
		t.Error("qbittorrent should have qbittorrent router label")
	}
}

// TestNonTraefikLabelsNotTransferred verifies that non-Traefik labels
// (like watchtower) remain on the original service and are not transferred
func TestNonTraefikLabelsNotTransferred(t *testing.T) {
	cfg := &config.Config{
		VPNEnabled: true,
		Domain:     "example.com",
	}

	compose := &ComposeFile{
		Name: "sdbx",
		Services: map[string]ComposeService{
			"gluetun": {
				Image:         "qmcgaw/gluetun:latest",
				ContainerName: "sdbx-gluetun",
				Networks:      []string{"proxy"},
				Labels: []string{
					"com.centurylinklabs.watchtower.enable=true",
				},
			},
			"qbittorrent": {
				Image:         "linuxserver/qbittorrent:latest",
				ContainerName: "sdbx-qbittorrent",
				NetworkMode:   "service:gluetun",
				Labels: []string{
					"com.centurylinklabs.watchtower.enable=true",
					"custom.label=value",
					"traefik.enable=true",
				},
			},
		},
	}

	gen := NewComposeGenerator(cfg, nil, nil)
	gen.transferLabelsForNetworkSharing(compose)

	qbt := compose.Services["qbittorrent"]
	gluetun := compose.Services["gluetun"]

	// Verify watchtower label stays on qBittorrent
	hasWatchtower := false
	hasCustomLabel := false
	for _, label := range qbt.Labels {
		if label == "com.centurylinklabs.watchtower.enable=true" {
			hasWatchtower = true
		}
		if label == "custom.label=value" {
			hasCustomLabel = true
		}
	}

	if !hasWatchtower {
		t.Error("qbittorrent should still have watchtower label")
	}
	if !hasCustomLabel {
		t.Error("qbittorrent should still have custom label")
	}

	// Verify gluetun did NOT get qbittorrent's non-Traefik labels
	gluetunLabelCount := 0
	for _, label := range gluetun.Labels {
		if label == "custom.label=value" {
			t.Error("gluetun should not have qbittorrent's custom label")
		}
		gluetunLabelCount++
	}

	// Gluetun should have its original watchtower label + traefik.enable
	// (at most 2 labels from qbittorrent)
	if gluetunLabelCount > 2 {
		t.Errorf("gluetun has unexpected number of labels: %d", gluetunLabelCount)
	}
}

// TestLabelTransferMissingHost verifies graceful handling when host service
// doesn't exist (edge case)
func TestLabelTransferMissingHost(t *testing.T) {
	cfg := &config.Config{
		VPNEnabled: true,
		Domain:     "example.com",
	}

	compose := &ComposeFile{
		Name: "sdbx",
		Services: map[string]ComposeService{
			"qbittorrent": {
				Image:         "linuxserver/qbittorrent:latest",
				ContainerName: "sdbx-qbittorrent",
				NetworkMode:   "service:nonexistent",
				Labels: []string{
					"traefik.enable=true",
					"traefik.http.routers.qbittorrent.rule=Host(`qbt.example.com`)",
				},
			},
		},
	}

	gen := NewComposeGenerator(cfg, nil, nil)

	// Should not panic when host service doesn't exist
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("transferLabelsForNetworkSharing panicked: %v", r)
		}
	}()

	gen.transferLabelsForNetworkSharing(compose)

	// Verify qbittorrent still has its labels (not transferred since host missing)
	qbt := compose.Services["qbittorrent"]
	hasTraefikLabel := false
	for _, label := range qbt.Labels {
		if strings.HasPrefix(label, "traefik.") {
			hasTraefikLabel = true
			break
		}
	}

	if !hasTraefikLabel {
		t.Error("qbittorrent should still have traefik labels when host service is missing")
	}
}

// TestMultipleServicesSharing verifies that multiple services can share the
// same network host (future-proofing)
func TestMultipleServicesSharing(t *testing.T) {
	cfg := &config.Config{
		VPNEnabled: true,
		Domain:     "example.com",
	}

	compose := &ComposeFile{
		Name: "sdbx",
		Services: map[string]ComposeService{
			"gluetun": {
				Image:         "qmcgaw/gluetun:latest",
				ContainerName: "sdbx-gluetun",
				Networks:      []string{"proxy"},
				Labels:        []string{},
			},
			"qbittorrent": {
				Image:       "linuxserver/qbittorrent:latest",
				NetworkMode: "service:gluetun",
				Labels: []string{
					"traefik.enable=true",
					"traefik.http.routers.qbittorrent.rule=Host(`qbt.example.com`)",
				},
			},
			"service2": {
				Image:       "example/service2:latest",
				NetworkMode: "service:gluetun",
				Labels: []string{
					"traefik.enable=true",
					"traefik.http.routers.service2.rule=Host(`svc2.example.com`)",
				},
			},
		},
	}

	gen := NewComposeGenerator(cfg, nil, nil)
	gen.transferLabelsForNetworkSharing(compose)

	gluetun := compose.Services["gluetun"]

	// Verify gluetun has labels from BOTH services
	hasQbtRouter := false
	hasSvc2Router := false

	for _, label := range gluetun.Labels {
		if strings.Contains(label, "traefik.http.routers.qbittorrent.rule") {
			hasQbtRouter = true
		}
		if strings.Contains(label, "traefik.http.routers.service2.rule") {
			hasSvc2Router = true
		}
	}

	if !hasQbtRouter {
		t.Error("gluetun should have qbittorrent router label")
	}
	if !hasSvc2Router {
		t.Error("gluetun should have service2 router label")
	}

	// Verify both services no longer have Traefik labels
	qbt := compose.Services["qbittorrent"]
	svc2 := compose.Services["service2"]

	for _, label := range qbt.Labels {
		if strings.HasPrefix(label, "traefik.") {
			t.Errorf("qbittorrent should not have traefik label: %s", label)
		}
	}

	for _, label := range svc2.Labels {
		if strings.HasPrefix(label, "traefik.") {
			t.Errorf("service2 should not have traefik label: %s", label)
		}
	}
}

// TestGenerateServiceExtraProperties verifies ShmSize, Sysctls, and GPU deploy
func TestGenerateServiceExtraProperties(t *testing.T) {
	cfg := &config.Config{
		Domain: "example.com",
		Routing: config.RoutingConfig{
			Strategy:   config.RoutingStrategySubdomain,
			BaseDomain: "sdbx",
		},
		Expose: config.ExposeConfig{
			Mode: config.ExposeModeCloudflared,
		},
	}

	gen := NewComposeGenerator(cfg, nil, nil)

	def := &registry.ServiceDefinition{
		Metadata: registry.ServiceMetadata{
			Name: "plex",
		},
		Spec: registry.ServiceSpec{
			Image: registry.ImageSpec{
				Repository: "linuxserver/plex",
				Tag:        "latest",
			},
			Container: registry.ContainerSpec{
				NameTemplate: "sdbx-plex",
				Restart:      "unless-stopped",
				ShmSize:      "2g",
				Sysctls: map[string]string{
					"net.ipv4.conf.all.src_valid_mark": "1",
				},
				GPUEnabled: true,
			},
		},
	}

	svc := gen.generateService(def)

	// Verify ShmSize
	if svc.ShmSize != "2g" {
		t.Errorf("ShmSize = %q, want '2g'", svc.ShmSize)
	}

	// Verify Sysctls
	if len(svc.Sysctls) != 1 {
		t.Errorf("expected 1 sysctl, got %d", len(svc.Sysctls))
	}
	if val, ok := svc.Sysctls["net.ipv4.conf.all.src_valid_mark"]; !ok || val != "1" {
		t.Error("Sysctls not set correctly")
	}

	// Verify GPU deploy
	if svc.Deploy == nil {
		t.Fatal("Deploy should not be nil when GPUEnabled is true")
	}
	if svc.Deploy.Resources == nil {
		t.Fatal("Deploy.Resources should not be nil")
	}
	if svc.Deploy.Resources.Reservations == nil {
		t.Fatal("Deploy.Resources.Reservations should not be nil")
	}
	if len(svc.Deploy.Resources.Reservations.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(svc.Deploy.Resources.Reservations.Devices))
	}

	device := svc.Deploy.Resources.Reservations.Devices[0]
	if device.Driver != "nvidia" {
		t.Errorf("device driver = %q, want 'nvidia'", device.Driver)
	}
	if device.Count != "all" {
		t.Errorf("device count = %q, want 'all'", device.Count)
	}
	if len(device.Capabilities) != 1 || device.Capabilities[0] != "gpu" {
		t.Errorf("device capabilities = %v, want [gpu]", device.Capabilities)
	}
}

// TestGenerateServiceNoExtraProperties verifies defaults when extra properties are not set
func TestGenerateServiceNoExtraProperties(t *testing.T) {
	cfg := &config.Config{
		Domain: "example.com",
		Routing: config.RoutingConfig{
			Strategy:   config.RoutingStrategySubdomain,
			BaseDomain: "sdbx",
		},
		Expose: config.ExposeConfig{
			Mode: config.ExposeModeCloudflared,
		},
	}

	gen := NewComposeGenerator(cfg, nil, nil)

	def := &registry.ServiceDefinition{
		Metadata: registry.ServiceMetadata{
			Name: "sonarr",
		},
		Spec: registry.ServiceSpec{
			Image: registry.ImageSpec{
				Repository: "linuxserver/sonarr",
				Tag:        "latest",
			},
			Container: registry.ContainerSpec{
				NameTemplate: "sdbx-sonarr",
			},
		},
	}

	svc := gen.generateService(def)

	if svc.ShmSize != "" {
		t.Errorf("ShmSize should be empty, got %q", svc.ShmSize)
	}
	if svc.Sysctls != nil {
		t.Error("Sysctls should be nil")
	}
	if svc.Deploy != nil {
		t.Error("Deploy should be nil when GPUEnabled is false")
	}
}

// TestCustomLabelsRendered verifies custom Traefik labels are added
func TestCustomLabelsRendered(t *testing.T) {
	cfg := &config.Config{
		Domain: "example.com",
		Routing: config.RoutingConfig{
			Strategy:   config.RoutingStrategySubdomain,
			BaseDomain: "sdbx",
		},
		Expose: config.ExposeConfig{
			Mode: config.ExposeModeCloudflared,
		},
	}

	gen := NewComposeGenerator(cfg, nil, nil)

	def := &registry.ServiceDefinition{
		Metadata: registry.ServiceMetadata{
			Name: "myservice",
		},
		Spec: registry.ServiceSpec{
			Image: registry.ImageSpec{
				Repository: "example/myservice",
				Tag:        "latest",
			},
			Container: registry.ContainerSpec{
				NameTemplate: "sdbx-myservice",
			},
		},
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Port:      8080,
			Subdomain: "myservice",
			Traefik: registry.TraefikConfig{
				CustomLabels: map[string]string{
					"traefik.http.routers.myservice.tls.certresolver": "letsencrypt",
					"traefik.http.services.myservice.loadbalancer.sticky.cookie": "true",
				},
			},
		},
	}

	ctx := TemplateContext{Config: cfg}
	labels := gen.buildTraefikLabels(def, ctx)

	foundCertResolver := false
	foundSticky := false
	for _, label := range labels {
		if label == "traefik.http.routers.myservice.tls.certresolver=letsencrypt" {
			foundCertResolver = true
		}
		if label == "traefik.http.services.myservice.loadbalancer.sticky.cookie=true" {
			foundSticky = true
		}
	}

	if !foundCertResolver {
		t.Error("expected custom label for certresolver")
	}
	if !foundSticky {
		t.Error("expected custom label for sticky cookie")
	}
}

// TestCustomLabelsEmpty verifies no custom labels when none defined
func TestCustomLabelsEmpty(t *testing.T) {
	cfg := &config.Config{
		Domain: "example.com",
		Routing: config.RoutingConfig{
			Strategy:   config.RoutingStrategySubdomain,
			BaseDomain: "sdbx",
		},
		Expose: config.ExposeConfig{
			Mode: config.ExposeModeCloudflared,
		},
	}

	gen := NewComposeGenerator(cfg, nil, nil)

	def := &registry.ServiceDefinition{
		Metadata: registry.ServiceMetadata{
			Name: "sonarr",
		},
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Port:      8989,
			Subdomain: "sonarr",
		},
	}

	ctx := TemplateContext{Config: cfg}
	labels := gen.buildTraefikLabels(def, ctx)

	// Should have standard traefik labels but no custom ones
	hasTraefikEnable := false
	for _, label := range labels {
		if label == "traefik.enable=true" {
			hasTraefikEnable = true
		}
	}

	if !hasTraefikEnable {
		t.Error("should have traefik.enable=true")
	}
}

// TestEvalTemplateWarnings verifies evalTemplate returns fallback on bad templates
func TestEvalTemplateWarnings(t *testing.T) {
	cfg := &config.Config{}
	gen := NewComposeGenerator(cfg, nil, nil)
	ctx := TemplateContext{Config: cfg}

	// Invalid template syntax should return the original string
	result := gen.evalTemplate("{{ .Invalid }", ctx)
	if result != "{{ .Invalid }" {
		t.Errorf("expected original string for parse error, got %q", result)
	}

	// Template referencing missing field should return original
	result = gen.evalTemplate("{{ .NonExistent.Field }}", ctx)
	if result != "{{ .NonExistent.Field }}" {
		t.Errorf("expected original string for execute error, got %q", result)
	}

	// Valid template should work normally
	result = gen.evalTemplate("hello-world", ctx)
	if result != "hello-world" {
		t.Errorf("expected 'hello-world', got %q", result)
	}
}
