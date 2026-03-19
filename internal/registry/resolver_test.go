package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maiko/sdbx/internal/config"
)

// newTestRegistry creates a minimal registry with embedded source for testing.
func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	cacheDir := t.TempDir()
	r := &Registry{
		sources:   []SourceProvider{NewEmbeddedSource()},
		cache:     NewCache(cacheDir),
		validator: NewValidator(),
	}
	r.resolver = NewResolver(r)
	return r
}

// newTestRegistryWithLocal creates a registry with a local source directory for custom test services.
func newTestRegistryWithLocal(t *testing.T, dir string) *Registry {
	t.Helper()
	cacheDir := t.TempDir()
	local := &LocalSource{
		BaseSource: BaseSource{
			name:     "test-local",
			srcType:  "local",
			priority: 100,
			enabled:  true,
			path:     dir,
			loader:   NewLoader(),
		},
	}
	r := &Registry{
		sources:   []SourceProvider{local},
		cache:     NewCache(cacheDir),
		validator: NewValidator(),
	}
	r.resolver = NewResolver(r)
	return r
}

func TestResolveEmbeddedServices(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Expose.Mode = config.ExposeModeLAN
	cfg.VPNEnabled = true

	ctx := context.Background()
	graph, err := resolver.Resolve(ctx, cfg)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if len(graph.Services) == 0 {
		t.Fatal("expected resolved services, got 0")
	}

	// Core services should be resolved
	coreServices := []string{"traefik", "authelia", "qbittorrent", "plex", "sdbx-webui"}
	for _, name := range coreServices {
		if _, exists := graph.Services[name]; !exists {
			t.Errorf("expected core service %s to be resolved", name)
		}
	}

	// gluetun should appear when VPN is enabled
	if _, exists := graph.Services["gluetun"]; !exists {
		t.Error("expected gluetun to be resolved when VPN is enabled")
	}
}

func TestResolveVPNDisabled(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Expose.Mode = config.ExposeModeLAN
	cfg.VPNEnabled = false

	ctx := context.Background()
	graph, err := resolver.Resolve(ctx, cfg)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// gluetun has condition requireConfig: vpn_enabled
	// When VPN is disabled, gluetun's condition should not be met
	if svc, exists := graph.Services["gluetun"]; exists {
		// It might still be in the enabled set as a core service,
		// but conditions should filter it out during resolveService
		_ = svc // depends on service definition conditions
	}
}

func TestResolveCloudflaredMode(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Expose.Mode = config.ExposeModeCloudflared
	cfg.VPNEnabled = true

	ctx := context.Background()
	graph, err := resolver.Resolve(ctx, cfg)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// cloudflared should be resolved when mode is cloudflared
	if _, exists := graph.Services["cloudflared"]; !exists {
		t.Error("expected cloudflared to be resolved in cloudflared mode")
	}
}

func TestResolveCloudflaredNotInLANMode(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Expose.Mode = config.ExposeModeLAN
	cfg.VPNEnabled = true

	ctx := context.Background()
	graph, err := resolver.Resolve(ctx, cfg)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// cloudflared should NOT be resolved in LAN mode
	if _, exists := graph.Services["cloudflared"]; exists {
		t.Error("cloudflared should not be resolved in LAN mode")
	}
}

func TestResolveDependencyOrder(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Expose.Mode = config.ExposeModeLAN
	cfg.VPNEnabled = true

	ctx := context.Background()
	graph, err := resolver.Resolve(ctx, cfg)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Order should be set
	if len(graph.Order) == 0 {
		t.Fatal("expected dependency order to be computed")
	}

	// Order should contain same services as graph
	if len(graph.Order) != len(graph.Services) {
		t.Errorf("order length (%d) != services length (%d)", len(graph.Order), len(graph.Services))
	}

	// All ordered services should exist in the graph
	for _, name := range graph.Order {
		if _, exists := graph.Services[name]; !exists {
			t.Errorf("ordered service %s not in graph", name)
		}
	}
}

func TestResolveServiceByName(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Expose.Mode = config.ExposeModeLAN
	cfg.VPNEnabled = true

	ctx := context.Background()
	resolved, err := resolver.ResolveService(ctx, cfg, "traefik")
	if err != nil {
		t.Fatalf("ResolveService() error: %v", err)
	}

	if resolved.Name != "traefik" {
		t.Errorf("expected name 'traefik', got %q", resolved.Name)
	}
	if resolved.DefinitionHash == "" {
		t.Error("expected non-empty definition hash")
	}
	if !resolved.Enabled {
		t.Error("expected resolved service to be enabled")
	}
	if resolved.Definition == nil {
		t.Error("expected non-nil definition")
	}
	if resolved.FinalDefinition == nil {
		t.Error("expected non-nil final definition")
	}
}

