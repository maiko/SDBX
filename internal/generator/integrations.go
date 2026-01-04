package generator

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
)

// IntegrationsGenerator generates integration configs (homepage, cloudflared, traefik)
type IntegrationsGenerator struct {
	Config  *config.Config
	Secrets map[string]string
}

// NewIntegrationsGenerator creates a new integrations generator
func NewIntegrationsGenerator(cfg *config.Config, secrets map[string]string) *IntegrationsGenerator {
	return &IntegrationsGenerator{
		Config:  cfg,
		Secrets: secrets,
	}
}

// HomepageServices represents homepage services.yaml
type HomepageServices struct {
	Groups []HomepageGroup `yaml:",inline"`
}

// HomepageGroup represents a group of services in homepage
type HomepageGroup struct {
	Name     string            `yaml:"-"`
	Services []HomepageService `yaml:"-"`
}

// HomepageService represents a single service in homepage
type HomepageService struct {
	Name        string `yaml:"-"`
	Icon        string `yaml:"icon,omitempty"`
	Href        string `yaml:"href,omitempty"`
	Description string `yaml:"description,omitempty"`
	Container   string `yaml:"container,omitempty"`
}

// GenerateHomepageServices generates homepage services.yaml content
func (g *IntegrationsGenerator) GenerateHomepageServices(graph *registry.ResolutionGraph) ([]byte, error) {
	groups := make(map[string][]HomepageService)

	// Process services in order
	for _, serviceName := range graph.Order {
		resolved := graph.Services[serviceName]
		if !resolved.Enabled {
			continue
		}

		def := resolved.FinalDefinition

		// Check if service has homepage integration
		if def.Integrations.Homepage == nil || !def.Integrations.Homepage.Enabled {
			continue
		}

		// Check conditions
		if !g.evaluateConditions(def.Conditions) {
			continue
		}

		homepage := def.Integrations.Homepage
		groupName := homepage.Group
		if groupName == "" {
			groupName = "Services"
		}

		svc := HomepageService{
			Name:        def.Metadata.Name,
			Icon:        homepage.Icon,
			Description: homepage.Description,
			Container:   fmt.Sprintf("sdbx-%s", def.Metadata.Name),
		}

		// Build URL
		svc.Href = g.getServiceURL(def)

		groups[groupName] = append(groups[groupName], svc)
	}

	// Convert to YAML structure
	// Homepage uses a specific YAML format: list of maps with group name as key
	var result []map[string][]map[string]interface{}

	// Define group order
	groupOrder := []string{"Media", "Downloads", "Management", "Services"}

	for _, groupName := range groupOrder {
		services, ok := groups[groupName]
		if !ok || len(services) == 0 {
			continue
		}

		var svcList []map[string]interface{}
		for _, svc := range services {
			svcEntry := map[string]interface{}{
				svc.Name: map[string]interface{}{
					"icon":        svc.Icon,
					"href":        svc.Href,
					"description": svc.Description,
					"container":   svc.Container,
				},
			}
			svcList = append(svcList, svcEntry)
		}

		result = append(result, map[string][]map[string]interface{}{
			groupName: svcList,
		})
	}

	return yaml.Marshal(result)
}

// CloudflaredConfig represents cloudflared config.yml
type CloudflaredConfig struct {
	Tunnel  string            `yaml:"tunnel"`
	Ingress []CloudflaredRule `yaml:"ingress"`
}

// CloudflaredRule represents a single ingress rule
type CloudflaredRule struct {
	Hostname string `yaml:"hostname,omitempty"`
	Service  string `yaml:"service"`
}

// GenerateCloudflaredConfig generates cloudflared config.yml content
func (g *IntegrationsGenerator) GenerateCloudflaredConfig(graph *registry.ResolutionGraph) ([]byte, error) {
	cfg := CloudflaredConfig{
		Tunnel:  "sdbx",
		Ingress: []CloudflaredRule{},
	}

	// Process services
	for _, serviceName := range graph.Order {
		resolved := graph.Services[serviceName]
		if !resolved.Enabled {
			continue
		}

		def := resolved.FinalDefinition

		// Check if service should be exposed via cloudflared
		if def.Integrations.Cloudflared == nil || !def.Integrations.Cloudflared.Enabled {
			continue
		}

		// Check conditions
		if !g.evaluateConditions(def.Conditions) {
			continue
		}

		// Only routed services
		if !def.Routing.Enabled {
			continue
		}

		// Determine hostname
		var hostname string
		if def.Routing.ForceSubdomain || g.Config.Routing.Strategy == config.RoutingStrategySubdomain {
			hostname = fmt.Sprintf("%s.%s", def.Routing.Subdomain, g.Config.Domain)
		} else {
			hostname = fmt.Sprintf("%s.%s", g.Config.Routing.BaseDomain, g.Config.Domain)
		}

		cfg.Ingress = append(cfg.Ingress, CloudflaredRule{
			Hostname: hostname,
			Service:  "http://traefik:80",
		})
	}

	// Add catch-all rule (required by cloudflared)
	cfg.Ingress = append(cfg.Ingress, CloudflaredRule{
		Service: "http_status:404",
	})

	return yaml.Marshal(cfg)
}

