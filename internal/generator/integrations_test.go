package generator

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
)

// makeTestGraph creates a ResolutionGraph with the given services for testing.
func makeTestGraph(services ...*registry.ResolvedService) *registry.ResolutionGraph {
	graph := &registry.ResolutionGraph{
		Services: make(map[string]*registry.ResolvedService),
	}
	for _, svc := range services {
		graph.Services[svc.Name] = svc
		graph.Order = append(graph.Order, svc.Name)
	}
	return graph
}

func makeResolvedService(name string, def *registry.ServiceDefinition) *registry.ResolvedService {
	return &registry.ResolvedService{
		Name:            name,
		Enabled:         true,
		Definition:      def,
		FinalDefinition: def,
	}
}

// --- GenerateHomepageServices ---

func TestGenerateHomepageServicesEmpty(t *testing.T) {
	gen := NewIntegrationsGenerator(config.DefaultConfig(), nil)
	graph := makeTestGraph()

	data, err := gen.GenerateHomepageServices(graph)
	if err != nil {
		t.Fatalf("GenerateHomepageServices() error: %v", err)
	}

	// Empty graph should produce valid YAML
	if len(data) == 0 {
		t.Error("expected non-empty output even for empty graph")
	}
}

func TestGenerateHomepageServicesWithServices(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Routing.Strategy = config.RoutingStrategySubdomain

	gen := NewIntegrationsGenerator(cfg, nil)

	sonarr := makeResolvedService("sonarr", &registry.ServiceDefinition{
		Metadata: registry.ServiceMetadata{Name: "sonarr"},
		Spec:     registry.ServiceSpec{},
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Subdomain: "sonarr",
			Port:      8989,
		},
		Conditions: registry.Conditions{Always: true},
		Integrations: registry.Integrations{
			Homepage: &registry.HomepageIntegration{
				Enabled:     true,
				Group:       "Media",
				Icon:        "sonarr.png",
				Description: "TV Shows",
			},
		},
	})

	radarr := makeResolvedService("radarr", &registry.ServiceDefinition{
		Metadata: registry.ServiceMetadata{Name: "radarr"},
		Spec:     registry.ServiceSpec{},
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Subdomain: "radarr",
			Port:      7878,
		},
		Conditions: registry.Conditions{Always: true},
		Integrations: registry.Integrations{
			Homepage: &registry.HomepageIntegration{
				Enabled:     true,
				Group:       "Media",
				Icon:        "radarr.png",
				Description: "Movies",
			},
		},
	})

	graph := makeTestGraph(sonarr, radarr)

	data, err := gen.GenerateHomepageServices(graph)
	if err != nil {
		t.Fatalf("GenerateHomepageServices() error: %v", err)
	}

	content := string(data)

	// Should contain both services
	if !strings.Contains(content, "sonarr") {
		t.Error("output should contain sonarr")
	}
	if !strings.Contains(content, "radarr") {
		t.Error("output should contain radarr")
	}

	// Should be valid YAML
	var parsed interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Errorf("output is not valid YAML: %v", err)
	}
}

func TestGenerateHomepageServicesSkipsDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"

	gen := NewIntegrationsGenerator(cfg, nil)

	disabled := &registry.ResolvedService{
		Name:    "disabled-svc",
		Enabled: false,
		FinalDefinition: &registry.ServiceDefinition{
			Metadata: registry.ServiceMetadata{Name: "disabled-svc"},
			Integrations: registry.Integrations{
				Homepage: &registry.HomepageIntegration{
					Enabled: true,
					Group:   "Services",
				},
			},
		},
	}

	graph := makeTestGraph(disabled)

	data, err := gen.GenerateHomepageServices(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if strings.Contains(string(data), "disabled-svc") {
		t.Error("disabled service should not appear in homepage config")
	}
}

