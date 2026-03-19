# Addons 🧩

SDBX is modular. The core binary embeds **7 core services** (Traefik, Authelia, Plex, qBittorrent, Gluetun, Cloudflared, SDBX Web UI). On top of that, **27 optional addons** are available from the official service repository — **34 services total**.

You can enable additional features and services using the `sdbx addon` command.

All service definitions use `apiVersion: sdbx.one/v1`.

## 📦 Popular Service Addons

| Addon | Purpose | Deployment Command |
|-------|---------|-------------------|
| **Sonarr** | TV show automation and management. | `sdbx addon enable sonarr` |
| **Radarr** | Movie automation and management. | `sdbx addon enable radarr` |
| **Prowlarr** | Centralized indexer manager for *arr apps. | `sdbx addon enable prowlarr` |
| **Wizarr** | Automated Plex/Jellyfin onboarding and user management. | `sdbx addon enable wizarr` |
| **Overseerr** | Request management for Plex. Users can request movies/TV shows. | `sdbx addon enable overseerr` |
| **Tautulli** | Detailed analytics and notifications for your Plex server. | `sdbx addon enable tautulli` |
| **FlareSolverr**| Proxy server to bypass Cloudflare protection for indexers. | `sdbx addon enable flaresolverr` |
| **Lidarr** | Music collection manager and automation tool. | `sdbx addon enable lidarr` |
| **Readarr** | Book and audiobook collection manager. | `sdbx addon enable readarr` |
| **Bazarr** | Automated subtitle management for Sonarr and Radarr. | `sdbx addon enable bazarr` |
| **Jellyfin** | Free and open source media server (Plex alternative). | `sdbx addon enable jellyfin` |
| **Recyclarr** | Automated *arr quality profile sync. | `sdbx addon enable recyclarr` |
| **Autoscan** | Real-time library update notifications. | `sdbx addon enable autoscan` |

For the full list of all 27 addons, run `sdbx addon list --all`.
## ⚙️ How Addons Work

When you enable an addon:
1. SDBX looks up the service definition from the registry (embedded or external sources).
2. It updates your configuration to include the addon.
3. Run `sdbx generate` to regenerate your `compose.yaml` with the new service.
4. Run `sdbx up` to start the updated stack.

## 🗑️ Disabling Addons

To remove an addon and its associated service:

```bash
sdbx addon disable NAME
```

> [!NOTE]
> Disabling an addon does **not** delete its data stored in the `data/` directory. If you want to purge its configuration and database, you must manually delete the corresponding folder in `data/`.
