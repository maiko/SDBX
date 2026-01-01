# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SDBX (Seedbox in a Box) is a Go CLI tool that bootstraps and manages a production-ready, security-first media automation stack. It generates Docker Compose configurations and manages a complete seedbox environment with VPN, authentication, and media management services.

**Key characteristics:**
- CLI built with Cobra + Viper for configuration management
- Interactive TUI wizards using Charmbracelet libraries (huh, lipgloss, bubbletea)
- **Registry-based service definitions** - Services are defined in YAML files (like Helm charts)
- **Multiple sources support** - Git repositories as "taps" (like Homebrew)
- Docker Compose orchestration for service management
- Security-first: SSO authentication (Authelia), VPN enforcement (Gluetun), zero-trust options (Cloudflare Tunnel)

## Development Commands

### Building & Running
```bash
make build              # Build the sdbx binary to bin/
make run ARGS="init"    # Run CLI with arguments (e.g., sdbx init)
make install            # Install to GOPATH/bin
make build-all          # Cross-compile for all platforms (linux/darwin, amd64/arm64)
```

### Testing & Quality
```bash
make test               # Run all tests
make test-coverage      # Generate coverage report (coverage.html)
go test -v ./internal/config/...  # Run tests for a specific package
go test -v -run TestValidate ./internal/config/...  # Run a single test
make lint               # Run golangci-lint
make fmt                # Format code with gofmt + goimports
```

### Release
```bash
make release-snapshot   # Test release locally (creates binaries in dist/)
make release            # Create production release via goreleaser
```

## Code Architecture

### Project Structure
```
cmd/sdbx/
  main.go              # Entry point, sets version info (version, commit, date)
  cmd/                 # Cobra command definitions
    root.go            # Root command + global flags (--no-tui, --json, --config)
    init.go            # Interactive wizard for project bootstrapping
    up.go, down.go     # Docker Compose lifecycle
    doctor.go          # Diagnostic checks
    status.go          # Service status display
    addon.go           # Addon management (search, enable, disable)
    source.go          # Source management (add, remove, list, update)
    lock.go            # Lock file management (lock, verify, diff)
    secrets.go         # Secret generation/rotation
    config.go          # Configuration get/set

internal/
  config/              # Configuration structs and loaders (Load, Save, Validate)
  secrets/             # Secret generation with crypto/rand, rotation with backups
  docker/              # Docker Compose wrapper (up, down, ps, logs, exec)
  doctor/              # Health checks (Docker, disk space, ports, permissions)
  generator/           # Compose and config file generation
    generator.go       # Main generator orchestrating all generation
    compose.go         # Docker Compose generation from registry
    integrations.go    # Homepage, Cloudflared, Traefik dynamic config generation
    templates/         # Static config templates (Authelia, Traefik static, etc.)
  registry/            # Service definition registry system
    types.go           # ServiceDefinition, Source, LockFile structs
    registry.go        # Main Registry interface and source management
    loader.go          # YAML loading and parsing
    validator.go       # Service definition validation
    resolver.go        # Service resolution with dependency ordering
    source.go          # Source interface + LocalSource
    git.go             # Git source implementation
    embedded.go        # Embedded source for bundled services
    cache.go           # Source caching
    lock.go            # Lock file management
    services/          # Embedded service definitions (YAML)
      core/            # Core services ONLY (6): traefik, authelia, plex, qbittorrent, gluetun, cloudflared
                       # NOTE: All addons (27) are in Git source only, not embedded
  tui/                 # Terminal UI styles and helpers
```

### Key Architectural Patterns

