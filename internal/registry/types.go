// Package registry provides a service registry for managing SDBX service definitions.
// It supports multiple sources (Git, local), YAML-based service definitions,
// and version pinning through lock files.
package registry

import "time"

// API version for service definitions
const (
	APIVersion = "sdbx.io/v1"
	KindService = "Service"
	KindServiceOverride = "ServiceOverride"
	KindSourceRepository = "SourceRepository"
	KindSourceConfig = "SourceConfig"
	KindLockFile = "LockFile"
)

// ServiceCategory defines the category of a service
type ServiceCategory string

const (
	CategoryMedia       ServiceCategory = "media"
	CategoryDownloads   ServiceCategory = "downloads"
	CategoryManagement  ServiceCategory = "management"
	CategoryUtility     ServiceCategory = "utility"
	CategoryNetworking  ServiceCategory = "networking"
	CategoryAuth        ServiceCategory = "auth"
)

// ServiceDefinition represents a complete service definition loaded from YAML
type ServiceDefinition struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   ServiceMetadata `yaml:"metadata"`
	Spec       ServiceSpec     `yaml:"spec"`
	Routing    RoutingConfig   `yaml:"routing,omitempty"`
	Secrets    []SecretDef     `yaml:"secrets,omitempty"`
	Integrations Integrations  `yaml:"integrations,omitempty"`
	Conditions Conditions      `yaml:"conditions,omitempty"`
}

// ServiceMetadata contains service identification and descriptive information
type ServiceMetadata struct {
	Name          string          `yaml:"name"`
	Version       string          `yaml:"version"`
	Category      ServiceCategory `yaml:"category"`
	Description   string          `yaml:"description"`
	Homepage      string          `yaml:"homepage,omitempty"`
	Documentation string          `yaml:"documentation,omitempty"`
	Maintainer    string          `yaml:"maintainer,omitempty"`
	Tags          []string        `yaml:"tags,omitempty"`
}

// ServiceSpec defines the container and runtime configuration
type ServiceSpec struct {
	Image       ImageSpec       `yaml:"image"`
	Container   ContainerSpec   `yaml:"container"`
	Environment EnvironmentSpec `yaml:"environment,omitempty"`
	Volumes     []VolumeMount   `yaml:"volumes,omitempty"`
	Ports       PortSpec        `yaml:"ports,omitempty"`
	Networking  NetworkSpec     `yaml:"networking,omitempty"`
	HealthCheck *HealthCheck    `yaml:"healthcheck,omitempty"`
	Dependencies DependencySpec `yaml:"dependencies,omitempty"`
}

// ImageSpec defines the container image configuration
type ImageSpec struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
	Registry   string `yaml:"registry,omitempty"`
}

// ContainerSpec defines container runtime settings
type ContainerSpec struct {
	NameTemplate string         `yaml:"name_template"`
	Restart      string         `yaml:"restart,omitempty"`
	Privileged   bool           `yaml:"privileged,omitempty"`
	Capabilities CapabilitiesSpec `yaml:"capabilities,omitempty"`
	Devices      []string       `yaml:"devices,omitempty"`
}

// CapabilitiesSpec defines Linux capabilities to add or drop
type CapabilitiesSpec struct {
	Add  []string `yaml:"add,omitempty"`
	Drop []string `yaml:"drop,omitempty"`
}

// EnvironmentSpec defines environment variables for the service
type EnvironmentSpec struct {
	Static      []EnvVar          `yaml:"static,omitempty"`
	Conditional []ConditionalEnvVar `yaml:"conditional,omitempty"`
	EnvFile     []string          `yaml:"envFile,omitempty"`
}

// EnvVar represents a single environment variable
type EnvVar struct {
	Name      string       `yaml:"name"`
	Value     string       `yaml:"value,omitempty"`
	ValueFrom *ValueSource `yaml:"valueFrom,omitempty"`
}

// ConditionalEnvVar is an environment variable with a condition
type ConditionalEnvVar struct {
	EnvVar `yaml:",inline"`
	When   string `yaml:"when,omitempty"`
}