func TestGenerateHomepageServicesSkipsNoIntegration(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"

	gen := NewIntegrationsGenerator(cfg, nil)

	noHomepage := makeResolvedService("no-homepage", &registry.ServiceDefinition{
		Metadata:     registry.ServiceMetadata{Name: "no-homepage"},
		Conditions:   registry.Conditions{Always: true},
		Integrations: registry.Integrations{}, // No homepage integration
	})

	graph := makeTestGraph(noHomepage)

	data, err := gen.GenerateHomepageServices(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if strings.Contains(string(data), "no-homepage") {
		t.Error("service without homepage integration should not appear")
	}
}

// --- getServiceURL ---

func TestGetServiceURLSubdomain(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategySubdomain
	cfg.Expose.Mode = config.ExposeModeDirect

	gen := NewIntegrationsGenerator(cfg, nil)

	def := &registry.ServiceDefinition{
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Subdomain: "sonarr",
		},
	}

	url := gen.getServiceURL(def)
	if url != "https://sonarr.example.com" {
		t.Errorf("getServiceURL() = %q, want %q", url, "https://sonarr.example.com")
	}
}

func TestGetServiceURLPath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategyPath
	cfg.Routing.BaseDomain = "sdbx"
	cfg.Expose.Mode = config.ExposeModeDirect

	gen := NewIntegrationsGenerator(cfg, nil)

	def := &registry.ServiceDefinition{
		Routing: registry.RoutingConfig{
			Enabled: true,
			Path:    "/sonarr",
		},
	}

	url := gen.getServiceURL(def)
	if url != "https://sdbx.example.com/sonarr" {
		t.Errorf("getServiceURL() = %q, want %q", url, "https://sdbx.example.com/sonarr")
	}
}

func TestGetServiceURLLANMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategySubdomain
	cfg.Expose.Mode = config.ExposeModeLAN

	gen := NewIntegrationsGenerator(cfg, nil)

	def := &registry.ServiceDefinition{
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Subdomain: "plex",
		},
	}

	url := gen.getServiceURL(def)
	if url != "http://plex.example.com" {
		t.Errorf("getServiceURL() = %q, want %q", url, "http://plex.example.com")
	}
}

func TestGetServiceURLForceSubdomain(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategyPath
	cfg.Routing.BaseDomain = "sdbx"
	cfg.Expose.Mode = config.ExposeModeDirect

	gen := NewIntegrationsGenerator(cfg, nil)

	def := &registry.ServiceDefinition{
		Routing: registry.RoutingConfig{
			Enabled:        true,
			Subdomain:      "auth",
			ForceSubdomain: true,
		},
	}

	url := gen.getServiceURL(def)
	// ForceSubdomain should use subdomain even in path routing mode
	if url != "https://auth.example.com" {
		t.Errorf("getServiceURL() = %q, want %q", url, "https://auth.example.com")
	}
}

func TestGetServiceURLEmptyDomain(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = ""
	cfg.Routing.Strategy = config.RoutingStrategySubdomain
	cfg.Expose.Mode = config.ExposeModeLAN

	gen := NewIntegrationsGenerator(cfg, nil)

	def := &registry.ServiceDefinition{
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Subdomain: "test",
		},
	}

	url := gen.getServiceURL(def)
	// With empty domain, URL will be malformed: "http://test."
	if !strings.HasPrefix(url, "http://test.") {
		t.Errorf("getServiceURL() with empty domain = %q", url)
	}
}

// --- GenerateCloudflaredConfig ---

func TestGenerateCloudflaredConfigEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"

	gen := NewIntegrationsGenerator(cfg, nil)
	graph := makeTestGraph()

	data, err := gen.GenerateCloudflaredConfig(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// Should always have catch-all rule
	var parsed CloudflaredConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}

	if len(parsed.Ingress) != 1 {
		t.Errorf("expected 1 catch-all rule, got %d", len(parsed.Ingress))
	}
	if parsed.Ingress[0].Service != "http_status:404" {
		t.Errorf("catch-all rule = %q, want http_status:404", parsed.Ingress[0].Service)
	}
}

