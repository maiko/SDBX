package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
)

// ComposeGenerator generates Docker Compose files from registry definitions
type ComposeGenerator struct {
	Config   *config.Config
	Registry *registry.Registry
	Secrets  map[string]string
	funcMap  template.FuncMap
}

// NewComposeGenerator creates a new compose generator
func NewComposeGenerator(cfg *config.Config, reg *registry.Registry, secrets map[string]string) *ComposeGenerator {
	g := &ComposeGenerator{
		Config:   cfg,
		Registry: reg,
		Secrets:  secrets,
	}
	g.initFuncMap()
	return g
}

// ComposeFile represents a Docker Compose file
type ComposeFile struct {
	Name     string                      `yaml:"name"`
	Services map[string]ComposeService   `yaml:"services"`
	Networks map[string]ComposeNetwork   `yaml:"networks,omitempty"`
	Secrets  map[string]ComposeSecretDef `yaml:"secrets,omitempty"`
}

// ComposeService represents a Docker Compose service
type ComposeService struct {
	Image         string                        `yaml:"image"`
	ContainerName string                        `yaml:"container_name"`
	Restart       string                        `yaml:"restart,omitempty"`
	Environment   []string                      `yaml:"environment,omitempty"`
	EnvFile       []string                      `yaml:"env_file,omitempty"`
	Volumes       []string                      `yaml:"volumes,omitempty"`
	Ports         []string                      `yaml:"ports,omitempty"`
	Networks      []string                      `yaml:"networks,omitempty"`
	NetworkMode   string                        `yaml:"network_mode,omitempty"`
	DependsOn     map[string]DependsOnCondition `yaml:"depends_on,omitempty"`
	Labels        []string                      `yaml:"labels,omitempty"`
	HealthCheck   *ComposeHealthCheck           `yaml:"healthcheck,omitempty"`
	CapAdd        []string                      `yaml:"cap_add,omitempty"`
	Devices       []string                      `yaml:"devices,omitempty"`
	Secrets       []string                      `yaml:"secrets,omitempty"`
	Command       string                        `yaml:"command,omitempty"`
}

// DependsOnCondition represents a depends_on condition
type DependsOnCondition struct {
	Condition string `yaml:"condition,omitempty"`
}

// ComposeHealthCheck represents a Docker Compose health check
type ComposeHealthCheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval,omitempty"`
	Timeout     string   `yaml:"timeout,omitempty"`
	Retries     int      `yaml:"retries,omitempty"`
	StartPeriod string   `yaml:"start_period,omitempty"`
}

// ComposeNetwork represents a Docker Compose network
type ComposeNetwork struct {
	Name string `yaml:"name,omitempty"`
}

// ComposeSecretDef represents a Docker Compose secret definition
type ComposeSecretDef struct {
	File string `yaml:"file"`
}

// initFuncMap initializes template functions
func (g *ComposeGenerator) initFuncMap() {
	g.funcMap = template.FuncMap{
		"eq":        func(a, b interface{}) bool { return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b) },
		"ne":        func(a, b interface{}) bool { return fmt.Sprintf("%v", a) != fmt.Sprintf("%v", b) },
		"not":       func(b bool) bool { return !b },
		"or":        func(a, b bool) bool { return a || b },
		"and":       func(a, b bool) bool { return a && b },
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"trim":      strings.TrimSpace,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
	}
}

// TemplateContext provides data for template evaluation
type TemplateContext struct {
	Config  *config.Config
	Secrets map[string]string
	Name    string
}

// Generate generates a Docker Compose file from resolved services
func (g *ComposeGenerator) Generate(graph *registry.ResolutionGraph) (*ComposeFile, error) {
	compose := &ComposeFile{
		Name:     "sdbx",
		Services: make(map[string]ComposeService),
		Networks: map[string]ComposeNetwork{
			"proxy": {Name: "sdbx_proxy"},
			"vpn":   {Name: "sdbx_vpn"},
		},
		Secrets: make(map[string]ComposeSecretDef),
	}

	// Generate services in dependency order
	for _, serviceName := range graph.Order {
		resolved := graph.Services[serviceName]
		if !resolved.Enabled {
			continue
		}

		def := resolved.FinalDefinition

		// Check conditions
		if !g.evaluateConditions(def.Conditions) {
			continue
		}

		// Generate compose service
		svc := g.generateService(def)
		compose.Services[serviceName] = svc

		// Collect secrets
		for _, secret := range def.Secrets {
			compose.Secrets[secret.Name] = ComposeSecretDef{
				File: fmt.Sprintf("./secrets/%s.txt", secret.Name),
			}
		}
	}

	return compose, nil
}