// ValueSource defines where to get a value from
type ValueSource struct {
	SecretRef string `yaml:"secretRef,omitempty"`
	ConfigRef string `yaml:"configRef,omitempty"`
}

// VolumeMount defines a volume mount for the container
type VolumeMount struct {
	Name          string `yaml:"name,omitempty"`
	HostPath      string `yaml:"hostPath"`
	ContainerPath string `yaml:"containerPath"`
	ReadOnly      bool   `yaml:"readOnly,omitempty"`
}

// PortSpec defines port mappings
type PortSpec struct {
	Static      []string           `yaml:"static,omitempty"`
	Conditional []ConditionalPort  `yaml:"conditional,omitempty"`
}

// ConditionalPort is a port mapping with a condition
type ConditionalPort struct {
	Port string `yaml:"port"`
	When string `yaml:"when"`
}

// NetworkSpec defines network configuration
type NetworkSpec struct {
	Networks     []NetworkRef `yaml:"networks,omitempty"`
	Mode         string       `yaml:"mode,omitempty"`
	ModeTemplate string       `yaml:"modeTemplate,omitempty"`
}

// NetworkRef is a network reference with optional condition
type NetworkRef struct {
	Name string `yaml:"name,omitempty"`
	When string `yaml:"when,omitempty"`
}

// HealthCheck defines container health check configuration
type HealthCheck struct {
	Test     []string `yaml:"test"`
	Interval string   `yaml:"interval,omitempty"`
	Timeout  string   `yaml:"timeout,omitempty"`
	Retries  int      `yaml:"retries,omitempty"`
	StartPeriod string `yaml:"start_period,omitempty"`
}

// DependencySpec defines service dependencies
type DependencySpec struct {
	Required    []string             `yaml:"required,omitempty"`
	Optional    []string             `yaml:"optional,omitempty"`
	Conditional []ConditionalDependency `yaml:"conditional,omitempty"`
}

// ConditionalDependency is a dependency with conditions
type ConditionalDependency struct {
	Name      string `yaml:"name"`
	Condition string `yaml:"condition,omitempty"`
	When      string `yaml:"when,omitempty"`
}

// RoutingConfig defines how the service is exposed via Traefik
type RoutingConfig struct {
	Enabled       bool              `yaml:"enabled"`
	Port          int               `yaml:"port,omitempty"`
	Subdomain     string            `yaml:"subdomain,omitempty"`
	Path          string            `yaml:"path,omitempty"`
	PathRouting   PathRoutingConfig `yaml:"pathRouting,omitempty"`
	Auth          AuthConfig        `yaml:"auth,omitempty"`
	ForceSubdomain bool             `yaml:"forceSubdomain,omitempty"`
	Traefik       TraefikConfig     `yaml:"traefik,omitempty"`
}

// PathRoutingConfig defines path-based routing behavior
type PathRoutingConfig struct {
	Strategy      string `yaml:"strategy,omitempty"`
	URLBaseEnvVar string `yaml:"urlBaseEnvVar,omitempty"`
}

// AuthConfig defines authentication requirements
type AuthConfig struct {
	Required bool `yaml:"required"`
	Bypass   bool `yaml:"bypass,omitempty"`
}

// TraefikConfig defines Traefik-specific labels
type TraefikConfig struct {
	Priority    *int     `yaml:"priority,omitempty"`
	Middlewares []string `yaml:"middlewares,omitempty"`
	CustomLabels map[string]string `yaml:"customLabels,omitempty"`
}