func TestGenerateCloudflaredConfigWithServices(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategySubdomain

	gen := NewIntegrationsGenerator(cfg, nil)

	sonarr := makeResolvedService("sonarr", &registry.ServiceDefinition{
		Metadata:   registry.ServiceMetadata{Name: "sonarr"},
		Conditions: registry.Conditions{Always: true},
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Subdomain: "sonarr",
		},
		Integrations: registry.Integrations{
			Cloudflared: &registry.CloudflaredIntegration{
				Enabled: true,
			},
		},
	})

	graph := makeTestGraph(sonarr)

	data, err := gen.GenerateCloudflaredConfig(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var parsed CloudflaredConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}

	// Should have sonarr rule + catch-all
	if len(parsed.Ingress) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(parsed.Ingress))
	}

	if parsed.Ingress[0].Hostname != "sonarr.example.com" {
		t.Errorf("hostname = %q, want sonarr.example.com", parsed.Ingress[0].Hostname)
	}
	if parsed.Ingress[0].Service != "http://sdbx-traefik:80" {
		t.Errorf("service = %q, want http://sdbx-traefik:80", parsed.Ingress[0].Service)
	}
}

func TestGenerateCloudflaredConfigDeduplicatesHostnames(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategyPath
	cfg.Routing.BaseDomain = "sdbx"

	gen := NewIntegrationsGenerator(cfg, nil)

	// Two services sharing the same hostname in path routing
	svcA := makeResolvedService("svc-a", &registry.ServiceDefinition{
		Metadata:   registry.ServiceMetadata{Name: "svc-a"},
		Conditions: registry.Conditions{Always: true},
		Routing: registry.RoutingConfig{
			Enabled: true,
			Path:    "/a",
		},
		Integrations: registry.Integrations{
			Cloudflared: &registry.CloudflaredIntegration{Enabled: true},
		},
	})

	svcB := makeResolvedService("svc-b", &registry.ServiceDefinition{
		Metadata:   registry.ServiceMetadata{Name: "svc-b"},
		Conditions: registry.Conditions{Always: true},
		Routing: registry.RoutingConfig{
			Enabled: true,
			Path:    "/b",
		},
		Integrations: registry.Integrations{
			Cloudflared: &registry.CloudflaredIntegration{Enabled: true},
		},
	})

	graph := makeTestGraph(svcA, svcB)

	data, err := gen.GenerateCloudflaredConfig(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var parsed CloudflaredConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}

	// Should have only 1 hostname rule + catch-all (deduplicated)
	if len(parsed.Ingress) != 2 {
		t.Errorf("expected 2 rules (1 deduped hostname + catch-all), got %d", len(parsed.Ingress))
	}
}

// --- GenerateTraefikDynamic ---

func TestGenerateTraefikDynamicSubdomain(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategySubdomain

	gen := NewIntegrationsGenerator(cfg, nil)
	graph := makeTestGraph()

	data, err := gen.GenerateTraefikDynamic(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var parsed TraefikDynamicConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}

	// Should always have authelia middleware
	authelia, ok := parsed.HTTP.Middlewares["authelia"]
	if !ok {
		t.Fatal("expected authelia middleware")
	}
	if authelia.ForwardAuth == nil {
		t.Fatal("expected ForwardAuth config")
	}
	if !strings.Contains(authelia.ForwardAuth.Address, "auth.example.com") {
		t.Errorf("authelia address = %q, want to contain auth.example.com", authelia.ForwardAuth.Address)
	}
	if !authelia.ForwardAuth.TrustForwardHeader {
		t.Error("TrustForwardHeader should be true")
	}
}

