package registry

import "github.com/maiko/sdbx/internal/config"

// EvaluateConditions checks whether a service's conditions are met given the
// current configuration. It returns true if the service should be included.
// Note: RequireAddon conditions are handled separately by the resolver.
func EvaluateConditions(cond Conditions, cfg *config.Config) bool {
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

	return true
}