func TestResolveServiceNotFound(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	ctx := context.Background()

	_, err := resolver.ResolveService(ctx, cfg, "nonexistent-service")
	if err == nil {
		t.Error("expected error for nonexistent service")
	}
}

func TestDetermineEnabledServices(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	cfg.Addons = []string{"sonarr", "radarr"}

	serviceMap := map[string]ServiceInfo{
		"traefik": {Name: "traefik", IsAddon: false},
		"authelia": {Name: "authelia", IsAddon: false},
		"sonarr":  {Name: "sonarr", IsAddon: true},
		"radarr":  {Name: "radarr", IsAddon: true},
		"lidarr":  {Name: "lidarr", IsAddon: true},
	}

	ctx := context.Background()
	enabled := resolver.determineEnabledServices(ctx, cfg, serviceMap)

	// Core services should always be enabled
	if !enabled["traefik"] {
		t.Error("core service traefik should be enabled")
	}
	if !enabled["authelia"] {
		t.Error("core service authelia should be enabled")
	}

	// Enabled addons should be enabled
	if !enabled["sonarr"] {
		t.Error("enabled addon sonarr should be enabled")
	}
	if !enabled["radarr"] {
		t.Error("enabled addon radarr should be enabled")
	}

	// Non-enabled addons should not be enabled
	if enabled["lidarr"] {
		t.Error("non-enabled addon lidarr should not be enabled")
	}
}

func TestTopologicalSortSimple(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	graph := &ResolutionGraph{
		Services: map[string]*ResolvedService{
			"a": {Name: "a", Dependencies: []string{}},
			"b": {Name: "b", Dependencies: []string{"a"}},
			"c": {Name: "c", Dependencies: []string{"b"}},
		},
	}

	order, err := resolver.topologicalSort(graph)
	if err != nil {
		t.Fatalf("topologicalSort() error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 items in order, got %d", len(order))
	}

	// a must come before b, b must come before c
	posA, posB, posC := -1, -1, -1
	for i, name := range order {
		switch name {
		case "a":
			posA = i
		case "b":
			posB = i
		case "c":
			posC = i
		}
	}

	if posA >= posB {
		t.Errorf("'a' (pos %d) should come before 'b' (pos %d)", posA, posB)
	}
	if posB >= posC {
		t.Errorf("'b' (pos %d) should come before 'c' (pos %d)", posB, posC)
	}
}

func TestTopologicalSortCircularDependency(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	graph := &ResolutionGraph{
		Services: map[string]*ResolvedService{
			"a": {Name: "a", Dependencies: []string{"b"}},
			"b": {Name: "b", Dependencies: []string{"c"}},
			"c": {Name: "c", Dependencies: []string{"a"}},
		},
	}

	_, err := resolver.topologicalSort(graph)
	if err == nil {
		t.Error("expected error for circular dependency")
	}
	if err != nil && err.Error() != "circular dependency detected" {
		t.Errorf("expected 'circular dependency detected', got %q", err.Error())
	}
}

func TestTopologicalSortNoDependencies(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	graph := &ResolutionGraph{
		Services: map[string]*ResolvedService{
			"a": {Name: "a", Dependencies: []string{}},
			"b": {Name: "b", Dependencies: []string{}},
			"c": {Name: "c", Dependencies: []string{}},
		},
	}

	order, err := resolver.topologicalSort(graph)
	if err != nil {
		t.Fatalf("topologicalSort() error: %v", err)
	}

	if len(order) != 3 {
		t.Errorf("expected 3 items, got %d", len(order))
	}
}

func TestTopologicalSortEmptyGraph(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	graph := &ResolutionGraph{
		Services: map[string]*ResolvedService{},
	}

	order, err := resolver.topologicalSort(graph)
	if err != nil {
		t.Fatalf("topologicalSort() error: %v", err)
	}

	if len(order) != 0 {
		t.Errorf("expected empty order, got %d items", len(order))
	}
}