func TestGenerateTraefikDynamicPathRouting(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategyPath
	cfg.Routing.BaseDomain = "sdbx"

	gen := NewIntegrationsGenerator(cfg, nil)

	sonarr := makeResolvedService("sonarr", &registry.ServiceDefinition{
		Metadata:   registry.ServiceMetadata{Name: "sonarr"},
		Conditions: registry.Conditions{Always: true},
		Routing: registry.RoutingConfig{
			Enabled: true,
			Path:    "/sonarr",
			PathRouting: registry.PathRoutingConfig{
				Strategy: "stripPrefix",
			},
		},
	})

	graph := makeTestGraph(sonarr)

	data, err := gen.GenerateTraefikDynamic(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var parsed TraefikDynamicConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}

	// Should have strip-sonarr middleware
	strip, ok := parsed.HTTP.Middlewares["strip-sonarr"]
	if !ok {
		t.Fatal("expected strip-sonarr middleware")
	}
	if strip.StripPrefix == nil {
		t.Fatal("expected StripPrefix config")
	}
	if len(strip.StripPrefix.Prefixes) != 1 || strip.StripPrefix.Prefixes[0] != "/sonarr" {
		t.Errorf("prefixes = %v, want [/sonarr]", strip.StripPrefix.Prefixes)
	}

	// Authelia address should use path format
	authelia := parsed.HTTP.Middlewares["authelia"]
	if !strings.Contains(authelia.ForwardAuth.Address, "sdbx.example.com") {
		t.Errorf("authelia address = %q, want to contain sdbx.example.com", authelia.ForwardAuth.Address)
	}
}

func TestGenerateTraefikDynamicSkipsForceSubdomain(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategyPath
	cfg.Routing.BaseDomain = "sdbx"

	gen := NewIntegrationsGenerator(cfg, nil)

	// ForceSubdomain services should not get strip prefix
	authelia := makeResolvedService("authelia", &registry.ServiceDefinition{
		Metadata:   registry.ServiceMetadata{Name: "authelia"},
		Conditions: registry.Conditions{Always: true},
		Routing: registry.RoutingConfig{
			Enabled:        true,
			Subdomain:      "auth",
			ForceSubdomain: true,
		},
	})

	graph := makeTestGraph(authelia)

	data, err := gen.GenerateTraefikDynamic(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var parsed TraefikDynamicConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid YAML: %v", err)
	}

	// Should NOT have strip-authelia middleware
	if _, ok := parsed.HTTP.Middlewares["strip-authelia"]; ok {
		t.Error("ForceSubdomain service should not get strip prefix middleware")
	}
}

// --- GenerateAutheliaAccessRules ---

func TestGenerateAutheliaAccessRulesEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	gen := NewIntegrationsGenerator(cfg, nil)
	graph := makeTestGraph()

	rules, err := gen.GenerateAutheliaAccessRules(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(rules) != 0 {
		t.Errorf("expected 0 rules for empty graph, got %d", len(rules))
	}
}

func TestGenerateAutheliaAccessRulesWithBypass(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategySubdomain

	gen := NewIntegrationsGenerator(cfg, nil)

	plex := makeResolvedService("plex", &registry.ServiceDefinition{
		Metadata:   registry.ServiceMetadata{Name: "plex"},
		Conditions: registry.Conditions{Always: true},
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Subdomain: "plex",
			Auth: registry.AuthConfig{
				Bypass: true,
			},
		},
	})

	sonarr := makeResolvedService("sonarr", &registry.ServiceDefinition{
		Metadata:   registry.ServiceMetadata{Name: "sonarr"},
		Conditions: registry.Conditions{Always: true},
		Routing: registry.RoutingConfig{
			Enabled:   true,
			Subdomain: "sonarr",
			Auth: registry.AuthConfig{
				Required: true,
			},
		},
	})

	graph := makeTestGraph(plex, sonarr)

	rules, err := gen.GenerateAutheliaAccessRules(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	// Find plex rule - should be bypass
	var plexRule, sonarrRule *AutheliaAccessRule
	for i := range rules {
		if rules[i].Domain == "plex.example.com" {
			plexRule = &rules[i]
		}
		if rules[i].Domain == "sonarr.example.com" {
			sonarrRule = &rules[i]
		}
	}

	if plexRule == nil {
		t.Fatal("expected plex rule")
	}
	if plexRule.Policy != "bypass" {
		t.Errorf("plex policy = %q, want bypass", plexRule.Policy)
	}

	if sonarrRule == nil {
		t.Fatal("expected sonarr rule")
	}
	if sonarrRule.Policy != "one_factor" {
		t.Errorf("sonarr policy = %q, want one_factor", sonarrRule.Policy)
	}
}

