# Quick Start: Zero to Plex in 10 Minutes

This guide gets you from nothing to a working Plex media server on your local network, protected by a login page. No domain name required, no VPN setup, no complicated networking. Just Plex, running and streaming your media.

## What you'll get

By the end of this guide, you'll have:

- **Plex Media Server** — Stream your movies and TV shows to any device on your network
- **Authelia** — A login page that protects your services with a username and password
- **Traefik** — A reverse proxy that ties everything together (you won't need to touch this)
- **SDBX Web UI** — A dashboard to monitor and manage your stack

All of this runs in Docker containers, which means it's isolated from the rest of your system and easy to remove if you change your mind.

## Prerequisites

You'll need three things:

1. **Docker** installed and running on your computer or server. If you don't have it yet, follow the [official Docker install guide](https://docs.docker.com/engine/install/). After installing, make sure you can run `docker ps` without errors.

2. **A folder with media files** (movies, TV shows, home videos, etc.) that you want to stream. This can be on an internal drive, an external drive, or a network share that's mounted on your machine.

3. **10 minutes** of your time.

## Step 1: Install SDBX

Open a terminal and run:

```bash
curl -fsSL https://raw.githubusercontent.com/maiko/SDBX/main/install.sh | bash
```

This downloads and installs the `sdbx` command on your system. Once it finishes, verify it worked:

```bash
sdbx version
```

You should see a version number printed. If you get "command not found," try opening a new terminal window or adding `/usr/local/bin` to your PATH.

<details>
<summary>Manual install (if the script doesn't work)</summary>

```bash
# Download the binary (change amd64 to arm64 if you're on an ARM machine like a Raspberry Pi)
curl -LO https://github.com/maiko/SDBX/releases/latest/download/sdbx_linux_amd64.tar.gz

# Extract it
tar -xzf sdbx_linux_amd64.tar.gz

# Move it somewhere your system can find it
sudo mv sdbx /usr/local/bin/
sudo chmod +x /usr/local/bin/sdbx
```

</details>

## Step 2: Run the setup wizard

Create a folder for your seedbox project and start the wizard:

```bash
mkdir ~/seedbox && cd ~/seedbox
sdbx init
```

The wizard walks you through a series of questions. Here's exactly what to pick for this minimal setup:

### Domain

Enter your server's local IP address (like `192.168.1.100`) or just type `localhost` if you'll only access Plex from this machine. To find your IP, run `hostname -I` and use the first address shown.

```
Domain: 192.168.1.100
```

### Exposure mode

Choose **lan**. This keeps everything on your local network using plain HTTP. No SSL certificates or domain configuration needed.

```
Exposure mode: lan
```

### Routing strategy

Choose **subdomain**. For LAN mode, this setting doesn't have a noticeable effect, so either option works.

```
Routing strategy: subdomain
```

### Admin credentials

Pick a username and password. This is what you'll use to log in to your services. Choose something you'll remember.

```
Admin username: admin
Admin password: (pick something secure)
```

### Storage paths

Point SDBX to your folders. You'll need three paths:

- **Media path** — Where your movies and TV shows live (e.g., `/home/you/media` or `/mnt/external/media`)
- **Downloads path** — Where downloaded files go temporarily (e.g., `/home/you/downloads`)
- **Config path** — Where SDBX stores service configurations (e.g., `/home/you/seedbox/configs` — the default is fine)

```
Media path: /home/you/media
Downloads path: /home/you/downloads
Config path: ./configs
```

### VPN

Choose **disabled** (or skip). We're keeping things simple for now. You can always add a VPN later.

```
VPN: disabled
```

### Addons

Choose the **Minimal** preset or don't enable any addons. For this guide, we only need the core services.

```
Addon preset: Minimal
```

The wizard will generate your configuration files. You should see a summary of what was created.

## Step 3: Start everything up

```bash
sdbx up
```

This pulls the Docker images and starts all the containers. The first run takes a few minutes because it needs to download the images. You'll see progress as each service starts.

## Step 4: Verify everything is healthy

```bash
sdbx doctor
```

This runs a series of health checks. You want to see green checkmarks across the board. If something shows a red X, the output will tell you what's wrong and how to fix it.

You can also check the status of individual services:

```bash
sdbx status
```

## Step 5: Access Plex

Open your web browser and go to:

```
http://YOUR-SERVER-IP:32400/web
```

Replace `YOUR-SERVER-IP` with the IP address you entered during setup (e.g., `http://192.168.1.100:32400/web`).

### Complete the Plex setup wizard

1. **Sign in** with your Plex account (create one at [plex.tv](https://www.plex.tv) if you don't have one — it's free).
2. **Name your server** — call it whatever you like.
3. **Add your media library** — click "Add Library," choose the type (Movies, TV Shows, etc.), and browse to the `/media` folder inside the container. Your files from the media path you configured will appear here.
4. **Finish setup** and let Plex scan your library. Depending on how many files you have, this could take a few minutes to a few hours.

Once the scan is done, your media will appear in Plex and you can stream it to any device on your network.

## Step 6: Access the SDBX dashboard

You can manage your stack through the SDBX Web UI:

```
http://YOUR-SERVER-IP/
```

Log in with the admin credentials you created during setup. From the dashboard you can monitor service health, view logs, and manage your stack.

## Next steps

Now that Plex is running, here are some things you might want to do next.

### Add media automation

Want Sonarr and Radarr to automatically find and download TV shows and movies? Enable them as addons:

```bash
cd ~/seedbox
sdbx addon enable sonarr
sdbx addon enable radarr
sdbx addon enable prowlarr
sdbx up
```

Then check out the [Post-Deployment Guide](post-deployment.md) for how to configure them to work together.

### Add a domain name

If you want to access your services from outside your home network, you can switch from LAN mode to either `direct` (with a domain and Let's Encrypt SSL) or `cloudflared` (with a Cloudflare Tunnel for zero-trust access). See the [Getting Started guide](getting-started.md) for details on exposure modes.

### Enable a VPN

To protect your download traffic with a VPN:

```bash
sdbx vpn configure
sdbx up
```

This walks you through choosing a VPN provider and entering your credentials. SDBX supports 17 providers including NordVPN, ProtonVPN, Mullvad, and more. Run `sdbx vpn providers` to see the full list.

### Explore more addons

SDBX has 27 optional addons. Browse them with:

```bash
sdbx addon list --all
sdbx addon search media
```

Popular picks include:
- **overseerr** — Let family and friends request movies and shows
- **tautulli** — See who's watching what on your Plex server
- **bazarr** — Automatically download subtitles

## Troubleshooting

### "command not found" after installing SDBX

Your shell doesn't know where the `sdbx` binary is. Try:

```bash
export PATH=$PATH:/usr/local/bin
```

Then add that line to your `~/.bashrc` or `~/.zshrc` so it persists.

### "Cannot connect to the Docker daemon"

Docker isn't running. Start it with:

```bash
sudo systemctl start docker
```

If you just installed Docker, you may also need to add your user to the docker group:

```bash
sudo usermod -aG docker $USER
```

Then **log out and log back in** for the group change to take effect.

### Port 32400 is already in use

Something else is using Plex's port. Check what it is:

```bash
sudo lsof -i :32400
```

If it's an existing Plex installation, stop it first. If it's something else, you'll need to free up that port or adjust your configuration.

### Plex can't find my media files

Double-check the media path you entered during setup. The path must be:

- An **absolute path** (starts with `/`), not a relative one
- A directory that **exists** on your machine
- **Readable** by the Docker process (run `ls -la /your/media/path` to check permissions)

If you need to change the path, edit your config:

```bash
sdbx config set media_path /correct/path/to/media
sdbx up
```

### Services show as unhealthy in `sdbx doctor`

Give them a minute. Some services (especially Plex on first run) take a little while to fully start. Run `sdbx doctor` again after 30 seconds. If the problem persists, check the logs:

```bash
sdbx logs plex
```

### Everything else

Check the [Troubleshooting Guide](troubleshooting.md) for more solutions, or open an issue on [GitHub](https://github.com/maiko/SDBX/issues).