// SecretDef defines a secret required by the service
type SecretDef struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Length      int    `yaml:"length,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// Integrations defines how the service integrates with other components
type Integrations struct {
	Homepage    *HomepageIntegration   `yaml:"homepage,omitempty"`
	Cloudflared *CloudflaredIntegration `yaml:"cloudflared,omitempty"`
	Watchtower  *WatchtowerIntegration `yaml:"watchtower,omitempty"`
	Unpackerr   *UnpackerrIntegration  `yaml:"unpackerr,omitempty"`
}

// HomepageIntegration defines Homepage dashboard integration
type HomepageIntegration struct {
	Enabled     bool   `yaml:"enabled"`
	Group       string `yaml:"group,omitempty"`
	Icon        string `yaml:"icon,omitempty"`
	Description string `yaml:"description,omitempty"`
	Widget      *HomepageWidget `yaml:"widget,omitempty"`
}

// HomepageWidget defines a Homepage widget configuration
type HomepageWidget struct {
	Type   string            `yaml:"type,omitempty"`
	Fields map[string]string `yaml:"fields,omitempty"`
}

// CloudflaredIntegration defines Cloudflare Tunnel integration
type CloudflaredIntegration struct {
	Enabled bool `yaml:"enabled"`
}

// WatchtowerIntegration defines Watchtower auto-update integration
type WatchtowerIntegration struct {
	Enabled bool `yaml:"enabled"`
}

// UnpackerrIntegration defines Unpackerr integration for *arr services
type UnpackerrIntegration struct {
	Enabled      bool   `yaml:"enabled"`
	URLEnvVar    string `yaml:"urlEnvVar,omitempty"`
	APIKeyEnvVar string `yaml:"apiKeyEnvVar,omitempty"`
	InternalURL  string `yaml:"internalUrl,omitempty"`
}

// Conditions defines when a service should be included
type Conditions struct {
	Always        bool   `yaml:"always,omitempty"`
	RequireAddon  bool   `yaml:"requireAddon,omitempty"`
	RequireConfig string `yaml:"requireConfig,omitempty"`
	RequireFeature string `yaml:"requireFeature,omitempty"`
}

// ServiceOverride allows partial overrides of service definitions
type ServiceOverride struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   OverrideMetadata       `yaml:"metadata"`
	Spec       *ServiceSpecOverride   `yaml:"spec,omitempty"`
	Routing    *RoutingConfigOverride `yaml:"routing,omitempty"`
}

// OverrideMetadata identifies the service being overridden
type OverrideMetadata struct {
	Name string `yaml:"name"`
}

// ServiceSpecOverride allows partial spec overrides
type ServiceSpecOverride struct {
	Image       *ImageSpec             `yaml:"image,omitempty"`
	Environment *EnvironmentOverride   `yaml:"environment,omitempty"`
	Volumes     *VolumeOverride        `yaml:"volumes,omitempty"`
}

// EnvironmentOverride allows adding environment variables
type EnvironmentOverride struct {
	Additional []EnvVar `yaml:"additional,omitempty"`
}

// VolumeOverride allows adding volume mounts
type VolumeOverride struct {
	Additional []VolumeMount `yaml:"additional,omitempty"`
}

// RoutingConfigOverride allows overriding routing settings
type RoutingConfigOverride struct {
	Subdomain *string `yaml:"subdomain,omitempty"`
	Path      *string `yaml:"path,omitempty"`
}

// SourceConfig defines the user's source configuration
type SourceConfig struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   SourceConfigMetadata `yaml:"metadata"`
	Sources    []Source     `yaml:"sources"`
	Cache      CacheConfig  `yaml:"cache,omitempty"`
	Security   SecurityConfig `yaml:"security,omitempty"`
}

// SourceConfigMetadata contains version info
type SourceConfigMetadata struct {
	Version int `yaml:"version"`
}

// Source defines a service definition source
type Source struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	URL      string `yaml:"url,omitempty"`
	Path     string `yaml:"path,omitempty"`
	Branch   string `yaml:"branch,omitempty"`
	SSHKey   string `yaml:"ssh_key,omitempty"`
	Priority int    `yaml:"priority"`
	Enabled  bool   `yaml:"enabled"`
	Verified bool   `yaml:"verified,omitempty"`
}

// CacheConfig defines source caching settings
type CacheConfig struct {
	Directory string `yaml:"directory,omitempty"`
	TTL       string `yaml:"ttl,omitempty"`
}

// SecurityConfig defines security settings for sources
type SecurityConfig struct {
	AllowUnverified   bool                    `yaml:"allowUnverified,omitempty"`
	RequireSignatures bool                    `yaml:"requireSignatures,omitempty"`
	TrustLevels       map[string]TrustLevel   `yaml:"trustLevels,omitempty"`
}

// TrustLevel defines what a source is allowed to do
type TrustLevel struct {
	AllowPrivileged   bool     `yaml:"allowPrivileged,omitempty"`
	AllowHostNetwork  bool     `yaml:"allowHostNetwork,omitempty"`
	AllowCapabilities []string `yaml:"allowCapabilities,omitempty"`
	AllowedRegistries []string `yaml:"allowedRegistries,omitempty"`
}

// SourceRepository is metadata for a service repository
type SourceRepository struct {
	APIVersion    string               `yaml:"apiVersion"`
	Kind          string               `yaml:"kind"`
	Metadata      SourceRepositoryMeta `yaml:"metadata"`
	SchemaVersion string               `yaml:"schemaVersion"`
	MinCLIVersion string               `yaml:"minCliVersion,omitempty"`
	Categories    []string             `yaml:"categories,omitempty"`
}

// SourceRepositoryMeta contains repository identification
type SourceRepositoryMeta struct {
	Name        string       `yaml:"name"`
	Version     string       `yaml:"version"`
	Description string       `yaml:"description,omitempty"`
	Maintainers []Maintainer `yaml:"maintainers,omitempty"`
	License     string       `yaml:"license,omitempty"`
}

// Maintainer represents a repository maintainer
type Maintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email,omitempty"`
}

