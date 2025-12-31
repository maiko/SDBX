# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- **Addon system** - Modular addons (Overseerr, Tautulli, Wizarr, etc.)

#### CLI Commands
- `sdbx init` - Initialize new project with interactive wizard
- `sdbx up` / `sdbx down` / `sdbx restart` - Service lifecycle management
- `sdbx status` - View service health dashboard
- `sdbx logs` - View service logs
- `sdbx doctor` - Run diagnostic checks
- `sdbx addon` - Manage optional addons (list, search, enable, disable)
- `sdbx source` - Manage service sources (list, add, remove, update)
- `sdbx lock` - Lock file management for reproducibility
- `sdbx secrets` - Manage secrets
- `sdbx config` - View and modify configuration

#### Services Included
**Core (13 services):**
- Traefik, Authelia, Homepage, qBittorrent, Gluetun
- Prowlarr, Sonarr, Radarr, Plex
- Recyclarr, Unpackerr, Watchtower, Cloudflared

**Addons (7 services):**
- Overseerr, Wizarr, Tautulli, Lidarr, Readarr, Bazarr, FlareSolverr

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

### v0.2.0 (Planned)
- Additional VPN providers
- More addon options
- Performance improvements

### v1.0.0 (Stable)
- Production-ready release
- Multi-host support
- Web-based configuration UI

---

For detailed information about any release, see the [GitHub Releases](https://github.com/maiko/SDBX/releases) page.
