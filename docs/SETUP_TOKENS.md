# SDBX Setup Tokens Guide

This guide explains the tokens required for SDBX setup and how to obtain them.

## Overview

SDBX uses two types of tokens for specific services:

1. **Cloudflare Tunnel Token** - Required when using `cloudflared` exposure mode
2. **Plex Claim Token** - Optional but recommended for linking your Plex server to your account

## Cloudflare Tunnel Token

### What is it?

The Cloudflare Tunnel token is a secure credential that allows the Cloudflare daemon (`cloudflared`) to establish a tunnel from your server to Cloudflare's edge network. This enables secure remote access to your services without opening firewall ports.

### When do I need it?

You need this token **only** if you selected `cloudflared` as your exposure mode during `sdbx init`. If you're using `lan` (local network only) or `direct` (public IP) modes, you can skip this.

### How to get it

1. Visit the [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/)
2. Sign in with your Cloudflare account (create one if needed - free tier is sufficient)
3. Navigate to **Networks → Tunnels**
4. Click **Create a tunnel**
5. Choose **Cloudflared** as the connector type
6. Name your tunnel (e.g., "sdbx-homeserver")
7. Click **Save tunnel**
8. On the next page, you'll see the tunnel token - a long base64-encoded string
9. Copy the entire token (starts with `eyJ...`)

### Where to provide it

**During Setup:**
- **CLI**: When running `sdbx init`, you'll be prompted for the token after selecting `cloudflared` mode
- **Web UI**: During the setup wizard, you'll see a dedicated "Cloudflare Tunnel Setup" page

**After Setup:**
If you skipped providing the token during setup:
1. Open `secrets/cloudflared_tunnel_token.txt` in your SDBX project directory
2. Paste the token and save
3. Restart containers: `sdbx down && sdbx up`

### Security Notes

- The token grants access to your Cloudflare tunnel - keep it secure
- Store it only in the `secrets/` directory (which is git-ignored)
- Never commit tokens to version control
- You can rotate the token by creating a new tunnel in the Cloudflare dashboard

## Plex Claim Token

### What is it?

The Plex claim token is a temporary credential (valid for 4 minutes) that links your Plex Media Server to your Plex account. This allows you to:
- Access your server remotely via plex.tv
- Share media with friends
- Use Plex apps on mobile/TV devices

### When do I need it?

You need this token **only** if you enabled the Plex addon during setup. The token is optional - you can claim your server via other methods.

### How to get it

1. Visit [https://plex.tv/claim](https://plex.tv/claim)
2. Sign in to your Plex account
3. Copy the claim token shown on the page (format: `claim-XXXXXXXXXXXXXXXXXXXX`)
4. **Important**: The token expires in 4 minutes - use it immediately!

### Where to provide it

**During Setup:**
- **CLI**: You'll be prompted during `sdbx up` (just before containers start) if Plex is enabled
- **Web UI**: You'll be prompted during `sdbx up` (same as CLI)

**Why not during init?**
The 4-minute expiration is too short for a complete setup wizard. By prompting right before starting containers, the token is used immediately.

### Alternative: Local Network Claiming

You don't need a claim token if you can access Plex via your local network:

1. Run `sdbx up` without providing a token (choose "Skip")
2. Wait for containers to start
3. Access `http://YOUR_SERVER_IP:32400/web` from a browser on the same network
4. Sign in to your Plex account
5. Your server will be automatically claimed

This method is recommended for local deployments or when you don't have a claim token ready.

### Security Notes

- Claim tokens are single-use and expire quickly (4 minutes)
- They only grant permission to link the server to your account
- After claiming, the server uses your Plex account credentials
- You can unclaim/reclaim servers anytime from plex.tv/web/app

## Token Storage

All tokens are stored in the `secrets/` directory with restrictive permissions (0600 - owner read/write only).

```
your-project/
├── secrets/
│   ├── cloudflared_tunnel_token.txt    # Cloudflare tunnel token
│   ├── plex_claim_token.txt            # Plex claim token
│   ├── authelia_jwt_secret.txt         # Auto-generated
│   └── ...other secrets...
```

The `secrets/` directory is automatically excluded from git via `.gitignore`.

## Troubleshooting

### Cloudflare Tunnel Issues

**Problem**: Tunnel not connecting, cloudflared container restarting
**Solution**:
1. Check token is correctly pasted in `secrets/cloudflared_tunnel_token.txt`
2. Verify token doesn't have extra whitespace/newlines
3. Check logs: `docker logs sdbx-cloudflared`
4. Verify tunnel exists in Cloudflare dashboard and is active

**Problem**: Services not accessible via tunnel domain
**Solution**:
1. Verify DNS is configured in Cloudflare dashboard
2. Check tunnel routes are configured correctly
3. Ensure Traefik is running: `docker ps | grep traefik`

### Plex Claim Issues

**Problem**: "Token expired" error
**Solution**:
- Get a fresh token from https://plex.tv/claim
- Run `sdbx up` immediately after copying the token

**Problem**: Server claimed but not showing in plex.tv
**Solution**:
- Wait 1-2 minutes for server to appear
- Restart Plex: `docker restart sdbx-plex`
- Check Plex logs: `docker logs sdbx-plex`

**Problem**: Want to claim server but containers already running
**Solution**:
Option 1 - Via local network (recommended):
1. Access `http://YOUR_SERVER_IP:32400/web`
2. Sign in to claim

Option 2 - Via claim token:
1. Get token from https://plex.tv/claim
2. Run: `docker exec -e PLEX_CLAIM="token" sdbx-plex /bin/bash -c "echo 'Claiming...'"`
3. Restart: `docker restart sdbx-plex`

Option 3 - Recreate with token:
1. Add token to `secrets/plex_claim_token.txt`
2. Run: `sdbx down && sdbx up`

## Token Rotation

### Cloudflare Tunnel Token

To rotate your tunnel token:
1. Create a new tunnel in Cloudflare dashboard
2. Copy the new token to `secrets/cloudflared_tunnel_token.txt`
3. Update DNS records to point to new tunnel
4. Restart: `docker restart sdbx-cloudflared`
5. Delete old tunnel from dashboard after verifying

### Plex Claim Token

Plex claim tokens are single-use. To reclaim your server:
1. Unclaim from plex.tv/web/app (Settings → Manage → Server → Remove)
2. Get new token from https://plex.tv/claim
3. Follow claiming process again

## Additional Resources

- [Cloudflare Tunnel Documentation](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)
- [Plex Server Claiming Guide](https://support.plex.tv/articles/204604227-why-am-i-locked-out-of-server-settings/)
- [SDBX GitHub Issues](https://github.com/maiko/sdbx/issues) - Report problems or ask questions

## Security Best Practices

1. **Never commit tokens to git** - They're excluded by default in `.gitignore`
2. **Restrict file permissions** - SDBX sets secrets to 0600 automatically
3. **Rotate tokens periodically** - Especially Cloudflare tunnel tokens
4. **Use strong Plex passwords** - Your Plex account secures your media
5. **Enable 2FA on Cloudflare** - Protects your tunnel configuration
6. **Backup secrets directory** - Include in your backup strategy

## Getting Help

If you encounter issues not covered in this guide:

1. Check the [SDBX documentation](https://github.com/maiko/sdbx/tree/main/docs)
2. Search [existing GitHub issues](https://github.com/maiko/sdbx/issues)
3. Create a new issue with:
   - SDBX version (`sdbx --version`)
   - Error messages (sanitize any tokens!)
   - Steps to reproduce

**Never share your actual tokens when asking for help!**
