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
    init.go            # Interactive wizard for project bootstrapping (7-step with progress)
    up.go, down.go     # Docker Compose lifecycle
    doctor.go          # Diagnostic checks (with CheckList TUI)
    status.go          # Service status display (with Table TUI)
    addon.go           # Addon management (search, enable, disable)
    source.go          # Source management (add, remove, list, update)
    lock.go            # Lock file management (lock, verify, diff)
    config.go          # Configuration get/set
    vpn.go             # VPN configuration (configure, status, providers)

internal/
  backup/              # Backup/restore functionality (tar.gz archives with metadata)
  config/              # Configuration structs and loaders (Load, Save, Validate)
    config.go          # Main Config struct with VPN credentials
    vpn_providers.go   # VPN provider definitions (17 providers with auth types)
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
      core/            # Core services (7): traefik, authelia, plex, qbittorrent, gluetun, cloudflared, sdbx-webui
                       # NOTE: All addons (27) are in Git source only, not embedded
  tui/                 # Terminal UI styles and components
    styles.go          # Lipgloss styles, icons, colors, render helpers
    spinner.go         # Animated spinner for long operations
    table.go           # Table rendering with auto-width columns and badges
    progress.go        # Step progress tracker and CheckList component
  web/                 # Web UI server (htmx + Go templates + WebSockets)
    server.go          # HTTP server with two-phase detection (pre-init vs post-init)
    embed.go           # go:embed directives for static assets and templates
    handlers/          # HTTP request handlers
      setup.go         # 7-step setup wizard (replaces `sdbx init`)
      dashboard.go     # Dashboard with service status
      services.go      # Service management (start/stop/restart)
      logs.go          # WebSocket log streaming
      addons.go        # Addon catalog and management
      config.go        # YAML configuration editor
      integration.go   # Integration center and backup management (uses internal/backup)
      common.go        # Shared utility functions
    middleware/        # HTTP middleware
      auth.go          # Two-phase auth (token for pre-init, Authelia for post-init)
      logging.go       # Request logging
      recovery.go      # Panic recovery
    templates/         # Go html/template files
      layouts/         # Base layouts (base.html, wizard.html)
      pages/           # Page templates (dashboard, services, logs, etc.)
      components/      # Reusable components (placeholders)
    static/            # Static assets (go:embed)
      css/             # Stylesheets (colors.css from TUI palette, main.css)
      js/              # JavaScript (htmx.min.js, websocket.js)
      icons/           # SVG icons (placeholders)
```

### Key Architectural Patterns

**1. Registry-Based Service Definitions**
- Services are defined in YAML files with a schema similar to Kubernetes/Helm
- Each service definition includes: metadata, container spec, routing, integrations, conditions
- **Embedded source** bundles 7 core services into the binary (essential infrastructure + web UI)
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
- **Embedded source** (priority -1) contains 7 core services, available offline as fallback
- **Official Git source** (priority 0) contains all 27 addons - auto-added on first run
- **Local source** (~/.config/sdbx/services, priority 100) can override anything
- Git sources can be added with `sdbx source add <name> <url>`
- Source config stored in `~/.config/sdbx/sources.yaml`
- **Official services repository**: https://github.com/maiko/SDBX-Services (7 core + 27 addons)

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

**8. Service Interconnection**
- Services communicate using Docker hostnames following the pattern `sdbx-{servicename}`
- Example: Sonarr connects to qBittorrent using `http://sdbx-qbittorrent:8080`
- Services must be manually configured to connect to each other
- API keys can be found in service config files (`configs/<service>/config.xml`)
- See `docs/service-interconnection.md` for detailed hostname reference

**9. Web UI Architecture (Two-Phase Deployment)**
- **Pre-init phase**: `sdbx serve` runs embedded HTTP server for setup wizard
  - Binds to 0.0.0.0:3000 for remote/headless setup
  - Generates one-time 256-bit crypto/rand token, displayed in CLI
  - Token-based authentication (query param + HttpOnly cookie)
  - Setup wizard replaces `sdbx init` CLI command
  - Creates `.sdbx.yaml`, generates compose.yaml, initializes Authelia users
- **Post-init phase**: Web UI runs as Docker service (sdbx-webui)
  - Deployed behind Traefik + Authelia (subdomain: sdbx.domain.tld)
  - Trusts Authelia Remote-User header for authentication
  - Replaces homepage addon as primary dashboard
