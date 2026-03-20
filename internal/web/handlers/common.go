package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/registry"
)

// formatServiceName formats a service name for display (converts kebab-case to Title Case)
func formatServiceName(name string) string {
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

// httpError logs the full error internally and returns a generic message to the client.
// This prevents exposing internal error details to users.
func httpError(w http.ResponseWriter, context string, err error, statusCode int) {
	log.Printf("Error [%s]: %v", context, err)
	http.Error(w, "An internal error occurred. Please try again later.", statusCode)
}

// jsonError logs the full error internally and returns a generic JSON error to the client.
// The userMessage is safe to show to clients; the err is only logged server-side.
func jsonError(w http.ResponseWriter, userMessage string, context string, err error, statusCode int) {
	log.Printf("Error [%s]: %v", context, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": userMessage,
	})
}

// respondJSON sends a JSON response with the given status code and data.
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// renderTemplate renders a named template with the given data.
// Output is buffered to prevent partial HTML on error.
func renderTemplate(tmpl *template.Template, w http.ResponseWriter, name string, ctx string, data interface{}) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		httpError(w, ctx+" template render", err, http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

// ServiceInfo represents service information for display across handlers.
type ServiceInfo struct {
	Name        string
	DisplayName string
	Status      string
	Health      string
	Running     bool
	Category    string
	Description string
	URL         string
	HasWebUI    bool
}

// buildServiceInfoMap creates a service map from registry metadata and Docker status.
func buildServiceInfoMap(compose *docker.Compose, reg *registry.Registry, ctx context.Context) (map[string]ServiceInfo, error) {
	dockerServices, err := compose.PS(ctx)
	if err != nil {
		log.Printf("Warning [buildServiceInfoMap]: Docker PS failed: %v", err)
		dockerServices = []docker.Service{}
	}

	registryServices, err := reg.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	// Load config for service URLs
	cfg, _ := config.Load()

	serviceMap := make(map[string]ServiceInfo)
	for _, regSvc := range registryServices {
		info := ServiceInfo{
			Name:        regSvc.Name,
			DisplayName: formatServiceName(regSvc.Name),
			Status:      "unknown",
			Health:      "",
			Running:     false,
			Category:    string(regSvc.Category),
			Description: regSvc.Description,
			HasWebUI:    regSvc.HasWebUI,
		}
		if cfg != nil && regSvc.HasWebUI {
			info.URL = cfg.GetServiceURL(regSvc.Name)
		}
		serviceMap[regSvc.Name] = info
	}

	for _, dockerSvc := range dockerServices {
		serviceName := strings.TrimPrefix(dockerSvc.Name, "sdbx-")
		if info, exists := serviceMap[serviceName]; exists {
			info.Status = dockerSvc.Status
			info.Health = dockerSvc.Health
			info.Running = dockerSvc.Running
			serviceMap[serviceName] = info
		}
	}

	return serviceMap, nil
}

// CategoryGroup represents a named group of services for stable ordering in templates.
type CategoryGroup struct {
	Name     string
	Services []ServiceInfo
}

// CategoryOrder defines the canonical display order for service categories.
var CategoryOrder = []string{"media", "downloads", "management", "auth", "networking", "utility", "other"}

// countRunningServices counts how many services are running.
func countRunningServices(services map[string]ServiceInfo) int {
	count := 0
	for _, svc := range services {
		if svc.Running {
			count++
		}
	}
	return count
}

// groupByCategory groups services by their category in a stable order.
func groupByCategory(serviceMap map[string]ServiceInfo) []CategoryGroup {
	byCategory := make(map[string][]ServiceInfo)
	for _, svc := range serviceMap {
		category := svc.Category
		if category == "" {
			category = "other"
		}
		byCategory[category] = append(byCategory[category], svc)
	}

	var groups []CategoryGroup
	for _, cat := range CategoryOrder {
		if services, ok := byCategory[cat]; ok {
			groups = append(groups, CategoryGroup{Name: cat, Services: services})
		}
	}
	// Include any categories not in CategoryOrder at the end
	for cat, services := range byCategory {
		found := false
		for _, known := range CategoryOrder {
			if cat == known {
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, CategoryGroup{Name: cat, Services: services})
		}
	}
	return groups
}