func TestGenerateAutheliaAccessRulesPathRouting(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = config.RoutingStrategyPath
	cfg.Routing.BaseDomain = "sdbx"

	gen := NewIntegrationsGenerator(cfg, nil)

	sonarr := makeResolvedService("sonarr", &registry.ServiceDefinition{
		Metadata:   registry.ServiceMetadata{Name: "sonarr"},
		Conditions: registry.Conditions{Always: true},
		Routing: registry.RoutingConfig{
			Enabled: true,
			Path:    "/sonarr",
			Auth: registry.AuthConfig{
				Required: true,
			},
		},
	})

	graph := makeTestGraph(sonarr)

	rules, err := gen.GenerateAutheliaAccessRules(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	if rules[0].Domain != "sdbx.example.com" {
		t.Errorf("domain = %q, want sdbx.example.com", rules[0].Domain)
	}
}

// --- GenerateEnvFile ---

func TestGenerateEnvFileBasic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Expose.Mode = config.ExposeModeLAN
	cfg.Timezone = "America/New_York"
	cfg.ConfigPath = "./configs"
	cfg.DataPath = "./data"
	cfg.DownloadsPath = "./downloads"
	cfg.MediaPath = "./media"
	cfg.PUID = 1000
	cfg.PGID = 1000
	cfg.Umask = "022"

	gen := NewIntegrationsGenerator(cfg, nil)
	graph := makeTestGraph()

	data, err := gen.GenerateEnvFile(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	content := string(data)

	checks := []string{
		"SDBX_DOMAIN=test.local",
		"SDBX_EXPOSE_MODE=lan",
		"SDBX_TIMEZONE=America/New_York",
		"SDBX_CONFIG_PATH=./configs",
		"SDBX_DOWNLOADS_PATH=./downloads",
		"SDBX_MEDIA_PATH=./media",
		"PUID=1000",
		"PGID=1000",
		"UMASK=022",
	}

	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("env file should contain %q", check)
		}
	}
}

func TestGenerateEnvFileWithVPN(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.VPNEnabled = true
	cfg.VPNProvider = "mullvad"
	cfg.VPNCountry = "Sweden"

	gen := NewIntegrationsGenerator(cfg, nil)
	graph := makeTestGraph()

	data, err := gen.GenerateEnvFile(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "SDBX_VPN_PROVIDER=mullvad") {
		t.Error("env should contain VPN provider when enabled")
	}
	if !strings.Contains(content, "SDBX_VPN_COUNTRY=Sweden") {
		t.Error("env should contain VPN country when enabled")
	}
}

func TestGenerateEnvFileWithoutVPN(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.VPNEnabled = false

	gen := NewIntegrationsGenerator(cfg, nil)
	graph := makeTestGraph()

	data, err := gen.GenerateEnvFile(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	content := string(data)

	if strings.Contains(content, "SDBX_VPN_PROVIDER=") {
		t.Error("env should not contain VPN provider when disabled")
	}
}

func TestGenerateEnvFileWithTLS(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Expose.Mode = config.ExposeModeDirect
	cfg.Expose.TLS.Email = "admin@test.local"

	gen := NewIntegrationsGenerator(cfg, nil)
	graph := makeTestGraph()

	data, err := gen.GenerateEnvFile(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "TRAEFIK_ACME_EMAIL=admin@test.local") {
		t.Error("env should contain ACME email in direct mode")
	}
}

func TestGenerateEnvFileWithAddons(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Addons = []string{"sonarr", "radarr"}

	gen := NewIntegrationsGenerator(cfg, nil)
	graph := makeTestGraph()

	data, err := gen.GenerateEnvFile(graph)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	content := string(data)

	if !strings.Contains(content, "sonarr, radarr") {
		t.Error("env should list enabled addons")
	}
}
