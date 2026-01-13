# Service Interconnection

This guide explains how SDBX services communicate with each other using Docker networking.

## Docker Hostname Pattern

All SDBX services follow a consistent naming pattern:

```
sdbx-{servicename}
```

Examples:
- Sonarr: `sdbx-sonarr`
- Radarr: `sdbx-radarr`
- qBittorrent: `sdbx-qbittorrent`
- Prowlarr: `sdbx-prowlarr`

## Service Reference Table

| Service | Docker Hostname | Internal Port | Purpose |
|---------|----------------|---------------|---------|
| **Core Services** |
| Traefik | sdbx-traefik | 8080 (dashboard) | Reverse proxy and routing |
| Authelia | sdbx-authelia | 9091 | Authentication and SSO |
| Plex | sdbx-plex | 32400 | Media server |
| qBittorrent | sdbx-qbittorrent | 8080 | Download client |
| Gluetun | sdbx-gluetun | - | VPN client (kill-switch) |
| Cloudflared | sdbx-cloudflared | - | Cloudflare Tunnel client |
| **Media Management** |
| Sonarr | sdbx-sonarr | 8989 | TV show automation |
| Radarr | sdbx-radarr | 7878 | Movie automation |
| Lidarr | sdbx-lidarr | 8686 | Music automation |
| Readarr | sdbx-readarr | 8787 | Book automation |
| Prowlarr | sdbx-prowlarr | 9696 | Indexer manager |
| Bazarr | sdbx-bazarr | 6767 | Subtitle automation |
| **Request & Monitoring** |
| Overseerr | sdbx-overseerr | 5055 | Request management |
| Jellyseerr | sdbx-jellyseerr | 5055 | Request management (Jellyfin) |
| Tautulli | sdbx-tautulli | 8181 | Plex monitoring |
| **Media Players** |
| Jellyfin | sdbx-jellyfin | 8096 | Media server (Plex alternative) |
| Emby | sdbx-emby | 8096 | Media server (Plex alternative) |
| **Utilities** |
| FlareSolverr | sdbx-flaresolverr | 8191 | Cloudflare bypass |
| Unpackerr | sdbx-unpackerr | - | Archive extractor |
| Recyclarr | sdbx-recyclarr | - | *arr configuration sync |
| Autoscan | sdbx-autoscan | 3030 | Real-time library updates |

## How Docker Networking Works

SDBX services run in Docker containers connected via Docker networks. Services can communicate
using their Docker hostnames without needing to know IP addresses.

**Example**: When Sonarr needs to connect to qBittorrent:
- **Hostname**: `sdbx-qbittorrent`
- **Port**: `8080`
- **Full URL**: `http://sdbx-qbittorrent:8080`

This works because both services are on the same Docker network (`sdbx-network`).

## External vs Internal URLs

- **External URL**: `https://sonarr.yourdomain.com` - Used by browsers, goes through Traefik/Authelia
- **Internal URL**: `http://sdbx-sonarr:8989` - Used by other Docker services, direct connection

**Never use external URLs for service-to-service communication.** They route through Traefik and may cause authentication loops or unnecessary overhead.

## Configuring Service Connections

When configuring services to communicate (e.g., adding a download client to Sonarr):

1. Use the Docker hostname (e.g., `sdbx-qbittorrent`)
2. Use the internal port (e.g., `8080` for qBittorrent)
3. Use `http://` (not `https://`) - internal traffic isn't encrypted
4. Don't use the external domain (e.g., `sonarr.yourdomain.com`) - that routes through Traefik

**Example Configuration**:
```
Host: sdbx-qbittorrent
Port: 8080
URL Base: (leave empty)
```

## Common Configuration Examples

### Adding qBittorrent to Sonarr

1. Open Sonarr web UI
2. Go to **Settings → Download Clients**
3. Click **+** to add a new download client
4. Select **qBittorrent**
5. Configure:
   - **Host**: `sdbx-qbittorrent`
   - **Port**: `8080`
   - **Username**: `admin`
   - **Password**: (from qBittorrent config)
   - **Category**: `sonarr` (optional, for organization)
6. Test and Save

### Adding Sonarr to Prowlarr

1. Open Prowlarr web UI
2. Go to **Settings → Apps**
3. Click **+** to add a new application
4. Select **Sonarr**
5. Configure:
   - **Prowlarr Server**: Leave as default or use `http://sdbx-prowlarr:9696`
   - **Sonarr Server**: `http://sdbx-sonarr:8989`
   - **API Key**: (found in Sonarr → Settings → General → Security → API Key)
   - **Sync Categories**: Select categories to sync
6. Test and Save

### Adding Plex to Overseerr

1. Open Overseerr web UI
2. Go to **Settings → Plex**
3. Click **Sign In** or manually configure:
   - **Hostname/IP**: `sdbx-plex`
   - **Port**: `32400`
4. Authenticate with your Plex account
5. Select libraries to scan

### Adding Plex to Tautulli

1. Open Tautulli web UI
2. Go to setup wizard or **Settings → Plex Media Server**
3. Configure:
   - **Plex IP Address or Hostname**: `sdbx-plex`
   - **Port**: `32400`
   - **Use SSL**: No (internal connection)
4. Verify Connection and Save

## Finding API Keys

Most *arr apps store their API keys in their configuration UI:

1. Open the service's web UI
2. Go to **Settings → General** (or similar)
3. Look for **Security** or **API** section
4. Copy the **API Key**

Alternatively, API keys are stored in config files:
- Sonarr: `configs/sonarr/config.xml` (look for `<ApiKey>`)
- Radarr: `configs/radarr/config.xml`
- Lidarr: `configs/lidarr/config.xml`
- Readarr: `configs/readarr/config.xml`
- Prowlarr: `configs/prowlarr/config.xml`

## Additional Resources

For service-specific configuration:
- Sonarr: [https://wiki.servarr.com/sonarr](https://wiki.servarr.com/sonarr)
- Radarr: [https://wiki.servarr.com/radarr](https://wiki.servarr.com/radarr)
- Lidarr: [https://wiki.servarr.com/lidarr](https://wiki.servarr.com/lidarr)
- Readarr: [https://wiki.servarr.com/readarr](https://wiki.servarr.com/readarr)
- Prowlarr: [https://wiki.servarr.com/prowlarr](https://wiki.servarr.com/prowlarr)
- qBittorrent: [https://github.com/qbittorrent/qBittorrent/wiki](https://github.com/qbittorrent/qBittorrent/wiki)
- Plex: [https://support.plex.tv](https://support.plex.tv)
- Overseerr: [https://docs.overseerr.dev](https://docs.overseerr.dev)

For Docker networking concepts:
- [https://docs.docker.com/network/](https://docs.docker.com/network/)