// LockFile represents a lock file for reproducible builds
type LockFile struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   LockFileMetadata `yaml:"metadata"`
	Sources    map[string]LockedSource `yaml:"sources"`
	Services   map[string]LockedService `yaml:"services"`
	InstallOrder []string       `yaml:"installOrder,omitempty"`
	GeneratedFiles map[string]string `yaml:"generatedFiles,omitempty"`
}

// LockFileMetadata contains lock file version info
type LockFileMetadata struct {
	Version     int       `yaml:"version"`
	GeneratedAt time.Time `yaml:"generatedAt"`
	CLIVersion  string    `yaml:"cliVersion"`
	ConfigHash  string    `yaml:"configHash"`
}

// LockedSource represents a pinned source
type LockedSource struct {
	URL       string    `yaml:"url"`
	Commit    string    `yaml:"commit"`
	Branch    string    `yaml:"branch,omitempty"`
	FetchedAt time.Time `yaml:"fetchedAt"`
}

// LockedService represents a pinned service
type LockedService struct {
	Source            string      `yaml:"source"`
	DefinitionVersion string      `yaml:"definitionVersion"`
	Image             LockedImage `yaml:"image"`
	ResolvedFrom      string      `yaml:"resolvedFrom"`
	Enabled           bool        `yaml:"enabled"`
}

// LockedImage represents a pinned container image
type LockedImage struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
	Digest     string `yaml:"digest,omitempty"`
}

// ResolvedService represents a fully resolved service ready for generation
type ResolvedService struct {
	Name            string
	Source          string
	SourcePath      string
	Definition      *ServiceDefinition
	DefinitionHash  string
	Overrides       []*ServiceOverride
	FinalDefinition *ServiceDefinition
	Dependencies    []string
	Enabled         bool
}

// ResolutionGraph represents the resolved dependency graph of services
type ResolutionGraph struct {
	Services map[string]*ResolvedService
	Order    []string
	Errors   []ResolutionError
}

// ResolutionError represents an error during service resolution
type ResolutionError struct {
	Service string
	Message string
	Cause   error
}

func (e ResolutionError) Error() string {
	if e.Cause != nil {
		return e.Service + ": " + e.Message + ": " + e.Cause.Error()
	}
	return e.Service + ": " + e.Message
}

// ValidationError represents a service definition validation error
type ValidationError struct {
	Field    string
	Message  string
	Severity string
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
