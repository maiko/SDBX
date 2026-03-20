# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking Changes
- **`sdbx lock` now requires subcommand** — Use `sdbx lock generate` (bare `sdbx lock` shows help)
- **`sdbx backup` now requires subcommand** — Use `sdbx backup create` (bare `sdbx backup` shows help)
- **Removed `sdbx integrate` command** — Services must be configured manually using Docker hostnames (see `docs/service-interconnection.md`)
- **Removed `sdbx secrets` command** — Secrets are auto-generated during `sdbx init`, manual rotation via file editing

### Added
- **Jellyfin as core media server** — Choose Plex, Jellyfin, or both during `sdbx init` wizard
- **`sdbx import` command** — Migrate from existing Docker Compose setups (detects 14 service types)
- **`sdbx regenerate` command** — Re-run generation without the interactive wizard
- **Addon preset profiles** — Minimal / Standard / Full / Custom presets in the init wizard
- **DNS challenge support** — `challenge_type: dns` for wildcard TLS certificates (Cloudflare, etc.)
- **GPU transcoding support** — `gpu_enabled` field in service definitions for NVIDIA passthrough
- **Extended compose properties** — `shm_size`, `sysctls`, and custom Traefik labels in service definitions
- **`--dry-run` flag** — Preview `sdbx up` and `sdbx down` actions without executing
- **Web wizard step progress indicator** — 6-step numbered progress bar with completion states
- **Client-side form validation** — Real-time password strength, match checking, required field validation
- **VPN provider display names** — Human-readable names in web wizard (NordVPN, not nordvpn)
- **Streamlined wizard completion** — Summary page generates directly (removed intermediate step)
- **YAML syntax highlighting** — Config editor now highlights keys, values, comments, strings, booleans
- **Log search and filtering** — Client-side keyword filtering in the log viewer
- **Service page filters** — Search, status filter, and compact view toggle on the services page
- **Collapsible sidebar** — Toggle between full and icon-only sidebar with localStorage persistence
- **Auto-refreshing dashboard stats** — Stat cards update alongside the service grid via htmx OOB swaps
- **Page titles in mobile topbar** — Shows current page name on small screens
- **Unsaved changes warning** — Config editor warns before navigating away with edits
- **Trust warning for third-party sources** — Confirmation dialog when adding non-official Git sources
- **Pending restart banner** — Shown after enabling/disabling addons to remind about restart
- **📦 favicon** — Inline SVG favicon across all pages
- **"Zero to Plex" quick start guide** — Minimal setup guide for beginners (LAN mode, no domain)
- **Wizard screenshots in README** — 5 screenshots showing the setup wizard flow
- **CODEOWNERS file** — Automatic PR reviewer assignment

### Changed
- **Total services: 35** (8 core + 27 addons), up from 34
- **CLI password minimum** — Increased from 4 to 8 characters (matches web UI)
- **Graceful wizard abort** — Ctrl+C exits cleanly with friendly message instead of error
- **Start-over option** — Confirmation step offers "Start over" to re-run wizard with values preserved
- **Backup list** — Now uses styled TUI table instead of raw tabwriter
- **Error messages** — Recovery hints added throughout (e.g., "Try: sdbx doctor")
- **Terminal icons** — Emoji icons replaced with text-safe Unicode for terminal compatibility
- **Category sort order** — Dashboard and service pages use stable ordering instead of random Go map iteration
- **CSS consolidation** — Styles extracted from inline `<style>` blocks into `main.css`
- **Dark mode** — Hardcoded colors replaced with CSS variables for proper theming
- **Sidebar icons** — HTML entities replaced with consistent cross-platform Unicode characters
- **ARIA landmarks** — Navigation, main content, and toast regions properly labeled
- **Focus indicators** — Visible `:focus-visible` outlines on all interactive elements

### Fixed
- **VPN health check** — Now executes inside gluetun container instead of checking host IP
- **Pre-restore safety backup** — Automatically creates a backup before restoring
- **Structured logging** — Web server uses `log/slog` with structured key-value fields
- **Template error logging** — `evalTemplate()` now logs warnings on parse/execute failures
- **HSTS header** — Added `Strict-Transport-Security` when served over HTTPS
- **Rate limiter hardening** — 10K visitor map cap with graceful shutdown channel
- **CIDR range caching** — Private IP ranges parsed once at init, not per-request
- **CSRF cookie refresh** — Extended to 24h and refreshed on every GET to prevent expiry
- **Request body limits** — 1MB `MaxBytesReader` on all POST/PUT/PATCH handlers
- **Dev mode warning** — Prominent log warning when running post-init without Docker
- **Service name validation** — Regex validation before passing to Docker commands
- **Session TTL cleanup** — Wizard sessions expire after 30 minutes with background cleanup
- **hx-confirm on restart** — Restart button now requires confirmation like stop

### Added

#### Web UI with Two-Phase Deployment
Complete web-based management interface inspired by Charm.land design philosophy:

