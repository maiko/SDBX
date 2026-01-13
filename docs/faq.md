# Frequently Asked Questions (FAQ)

## General Questions

### What is SDBX?

SDBX (Seedbox in a Box) is a complete, production-ready CLI tool for deploying and managing a seedbox stack. It bundles Plex, Sonarr, Radarr, qBittorrent, and other services with authentication, VPN enforcement, and beautiful interfaces - all configured automatically.

### Who is SDBX for?

- **Media enthusiasts** wanting automated TV/movie management
- **Privacy-conscious users** needing VPN-enforced downloads
- **Self-hosters** looking for an all-in-one solution
- **Beginners** wanting a simple setup process
- **Advanced users** needing flexibility and control

### What makes SDBX different from similar projects?

- **Zero-configuration setup** - Interactive wizard handles everything
- **Production-ready security** - SSO, 2FA, VPN kill-switch built-in
- **Multiple deployment options** - LAN, direct HTTPS, or Cloudflare Tunnel
- **Modular design** - Enable only what you need
- **Beautiful TUI** - Modern command-line interface
- **Active development** - Regular updates and community support

## Deployment & Setup

### Which exposure mode should I choose?

| Mode | Best For | Pros | Cons |
|------|----------|------|------|
| **Cloudflare Tunnel** | Remote access without port forwarding | No open ports, free SSL, DDoS protection | Requires Cloudflare account |
| **Direct HTTPS** | Full control, static IP | Best performance, no third-party | Requires ports 80/443, domain DNS |
| **LAN Only** | Home lab, local network | Simple, no internet exposure | No remote access |

**Recommendation**: Start with Cloudflare Tunnel for easiest setup and best security.

### Which routing strategy should I use?

**Subdomain** (`radarr.domain.com`, `sonarr.domain.com`):
- ✅ Cleaner URLs
- ✅ Better SSL certificate handling
- ✅ Easier to remember
- ❌ Requires wildcard DNS
- ❌ More subdomains to manage

**Path** (`sdbx.domain.com/radarr`, `sdbx.domain.com/sonarr`):
- ✅ Single domain/certificate
- ✅ Simpler DNS setup
- ✅ Better for limited domain availability
- ❌ Some apps have path routing issues
- ❌ Longer URLs

**Recommendation**: Use subdomain routing unless you have domain limitations.

### Which VPN provider should I use?

**Recommended for P2P:**
- **Mullvad** - Best privacy, no personal info required
- **Private Internet Access** - Good performance, proven no-logs
- **NordVPN** - User-friendly, fast
- **ProtonVPN** - Privacy-focused, Swiss-based
- **Surfshark** - Budget-friendly, unlimited devices

**Avoid**: Free VPNs (slow, unreliable, questionable privacy)

**Note**: Ensure your VPN provider allows P2P/torrenting and provides OpenVPN configs.

### How much disk space do I need?

**Minimum**:
- System: 20 GB
- Config: 2 GB
- Initial setup: 30 GB total

**Recommended**:
- Movies (1080p): 4-8 GB each
- TV shows (1080p): 1-3 GB per episode
- 4K movies: 20-60 GB each

**Example**:
- 100 movies (1080p): ~600 GB
- 20 TV series (10 seasons): ~1.2 TB
- **Total**: 2 TB minimum, 4-8 TB recommended

### What are the bandwidth requirements?

**Typical usage**:
- Initial setup: < 5 GB (Docker images)
- Per movie download: 4-40 GB
- Streaming (local): Minimal
- Streaming (remote): 2-15 Mbps per stream

**Recommended internet**:
- **Download**: 100 Mbps+ (for fast torrenting)
- **Upload**: 20 Mbps+ (for remote Plex streaming)

### Can I run this on a Raspberry Pi?

**Raspberry Pi 4 (4GB+)**:
- ✅ Works but limited
- ✅ Good for: *arr apps, qBittorrent, lightweight
- ❌ Struggles with: Plex transcoding, multiple streams
- **Verdict**: OK for personal use, not ideal for families

**Raspberry Pi 5 (8GB)**:
- ✅ Better performance
- ✅ Can handle 1-2 transcodes (1080p)
- **Verdict**: Good for small setups

**Better alternatives**:
- Intel NUC (N100/N305)
- Used business PC (Dell Optiplex, HP EliteDesk)
- Budget VPS with good storage

### What are the hardware requirements?

**Minimum**:
- CPU: 2 cores, 2.0 GHz
- RAM: 4 GB
- Storage: 100 GB (config + cache)
- Bandwidth: 50 Mbps down, 10 Mbps up

**Recommended**:
- CPU: 4+ cores, Intel/AMD with QuickSync (for Plex)
- RAM: 8-16 GB
- Storage: 2-8 TB HDD/SSD
- Bandwidth: 100+ Mbps down, 20+ Mbps up

**Optimal** (Plex transcoding):
- CPU: Intel i3-N305 or better (QuickSync)
- RAM: 16 GB
- Storage: 8+ TB
- Bandwidth: Gigabit

