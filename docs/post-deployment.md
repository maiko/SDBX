# Post-Deployment Configuration

This guide walks you through configuring your services after running `sdbx up` for the first time.

## Table of Contents

- [First Steps](#first-steps)
- [Authelia Configuration](#authelia-configuration)
- [Prowlarr Setup](#prowlarr-setup)
- [Download Client Configuration](#download-client-configuration)
- [Sonarr Configuration](#sonarr-configuration)
- [Radarr Configuration](#radarr-configuration)
- [Plex Setup](#plex-setup)
- [Optional Services](#optional-services)
- [Quality Profiles](#quality-profiles)
- [Automation](#automation)

## First Steps

### 1. Verify Services Are Running

```bash
# Check service health
sdbx status

# Run diagnostics
sdbx doctor

# View logs if any issues
sdbx logs
```

All services should show as "healthy" or "running".

### 2. Access Your Dashboard

Open Homepage in your browser:

```bash
# Open automatically
sdbx open homepage

# Or manually:
# - LAN mode: http://YOUR_IP/
# - Direct/Cloudflare: https://sdbx.yourdomain.com/
```

## Authelia Configuration

### Initial Login

1. Navigate to Authelia:
   ```bash
   sdbx open authelia
   ```

2. Login with credentials from `sdbx init`:
   - Username: (what you set during init, default: `admin`)
   - Password: (what you set during init)

### Enable Two-Factor Authentication (2FA)

**Highly recommended for security!**

1. Click your username (top right) → **Settings**
2. Go to **Two-Factor Authentication**
3. Click **Register device**
4. Scan QR code with your authenticator app:
   - Google Authenticator
   - Authy
   - Microsoft Authenticator
   - 1Password
   - Bitwarden

5. Enter the 6-digit code to verify
6. **Save your backup codes** somewhere safe
7. Test 2FA by logging out and back in

### Add Additional Users

1. Edit `configs/authelia/users_database.yml`:
   ```yaml
   users:
     admin:
       # ... existing admin user ...

     family_member:
       displayname: "Family Member"
       password: "$argon2id$..."  # See below
       email: "member@example.com"
       groups:
         - users  # or 'admins' for full access
   ```

2. Generate password hash:
   ```bash
   docker run --rm authelia/authelia:latest authelia hash-password 'their_password'
   ```

3. Copy the hash (starting with `$argon2id$...`) into the YAML

4. Restart Authelia:
   ```bash
   sdbx restart authelia
   ```

5. User can now login and set up their own 2FA

## Prowlarr Setup

Prowlarr manages indexers (torrent/usenet sites) for all *arr apps.

### 1. Access Prowlarr

```bash
sdbx open prowlarr
```

### 2. Add Indexers

#### Public Indexers (No Account Needed)

1. Go to **Indexers** → **Add Indexer**
2. Search for popular public trackers:
   - **1337x**
   - **The Pirate Bay**
   - **RARBG** (if still available)
   - **EZTV** (TV)
   - **YTS** (Movies)

3. For each:
   - Click indexer name
   - Click **Test** to verify it works
   - Click **Save**

#### Private Indexers (Require Account)

If you have accounts with private trackers:

1. **Add Indexer** → Search for your tracker
2. Enter your credentials:
   - API key (if available)
   - Or username/password
   - Cookie (some trackers)

3. **Test** and **Save**

#### FlareSolverr for Cloudflare-Protected Sites

Some indexers use Cloudflare protection. If enabled:

1. Ensure FlareSolverr addon is enabled:
   ```bash
   sdbx addon enable flaresolverr
   sdbx up
   ```

2. In Prowlarr: **Settings** → **Indexers** → **FlareSolverr**
   - Host: `http://flaresolverr:8191`
   - Tags: Add tag (e.g., `flaresolverr`)

3. Tag indexers that need it:
   - Edit indexer → Tags → Add `flaresolverr`

### 3. Connect Prowlarr to Apps

Prowlarr can automatically sync indexers to Sonarr/Radarr/etc.

1. Go to **Settings** → **Apps**
2. Click **+** → Select **Sonarr**
3. Configure:
   - Prowlarr Server: `http://prowlarr:9696`
   - Sonarr Server: `http://sonarr:8989`
   - API Key: Get from Sonarr (Settings → General → Security → API Key)
   - Sync Level: **Add and Remove Only** (recommended)

4. **Test** and **Save**
5. Repeat for Radarr, Lidarr, Readarr

**Result**: All indexers will automatically appear in your *arr apps!

## Download Client Configuration

Configure qBittorrent as the download client for all *arr apps.

### 1. Access qBittorrent

```bash
sdbx open qbittorrent
```

Login:
- Username: `admin`
- Password: (from `secrets/qbittorrent_password.txt`)
  ```bash
  cat secrets/qbittorrent_password.txt
  ```

### 2. Configure qBittorrent

#### Basic Settings

1. **Tools** → **Options**
2. **Downloads**:
   - Default Save Path: `/data/downloads/complete`
   - Temp Path: `/data/downloads/incomplete`
   - ✅ Append .!qB extension to incomplete files
   - ✅ Pre-allocate disk space
   - ✅ Keep incomplete torrents in: `/data/downloads/incomplete`

3. **Connection**:
   - Port: `8080` (default)
   - ✅ Use UPnP/NAT-PMP: **Disabled** (VPN handles this)

4. **Speed**:
   - Set upload/download limits based on your connection
   - Alternative Rate Limits: For daytime throttling

5. **BitTorrent**:
   - Privacy:
     - ✅ Enable DHT
     - ✅ Enable PeX
     - ✅ Enable Local Peer Discovery
   - Seeding Limits:
     - Ratio: 2.0 (adjust based on tracker requirements)
     - Time: 0 (unlimited)

6. **Web UI**:
   - ✅ Enable Web UI
   - Port: `8080`

7. Click **Save**

#### Categories (Organizational)

1. Right-click in qBittorrent → **Add Category**
2. Create categories:
   - `sonarr` → Save Path: `/data/downloads/complete/sonarr`
   - `radarr` → Save Path: `/data/downloads/complete/radarr`
   - `lidarr` → Save Path: `/data/downloads/complete/lidarr`
   - `readarr` → Save Path: `/data/downloads/complete/readarr`
   - `manual` → Save Path: `/data/downloads/complete`

### 3. Verify VPN

**Critical**: Ensure downloads go through VPN!

1. In qBittorrent, go to **Tools** → **Options** → **Advanced**
2. **Network Interface**: Should be `tun0` (VPN interface)
3. Test your IP:
   - Add a torrent (any test torrent)
   - Check: https://ipleak.net/
   - Your real IP should NOT appear

```bash
# Verify VPN from command line
docker exec sdbx-qbittorrent-1 curl https://ifconfig.me

# Should show VPN IP, not your real IP
```

## Sonarr Configuration

Sonarr automates TV show downloads.

### 1. Access Sonarr

```bash
sdbx open sonarr
```

### 2. Add Root Folder

1. **Settings** → **Media Management**
2. Scroll to **Root Folders**
3. Click **Add Root Folder**
4. Enter: `/data/media/tv`
5. **OK**

### 3. Add Download Client

1. **Settings** → **Download Clients**
2. Click **+** → **qBittorrent**
3. Configure:
   - Name: `qBittorrent`
   - Host: `qbittorrent`
   - Port: `8080`
   - Username: `admin`
   - Password: (from `secrets/qbittorrent_password.txt`)
   - Category: `sonarr`
4. **Test** → **Save**

### 4. Quality Profile

1. **Settings** → **Profiles**
2. **Quality Profiles** → Edit **HD-1080p** (or create new):
   - Allowed qualities: HD-1080p, HDTV-1080p, WEB-1080p
   - Upgrade until: WEB-1080p (highest quality)

### 5. Add Indexers (if not using Prowlarr sync)

If Prowlarr isn't syncing:
1. **Settings** → **Indexers**
2. Add manually (Prowlarr is easier though!)

### 6. Add Your First Show

1. **Series** → **Add New Series**
2. Search for a show (e.g., "Breaking Bad")
3. Select show from results
4. Configure:
   - Root Folder: `/data/media/tv`
   - Quality Profile: **HD-1080p**
   - Series Type: **Standard** (or Anime if applicable)
   - ✅ Monitor: **All Episodes**
   - ✅ Search for missing episodes
5. **Add Series**

Sonarr will now:
- Search for all episodes
- Download them via qBittorrent
- Rename and move to `/data/media/tv/Breaking Bad/`
- Continue monitoring for new episodes

## Radarr Configuration

Radarr automates movie downloads. Setup is nearly identical to Sonarr.

### 1. Access Radarr

```bash
sdbx open radarr
```

### 2. Add Root Folder

1. **Settings** → **Media Management**
2. **Root Folders** → **Add Root Folder**
3. Enter: `/data/media/movies`

### 3. Add Download Client

1. **Settings** → **Download Clients**
2. Click **+** → **qBittorrent**
3. Configure (same as Sonarr, but use category `radarr`)

### 4. Quality Profile

1. **Settings** → **Profiles**
2. Edit **HD-1080p** or create:
   - Allowed: Bluray-1080p, WEB-1080p
   - Upgrade Until: Bluray-1080p

### 5. Add Your First Movie

1. **Movies** → **Add New Movie**
2. Search for a movie
3. Configure:
   - Root Folder: `/data/media/movies`
   - Quality Profile: **HD-1080p**
   - ✅ Monitor: **Yes**
   - ✅ Search for movie
4. **Add Movie**

## Plex Setup

### 1. Initial Access

```bash
sdbx open plex
```

### 2. Claim Your Server

If this is your first time:

1. You'll be prompted to sign in to Plex
2. Sign in or create account
3. Give your server a name (e.g., "SDBX Media Server")

If not prompted:
```bash
# Get a claim token
# Visit: https://www.plex.tv/claim
# Add to .env:
echo "PLEX_CLAIM=claim-XXXXXXXXX" >> .env

# Restart Plex
sdbx restart plex
```

### 3. Add Libraries

1. Click **Add Library**

2. **Movies**:
   - Library type: Movies
   - Name: Movies
   - Add Folder: `/data/media/movies`
   - **Add Library**

3. **TV Shows**:
   - Library type: TV Shows
   - Name: TV Shows
   - Add Folder: `/data/media/tv`
   - **Add Library**

4. **Music** (if using Lidarr):
   - Library type: Music
   - Add Folder: `/data/media/music`

5. **Books** (if using Readarr):
   - Library type: Other
   - Add Folder: `/data/media/books`

### 4. Configure Settings

1. **Settings** → **Server** → **Network**
   - Custom server access URLs: (your domain)

2. **Settings** → **Server** → **Transcoder**
   - Transcoder quality: Automatic
   - ✅ Use hardware acceleration when available

3. **Settings** → **Server** → **Library**
   - ✅ Scan my library automatically
   - ✅ Run a partial scan when changes are detected

4. **Save Changes**

### 5. Share with Users

1. **Settings** → **Users & Sharing**
2. **Invite Friend** → Enter email
3. Select libraries to share
4. Set restrictions if needed
5. **Send Invite**

## Optional Services

### Overseerr (Media Requests)

If enabled, users can request movies/TV shows:

1. Access Overseerr:
   ```bash
   sdbx open overseerr
   ```

2. Sign in with Plex
3. **Settings**:
   - **Plex**: Connect to Plex server
   - **Services**:
     - Add Sonarr: http://sonarr:8989
     - Add Radarr: http://radarr:7878

4. Configure:
   - Default quality profiles
   - Root folders
   - Permissions for users

Now users can request content via Overseerr!

### Wizarr (User Onboarding)

Simplifies inviting users to Plex:

1. Access Wizarr:
   ```bash
   sdbx open wizarr
   ```

2. Connect to Plex
3. Create invitation link
4. Share link with users
5. They follow wizard to create account

### Tautulli (Plex Analytics)

Monitor Plex usage:

1. Access Tautulli:
   ```bash
   sdbx open tautulli
   ```

2. Connect to Plex
3. View:
   - Current streams
   - Watch history
   - User statistics
   - Most popular content

### Bazarr (Subtitles)

Automate subtitle downloads:

1. Access Bazarr:
   ```bash
   sdbx open bazarr
   ```

2. **Settings** → **Languages**:
   - Add your preferred subtitle languages

3. **Settings** → **Providers**:
   - Enable subtitle providers (OpenSubtitles, etc.)
   - Add credentials if needed

4. **Settings** → **Sonarr/Radarr**:
   - Connect to Sonarr/Radarr

## Quality Profiles

### Using Recyclarr (Recommended)

Recyclarr automatically configures optimal quality profiles:

1. Recyclarr runs automatically
2. Check logs:
   ```bash
   sdbx logs recyclarr
   ```

3. Profiles applied to Sonarr/Radarr:
   - Release priorities
   - Format preferences
   - Size limits
   - Quality tiers

### Manual Configuration

If you prefer custom profiles:

#### Sonarr Quality Profile

1. **Settings** → **Profiles**
2. **Quality Profiles** → Edit or create new
3. Set preferences:
   - Minimum: HDTV-720p
   - Cutoff: WEB-1080p
   - Allowed: HDTV-720p, HDTV-1080p, WEB-720p, WEB-1080p

#### Radarr Quality Profile

1. **Settings** → **Profiles**
2. Similar to Sonarr, but prefer:
   - Bluray over WEB
   - Remuxes for quality enthusiasts

## Automation

### Automatic Show/Movie Management

#### Sonarr Lists

Auto-add shows from lists:

1. **Settings** → **Import Lists**
2. Examples:
   - **Trakt Lists**: Popular shows
   - **IMDb Lists**: Your watchlist
   - **Sonarr Lists**: Another Sonarr instance

#### Radarr Lists

Similar to Sonarr:
- Trakt, IMDb, TMDb lists
- Auto-discover new releases
- Popular movies

### Notifications

Set up notifications for completed downloads:

#### Discord

1. In Sonarr/Radarr: **Settings** → **Connect**
2. Click **+** → **Discord**
3. Add Discord webhook URL
4. Select triggers:
   - On Download
   - On Import
   - On Upgrade

#### Email

1. **Settings** → **Connect** → **Email**
2. SMTP configuration
3. Select triggers

### Scheduling

#### Download Times

If you have bandwidth caps:

1. qBittorrent: **Options** → **Speed** → **Schedule**
2. Set slower speeds during day
3. Full speed at night

#### Scan Times

Prevent scans during peak viewing:

1. Sonarr/Radarr: **Settings** → **General**
2. **RSS Sync Interval**: Adjust frequency

## Post-Configuration Checklist

### Security

- ✅ 2FA enabled in Authelia
- ✅ Strong passwords used
- ✅ VPN verified working
- ✅ Services only accessible via authentication

### Configuration

- ✅ All root folders added
- ✅ Download client connected
- ✅ Indexers added (via Prowlarr)
- ✅ Quality profiles set
- ✅ Plex libraries created

### Testing

- ✅ Add a test TV show → Verify it downloads
- ✅ Add a test movie → Verify it downloads
- ✅ Stream on Plex → Verify it plays
- ✅ Check qBittorrent → Verify VPN IP

### Monitoring

- ✅ Homepage accessible
- ✅ All services green in `sdbx status`
- ✅ `sdbx doctor` passes all checks

## Next Steps

1. **Explore your services**: Each has many features!
2. **Read service docs**: Sonarr/Radarr have extensive wikis
3. **Join communities**:
   - /r/sonarr
   - /r/radarr
   - /r/plex
   - SDBX Discussions

4. **Set up backups**:
   ```bash
   # Create regular backups
   sdbx backup run

   # Consider automating with cron
   0 2 * * 0 /usr/local/bin/sdbx backup run
   ```

5. **Monitor regularly**:
   ```bash
   sdbx status
   sdbx doctor
   ```

## Troubleshooting

### Service Not Accessible

- Check: `sdbx status`
- View logs: `sdbx logs <service>`
- Verify DNS/routing
- Check Authelia logs

### Downloads Not Starting

- Check Prowlarr has indexers
- Verify download client in *arr app
- Check qBittorrent is running
- View *arr app logs

### VPN Issues

- Check: `sdbx logs gluetun`
- Verify credentials in secrets/
- Test VPN IP as shown above

---

**Configuration complete?** Enjoy your fully automated media server! 🎉

For more help, see:
- [FAQ](faq.md)
- [Troubleshooting](troubleshooting.md)
- [GitHub Discussions](https://github.com/maiko/sdbx/discussions)