**1. Registry-Based Service Definitions**
- Services are defined in YAML files with a schema similar to Kubernetes/Helm
- Each service definition includes: metadata, container spec, routing, integrations, conditions
- **Embedded source** bundles only 6 core services into the binary (essential infrastructure for offline init)
- **Git source** (https://github.com/maiko/SDBX-Services) contains all 27 addons
- Multiple Git sources can be added like Homebrew taps
- Lock files (`.sdbx.lock`) pin versions for reproducibility

Example service definition (from Git source, e.g., `addons/sonarr/service.yaml`):
```yaml
apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: sonarr
  version: 1.0.0
  category: media
  description: "TV Shows automation and management"
spec:
  image:
    repository: linuxserver/sonarr
    tag: latest
  container:
    name_template: "sdbx-{{ .Name }}"
  environment:
    static:
      - name: TZ
        value: "{{ .Config.Timezone }}"
routing:
  enabled: true
  port: 8989
  auth:
    required: true
conditions:
  requireAddon: true  # Addon service, requires explicit enable
```

**2. Source Management**
- Sources are Git repositories or local directories containing service definitions
- **Embedded source** (priority -1) contains ONLY 6 core services, available offline as fallback
- **Official Git source** (priority 0) contains all 27 addons - auto-added on first run
- **Local source** (~/.config/sdbx/services, priority 100) can override anything
- Git sources can be added with `sdbx source add <name> <url>`
- Source config stored in `~/.config/sdbx/sources.yaml`
- **Official services repository**: https://github.com/maiko/SDBX-Services (6 core + 27 addons)

**3. Generator Pipeline**
- `init` command collects user preferences via TUI wizard
- Registry resolves services based on config (addons, VPN, etc.)
- ComposeGenerator creates `compose.yaml` from resolved services
- IntegrationsGenerator creates homepage, cloudflared, traefik configs
- Static templates handle service-specific configs (Authelia, etc.)

**4. Docker Compose Orchestration**
- `internal/docker/compose.go` wraps `docker compose` commands
- All operations use context for cancellation and timeouts
- Service health checks use `docker compose ps --format json` for structured output

**5. TUI Mode Detection**
- Commands respect `--no-tui` and `--json` flags
- Interactive wizards only run when `IsTUIEnabled()` returns true
- JSON output mode available for scripting/automation

**6. Configuration Management**
- Viper loads config from `.sdbx.yaml` or `--config` flag
- Environment variables with `SDBX_` prefix override config
- Three exposure modes: `lan` (local), `direct` (public), `cloudflared` (tunnel)
- Two routing strategies: `subdomain` (radarr.domain.tld) vs `path` (domain.tld/radarr)

**7. Secrets Management**
- Secrets stored in `secrets/*.txt` files (JWT secret, session secret, etc.)
- Generated with `crypto/rand` for cryptographic security
- Rotation creates timestamped backups before overwriting
- Never committed to git (`.gitignore` includes `secrets/`)

### Important Implementation Details

**VPN Enforcement**
- qBittorrent container uses `network_mode: service:gluetun` to route all traffic through VPN
- If VPN drops, qBittorrent has no network access (kill-switch)

**Path vs Subdomain Routing**
- Path routing requires services to support base path configuration
- Traefik middlewares handle path stripping: `StripPrefix` or `server.path` depending on service
- Authelia uses `server.path` instead of `StripPrefix` for proper path routing

**Authelia Integration**
- All services except Plex/Homepage require authentication via Traefik middleware
- User database stored in `configs/authelia/users_database.yml` with Argon2 hashed passwords
- Admin credentials configured during `init` wizard

## CLI Commands Reference

### Source Management
```bash
sdbx source list                    # List configured sources
sdbx source add <name> <url>        # Add a Git source
sdbx source remove <name>           # Remove a source
sdbx source update [name]           # Update sources (pull latest)
sdbx source info <name>             # Show source details
```

### Addon Management
```bash
sdbx addon list [--all]             # List enabled/all addons
sdbx addon search <query>           # Search for addons
sdbx addon info <name>              # Show addon details
sdbx addon enable <name>            # Enable an addon
sdbx addon disable <name>           # Disable an addon
```

### Lock File Management
```bash
sdbx lock                           # Generate/update lock file
sdbx lock verify                    # Verify lock file integrity
sdbx lock diff                      # Show differences from lock
sdbx lock update [service...]       # Update services in lock
```

## Testing Notes

- Go 1.25.5+ required (see go.mod)
- Tests across 8 packages including registry
- Package coverage: tui (100%), generator (82%), registry, secrets (76%), doctor (71%), config (43%), cmd (34%)
- No integration tests for Docker Compose operations (would require Docker in CI)

## Common Patterns

**Adding a New Command**
1. Create `cmd/sdbx/cmd/<command>.go`
2. Define `cobra.Command` with Use, Short, Long, RunE
3. Register with `rootCmd.AddCommand()` in `init()`
4. Use `IsTUIEnabled()` for conditional TUI rendering
5. Use `IsJSONOutput()` for structured output mode

**Adding a New Service Definition**
1. Create `internal/registry/services/{core|addons}/<name>/service.yaml`
2. Define service with apiVersion, kind, metadata, spec, routing, conditions
3. Set `conditions.always: true` for core, `conditions.requireAddon: true` for addons
4. Add to `internal/registry/embedded_test.go` expected services list
5. Run `go test ./internal/registry/...` to verify

**Service Definition Schema**
```yaml
apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: string           # Service identifier
  version: string        # Definition version (semver)
  category: string       # media, downloads, management, utility, networking, auth
  description: string    # Human-readable description
spec:
  image:
    repository: string   # Docker image repository
    tag: string          # Docker image tag
  container:
    name_template: string # Container name template
    restart: string       # Restart policy
  environment:
    static: []           # Always-applied env vars
    conditional: []      # Condition-based env vars
  volumes: []            # Volume mounts
  ports:
    static: []           # Always-exposed ports
    conditional: []      # Condition-based ports
  networking:
    mode: string         # bridge, host, or service:<name>
    networks: []         # Networks to join
routing:
  enabled: bool          # Whether service has web UI
  port: int              # Internal port
  subdomain: string      # For subdomain routing
  path: string           # For path routing
  auth:
    required: bool       # Whether auth is required
    bypass: bool         # Bypass auth for this service
conditions:
  always: bool           # Core service (always enabled)
  requireAddon: bool     # Addon (requires explicit enable)
  requireConfig: string  # Config condition (e.g., "vpn_enabled")
integrations:
  homepage:              # Homepage dashboard integration
    enabled: bool
    group: string
    icon: string
  cloudflared:           # Cloudflare Tunnel integration
    enabled: bool
  watchtower:            # Auto-update integration
    enabled: bool
```

**Doctor Check Implementation**
- Add check function to `internal/doctor/checks.go` with signature `func (d *Doctor) checkX(context.Context) (bool, string)`
- Register in `RunAll()` slice
- Return true + success message or false + error description
