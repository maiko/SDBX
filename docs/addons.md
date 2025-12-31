# Addons 🧩

SDBX is modular. You can enable additional features and services using the `sdbx addon` command.

## 📦 Service Addons

| Addon | Purpose | Deployment Command |
|-------|---------|-------------------|
| **Wizarr** | Automated Plex/Jellyfin onboarding and user management. | `sdbx addon enable wizarr` |
| **Overseerr** | Request management for Plex. Users can request movies/TV shows. | `sdbx addon enable overseerr` |
| **Tautulli** | Detailed analytics and notifications for your Plex server. | `sdbx addon enable tautulli` |
| **FlareSolverr**| Proxy server to bypass Cloudflare protection for indexers. | `sdbx addon enable flaresolverr` |
| **Lidarr** | Music collection manager and automation tool. | `sdbx addon enable lidarr` |
| **Readarr** | Book and audiobook collection manager. | `sdbx addon enable readarr` |
| **Bazarr** | Automated subtitle management for Sonarr and Radarr. | `sdbx addon enable bazarr` |
| **Unpackerr** | Automated extraction of archives for the *arr suite. | `sdbx addon enable unpackerr` |

## 🛠️ Performance & Utility Addons

### `prometheus-metrics`
Enables a Prometheus/Grafana stack to monitor host metrics and Docker container health.
Usage: `sdbx addon enable prometheus-metrics`

### `wireguard-client`
Allows you to connect your server to another Wireguard network, useful for site-to-site connectivity.
Usage: `sdbx addon enable wireguard-client`

## ⚙️ How Addons Work

When you enable an addon:
1. SDBX looks for a template in `internal/generator/templates/addons/`.
2. It merges the service definition into your `compose.yaml`.
3. It adds necessary environment variables to your `.env` (prefixed with the addon name).
4. it restarts the stack to apply changes.

## 🗑️ Disabling Addons

To remove an addon and its associated service:

```bash
sdbx addon disable NAME
```

> [!NOTE]
> Disabling an addon does **not** delete its data stored in the `data/` directory. If you want to purge its configuration and database, you must manually delete the corresponding folder in `data/`.
