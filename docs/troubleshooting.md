# Troubleshooting & Support 🩺

Even with a magic wand like SDBX, things can sometimes go sideways. This guide helps you diagnose and fix common issues.

## 🛠️ The First Line of Defense: `sdbx doctor`

Always start by running the doctor. It checks the health of your host and your SDBX stack:

```bash
sdbx doctor
```

It will check for:
- Docker and Compose versions.
- Disk space availability.
- File and directory permissions.
- Critical ports (80, 443, 32400) being in use.
- Configuration and secret validity.

## 📝 Checking Logs

When a specific service isn't behaving, look at its logs:

```bash
sdbx logs sonarr                # Last 100 lines
sdbx logs -f radarr             # Stream logs in real-time
sdbx logs --tail 500 authelia   # View more history
```

## 🚩 Common Issues

### 1. "Can't access the dashboard (Timeout/Connection Refused)"
- **Check if services are running**: `sdbx status`
- **Check Exposure Mode**: If using `cloudflared`, ensure the tunnel is active in your Cloudflare Dashboard. If using `direct`, ensure ports 80 and 443 are open on your router.
- **Check DNS**: Ensure your domain points to your server's IP (for direct mode) or is correctly configured in Cloudflare (for tunnel mode).

### 2. "Downloads are stalled"
- **Check VPN Connectivity**: `sdbx doctor` includes a VPN check. If it fails, check your VPN credentials in `secrets/vpn_password.txt`.
- **Check qBittorrent Logs**: `sdbx logs qbittorrent`. Look for errors reaching tracker sites.
- **Update Docker Images**: Sometimes a provider's API changes. Run `sdbx update --safe`.

### 3. "Permission Denied" errors in *arr apps
- **Check UID/GID**: Ensure the `PUID` and `PGID` in your `.env` match the owner of your media library files.
- **Fix Permissions**:
  ```bash
  sudo chown -R $USER:$USER /path/to/media
  chmod -R 775 /path/to/media
  ```

### 4. "Authelia: Invalid Username or Password"
- **Check secrets**: Regenerate them if they were corrupted: `sdbx secrets generate`.
- **Reset Admin Password**: Follow the instructions in the [README](../README.md#5-first-login) to generate a new Argon2 hash.

## 🆘 Getting More Help

If you're still stuck:
1. **Search GitHub Issues**: Your problem might have been solved before.
2. **Open a New Issue**: Provide the output of `sdbx doctor` and the logs of the failing service.
3. **Community Discord**: Join our community for real-time help (link in GitHub repository).

---

> [!TIP]
> **Pro Tip**: Use `sdbx restart` as a first attempt for transient issues. It's the "turn it off and on again" of the SDBX world. 🔄