// generateService generates a single compose service
func (g *ComposeGenerator) generateService(def *registry.ServiceDefinition) ComposeService {
	ctx := TemplateContext{
		Config:  g.Config,
		Secrets: g.Secrets,
		Name:    def.Metadata.Name,
	}

	svc := ComposeService{
		Image:         g.resolveImage(def),
		ContainerName: g.evalTemplate(def.Spec.Container.NameTemplate, ctx),
		Restart:       def.Spec.Container.Restart,
		Command:       def.Spec.Container.Command,
	}

	// Environment variables
	svc.Environment = g.buildEnvironment(def, ctx)

	// Env files
	svc.EnvFile = def.Spec.Environment.EnvFile

	// Volumes
	svc.Volumes = g.buildVolumes(def, ctx)

	// Ports
	svc.Ports = g.buildPorts(def, ctx)

	// Networks
	svc.Networks, svc.NetworkMode = g.buildNetworking(def, ctx)

	// Dependencies
	svc.DependsOn = g.buildDependsOn(def, ctx)

	// Labels (including Traefik)
	svc.Labels = g.buildLabels(def, ctx)

	// Health check
	if def.Spec.HealthCheck != nil {
		svc.HealthCheck = &ComposeHealthCheck{
			Test:     def.Spec.HealthCheck.Test,
			Interval: def.Spec.HealthCheck.Interval,
			Timeout:  def.Spec.HealthCheck.Timeout,
			Retries:  def.Spec.HealthCheck.Retries,
		}
	}

	// Capabilities
	svc.CapAdd = def.Spec.Container.Capabilities.Add

	// Devices
	svc.Devices = def.Spec.Container.Devices

	// Secrets
	for _, secret := range def.Secrets {
		svc.Secrets = append(svc.Secrets, secret.Name)
	}

	return svc
}

// resolveImage builds the full image reference
func (g *ComposeGenerator) resolveImage(def *registry.ServiceDefinition) string {
	img := def.Spec.Image.Repository
	if def.Spec.Image.Tag != "" {
		img += ":" + def.Spec.Image.Tag
	}
	return img
}

// buildEnvironment builds environment variables
func (g *ComposeGenerator) buildEnvironment(def *registry.ServiceDefinition, ctx TemplateContext) []string {
	var env []string

	// Static environment variables
	for _, e := range def.Spec.Environment.Static {
		value := e.Value
		if e.ValueFrom != nil && e.ValueFrom.SecretRef != "" {
			// Secret reference
			if secret, ok := g.Secrets[e.ValueFrom.SecretRef+".txt"]; ok {
				value = secret
			}
		}
		value = g.evalTemplate(value, ctx)
		env = append(env, fmt.Sprintf("%s=%s", e.Name, value))
	}

	// Conditional environment variables
	for _, e := range def.Spec.Environment.Conditional {
		if g.evalCondition(e.When, ctx) {
			value := e.Value
			if e.ValueFrom != nil && e.ValueFrom.SecretRef != "" {
				if secret, ok := g.Secrets[e.ValueFrom.SecretRef+".txt"]; ok {
					value = secret
				}
			}
			value = g.evalTemplate(value, ctx)
			env = append(env, fmt.Sprintf("%s=%s", e.Name, value))
		}
	}

	return env
}

// buildVolumes builds volume mounts
func (g *ComposeGenerator) buildVolumes(def *registry.ServiceDefinition, ctx TemplateContext) []string {
	var volumes []string
	for _, v := range def.Spec.Volumes {
		hostPath := g.evalTemplate(v.HostPath, ctx)
		mount := fmt.Sprintf("%s:%s", hostPath, v.ContainerPath)
		if v.ReadOnly {
			mount += ":ro"
		}
		volumes = append(volumes, mount)
	}
	return volumes
}

// buildPorts builds port mappings
func (g *ComposeGenerator) buildPorts(def *registry.ServiceDefinition, ctx TemplateContext) []string {
	var ports []string

	// Static ports
	ports = append(ports, def.Spec.Ports.Static...)

	// Conditional ports
	for _, p := range def.Spec.Ports.Conditional {
		if g.evalCondition(p.When, ctx) {
			ports = append(ports, p.Port)
		}
	}

	return ports
}

// buildNetworking builds network configuration
func (g *ComposeGenerator) buildNetworking(def *registry.ServiceDefinition, ctx TemplateContext) ([]string, string) {
	var networks []string
	var networkMode string

	// Check for network mode template
	if def.Spec.Networking.ModeTemplate != "" {
		networkMode = g.evalTemplate(def.Spec.Networking.ModeTemplate, ctx)
	} else if def.Spec.Networking.Mode == "bridge" || def.Spec.Networking.Mode == "" {
		// Default bridge mode - use networks
		for _, n := range def.Spec.Networking.Networks {
			if n.When == "" || g.evalCondition(n.When, ctx) {
				name := n.Name
				if name == "" {
					name = "proxy"
				}
				networks = append(networks, name)
			}
		}
	} else {
		networkMode = def.Spec.Networking.Mode
	}

	return networks, networkMode
}

