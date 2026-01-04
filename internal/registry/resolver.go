package registry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/maiko/sdbx/internal/config"
)

// Resolver handles service resolution and dependency ordering
type Resolver struct {
	registry *Registry
	loader   *Loader
}

// NewResolver creates a new Resolver
func NewResolver(registry *Registry) *Resolver {
	return &Resolver{
		registry: registry,
		loader:   NewLoader(),
	}
}

// Resolve resolves all services based on configuration
func (r *Resolver) Resolve(ctx context.Context, cfg *config.Config) (*ResolutionGraph, error) {
	graph := &ResolutionGraph{
		Services: make(map[string]*ResolvedService),
		Errors:   make([]ResolutionError, 0),
	}

	// Get all available services
	services, err := r.registry.ListServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	// Build service map
	serviceMap := make(map[string]ServiceInfo)
	for _, svc := range services {
		serviceMap[svc.Name] = svc
	}

	// Determine which services to include
	enabledServices := r.determineEnabledServices(ctx, cfg, serviceMap)

	// Resolve each enabled service
	for serviceName := range enabledServices {
		if err := r.resolveService(ctx, cfg, graph, serviceName); err != nil {
			graph.Errors = append(graph.Errors, ResolutionError{
				Service: serviceName,
				Message: "failed to resolve",
				Cause:   err,
			})
		}
	}

	// Calculate dependency order
	order, err := r.topologicalSort(graph)
	if err != nil {
		graph.Errors = append(graph.Errors, ResolutionError{
			Service: "",
			Message: "dependency resolution failed",
			Cause:   err,
		})
	}
	graph.Order = order

	return graph, nil
}

// determineEnabledServices determines which services should be enabled
func (r *Resolver) determineEnabledServices(_ context.Context, cfg *config.Config, serviceMap map[string]ServiceInfo) map[string]bool {
	enabled := make(map[string]bool)

	for name, svc := range serviceMap {
		// Core services (not addons) are always candidates
		if !svc.IsAddon {
			enabled[name] = true
			continue
		}

		// Addons need to be explicitly enabled
		if cfg.IsAddonEnabled(name) {
			enabled[name] = true
		}
	}

	return enabled
}

// resolveService resolves a single service
func (r *Resolver) resolveService(ctx context.Context, cfg *config.Config, graph *ResolutionGraph, serviceName string) error {
	// Check if already resolved
	if _, exists := graph.Services[serviceName]; exists {
		return nil
	}

	// Get service definition
	def, source, err := r.registry.GetService(ctx, serviceName)
	if err != nil {
		return err
	}

	// Check conditions
	if !r.evaluateConditions(def.Conditions, cfg) {
		return nil // Service doesn't meet conditions
	}

	// Calculate definition hash
	hash := r.calculateHash(def)

	// Look for overrides (optional)
	overrides := r.loadOverrides(ctx, serviceName)

	// Merge overrides to get final definition
	finalDef := def
	for _, override := range overrides {
		finalDef = r.loader.MergeOverride(finalDef, override)
	}

	// Get source path
	sourceProvider, _ := r.registry.GetSource(source)
	sourcePath := ""
	if sourceProvider != nil {
		sourcePath = sourceProvider.GetServicePath(serviceName)
	}

	// Create resolved service
	resolved := &ResolvedService{
		Name:            serviceName,
		Source:          source,
		SourcePath:      sourcePath,
		Definition:      def,
		DefinitionHash:  hash,
		Overrides:       overrides,
		FinalDefinition: finalDef,
		Dependencies:    r.collectDependencies(finalDef, cfg),
		Enabled:         true,
	}

	graph.Services[serviceName] = resolved

	// Recursively resolve dependencies
	for _, depName := range resolved.Dependencies {
		if err := r.resolveService(ctx, cfg, graph, depName); err != nil {
			// Dependency failed to resolve, but continue
			graph.Errors = append(graph.Errors, ResolutionError{
				Service: serviceName,
				Message: fmt.Sprintf("dependency %s failed", depName),
				Cause:   err,
			})
		}
	}

	return nil
}

// evaluateConditions checks if a service's conditions are met
func (r *Resolver) evaluateConditions(cond Conditions, cfg *config.Config) bool {
	// Always-on services
	if cond.Always {
		return true
	}

	// Config-based conditions
	if cond.RequireConfig != "" {
		switch cond.RequireConfig {
		case "vpn_enabled":
			if !cfg.VPNEnabled {
				return false
			}
		case "cloudflared":
			if cfg.Expose.Mode != config.ExposeModeCloudflared {
				return false
			}
		}
	}

	// Addon-based conditions are handled separately in determineEnabledServices

	return true
}