## Configuration

### How do I change my domain after setup?

```bash
# Edit config
sdbx config set domain new.sdbx.one

# Regenerate configs
sdbx init --skip-wizard

# Restart services
sdbx down && sdbx up
```

### How do I add/remove addons?

```bash
# List available addons
sdbx addon list

# Enable addon
sdbx addon enable overseerr

# Disable addon
sdbx addon disable tautulli

# Restart to apply
sdbx down && sdbx up
```

### How do I change VPN settings?

Edit `configs/gluetun/gluetun.env`:
```bash
VPN_SERVICE_PROVIDER=mullvad
WIREGUARD_PRIVATE_KEY=your_private_key
SERVER_COUNTRIES=Netherlands,Sweden
```

Then restart:
```bash
sdbx restart gluetun
sdbx restart qbittorrent
```

### How do I add custom Plex libraries?

1. Access Plex web UI: `sdbx open plex`
2. Go to Settings → Manage → Libraries
3. Add library with paths:
   - Movies: `/data/media/movies`
   - TV Shows: `/data/media/tv`
   - Music: `/data/media/music`
   - Books: `/data/media/books`

### Can I use my existing Plex server?

Currently, SDBX manages its own Plex instance. To migrate:

1. Backup existing Plex database
2. Deploy SDBX
3. Copy Plex database to `config/plex/Library/Application Support/Plex Media Server/`
4. Restart: `sdbx restart plex`

## Usage & Operations

### How do I backup my configuration?

```bash
# Create backup
sdbx backup run

# List backups
sdbx backup list

# Restore backup
sdbx backup restore sdbx-backup-20250101-120000.tar.gz
```

**What's included**:
- Configuration files
- Secrets
- Service configs (*arr settings, etc.)

**Not included**:
- Media files (too large)
- Plex database (backup separately if needed)

### How do I update SDBX?

```bash
# Check current version
sdbx version

# Update binary
curl -fsSL https://github.com/maiko/sdbx/releases/latest/download/install.sh | bash

# Update service images
sdbx update
```

### How do I rotate secrets?

Secrets are auto-generated during `sdbx init`. To rotate them manually:

```bash
# Delete the secret files you want to rotate
rm secrets/authelia_jwt_secret.txt

# Restart services (will auto-regenerate missing secrets)
sdbx down && sdbx up
```

**Note**: Rotating secrets may require re-authenticating users. Keep backups of `secrets/` directory before deletion if needed.

### How do I view logs?

```bash
# All services
sdbx logs

# Specific service
sdbx logs traefik

# Follow logs (live)
sdbx logs -f qbittorrent

# Last 100 lines
sdbx logs --tail 100 sonarr
```

### How do I restart a single service?

```bash
# Restart specific service
sdbx restart plex

# Restart all
sdbx restart
```

## Troubleshooting

### Services won't start

```bash
# Check service health
sdbx status

# Run diagnostics
sdbx doctor

# Check logs
sdbx logs

# Common fixes:
# 1. Port conflict - check ports with: sudo netstat -tulpn
# 2. Permissions - check PUID/PGID in .env
# 3. Disk space - check: df -h
```

### Can't access services

**Check exposure mode**:
```bash
# LAN mode - use local IP
http://192.168.1.100/

# Direct/Cloudflare - use domain
https://box.sdbx.one/
```

**Verify DNS**:
```bash
# Check domain resolves
nslookup box.sdbx.one

# Cloudflare Tunnel status
sdbx logs cloudflared
```

**Check Authelia**:
```bash
# View auth logs
sdbx logs authelia

# Reset password in configs/authelia/users_database.yml
```

### Downloads not working through VPN

```bash
# Check VPN status
sdbx logs gluetun | grep "ip address"

# Verify kill-switch
sdbx doctor

# Test from qBittorrent container
docker exec sdbx-qbittorrent-1 curl https://ifconfig.me

# Should show VPN IP, not your real IP
```

### Plex not transcoding

**Enable hardware acceleration**:

Edit `compose.yaml`:
```yaml
plex:
  devices:
    - /dev/dri:/dev/dri  # Intel QuickSync
```

Restart:
```bash
sdbx down && sdbx up
```

### Why is Plex limiting my streaming quality to 720p?

If Plex shows "Indirect" connection or limits quality to 720p 2Mbps, it's routing through Plex's relay servers instead of connecting directly.

**Cause**: Plex doesn't know where it can be reached for direct connections.

**Solution**: Configure `PLEX_ADVERTISE_IP`

1. Edit `.sdbx.yaml` and add:
   ```yaml
   plex_advertise_urls: "https://plex.yourdomain.com:443"
   ```

2. For multiple locations (LAN + Remote with Cloudflare Tunnel):
   ```yaml
   plex_advertise_urls: "https://plex.yourdomain.com:443,http://192.168.1.100:32400"
   ```