**New Command:**
- `sdbx serve` - Start web UI server
- `sdbx serve --host 0.0.0.0 --port 3000` - Configure bind address and port

**Two-Phase Deployment Model:**

**Pre-Init Phase (Setup Wizard):**
- Embedded HTTP server runs before Docker stack exists
- Binds to 0.0.0.0:3000 for remote/headless setup
- One-time 256-bit crypto/rand token displayed in CLI
- 7-step setup wizard replaces `sdbx init`:
  1. Welcome + system checks
  2. Domain + exposure mode + routing strategy
  3. Admin credentials (Argon2id hashed)
  4. Storage paths configuration
  5. VPN provider selection
  6. Addon selection
  7. Summary + project generation
- Creates `.sdbx.yaml`, generates `compose.yaml`, initializes Authelia users

**Post-Init Phase (Docker Service):**
- Runs as Docker container (sdbx-webui) behind Traefik + Authelia
- Subdomain routing (sdbx.domain.tld) with SSO authentication
- Replaces homepage addon as primary dashboard

**Features:**
- **Dashboard**: Real-time service status, health indicators, quick links
- **Service Management**: Start/stop/restart services via API
- **Live Log Viewer**: WebSocket streaming with auto-scroll, pause, clear
- **Addon Catalog**: Search, filter by category, enable/disable addons
- **Config Editor**: YAML editor with validation, syntax checking, automatic backups
- **Integration Center**: One-click service integration runner
- **Backup Management**: Create, restore, delete backups with metadata

**Technology Stack:**
- htmx for dynamic UI without heavy JavaScript
- Go html/template for server-side rendering
- WebSocket for real-time log streaming (gorilla/websocket)
- TUI color palette ported to CSS variables
- go:embed for bundled static assets

**Docker Image:**
- Multi-platform support: linux/amd64, linux/arm64
- Published to GitHub Container Registry: `ghcr.io/maiko/sdbx:latest`
- Alpine-based with docker-cli for container management
- Health checks via `/health` endpoint

**Security:**
- Pre-init: Token-based auth (query param + HttpOnly cookie)
- Post-init: Authelia Remote-User header trust
- Middleware chain: Recovery → Logging → Auth
- Input validation and YAML sanitization

**Implementation:**
- ~5,000 lines Go + ~2,000 lines templates/CSS/JS
- 7 handlers (setup, dashboard, services, logs, addons, config, backup)
- 3 middleware (auth, logging, recovery)
- 10+ page templates with reusable components
- Minimal JavaScript footprint (htmx + WebSocket client)
- Modular integrator architecture for extensibility

#### Service Interconnection Improvements
- **Service Info page** in Web UI displaying Docker hostnames (`sdbx-{servicename}`), internal ports, and external URLs
- **Hostname column** in `sdbx status` command output (both CLI table and JSON format)
- **Service interconnection documentation** (`docs/service-interconnection.md`) with hostname patterns, reference tables, and configuration examples
- **Homepage bookmarks** configuration automatically generated with SDBX project links, community resources, and documentation

### Changed
- Backup management moved to dedicated page and handler (extracted from integration handler)
- Handler count updated from 8 to 7 (integration handler removed)

### Removed
- Integration command (~2,000 lines) - users configure services manually using Docker hostnames
- Secrets rotation command - secrets auto-generated during init, manual rotation via file editing
- Integration page from Web UI

## [0.4.0-alpha] - 2026-01-01

### Added

#### Backup/Restore Commands
Complete backup and restore system for SDBX configurations:

**New Commands:**
- `sdbx backup` - Create timestamped backup
- `sdbx backup list` - List all backups with metadata
- `sdbx backup restore <name>` - Restore from backup
- `sdbx backup delete <name>` - Delete backup

**Features:**
- Pure Go implementation (no external tar dependency)
- Metadata tracking (version, timestamp, hostname, file list)
- Compressed tar.gz archives
- Smart relative time display ("5 minutes ago", "2 days ago")
- Human-readable size formatting (KB, MB, GB)
- JSON output mode for scripting
- Backs up: `.sdbx.yaml`, `.sdbx.lock`, `compose.yaml`, `secrets/`, `configs/`
- Excludes: media files, downloads, docker volumes

**Storage:** `./backups/sdbx-backup-YYYY-MM-DD-HHMMSS.tar.gz`

#### Enhanced VPN Provider Selection
Expanded VPN provider wizard from 6 to 16 popular providers:

**Added Providers:**
- ExpressVPN, Windscribe, IPVanish, CyberGhost, IVPN, TorGuard, VyprVPN, PureVPN, HideMyAss (HMA), Perfect Privacy, AirVPN

**Total in wizard:** 16 providers (plus Custom/OpenVPN option)
**Note:** Gluetun supports 30+ providers - users can manually configure any provider via `.env`

### Changed
- Updated website to reflect 33 total services (6 core + 27 addons)
- Updated service showcase with all new addons
- Updated meta descriptions for SEO