// collectDependencies collects all dependencies for a service
func (r *Resolver) collectDependencies(def *ServiceDefinition, cfg *config.Config) []string {
	deps := make(map[string]bool)

	// Required dependencies
	for _, dep := range def.Spec.Dependencies.Required {
		deps[dep] = true
	}

	// Conditional dependencies
	for _, dep := range def.Spec.Dependencies.Conditional {
		if r.evaluateConditionString(dep.When, cfg) {
			deps[dep.Name] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}

	return result
}

// evaluateConditionString evaluates a condition template string
func (r *Resolver) evaluateConditionString(condition string, cfg *config.Config) bool {
	if condition == "" {
		return true
	}

	// Simple condition evaluation
	// In a real implementation, this would use text/template
	switch condition {
	case "{{ .Config.VPNEnabled }}":
		return cfg.VPNEnabled
	case "{{ not .Config.VPNEnabled }}":
		return !cfg.VPNEnabled
	case "{{ eq .Config.Expose.Mode \"cloudflared\" }}":
		return cfg.Expose.Mode == config.ExposeModeCloudflared
	case "{{ eq .Config.Routing.Strategy \"path\" }}":
		return cfg.Routing.Strategy == config.RoutingStrategyPath
	default:
		// Log warning for unknown conditions and default to false
		// This prevents unrecognized conditions from silently enabling features
		log.Printf("Warning: unknown condition '%s', defaulting to false", condition)
		return false
	}
}

// loadOverrides loads all overrides for a service
func (r *Resolver) loadOverrides(_ context.Context, serviceName string) []*ServiceOverride {
	var overrides []*ServiceOverride

	// Get all sources and sort by priority (lowest first, so high priority wins when applied)
	sources := r.registry.Sources()
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Priority() < sources[j].Priority()
	})

	// Check each source for overrides
	for _, source := range sources {
		if !source.IsEnabled() {
			continue
		}

		// Get service path and derive override path
		servicePath := source.GetServicePath(serviceName)
		if servicePath == "" {
			continue
		}

		// Skip embedded sources (they start with "embedded://")
		if strings.HasPrefix(servicePath, "embedded://") {
			continue
		}

		// Derive override path from service path
		overridePath := filepath.Join(filepath.Dir(servicePath), "override.yaml")

		// Check if override file exists
		if _, err := os.Stat(overridePath); err != nil {
			continue // No override in this source
		}

		// Load the override
		override, err := r.loader.LoadServiceOverride(overridePath)
		if err != nil {
			// Log but don't fail - overrides are optional
			continue
		}

		// Verify override is for the correct service
		if override.Metadata.Name != serviceName {
			continue
		}

		overrides = append(overrides, override)
	}

	return overrides
}

// calculateHash calculates a hash of the service definition
func (r *Resolver) calculateHash(def *ServiceDefinition) string {
	data, _ := yaml.Marshal(def)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash[:8])
}

// topologicalSort performs topological sort on the dependency graph
func (r *Resolver) topologicalSort(graph *ResolutionGraph) ([]string, error) {
	// Build adjacency list
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for name, svc := range graph.Services {
		if _, exists := inDegree[name]; !exists {
			inDegree[name] = 0
		}
		// Only initialize adjList entry if not already set
		// (prevents overwriting entries created by processing dependencies of other services)
		if _, exists := adjList[name]; !exists {
			adjList[name] = []string{}
		}

		for _, dep := range svc.Dependencies {
			// Only count dependencies that are in our graph
			if _, exists := graph.Services[dep]; exists {
				adjList[dep] = append(adjList[dep], name)
				inDegree[name]++
			}
		}
	}

	// Kahn's algorithm
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var order []string
	for len(queue) > 0 {
		// Pop from queue
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		// Reduce in-degree for dependents
		for _, dependent := range adjList[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check for cycles
	if len(order) != len(graph.Services) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return order, nil
}

// ResolveService resolves a single service by name
func (r *Resolver) ResolveService(ctx context.Context, cfg *config.Config, serviceName string) (*ResolvedService, error) {
	graph := &ResolutionGraph{
		Services: make(map[string]*ResolvedService),
		Errors:   make([]ResolutionError, 0),
	}

	if err := r.resolveService(ctx, cfg, graph, serviceName); err != nil {
		return nil, err
	}

	resolved, exists := graph.Services[serviceName]
	if !exists {
		return nil, fmt.Errorf("service %s not found after resolution", serviceName)
	}

	return resolved, nil
}

// GetDependencyOrder returns services in dependency order
func GetDependencyOrder(graph *ResolutionGraph) []string {
	return graph.Order
}

// GetEnabledServices returns only enabled services
func GetEnabledServices(graph *ResolutionGraph) map[string]*ResolvedService {
	enabled := make(map[string]*ResolvedService)
	for name, svc := range graph.Services {
		if svc.Enabled {
			enabled[name] = svc
		}
	}
	return enabled
}