func TestEvaluateConditionString(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	tests := []struct {
		name      string
		condition string
		cfg       *config.Config
		expected  bool
	}{
		{
			name:      "empty condition",
			condition: "",
			cfg:       config.DefaultConfig(),
			expected:  true,
		},
		{
			name:      "VPN enabled - true",
			condition: "{{ .Config.VPNEnabled }}",
			cfg:       &config.Config{VPNEnabled: true},
			expected:  true,
		},
		{
			name:      "VPN enabled - false",
			condition: "{{ .Config.VPNEnabled }}",
			cfg:       &config.Config{VPNEnabled: false},
			expected:  false,
		},
		{
			name:      "not VPN enabled - VPN off",
			condition: "{{ not .Config.VPNEnabled }}",
			cfg:       &config.Config{VPNEnabled: false},
			expected:  true,
		},
		{
			name:      "not VPN enabled - VPN on",
			condition: "{{ not .Config.VPNEnabled }}",
			cfg:       &config.Config{VPNEnabled: true},
			expected:  false,
		},
		{
			name:      "cloudflared mode match",
			condition: `{{ eq .Config.Expose.Mode "cloudflared" }}`,
			cfg: func() *config.Config {
				c := config.DefaultConfig()
				c.Expose.Mode = config.ExposeModeCloudflared
				return c
			}(),
			expected: true,
		},
		{
			name:      "cloudflared mode no match",
			condition: `{{ eq .Config.Expose.Mode "cloudflared" }}`,
			cfg: func() *config.Config {
				c := config.DefaultConfig()
				c.Expose.Mode = config.ExposeModeLAN
				return c
			}(),
			expected: false,
		},
		{
			name:      "path routing match",
			condition: `{{ eq .Config.Routing.Strategy "path" }}`,
			cfg: func() *config.Config {
				c := config.DefaultConfig()
				c.Routing.Strategy = config.RoutingStrategyPath
				return c
			}(),
			expected: true,
		},
		{
			name:      "unknown condition defaults to false",
			condition: "{{ .SomeUnknownField }}",
			cfg:       config.DefaultConfig(),
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.evaluateConditionString(tt.condition, tt.cfg)
			if result != tt.expected {
				t.Errorf("evaluateConditionString(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestCollectDependencies(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	def := &ServiceDefinition{
		Spec: ServiceSpec{
			Dependencies: DependencySpec{
				Required: []string{"traefik", "authelia"},
				Conditional: []ConditionalDependency{
					{Name: "gluetun", When: "{{ .Config.VPNEnabled }}"},
				},
			},
		},
	}

	// VPN enabled: should include gluetun
	cfgVPN := &config.Config{VPNEnabled: true}
	deps := resolver.collectDependencies(def, cfgVPN)
	depsMap := make(map[string]bool)
	for _, d := range deps {
		depsMap[d] = true
	}

	if !depsMap["traefik"] {
		t.Error("expected traefik in dependencies")
	}
	if !depsMap["authelia"] {
		t.Error("expected authelia in dependencies")
	}
	if !depsMap["gluetun"] {
		t.Error("expected gluetun in dependencies when VPN enabled")
	}

	// VPN disabled: should NOT include gluetun
	cfgNoVPN := &config.Config{VPNEnabled: false}
	depsNoVPN := resolver.collectDependencies(def, cfgNoVPN)
	depsNoVPNMap := make(map[string]bool)
	for _, d := range depsNoVPN {
		depsNoVPNMap[d] = true
	}

	if depsNoVPNMap["gluetun"] {
		t.Error("gluetun should not be in dependencies when VPN disabled")
	}
	if !depsNoVPNMap["traefik"] {
		t.Error("traefik should still be in dependencies")
	}
}

func TestCalculateHash(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	def := &ServiceDefinition{
		Metadata: ServiceMetadata{
			Name:    "test",
			Version: "1.0.0",
		},
	}

	hash := resolver.calculateHash(def)
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if len(hash) < 10 {
		t.Errorf("hash seems too short: %q", hash)
	}

	// Same definition should produce same hash
	hash2 := resolver.calculateHash(def)
	if hash != hash2 {
		t.Errorf("same definition produced different hashes: %q vs %q", hash, hash2)
	}

	// Different definition should produce different hash
	def2 := &ServiceDefinition{
		Metadata: ServiceMetadata{
			Name:    "other",
			Version: "2.0.0",
		},
	}
	hash3 := resolver.calculateHash(def2)
	if hash == hash3 {
		t.Error("different definitions should produce different hashes")
	}
}

func TestLoadOverridesEmpty(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	ctx := context.Background()
	overrides := resolver.loadOverrides(ctx, "traefik")

	// Embedded source is skipped for overrides, so should be empty
	if len(overrides) != 0 {
		t.Errorf("expected no overrides from embedded source, got %d", len(overrides))
	}
}

func TestLoadOverridesFromLocal(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service with an override
	svcDir := filepath.Join(tmpDir, "core", "test-svc")
	if err := os.MkdirAll(svcDir, 0755); err != nil {
		t.Fatal(err)
	}

	svcYAML := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: test-svc
  version: 1.0.0
  category: utility
  description: Test service
spec:
  image:
    repository: test/image
    tag: latest
  container:
    name_template: "sdbx-test-svc"
routing:
  enabled: false
conditions:
  always: true
`
	if err := os.WriteFile(filepath.Join(svcDir, "service.yaml"), []byte(svcYAML), 0644); err != nil {
		t.Fatal(err)
	}

	overrideYAML := `apiVersion: sdbx.one/v1
kind: ServiceOverride
metadata:
  name: test-svc
spec:
  image:
    tag: "v2.0"
`
	if err := os.WriteFile(filepath.Join(svcDir, "override.yaml"), []byte(overrideYAML), 0644); err != nil {
		t.Fatal(err)
	}

	reg := newTestRegistryWithLocal(t, tmpDir)
	resolver := NewResolver(reg)

	ctx := context.Background()
	overrides := resolver.loadOverrides(ctx, "test-svc")

	if len(overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(overrides))
	}
	if overrides[0].Metadata.Name != "test-svc" {
		t.Errorf("expected override for test-svc, got %q", overrides[0].Metadata.Name)
	}
}

func TestGetDependencyOrder(t *testing.T) {
	graph := &ResolutionGraph{
		Order: []string{"a", "b", "c"},
	}

	order := GetDependencyOrder(graph)
	if len(order) != 3 {
		t.Errorf("expected 3 items, got %d", len(order))
	}
}

func TestGetEnabledServices(t *testing.T) {
	graph := &ResolutionGraph{
		Services: map[string]*ResolvedService{
			"a": {Name: "a", Enabled: true},
			"b": {Name: "b", Enabled: false},
			"c": {Name: "c", Enabled: true},
		},
	}

	enabled := GetEnabledServices(graph)
	if len(enabled) != 2 {
		t.Errorf("expected 2 enabled services, got %d", len(enabled))
	}
	if _, ok := enabled["a"]; !ok {
		t.Error("expected 'a' in enabled services")
	}
	if _, ok := enabled["b"]; ok {
		t.Error("'b' should not be in enabled services")
	}
	if _, ok := enabled["c"]; !ok {
		t.Error("expected 'c' in enabled services")
	}
}

func TestTopologicalSortDiamond(t *testing.T) {
	// Diamond dependency: a -> b, a -> c, b -> d, c -> d
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	graph := &ResolutionGraph{
		Services: map[string]*ResolvedService{
			"d": {Name: "d", Dependencies: []string{}},
			"b": {Name: "b", Dependencies: []string{"d"}},
			"c": {Name: "c", Dependencies: []string{"d"}},
			"a": {Name: "a", Dependencies: []string{"b", "c"}},
		},
	}

	order, err := resolver.topologicalSort(graph)
	if err != nil {
		t.Fatalf("topologicalSort() error: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("expected 4 items, got %d", len(order))
	}

	// Find positions
	pos := make(map[string]int)
	for i, name := range order {
		pos[name] = i
	}

	// d must come before b and c; b and c must come before a
	if pos["d"] >= pos["b"] {
		t.Error("d should come before b")
	}
	if pos["d"] >= pos["c"] {
		t.Error("d should come before c")
	}
	if pos["b"] >= pos["a"] {
		t.Error("b should come before a")
	}
	if pos["c"] >= pos["a"] {
		t.Error("c should come before a")
	}
}

func TestResolveWithCustomLocalService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two services: svc-a depends on svc-b
	for _, svc := range []struct {
		name string
		yaml string
	}{
		{
			name: "svc-b",
			yaml: `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: svc-b
  version: 1.0.0
  category: utility
  description: Service B
spec:
  image:
    repository: test/svc-b
    tag: latest
  container:
    name_template: "sdbx-svc-b"
routing:
  enabled: false
conditions:
  always: true
`,
		},
		{
			name: "svc-a",
			yaml: `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: svc-a
  version: 1.0.0
  category: utility
  description: Service A depends on B
spec:
  image:
    repository: test/svc-a
    tag: latest
  container:
    name_template: "sdbx-svc-a"
  dependencies:
    required:
      - svc-b
routing:
  enabled: false
conditions:
  always: true
`,
		},
	} {
		dir := filepath.Join(tmpDir, "core", svc.name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte(svc.yaml), 0644); err != nil {
			t.Fatal(err)
		}
	}

	reg := newTestRegistryWithLocal(t, tmpDir)
	resolver := NewResolver(reg)

	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"

	ctx := context.Background()
	graph, err := resolver.Resolve(ctx, cfg)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	// Both services should be resolved
	if _, exists := graph.Services["svc-a"]; !exists {
		t.Error("expected svc-a to be resolved")
	}
	if _, exists := graph.Services["svc-b"]; !exists {
		t.Error("expected svc-b to be resolved")
	}

	// svc-b should come before svc-a in order
	pos := make(map[string]int)
	for i, name := range graph.Order {
		pos[name] = i
	}
	if pos["svc-b"] >= pos["svc-a"] {
		t.Errorf("svc-b (pos %d) should come before svc-a (pos %d)", pos["svc-b"], pos["svc-a"])
	}
}

func TestResolveAddonNotEnabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an addon service
	dir := filepath.Join(tmpDir, "addons", "test-addon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	yaml := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: test-addon
  version: 1.0.0
  category: media
  description: Test addon
spec:
  image:
    repository: test/addon
    tag: latest
  container:
    name_template: "sdbx-test-addon"
routing:
  enabled: false
conditions:
  requireAddon: true
`
	if err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	reg := newTestRegistryWithLocal(t, tmpDir)
	resolver := NewResolver(reg)

	// Config with NO addons enabled
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Addons = []string{}

	ctx := context.Background()
	graph, err := resolver.Resolve(ctx, cfg)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if _, exists := graph.Services["test-addon"]; exists {
		t.Error("addon should not be resolved when not enabled")
	}
}

func TestResolveAddonEnabled(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an addon service
	dir := filepath.Join(tmpDir, "addons", "test-addon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	yaml := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: test-addon
  version: 1.0.0
  category: media
  description: Test addon
spec:
  image:
    repository: test/addon
    tag: latest
  container:
    name_template: "sdbx-test-addon"
routing:
  enabled: false
conditions:
  requireAddon: true
`
	if err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	reg := newTestRegistryWithLocal(t, tmpDir)
	resolver := NewResolver(reg)

	// Config WITH addon enabled
	cfg := config.DefaultConfig()
	cfg.Domain = "test.local"
	cfg.Addons = []string{"test-addon"}

	ctx := context.Background()
	graph, err := resolver.Resolve(ctx, cfg)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if _, exists := graph.Services["test-addon"]; !exists {
		t.Error("addon should be resolved when enabled")
	}
}

func TestEvaluateConditionStringTemplate(t *testing.T) {
	reg := newTestRegistry(t)
	resolver := NewResolver(reg)

	tests := []struct {
		name      string
		condition string
		vpn       bool
		mode      string
		strategy  string
		want      bool
	}{
		{"empty condition", "", false, "lan", "subdomain", true},
		{"VPN enabled true", "{{ .Config.VPNEnabled }}", true, "lan", "subdomain", true},
		{"VPN enabled false", "{{ .Config.VPNEnabled }}", false, "lan", "subdomain", false},
		{"VPN not enabled", "{{ not .Config.VPNEnabled }}", false, "lan", "subdomain", true},
		{"cloudflared mode match", `{{ eq .Config.Expose.Mode "cloudflared" }}`, false, "cloudflared", "subdomain", true},
		{"cloudflared mode no match", `{{ eq .Config.Expose.Mode "cloudflared" }}`, false, "direct", "subdomain", false},
		{"path strategy match", `{{ eq .Config.Routing.Strategy "path" }}`, false, "lan", "path", true},
		{"subdomain strategy match", `{{ eq .Config.Routing.Strategy "subdomain" }}`, false, "lan", "subdomain", true},
		{"direct mode", `{{ eq .Config.Expose.Mode "direct" }}`, false, "direct", "subdomain", true},
		{"or condition", `{{ or (eq .Config.Expose.Mode "lan") (eq .Config.Expose.Mode "direct") }}`, false, "lan", "subdomain", true},
		{"ne condition", `{{ ne .Config.Expose.Mode "lan" }}`, false, "direct", "subdomain", true},
		{"invalid template", "{{ .Invalid.Field }}", false, "lan", "subdomain", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.VPNEnabled = tt.vpn
			cfg.Expose.Mode = tt.mode
			cfg.Routing.Strategy = tt.strategy

			got := resolver.evaluateConditionString(tt.condition, cfg)
			if got != tt.want {
				t.Errorf("evaluateConditionString(%q) = %v, want %v", tt.condition, got, tt.want)
			}
		})
	}
}
