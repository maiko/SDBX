# CLI Reference 🕹️

The `sdbx` CLI is your primary tool for managing your seedbox stack. This page provides a comprehensive reference for all available commands.

## 🏗️ Core Commands

### `sdbx init`
Initializes a new SDBX project in the current directory. It runs an interactive TUI wizard to collect configuration details.
- **Flags**:
  - `--domain STRING`: Base domain (e.g., `box.sdbx.one`)
  - `--expose STRING`: Exposure mode: `cloudflared`, `direct`, or `lan`
  - `--routing STRING`: Routing strategy: `subdomain` or `path`
  - `--timezone STRING`: Timezone (e.g., `Europe/Paris`)
  - `--vpn`: Enable VPN for downloads
  - `--vpn-provider STRING`: VPN provider (nordvpn, mullvad, pia, etc.)
  - `--vpn-country STRING`: VPN server country
  - `--admin-user STRING`: Admin username for Authelia
  - `--admin-password STRING`: Admin password for Authelia
  - `--skip-wizard`: Skip interactive wizard (use flags only)
  - `--force`: Overwrite existing configuration files

### `sdbx up`
Starts all services defined in your `compose.yaml`.
- **Flags**:
  - `-d, --detach`: Run in background (default).
  - `--build`: Rebuild images before starting.

### `sdbx down`
Stops and removes all containers, networks, and images defined in `compose.yaml`.

### `sdbx restart [service]`
Restarts all services or a specific service.
- **Arguments**:
  - `[service]`: (Optional) Name of the service to restart (e.g., `authelia`).

### `sdbx status`
Displays the current status of all services, including health and public URLs.

### `sdbx logs [service]`
Views logs for all or a specific service.
- **Flags**:
  - `-f, --follow`: Stream logs.
  - `--tail N`: Show last N lines.

### `sdbx doctor`
Runs a suite of diagnostic checks to ensure the host and the stack are healthy. 
Checks include Docker version, disk space, file permissions, and connectivity.

### `sdbx open [service]`
Opens the dashboard or a specific service's URL in your default web browser.

### `sdbx serve`
Starts the embedded web UI server. Behavior depends on whether the project has been initialized.
- **Flags**:
  - `--host STRING`: Bind address (default: `0.0.0.0`)
  - `--port INT`: Listen port (default: `3000`)

**Pre-init mode** (no `.sdbx.yaml` exists):
Runs a 7-step setup wizard that replaces `sdbx init`. A one-time 256-bit token is generated and printed to the terminal as a URL (e.g., `http://192.168.1.100:3000?token=abc123`). The token is required for access.

**Post-init mode** (`.sdbx.yaml` exists):
Serves the full dashboard and management interface. In production, the web UI runs as a Docker service behind Traefik + Authelia.

The web UI provides **12 pages** organized into four sidebar groups:

| Group | Page | Description |
|-------|------|-------------|
| Operations | **Dashboard** | Service overview with Quick Access links to each service URL |
| Operations | **Services** | Start, stop, and restart individual services |
| Config | **Addons** | Browse, enable, and disable addon services |
| Config | **VPN** | Configure VPN provider and credentials |
| Config | **Sources** | Manage service definition sources |
| Config | **Config** | Edit YAML configuration |
| System | **Doctor** | Run diagnostic health checks |
| System | **Compose** | View generated Docker Compose file |
| System | **Lock File** | Inspect and verify the lock file |
| System | **Backup** | Create and restore configuration backups |
| Reference | **Service Info** | Detailed service definitions and metadata |
| — | **Logs** | Live WebSocket log streaming per service |

Additional features: dark mode toggle (persisted via localStorage), CSRF protection via `csrfFetch()` wrapper, htmx bundled locally (no CDN dependency), and service control endpoints returning HTML fragments for htmx partial updates.

---

## ⚙️ Configuration

### `sdbx config get KEY`
Retrieves a configuration value from the `.env` file.

### `sdbx config set KEY VALUE`
Updates a configuration value in the `.env` file and applies changes to relevant templates.

---

## 🧩 Addons

### `sdbx addon list`
Lists all available and currently enabled addons.

### `sdbx addon enable NAME`
Enables a specific addon (e.g., `sdbx addon enable overseerr`). This will update your `compose.yaml` and restart necessary services.

### `sdbx addon disable NAME`
Disables and removes a specific addon.

### `sdbx addon search QUERY`
Searches for addons matching the query.

### `sdbx addon info NAME`
Shows detailed information about a specific addon.

---

## 📦 Source Management

### `sdbx source list`
Lists all configured service definition sources.

### `sdbx source add NAME URL`
Adds a new Git repository as a service source (like Homebrew taps).

> [!NOTE]
> Adding a third-party source (any source not from the official SDBX repository) will display a trust warning. Third-party sources can contain arbitrary service definitions that run Docker containers on your system. Only add sources you trust.

Source configuration is stored in `sources.yaml` (Kind: `SourceRepository`). The CLI enforces `minCliVersion` from source metadata, ensuring your CLI is compatible with the source's service definitions.

### `sdbx source remove NAME`
Removes a configured source.

### `sdbx source update [NAME]`
Updates sources to fetch latest service definitions. Updates all if no name specified.

---

## 🔒 Lock File

### `sdbx lock generate`
Generates or updates the `.sdbx.lock` file to pin service versions.

### `sdbx lock verify`
Verifies the lock file integrity against current sources.

### `sdbx lock diff`
Shows differences between current state and lock file.

---

## 🔧 Operations

### `sdbx update [--safe]`
Updates all Docker images to their latest versions.
- **Flags**:
  - `--safe`: (Recommended) Updates services one by one and runs health checks before proceeding to the next.

### `sdbx backup create`
Creates a timestamped backup of your configuration and database volumes.

### `sdbx backup restore`
Lists available backups and allows you to restore to a previous state.

### `sdbx import`
Imports services from an existing Docker Compose file into SDBX configuration.

### `sdbx regenerate`
Regenerates `compose.yaml` from the current `.sdbx.yaml` configuration. Useful after editing config or enabling/disabling addons. Alias: `regen`.

### `sdbx version`
Prints the current version of the `sdbx` CLI.