// buildDependsOn builds service dependencies
func (g *ComposeGenerator) buildDependsOn(def *registry.ServiceDefinition, ctx TemplateContext) map[string]DependsOnCondition {
	deps := make(map[string]DependsOnCondition)

	// Required dependencies - default to service_started condition
	for _, dep := range def.Spec.Dependencies.Required {
		deps[dep] = DependsOnCondition{Condition: "service_started"}
	}

	// Conditional dependencies
	for _, dep := range def.Spec.Dependencies.Conditional {
		if g.evalCondition(dep.When, ctx) {
			condition := dep.Condition
			if condition == "" {
				condition = "service_started"
			}
			deps[dep.Name] = DependsOnCondition{
				Condition: condition,
			}
		}
	}

	if len(deps) == 0 {
		return nil
	}
	return deps
}

// buildLabels builds Docker labels including Traefik configuration
func (g *ComposeGenerator) buildLabels(def *registry.ServiceDefinition, ctx TemplateContext) []string {
	var labels []string

	// Watchtower label
	if def.Integrations.Watchtower != nil && def.Integrations.Watchtower.Enabled {
		labels = append(labels, "com.centurylinklabs.watchtower.enable=true")
	}

	// Traefik labels for routed services
	if def.Routing.Enabled {
		labels = append(labels, g.buildTraefikLabels(def, ctx)...)
	}

	return labels
}

// buildTraefikLabels generates Traefik routing labels
func (g *ComposeGenerator) buildTraefikLabels(def *registry.ServiceDefinition, _ TemplateContext) []string {
	var labels []string
	name := def.Metadata.Name

	labels = append(labels, "traefik.enable=true")

	// Router rule
	var rule string
	if def.Routing.ForceSubdomain || g.Config.Routing.Strategy == config.RoutingStrategySubdomain {
		// Subdomain routing
		subdomain := def.Routing.Subdomain
		rule = fmt.Sprintf("Host(`%s.%s`)", subdomain, g.Config.Domain)
	} else {
		// Path routing
		path := def.Routing.Path
		baseDomain := g.Config.Routing.BaseDomain
		rule = fmt.Sprintf("Host(`%s.%s`) && PathPrefix(`%s`)", baseDomain, g.Config.Domain, path)
	}
	labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.rule=%s", name, rule))

	// Entrypoint
	var entrypoint string
	switch g.Config.Expose.Mode {
	case config.ExposeModeCloudflared, config.ExposeModeLAN:
		entrypoint = "web"
	default:
		entrypoint = "websecure"
	}
	labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.entrypoints=%s", name, entrypoint))

	// TLS for direct mode
	if g.Config.Expose.Mode == config.ExposeModeDirect {
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.tls=true", name))
	}

	// Middlewares
	var middlewares []string

	// Strip prefix middleware for path routing
	if !def.Routing.ForceSubdomain && g.Config.Routing.Strategy == config.RoutingStrategyPath {
		if def.Routing.PathRouting.Strategy == "stripPrefix" {
			middlewares = append(middlewares, fmt.Sprintf("strip-%s@file", name))
		}
	}

	// Auth middleware
	if def.Routing.Auth.Required && !def.Routing.Auth.Bypass {
		middlewares = append(middlewares, "authelia@file")
	}

	if len(middlewares) > 0 {
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.middlewares=%s", name, strings.Join(middlewares, ",")))
	}

	// Service port
	labels = append(labels, fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", name, def.Routing.Port))

	// Priority for homepage
	if def.Routing.Traefik.Priority != nil {
		labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.priority=%d", name, *def.Routing.Traefik.Priority))
	}

	return labels
}

// evaluateConditions checks if a service's conditions are met
func (g *ComposeGenerator) evaluateConditions(cond registry.Conditions) bool {
	if cond.Always {
		return true
	}

	if cond.RequireConfig != "" {
		switch cond.RequireConfig {
		case "vpn_enabled":
			if !g.Config.VPNEnabled {
				return false
			}
		case "cloudflared":
			if g.Config.Expose.Mode != config.ExposeModeCloudflared {
				return false
			}
		}
	}

	return true
}

// evalTemplate evaluates a Go template string
func (g *ComposeGenerator) evalTemplate(tmpl string, ctx TemplateContext) string {
	if !strings.Contains(tmpl, "{{") {
		return tmpl
	}

	t, err := template.New("").Funcs(g.funcMap).Parse(tmpl)
	if err != nil {
		return tmpl
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return tmpl
	}

	return buf.String()
}

// evalCondition evaluates a condition template and returns boolean
func (g *ComposeGenerator) evalCondition(condition string, ctx TemplateContext) bool {
	if condition == "" {
		return true
	}

	result := g.evalTemplate(condition, ctx)
	return result == "true"
}

// ToYAML converts the compose file to YAML
func (c *ComposeFile) ToYAML() ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