// TraefikDynamicConfig represents traefik dynamic configuration
type TraefikDynamicConfig struct {
	HTTP TraefikHTTP `yaml:"http"`
}

// TraefikHTTP represents Traefik HTTP configuration
type TraefikHTTP struct {
	Middlewares map[string]TraefikMiddleware `yaml:"middlewares"`
}

// TraefikMiddleware represents a Traefik middleware
type TraefikMiddleware struct {
	StripPrefix *StripPrefixMiddleware `yaml:"stripPrefix,omitempty"`
	ForwardAuth *ForwardAuthMiddleware `yaml:"forwardAuth,omitempty"`
}

// StripPrefixMiddleware represents StripPrefix middleware config
type StripPrefixMiddleware struct {
	Prefixes []string `yaml:"prefixes"`
}

// ForwardAuthMiddleware represents ForwardAuth middleware config
type ForwardAuthMiddleware struct {
	Address             string   `yaml:"address"`
	TrustForwardHeader  bool     `yaml:"trustForwardHeader"`
	AuthResponseHeaders []string `yaml:"authResponseHeaders,omitempty"`
}

// GenerateTraefikDynamic generates traefik dynamic middlewares config
func (g *IntegrationsGenerator) GenerateTraefikDynamic(graph *registry.ResolutionGraph) ([]byte, error) {
	cfg := TraefikDynamicConfig{
		HTTP: TraefikHTTP{
			Middlewares: make(map[string]TraefikMiddleware),
		},
	}

	// Add Authelia forward auth middleware
	var authAddr string
	if g.Config.Routing.Strategy == config.RoutingStrategyPath {
		authAddr = fmt.Sprintf("http://authelia:9091/api/verify?rd=https://%s.%s/auth/",
			g.Config.Routing.BaseDomain, g.Config.Domain)
	} else {
		authAddr = fmt.Sprintf("http://authelia:9091/api/verify?rd=https://auth.%s/",
			g.Config.Domain)
	}

	cfg.HTTP.Middlewares["authelia"] = TraefikMiddleware{
		ForwardAuth: &ForwardAuthMiddleware{
			Address:            authAddr,
			TrustForwardHeader: true,
			AuthResponseHeaders: []string{
				"Remote-User",
				"Remote-Groups",
				"Remote-Name",
				"Remote-Email",
			},
		},
	}

	// Add strip prefix middlewares for path routing
	if g.Config.Routing.Strategy == config.RoutingStrategyPath {
		for _, serviceName := range graph.Order {
			resolved := graph.Services[serviceName]
			if !resolved.Enabled {
				continue
			}

			def := resolved.FinalDefinition

			// Only for routed services using strip prefix
			if !def.Routing.Enabled || def.Routing.ForceSubdomain {
				continue
			}

			if def.Routing.PathRouting.Strategy != "stripPrefix" {
				continue
			}

			// Check conditions
			if !g.evaluateConditions(def.Conditions) {
				continue
			}

			middlewareName := fmt.Sprintf("strip-%s", def.Metadata.Name)
			cfg.HTTP.Middlewares[middlewareName] = TraefikMiddleware{
				StripPrefix: &StripPrefixMiddleware{
					Prefixes: []string{def.Routing.Path},
				},
			}
		}
	}

	return yaml.Marshal(cfg)
}

// AutheliaAccessRule represents an Authelia access control rule
type AutheliaAccessRule struct {
	Domain string `yaml:"domain"`
	Policy string `yaml:"policy"`
}