- **Technology stack**: htmx (no heavy JS), Go html/template, WebSockets (logs only)
- **Features**: Dashboard, service control, live logs, addon management, config editor, integration center, backup/restore
- **Design**: Minimal aesthetic inspired by Charm.land, TUI color palette ported to CSS
- **go:embed**: All templates, CSS, and JS bundled in binary

### Important Implementation Details

**VPN Enforcement**
- qBittorrent container uses `network_mode: service:gluetun` to route all traffic through VPN
- If VPN drops, qBittorrent has no network access (kill-switch)

**VPN Network Sharing & Routing Pass-Through**
- When VPN is enabled, qBittorrent uses `network_mode: service:gluetun` (shares network namespace)
- This provides kill-switch protection: if VPN drops, qBittorrent has no network access
- ComposeGenerator automatically transfers Traefik labels from qBittorrent to gluetun
- Gluetun exposes qBittorrent's ports (8080, 6881) and handles routing
- Pattern is generalizable: any service using `network_mode: service:X` gets routing pass-through
- Implementation: `transferLabelsForNetworkSharing()` in `internal/generator/compose.go`
- Non-Traefik labels (watchtower) remain on original service
- When VPN is disabled, qBittorrent uses normal bridge networking with labels on itself

**Path vs Subdomain Routing**
- Path routing requires services to support base path configuration
- Traefik middlewares handle path stripping: `StripPrefix` or `server.path` depending on service
- Authelia uses `server.path` instead of `StripPrefix` for proper path routing

**Authelia Integration**
- All services except Plex/Homepage require authentication via Traefik middleware
- User database stored in `configs/authelia/users_database.yml` with Argon2 hashed passwords
- Admin credentials configured during `init` wizard

## CLI Commands Reference

### Web UI
```bash
sdbx serve                          # Start web UI server
sdbx serve --host 0.0.0.0           # Bind to all interfaces (default)
sdbx serve --port 3000              # Custom port (default: 3000)
```

**Pre-init mode** (no .sdbx.yaml exists):
- Displays setup token URL in CLI: `http://192.168.1.100:3000?token=abc123`
- Token required for access (256-bit crypto/rand)
- Serves 7-step setup wizard
- Creates project configuration on completion

**Post-init mode** (.sdbx.yaml exists):
- Serves dashboard and management UI
- Development mode: Direct access (shows warning)
- Production mode: Deploy as Docker service behind Traefik + Authelia

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

### Service Interconnection
Services communicate using Docker hostnames following the pattern `sdbx-{servicename}`.

Examples:
- Sonarr → qBittorrent: `http://sdbx-qbittorrent:8080`
- Prowlarr → Sonarr: `http://sdbx-sonarr:8989`
- Radarr → qBittorrent: `http://sdbx-qbittorrent:8080`

See `docs/service-interconnection.md` for complete hostname reference and configuration examples.

### VPN Configuration
```bash
sdbx vpn configure                  # Interactive VPN credential configuration
sdbx vpn status                     # Show VPN configuration status
sdbx vpn providers                  # List supported VPN providers with auth types
```

Supported VPN providers (17 total) with different authentication types:
- **Username/Password**: NordVPN, Surfshark, ExpressVPN, IPVanish, CyberGhost, TorGuard, VyprVPN, PureVPN, HMA, PIA
- **Token/Key**: Mullvad (account number), AirVPN (device key), IVPN (account ID)
- **Service Credentials**: ProtonVPN (OpenVPN creds from dashboard)
- **Wireguard**: Mullvad, ProtonVPN, AirVPN, IVPN (generate keys via provider dashboard)
- **Config File**: Custom OpenVPN configuration

The init wizard collects provider-specific credentials and generates the appropriate `gluetun.env` file.

### Backup Management
```bash
sdbx backup                         # Create timestamped backup
sdbx backup list                    # List all backups with metadata
sdbx backup restore <name>          # Restore from backup
sdbx backup delete <name>           # Delete backup
```

Backups are stored in `./backups/` as tar.gz archives containing `.sdbx.yaml`, `.sdbx.lock`, `compose.yaml`, `secrets/`, and `configs/`.

## Testing Notes

- Go 1.25.5+ required (see go.mod)
- Tests across 10+ packages including registry, web, tui
- Package coverage: tui (100%), generator (82%), web (80%), registry (76%), secrets (76%), doctor (71%), config (43%), cmd (34%)
- Web tests cover template loading, phase detection, token generation, health endpoint
- TUI tests cover table rendering, progress tracking, checklist, badges
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