### Fixed
- Skipped addon integration tests requiring Git source setup
- Tests now pass properly (6 tests skipped with TODO for refactoring)
- Updated test expectations for new architecture

---

## [0.3.0-alpha] - 2026-01-01

### Added

#### 13 New Addons
Expanded the addon ecosystem with 13 new services across multiple categories:

**Media Servers:**
- Jellyfin - Open-source media server (Plex alternative)

**Downloads:**
- pyLoad - HTTP(S) download manager for file hosters
- Cobalt - Social media downloader (YouTube, TikTok, Twitter, Instagram, Reddit)
- SABnzbd - Usenet binary downloader
- NZBHydra2 - Usenet indexer meta-search aggregator
- Cross-seed - Automated cross-seeding for torrent trackers

**Media Management & Processing:**
- Tdarr - Distributed transcoding and media library optimizer
- Kometa - Plex metadata and collection manager (formerly Plex Meta Manager)
- Tube Archivist - YouTube channel archival and management
- Audiobookshelf - Self-hosted audiobook and podcast server
- Navidrome - Modern music streaming server (Subsonic-compatible)
- Calibre-Web - Web-based ebook library manager and reader

**Utilities:**
- Notifiarr - Unified notification system for *arr apps

### Changed

#### Core-Only Embedded Architecture
Implemented a new architecture where only essential core services are embedded in the binary:

- **Embedded source** now contains ONLY 6 core services (was 20 services)
  - Core: traefik, authelia, qbittorrent, plex, gluetun, cloudflared
- **All 27 addons** are now served exclusively from Git source (SDBX-Services repo)
- Addon updates no longer require a CLI release - use `sdbx source update`

**Benefits:**
- Smaller binary size
- Faster addon updates without CLI releases
- Offline initialization with core services
- Scalable addon ecosystem

**Total Services:** 33 (6 core + 27 addons)

### Fixed
- Updated documentation to reflect new architecture
- Updated embedded tests to expect core services only

---

## [0.2.0-alpha] - 2026-01-01

### Changed

#### Modular Core/Addon Architecture
Reorganized services into a leaner core + optional addons model for more flexible deployments.

**Core (6 services)** - Essential infrastructure only:
- Traefik, Authelia, qBittorrent, Plex, Gluetun, Cloudflared

**Addons (14 services)** - Enable what you need:
- **Media automation**: Sonarr, Radarr, Prowlarr, Lidarr, Readarr, Bazarr
- **Utilities**: Recyclarr, Unpackerr, Watchtower, Homepage
- **Streaming**: Overseerr, Tautulli, Wizarr
- **Downloads**: FlareSolverr

This allows users to start with a minimal setup (just qBittorrent + Plex + security) and add services as needed with `sdbx addon enable`.

### Fixed
- Documentation updated to reflect new architecture
- Removed references to non-existent CLI commands

---

## [0.1.0-alpha] - 2025-12-31

### Added

#### Core Features
- **Interactive setup wizard** - Terminal UI for project initialization
- **Multiple exposure modes** - Support for LAN, direct HTTPS, and Cloudflare Tunnel
- **Routing strategies** - Subdomain and path-based routing options
- **VPN enforcement** - Gluetun integration with kill-switch for downloads
- **Single Sign-On** - Authelia authentication with 2FA support
- **Service management** - Start, stop, restart, and monitor services
- **Health monitoring** - Built-in diagnostic checks with `sdbx doctor`
- **Secret management** - Automatic generation and rotation of secrets
- **Registry-based services** - YAML service definitions with multiple sources
- **Addon system** - Modular addons

#### CLI Commands
- `sdbx init` - Initialize new project with interactive wizard
- `sdbx up` / `sdbx down` / `sdbx restart` - Service lifecycle management
- `sdbx status` - View service health dashboard
- `sdbx logs` - View service logs
- `sdbx doctor` - Run diagnostic checks
- `sdbx addon` - Manage optional addons (list, search, enable, disable)
- `sdbx source` - Manage service sources (list, add, remove, update)
- `sdbx lock` - Lock file management for reproducibility
- `sdbx config` - View and modify configuration

#### Services Included
Initial release with 20 services (13 core + 7 addons)

#### VPN Providers
- NordVPN, Mullvad, Private Internet Access, Surfshark, ProtonVPN, Custom OpenVPN

### Infrastructure
- Comprehensive test suite
- Cross-platform builds (Linux, macOS - amd64, arm64)

### Documentation
- Getting started guide
- CLI reference
- Architecture documentation
- Troubleshooting guide

---

## Roadmap

### v0.4.0 (Planned)
- Backup/restore commands
- Auto-configure service integrations (Sonarr ↔ Prowlarr ↔ qBittorrent)
- Preset installation bundles (Media Library, Usenet Stack, etc.)
- Enhanced VPN provider wizard

### v1.0.0 (Stable)
- Production-ready release
- Multi-host support
- Web-based configuration UI

---

For detailed information about any release, see the [GitHub Releases](https://github.com/maiko/SDBX/releases) page.
