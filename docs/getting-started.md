# Getting Started with SDBX 🚀

Welcome to **SDBX**! This guide will walk you through setting up your own professional-grade seedbox automation stack from scratch.

## 🏁 Introduction

SDBX is designed to be a **"set it and forget it"** solution for managing your media collection. It combines the best open-source tools into a cohesive, secure, and easy-to-manage platform.

### What you'll get
- 📺 **Media Server** (Plex)
- 🤖 **Automation** (Sonarr, Radarr, Prowlarr)
- 📥 **Download Client** (qBittorrent via VPN)
- 🛡️ **SSO Security** (Authelia)
- 📊 **Dashboard** (Homepage)

---

## 🛠️ Step 1: Preparation

Before you begin, ensure you have the following:

- **🐧 A Linux Server**: Debian 11+ or Ubuntu 20.04+.
  - *Minimum specs*: 2 CPU, 4GB RAM.
  - *Recommended*: 4 CPU, 8GB+ RAM (especially for Plex transcoding).
- **🌐 A Domain Name**: You need a domain (e.g., `myseedbox.com`) to access your services securily.
- **🛡️ VPN Provider (Optional)**: Highly recommended for privacy. SDBX supports:
  - NordVPN, ProtonVPN, PIA, Mullvad, Surfshark, and more via Gluetun.

---

## 📥 Step 2: Installation

Install the SDBX CLI with a single command:

```bash
curl -fsSL https://github.com/maiko/sdbx/releases/latest/download/sdbx-linux-amd64 -o sdbx
chmod +x sdbx
sudo mv sdbx /usr/local/bin/
```

Verify it works:
```bash
sdbx --version
```

---

## 🧙‍♂️ Step 3: Initialization

Create a directory for your project and summon the setup wizard:

```bash
mkdir ~/seedbox && cd ~/seedbox
sdbx init
```

The graphical wizard will ask you for:

1. **Domain Name**: e.g., `yourdomain.com`
2. **Exposure Mode**:
   - `cloudflared` 🛡️ — **Recommended**. No open ports, uses Cloudflare Tunnel (Zero Trust).
   - `direct` 🌐 — Standard setup. Requires ports 80 & 443 open. Uses Let's Encrypt.
   - `lan` 🏠 — HTTP only. Best for home labs behind a separate proxy.
3. **Routing Strategy**:
   - `subdomain` (Default) — Services at `sonarr.domain.com`, `radarr.domain.com`.
   - `path` — Services at `box.domain.com/sonarr`, `box.domain.com/radarr`.
4. **VPN Credentials**: Your provider's username/password (safely stored in secrets).
5. **Admin User**: The master account for Authelia SSO.

---

## 🚀 Step 4: Deploy

Ignition! 🚀

```bash
sdbx up
```

This will:
1. Pull all Docker images
2. Create secure networks
3. Start the VPN tunnel
4. Launch all services

Watch the logs if you're curious:
```bash
sdbx logs -f
```

---

## 🩺 Step 5: Health Check

Ensure all systems are operational:

```bash
sdbx doctor
```

✅ **Green checks** mean you're good to go!
❌ **Red crosses** will come with suggestions on how to fix common issues.

---

## 📺 Step 6: First Login

Time to see your creation.

**If you chose Subdomain routing:**
👉 Go to `https://home.yourdomain.com`

**If you chose Path routing:**
👉 Go to `https://yourdomain.com/` (or your configured base subdomain)

Login with the **Admin** credentials you created during init.

### 🎉 Success!
You should see the **Homepage** dashboard with status indicators for all your services.

**Next Steps:**
1. Open **Plex** (`sdbx open plex`) and claim your server.
2. Open **Sonarr/Radarr** (`sdbx open sonarr`) and add your root folders (`/tv`, `/movies`).
3. Open **Prowlarr** (`sdbx open prowlarr`) and add your indexers.

---

## 🛡️ Security Best Practices

By default, SDBX is secure. But for production use, we recommend:

1. **Enable 2FA**: Switch Authelia to `two_factor` policy in `configs/authelia/configuration.yml`.
2. **Backup Secrets**: Keep your `secrets/` folder safe (and **never** commit it to git).
3. **Updates**: Run `sdbx update` regularly to keep containers patched.

---

**Need help?**
Check out the [Troubleshooting Guide](troubleshooting.md) or open an issue on GitHub.