// GenerateAutheliaAccessRules generates Authelia access control rules
func (g *IntegrationsGenerator) GenerateAutheliaAccessRules(graph *registry.ResolutionGraph) ([]AutheliaAccessRule, error) {
	var rules []AutheliaAccessRule

	for _, serviceName := range graph.Order {
		resolved := graph.Services[serviceName]
		if !resolved.Enabled {
			continue
		}

		def := resolved.FinalDefinition

		// Only routed services with auth
		if !def.Routing.Enabled {
			continue
		}

		// Check conditions
		if !g.evaluateConditions(def.Conditions) {
			continue
		}

		// Determine domain
		var domain string
		if def.Routing.ForceSubdomain || g.Config.Routing.Strategy == config.RoutingStrategySubdomain {
			domain = fmt.Sprintf("%s.%s", def.Routing.Subdomain, g.Config.Domain)
		} else {
			domain = fmt.Sprintf("%s.%s", g.Config.Routing.BaseDomain, g.Config.Domain)
		}

		// Determine policy
		policy := "one_factor"
		if def.Routing.Auth.Bypass {
			policy = "bypass"
		}

		rules = append(rules, AutheliaAccessRule{
			Domain: domain,
			Policy: policy,
		})
	}

	return rules, nil
}

// getServiceURL returns the full URL for a service
func (g *IntegrationsGenerator) getServiceURL(def *registry.ServiceDefinition) string {
	var scheme string
	if g.Config.Expose.Mode == config.ExposeModeLAN {
		scheme = "http"
	} else {
		scheme = "https"
	}

	if def.Routing.ForceSubdomain || g.Config.Routing.Strategy == config.RoutingStrategySubdomain {
		return fmt.Sprintf("%s://%s.%s", scheme, def.Routing.Subdomain, g.Config.Domain)
	}

	return fmt.Sprintf("%s://%s.%s%s", scheme, g.Config.Routing.BaseDomain, g.Config.Domain, def.Routing.Path)
}

// evaluateConditions checks if conditions are met
func (g *IntegrationsGenerator) evaluateConditions(cond registry.Conditions) bool {
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

// GenerateEnvFile generates the .env file content
func (g *IntegrationsGenerator) GenerateEnvFile(graph *registry.ResolutionGraph) ([]byte, error) {
	var lines []string

	lines = append(lines, "# SDBX Environment Configuration")
	lines = append(lines, "# Generated by sdbx init")
	lines = append(lines, "")

	// Core settings
	lines = append(lines, fmt.Sprintf("SDBX_DOMAIN=%s", g.Config.Domain))
	lines = append(lines, fmt.Sprintf("SDBX_EXPOSE_MODE=%s", g.Config.Expose.Mode))
	lines = append(lines, fmt.Sprintf("SDBX_TIMEZONE=%s", g.Config.Timezone))
	lines = append(lines, "")

	// Paths
	lines = append(lines, fmt.Sprintf("SDBX_CONFIG_PATH=%s", g.Config.ConfigPath))
	lines = append(lines, fmt.Sprintf("SDBX_DATA_PATH=%s", g.Config.DataPath))
	lines = append(lines, fmt.Sprintf("SDBX_DOWNLOADS_PATH=%s", g.Config.DownloadsPath))
	lines = append(lines, fmt.Sprintf("SDBX_MEDIA_PATH=%s", g.Config.MediaPath))
	lines = append(lines, "")

	// Permissions
	lines = append(lines, fmt.Sprintf("PUID=%d", g.Config.PUID))
	lines = append(lines, fmt.Sprintf("PGID=%d", g.Config.PGID))
	lines = append(lines, fmt.Sprintf("UMASK=%s", g.Config.Umask))
	lines = append(lines, "")

	// VPN
	if g.Config.VPNEnabled {
		lines = append(lines, fmt.Sprintf("SDBX_VPN_PROVIDER=%s", g.Config.VPNProvider))
		lines = append(lines, fmt.Sprintf("SDBX_VPN_COUNTRY=%s", g.Config.VPNCountry))
		lines = append(lines, "")
	}

	// TLS
	if g.Config.Expose.Mode == config.ExposeModeDirect {
		lines = append(lines, fmt.Sprintf("TRAEFIK_ACME_EMAIL=%s", g.Config.Expose.TLS.Email))
		lines = append(lines, "")
	}

	// Plex claim (user fills in)
	lines = append(lines, "# Get your Plex claim token from https://plex.tv/claim")
	lines = append(lines, "PLEX_CLAIM=")
	lines = append(lines, "")

	// Enabled addons
	if len(g.Config.Addons) > 0 {
		lines = append(lines, fmt.Sprintf("# Addons: %s", strings.Join(g.Config.Addons, ", ")))
	}

	return []byte(strings.Join(lines, "\n") + "\n"), nil
}
