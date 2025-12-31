# Migration Guide

This guide helps you upgrade SDBX between versions and migrate from other seedbox solutions.

## Table of Contents

- [Upgrading SDBX](#upgrading-sdbx)
- [Migrating from Other Solutions](#migrating-from-other-solutions)
- [Version-Specific Upgrades](#version-specific-upgrades)
- [Troubleshooting Migrations](#troubleshooting-migrations)

## Upgrading SDBX

### Pre-Upgrade Checklist

Before upgrading, always:

1. **Backup your configuration**
   ```bash
   sdbx backup run
   ```

2. **Check the changelog**
   ```bash
   # View release notes
   curl -s https://api.github.com/repos/maiko/sdbx/releases/latest | jq -r '.body'
   ```

3. **Note your current version**
   ```bash
   sdbx version
   ```

4. **Stop services** (optional, for safety)
   ```bash
   sdbx down
   ```

### Standard Upgrade Process

#### Method 1: Automated (Recommended)

```bash
# Update SDBX binary
curl -fsSL https://github.com/maiko/sdbx/releases/latest/download/install.sh | bash

# Verify new version
sdbx version

# Update service images
sdbx update

# Check health
sdbx doctor
```

#### Method 2: Manual

```bash
# Download latest release
VERSION="v0.1.0-alpha"  # Replace with desired version
curl -LO https://github.com/maiko/SDBX/releases/download/$VERSION/sdbx_Linux_x86_64.tar.gz

# Extract
tar -xzf sdbx_Linux_x86_64.tar.gz

# Install
sudo mv sdbx /usr/local/bin/
sudo chmod +x /usr/local/bin/sdbx

# Verify
sdbx version

# Update services
sdbx update
```

### Post-Upgrade Steps

1. **Regenerate configs** (if recommended in changelog):
   ```bash
   sdbx init --skip-wizard
   ```

2. **Start services**:
   ```bash
   sdbx up
   ```

3. **Verify operation**:
   ```bash
   sdbx status
   sdbx doctor
   ```

4. **Test key functions**:
   - Access Authelia
   - Check VPN status
   - Test a service (Radarr, Sonarr)
   - Verify downloads work

## Migrating from Other Solutions

### From docker-compose Seedbox

If you're running a similar stack with raw docker-compose:

1. **Document your current setup**:
   - Note all enabled services
   - Export service configurations
   - Document domain/DNS settings
   - List custom modifications

2. **Backup existing data**:
   ```bash
   # Backup configs
   tar -czf ~/seedbox-backup.tar.gz \
     compose.yaml \
     .env \
     config/

   # Media files are safe (SDBX uses same paths)
   ```

3. **Stop existing stack**:
   ```bash
   docker-compose down
   ```

4. **Initialize SDBX**:
   ```bash
   cd /opt/sdbx  # or your preferred location
   sdbx init
   ```

5. **Migrate configurations**:

   **Sonarr/Radarr/Lidarr/Readarr**:
   ```bash
   # Copy databases
   cp -r ~/old-config/sonarr/* config/sonarr/
   cp -r ~/old-config/radarr/* config/radarr/

   # Set permissions
   chown -R $(id -u):$(id -g) config/sonarr config/radarr
   ```

   **qBittorrent**:
   ```bash
   # Copy settings
   cp ~/old-config/qbittorrent/qBittorrent.conf config/qbittorrent/
   ```

   **Plex**:
   ```bash
   # Copy entire Plex database
   cp -r ~/old-config/plex/* config/plex/
   ```

6. **Update service settings**:
   - URLs may have changed (new domain/routing)
   - Update indexer URLs in Prowlarr
   - Update download client in *arr apps
   - Verify Plex libraries

7. **Start SDBX**:
   ```bash
   sdbx up
   ```

8. **Verify migration**:
   ```bash
   sdbx status
   sdbx doctor
   ```

### From Swizzin

**Swizzin users**: SDBX uses Docker, while Swizzin uses systemd services. Migration requires reconfiguration.

1. **Export Swizzin configs**:
   ```bash
   # Backup application data
   cd /home/$USER
   tar -czf ~/swizzin-apps-backup.tar.gz \
     .apps/sonarr \
     .apps/radarr \
     .apps/lidarr \
     .apps/prowlarr
   ```

2. **Stop Swizzin services**:
   ```bash
   box stop
   ```

3. **Install SDBX**:
   ```bash
   cd /opt/sdbx
   sdbx init
   ```

4. **Copy databases** (similar to docker-compose migration above)

5. **Reconfigure services**:
   - Applications will have new URLs
   - Update all inter-service connections
   - Re-add download clients
   - Re-add indexers

### From QuickBox

Similar to Swizzin migration. Key differences:

- QuickBox uses `/home/username/` for configs
- nginx reverse proxy → Traefik migration
- Recreate Authelia users (different auth system)

### From Cloudbox/Saltbox

Cloudbox/Saltbox also use Docker, making migration easier:

1. **Backup Cloudbox data**:
   ```bash
   cd ~/cloudbox
   ansible-playbook backup.yml
   ```

2. **Note inventory settings**:
   - Domain configuration
   - Cloudflare settings
   - Plex claim token

3. **Stop Cloudbox**:
   ```bash
   sudo ansible-playbook cloudbox.yml --tags stop
   ```

4. **Initialize SDBX**:
   ```bash
   sdbx init
   # Use same domain settings
   ```

5. **Migrate configs**:
   ```bash
   # Cloudbox stores in /opt/
   cp -r /opt/sonarr config/sonarr/
   cp -r /opt/radarr config/radarr/
   # etc.
   ```

6. **Handle Cloudflare**:
   - SDBX can use Cloudflare Tunnel (different from Cloudbox)
   - Or continue with direct DNS (choose "direct" mode)
   - Update DNS records if needed

## Version-Specific Upgrades

### Upgrading to v1.1.0 (from v1.0.x)

**Breaking Changes**: None

**New Features**:
- Additional addons
- Performance improvements

**Upgrade Steps**:
1. Standard upgrade process (above)
2. Enable new addons if desired: `sdbx addon enable <name>`

### Upgrading to v1.2.0 (from v1.1.x)

**Breaking Changes**: None

**Upgrade Steps**:
1. Standard upgrade process
2. Check for new configuration options: `sdbx config get`

### Upgrading to v2.0.0 (from v1.x)

**Breaking Changes**: TBD (when v2.0.0 is released)

**Major Changes**: TBD

**Upgrade Steps**: TBD

---

## Configuration Migration

### Migrating Authentication

#### From nginx-proxy + organizr

SDBX uses Authelia for SSO. To recreate users:

1. Edit `configs/authelia/users_database.yml`
2. Add users:
   ```yaml
   users:
     username:
       displayname: "Display Name"
       password: "$argon2id$..."  # Generate with sdbx
       email: user@example.com
       groups:
         - admins
   ```

3. Generate password hash:
   ```bash
   # Use SDBX's hash generator
   docker run --rm authelia/authelia:latest authelia hash-password 'yourpassword'
   ```

4. Restart Authelia:
   ```bash
   sdbx restart authelia
   ```

#### From Organizr

Export user list, then recreate in Authelia (no automated migration available).

### Migrating Download Client Settings

All *arr apps need to point to the new qBittorrent instance:

1. **In each *arr app** (Sonarr, Radarr, etc.):
   - Settings → Download Clients → Edit qBittorrent
   - Host: `qbittorrent` (Docker network name)
   - Port: `8080`
   - Username: `admin` (default)
   - Password: (from `secrets/qbittorrent_password.txt`)

2. **Test connection** in each app

### Migrating Indexers

If you used Jackett before, Prowlarr is the replacement:

1. **Export from Jackett** (if possible): Settings → Export
2. **In Prowlarr**:
   - Add indexers manually or
   - Use Prowlarr's built-in search to find your trackers
   - Sync to *arr apps: Settings → Apps

### Migrating Plex

To keep watch history, continue watched status, etc.:

1. **Stop Plex on both systems**

2. **Copy entire Plex directory**:
   ```bash
   rsync -av /old/plex/location/ /opt/sdbx/config/plex/
   ```

3. **Critical files**:
   - `Library/Application Support/Plex Media Server/Plug-in Support/Databases/com.plexapp.plugins.library.db`
   - `Library/Application Support/Plex Media Server/Preferences.xml`

4. **Update library paths** if media location changed:
   - Start Plex: `sdbx start plex`
   - Settings → Manage → Libraries → Edit each library
   - Update folder paths to new locations

5. **Claim server** if needed (first-time setup):
   - Get claim token from https://www.plex.tv/claim
   - Add to `.env`: `PLEX_CLAIM=claim-XXXXX`
   - Restart: `sdbx restart plex`

## Data Migration

### Moving Media Files

Media files can be large. Best practices:

#### Same Server (New Path)

```bash
# Use rsync to move with progress
rsync -ah --progress /old/media/ /new/media/

# Verify sizes match
du -sh /old/media /new/media

# Update mount points in compose.yaml if needed
```

#### To New Server

```bash
# From old server, push to new
rsync -avz --progress /media/ user@newserver:/new/media/

# Or from new server, pull from old
rsync -avz --progress user@oldserver:/media/ /new/media/

# For large transfers, use screen/tmux to avoid interruption
screen -S media-transfer
rsync -avz --progress --partial user@oldserver:/media/ /new/media/
```

**Tips**:
- Use `--partial` to resume interrupted transfers
- Add `--dry-run` first to preview
- Transfer during off-peak hours
- Verify with checksums for critical data: `rsync -c`

### Migrating VPN Configuration

From other solutions to SDBX's Gluetun:

1. **Locate old VPN configs**:
   - OpenVPN: `*.ovpn` files
   - WireGuard: `wg0.conf`, private keys

2. **For OpenVPN** (custom provider):
   ```bash
   # Copy .ovpn file to configs/gluetun/
   cp your-vpn.ovpn configs/gluetun/

   # Edit configs/gluetun/gluetun.env
   VPN_SERVICE_PROVIDER=custom
   VPN_TYPE=openvpn
   OPENVPN_CUSTOM_CONFIG=/gluetun/your-vpn.ovpn
   ```

3. **For WireGuard**:
   ```bash
   # Extract values from old config
   # Add to configs/gluetun/gluetun.env
   VPN_TYPE=wireguard
   WIREGUARD_PRIVATE_KEY=xxx
   WIREGUARD_ADDRESSES=10.x.x.x/32
   SERVER_COUNTRIES=Sweden
   ```

4. **Add credentials**:
   ```bash
   echo "your-username" > secrets/vpn_username.txt
   echo "your-password" > secrets/vpn_password.txt
   ```

5. **Test VPN**:
   ```bash
   sdbx restart gluetun
   sdbx logs gluetun | grep "ip address"
   ```

## Troubleshooting Migrations

### Services Won't Start After Upgrade

```bash
# Check logs
sdbx logs <service>

# Common issues:
# 1. Configuration format changed
#    Solution: Regenerate configs
   sdbx init --skip-wizard

# 2. Database migration needed
#    Solution: Check service-specific logs

# 3. Port conflicts
#    Solution: Check running services
   sudo netstat -tulpn | grep <port>
```

### Lost Data After Migration

```bash
# Restore from backup
sdbx backup restore <backup-file>

# If no backup, check Docker volumes
docker volume ls
docker volume inspect sdbx_<service>_data

# Mount volume to access data
docker run --rm -v sdbx_sonarr_data:/data alpine ls -la /data
```

### Authentication Not Working

```bash
# Reset Authelia password
# Edit configs/authelia/users_database.yml
# Generate new password hash:
docker run --rm authelia/authelia:latest authelia hash-password 'newpassword'

# Restart Authelia
sdbx restart authelia

# Clear browser cookies/cache for your domain
```

### VPN Not Connecting After Migration

```bash
# Check VPN logs
sdbx logs gluetun

# Common issues:
# 1. Wrong credentials
   cat secrets/vpn_username.txt
   cat secrets/vpn_password.txt

# 2. Unsupported server
#    Check Gluetun wiki for supported servers

# 3. Firewall blocking
   sudo ufw allow out on tun0
```

### Media Libraries Missing in Plex

```bash
# Check Plex has access to media
docker exec sdbx-plex-1 ls -la /data/media

# Fix permissions
sudo chown -R $(id -u):$(id -g) /path/to/media

# Rescan libraries in Plex UI
# Settings → Manage → Libraries → Scan Library Files
```

### *arr Apps Lost Settings

If regenerating configs with `sdbx init --skip-wizard` overwrites settings:

```bash
# Restore from backup
sdbx backup restore <backup-file>

# Or manually restore service config
cp backup/configs/sonarr/* config/sonarr/
sdbx restart sonarr
```

## Rollback Procedure

If upgrade causes issues:

1. **Stop services**:
   ```bash
   sdbx down
   ```

2. **Restore backup**:
   ```bash
   sdbx backup restore <pre-upgrade-backup>
   ```

3. **Downgrade SDBX binary**:
   ```bash
   # Install specific version
   VERSION="v0.1.0-alpha"
   curl -LO https://github.com/maiko/SDBX/releases/download/$VERSION/sdbx_Linux_x86_64.tar.gz
   tar -xzf sdbx_Linux_x86_64.tar.gz
   sudo mv sdbx /usr/local/bin/
   ```

4. **Restart services**:
   ```bash
   sdbx up
   ```

5. **Verify**:
   ```bash
   sdbx version
   sdbx status
   ```

## Best Practices

### Before Any Migration

1. ✅ **Full backup** (config + media if possible)
2. ✅ **Document current state** (screenshots, configs)
3. ✅ **Test in staging** (if possible)
4. ✅ **Read changelog** completely
5. ✅ **Plan rollback strategy**

### During Migration

1. ✅ **Work during low-usage hours**
2. ✅ **Inform users** of potential downtime
3. ✅ **Monitor logs** actively
4. ✅ **Test incrementally** (one service at a time)
5. ✅ **Keep notes** of changes made

### After Migration

1. ✅ **Verify all services** working
2. ✅ **Test key workflows** (download, stream, etc.)
3. ✅ **Monitor for 24-48 hours**
4. ✅ **Update documentation** with any findings
5. ✅ **Create new backup** of working state

## Getting Help

If you encounter issues during migration:

1. **Check documentation**: All docs in `/docs`
2. **Search issues**: https://github.com/maiko/sdbx/issues
3. **Ask community**: GitHub Discussions
4. **Open issue**: Provide migration context, logs, and steps taken

---

**Successful migration?** Consider helping others by:
- Documenting your specific migration path
- Contributing to this guide via PR
- Answering questions in Discussions