3. Regenerate and restart:
   ```bash
   sdbx up --force-recreate plex
   ```

**For Cloudflare Tunnel users (remote servers)**:
- Use your Cloudflare Tunnel URL: `https://plex.yourdomain.com:443`
- Modern Plex supports HTTPS streaming through reverse proxies and tunnels
- This keeps all traffic within Cloudflare network (no need for direct port exposure)
- For LAN access: Add local IP: `https://plex.domain.com:443,http://192.168.x.x:32400`

**Finding your local IP**:
- Linux/Mac: `ip addr` or `ifconfig`
- Windows: `ipconfig`
- Look for `192.168.x.x` or `10.x.x.x`

### Out of disk space

```bash
# Check usage
df -h

# Clear Docker cache
docker system prune -a

# Remove old torrents
# In qBittorrent: Right-click → Delete → Check "Also delete files"

# Clear Plex cache
rm -rf config/plex/Library/Application\ Support/Plex\ Media\ Server/Cache/*
```

## Security

### How secure is SDBX?

- ✅ SSO with 2FA (Authelia)
- ✅ VPN kill-switch for downloads
- ✅ Automatic secret generation
- ✅ HTTPS everywhere
- ✅ No default passwords
- ✅ Regular security updates

### Should I expose services to the internet?

**Yes, with precautions**:
- Use Cloudflare Tunnel (no port forwarding)
- Enable 2FA in Authelia
- Use strong passwords
- Keep SDBX updated
- Monitor logs regularly

**No, if**:
- You only need local access
- You're uncomfortable with security management
- You have sensitive personal data

### How do I enable 2FA?

1. Access Authelia: `sdbx open authelia`
2. Log in with your credentials
3. Go to Settings → Two-Factor Authentication
4. Scan QR code with authenticator app (Google Authenticator, Authy, etc.)
5. Enter code to verify
6. Save backup codes safely

### Is my VPN password safe?

VPN credentials are stored in `secrets/vpn_password.txt` as plain text (required by Gluetun). To protect:

- Use restrictive file permissions: `chmod 600 secrets/*`
- Use VPN-specific credentials (not your main account)
- Enable disk encryption
- Don't commit secrets/ to git (already in .gitignore)

### Can I use Let's Encrypt instead of Cloudflare?

Yes, with direct HTTPS mode:

```bash
# During init, select "Direct HTTPS" exposure mode
# Traefik will automatically obtain Let's Encrypt certificates

# Requires:
# - Ports 80/443 open
# - Domain pointing to your IP
# - Valid email for Let's Encrypt
```

## Advanced

### Can I customize service configurations?

**Yes, but carefully**:

1. Service configs are in `configs/<service>/`
2. Edit configs directly
3. Restart service: `sdbx restart <service>`

**Note**: Running `sdbx init` again will overwrite custom configs. Back them up first!

### Can I add custom Docker services?

**Yes**, edit `compose.yaml`:

```yaml
services:
  my-custom-service:
    image: custom/image:latest
    container_name: sdbx-custom-1
    networks:
      - sdbx
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.custom.rule=Host(`custom.domain.com`)"
```

Restart: `sdbx down && sdbx up`

### How do I run SDBX on a different port?

Edit `compose.yaml` and change Traefik ports:

```yaml
traefik:
  ports:
    - "8080:80"    # HTTP
    - "8443:443"   # HTTPS
```

Update firewall/router accordingly.

### Can I use a different timezone?

```bash
# Change timezone
sdbx config set timezone America/New_York

# Regenerate configs
sdbx init --skip-wizard

# Restart
sdbx down && sdbx up
```

### How do I migrate to a new server?

1. **On old server**:
   ```bash
   sdbx backup run
   ```

2. **Transfer backup** to new server

3. **On new server**:
   ```bash
   # Install SDBX
   curl -fsSL https://[...]/install.sh | bash

   # Restore backup
   sdbx backup restore sdbx-backup-XXX.tar.gz

   # Start services
   sdbx up
   ```

4. **Update DNS** to point to new IP

5. **Verify** everything works

6. **Copy media files** separately (rsync recommended)

## Community & Support

### Where can I get help?

- **Documentation**: https://github.com/maiko/sdbx/docs
- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and community help
- **Discord**: (if available)

### How can I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md) for detailed guidelines.

Quick start:
1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

### How do I report a bug?

1. Check existing issues first
2. Create a new issue with:
   - SDBX version (`sdbx version`)
   - Steps to reproduce
   - Expected vs actual behavior
   - Logs (`sdbx logs <service>`)
   - Environment info (OS, Docker version)

### How do I request a feature?

Open a GitHub issue with:
- Clear description of the feature
- Use case / problem it solves
- Proposed implementation (if you have ideas)
- Willingness to contribute

---

**Still have questions?** Check [GitHub Discussions](https://github.com/maiko/sdbx/discussions) or [open an issue](https://github.com/maiko/sdbx/issues/new).
