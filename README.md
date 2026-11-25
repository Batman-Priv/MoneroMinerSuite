# 🚀 Monero Miner Suite

<img src="https://github.com/user-attachments/assets/b20a6dda-30a8-4fc6-9c01-025ad9172035">

**Enterprise-grade Monero (XMR) mining management platform** with a beautiful web dashboard, auto-healing agents, GPU support, and a unified control plane for fleets of miners.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Windows-lightgrey)](https://github.com/Batman-Priv/MoneroMinerSuite)
[![XMRig](https://img.shields.io/badge/XMRig-v6.26.0-orange)](https://github.com/xmrig/xmrig)

---

## 📖 Table of Contents

- [Overview](#-overview)
- [Architecture](#-architecture)
- [Features](#-features)
- [Prerequisites](#-prerequisites)
- [Quick Start](#-quick-start)
  - [1. Build the Binaries](#1-build-the-binaries)
  - [2. Install the Agent (On Each Mining Server)](#2-install-the-agent-on-each-mining-server)
  - [3. Launch the Orchestrator Panel](#3-launch-the-orchestrator-panel)
  - [4. Add Servers to the Panel](#4-add-servers-to-the-panel)
- [Configuration](#-configuration)
  - [Working Modes](#working-modes)
  - [Changing Settings After Installation](#changing-settings-after-installation)
  - [Uninstalling](#uninstalling)
- [Dashboard Overview](#-dashboard-overview)
- [Security Best Practices](#-security-best-practices)
- [File Structure Reference](#-file-structure-reference)
- [Troubleshooting](#-troubleshooting)
- [Frequently Asked Questions (FAQ)](#-frequently-asked-questions-faq)
- [Contributing](#-contributing)
- [License](#-license)
- [Disclaimer](#-disclaimer)
- [Acknowledgements](#-acknowledgements)

---

## ✨ Overview

**MoneroMinerSuite** is a complete, self-hosted solution for managing Monero (XMR) mining operations across multiple Linux and Windows servers. It combines:

- **Auto-healing agents** that download, configure, and run XMRig as a background service
- **A central orchestrator** with a beautiful web dashboard for monitoring and control
- **Automatic XMRig updates**, crash-loop protection, and graceful shutdowns
- **Real-time hashrate graphs**, live CPU temperature gauges, and earnings estimates

Whether you're running a single rig or a hundred, MoneroMinerSuite gives you enterprise-grade tooling without the enterprise price tag.

### Why Monero?
Monero (XMR) is the leading privacy-focused cryptocurrency. Unlike Bitcoin, Monero transactions are completely anonymous and untraceable. Its RandomX proof-of-work algorithm is designed to be ASIC-resistant, making CPU mining viable and decentralized.

### Why This Suite?
Most mining setups require manual configuration on each server, separate monitoring tools, and constant babysitting. MoneroMinerSuite automates everything:
- ✅ One command per server to get mining
- ✅ Central dashboard to monitor everything
- ✅ Automatic recovery from failures
- ✅ No cloud dependencies — runs entirely on your infrastructure

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Your Network                                      │
│                                                                                 │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐                    │
│  │   Server 1     │  │   Server 2     │  │   Server N     │                    │
│  │   ──────────   │  │   ──────────   │  │   ──────────   │                    │
│  │  miner-agent   │  │  miner-agent   │  │  miner-agent   │                    │
│  │  (systemd/win) │  │  (systemd/win) │  │  (systemd/win) │                    │
│  │       ↓        │  │       ↓        │  │       ↓        │                    │
│  │    XMRig       │  │    XMRig       │  │    XMRig       │                    │
│  │  (mining)      │  │  (mining)      │  │  (mining)      │                    │
│  └────────┬───────┘  └────────┬───────┘  └────────┬───────┘                    │
│           │                   │                   │                            │
│           │   REST API (port 9090, Bearer Token auth)                          │
│           └───────────────────┼───────────────────┘                            │
│                               │                                                │
│                        ┌──────▼──────┐                                         │
│                        │ Orchestrator│  ←── Web Dashboard :8080                │
│                        │   Panel     │       (User/Pass protected)             │
│                        └─────────────┘                                         │
│                                                                                 │
│  ┌──────────────────────────────────────────────────────────────┐              │
│  │  All agents communicate with orchestrator via REST APIs      │              │
│  │  using per-agent bearer tokens for authentication.           │              │
│  └──────────────────────────────────────────────────────────────┘              │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

| Component | Language | Purpose |
|-----------|----------|---------|
| **miner-agent** | Go | Runs on each mining server. Downloads XMRig, configures it, monitors temperature, auto-updates, and reports status. |
| **miner-orchestrator** | Go | Central control plane. Hosts the web dashboard, aggregates data, and sends commands to agents. |
| **XMRig** | C++ | The actual miner binary (auto-downloaded by the agent). |
| **Web Dashboard** | HTML/CSS/JS | Beautiful, responsive interface for monitoring and control. |

### Communication Flow
1. **Agent → Orchestrator**: Periodically or on-demand, the orchestrator queries each agent's `/status` endpoint.
2. **Orchestrator → Agent**: Commands (start/stop/mode change) are sent via POST requests.
3. **Authentication**: Each request includes a Bearer token that both sides validate.
4. **Firewall**: Agent port 9090 should be restricted to the orchestrator's IP only.

---

## 🎯 Features

### 🛡️ Reliability & Stability
- **Crash-loop detection** — stops restarting after 6 crashes in 3 minutes (prevents resource waste on configuration errors)
- **Graceful shutdown** — sends SIGTERM (Linux) or CTRL_BREAK (Windows) and waits up to 15 seconds for clean exit
- **Auto-restart** — restarts XMRig immediately if it dies (with exponential backoff up to 30 seconds)
- **Log rotation** — agent logs auto-rotate at 20 MB to prevent disk filling
- **In-memory XMRig logs** — last 500 lines stored in RAM, served via API (no disk I/O for miner logs)
- **Service-level recovery** — both systemd (Linux) and Windows Service Manager automatically restart the agent if it crashes

### 🔄 Self-Healing & Maintenance
- **Automatic XMRig updates** — checks GitHub every 24 hours and updates silently (can be disabled)
- **Pool failover** — configure multiple pools; XMRig switches automatically if the primary fails
- **Wallet address validation** — CryptoNote base58 checksum validation before installation (catches typos early)
- **System tuning** — automatically enables huge pages, MSR mod (Linux), and Lock Pages in Memory (Windows)
- **Pool-side worker naming** — worker name is sent to the pool for easy tracking

### 📊 Central Dashboard
- **Live hashrate** — updated every 5 seconds
- **Sparkline graphs** — 1-hour hashrate history per server (120 data points)
- **CPU temperature gauge** — color-coded (green/yellow/red) with smooth animations
- **Earnings estimates** — daily XMR and USD based on network difficulty and live price (refreshed every 10 minutes)
- **Live logs** — view real-time XMRig output for any server (auto-refreshes)
- **Crash-loop alerts** — clear warnings when a server is in crash loop
- **Add/remove servers** — without touching config files (via web UI or `servers.json`)
- **Bulk operations** — start/stop/change mode on all servers or a single server

### 🖥️ Platform Support
- **Linux** — systemd service with automatic restarts
- **Windows** — native Windows Service (no NSSM required)
- **ARM64** — support for Raspberry Pi and cloud ARM instances
- **Cross-compilation** — build once, deploy to any platform

### 🎮 GPU Support
- **Real vendor detection** — uses `nvidia-smi`, `rocm-smi`, and `clinfo` with actual Platform Vendor parsing
- **CUDA** — enabled for NVIDIA GPUs automatically
- **OpenCL** — enabled for AMD/Intel GPUs automatically
- **Manual override** — force enable/disable specific backends (`--gpu cuda|opencl|none`)
- **Note**: RandomX is optimized for CPU; GPU contribution is modest but supported

### 🔐 Security
- **Bearer token authentication** — each agent has a unique token
- **Session-based panel auth** — username/password with SHA256 + salt
- **Constant-time comparison** — prevents timing attacks
- **HttpOnly cookies** — prevents XSS session theft
- **No telemetry** — only connects to GitHub for XMRig updates

---

## 📋 Prerequisites

### For All Servers
| Requirement | Details |
|-------------|---------|
| **Go 1.21+** | Only for building from source. Skip if using pre-built binaries. |
| **Root/Administrator** | Required for service installation and system tuning |
| **Internet access** | For downloading XMRig from GitHub |
| **Network connectivity** | Agents must be reachable from orchestrator on port 9090 |

### For Linux Servers
| Requirement | Details | Installation |
|-------------|---------|--------------|
| **Systemd** | For service management | Built-in on most modern distros |
| **lm-sensors** (optional) | For CPU temperature monitoring | `apt install lm-sensors` / `yum install lm_sensors` |
| **clinfo** (optional) | For OpenCL GPU detection | `apt install clinfo` / `yum install clinfo` |
| **sudo** | For privilege escalation | Built-in on most distros |

### For Windows Servers
| Requirement | Details |
|-------------|---------|
| **PowerShell** | For installation commands |
| **NVIDIA drivers** | Required for CUDA support |
| **Administrator privileges** | For service installation |

### 📱 For Monero Wallets
You'll need a Monero wallet address to receive mined coins. We **strongly recommend [Cake Wallet](https://cakewallet.com/)** for:

- ✅ **Mobile & desktop support** — iOS, Android, macOS, Windows, Linux
- ✅ **Intuitive UI** — easy to use for beginners
- ✅ **Strong privacy** — built-in Tor support
- ✅ **Exchange integration** — buy/sell XMR directly
- ✅ **Open source** — code is public for auditing

**Other excellent options:**
- [Monero GUI Wallet](https://www.getmonero.org/downloads/) — official wallet, full node support
- [Feather Wallet](https://featherwallet.org/) — lightweight desktop wallet with Tor integration
- [Monerujo](https://www.monerujo.io/) — dedicated Android wallet
- [Edge Wallet](https://edge.app/) — multi-currency mobile wallet with XMR support

> ⚠️ **IMPORTANT**: Never share your **private keys** or **seed phrase** with anyone. The pool only needs your **public address** (the 95-character string starting with `4`).

---

## 🚀 Quick Start

### 1. Build the Binaries

Clone the repository and build the agent and orchestrator:

```bash
# Clone the repository
git clone https://github.com/Batman-Priv/MoneroMinerSuite.git
cd MoneroMinerSuite

# Build the agent for all platforms
cd agent
go mod tidy

# Linux (64-bit)
GOOS=linux GOARCH=amd64 go build -o miner-agent .

# Windows (64-bit)
GOOS=windows GOARCH=amd64 go build -o miner-agent.exe .

# ARM64 (Raspberry Pi, AWS Graviton)
GOOS=linux GOARCH=arm64 go build -o miner-agent .

# Build the orchestrator
cd ../orchestrator
go build -o miner-orchestrator .
```

**Optional: Build everything at once**
```bash
# Create a build script
cat > build.sh << 'EOF'
#!/bin/bash
cd agent
GOOS=linux GOARCH=amd64 go build -o miner-agent-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o miner-agent-windows-amd64.exe .
GOOS=linux GOARCH=arm64 go build -o miner-agent-linux-arm64 .
cd ../orchestrator
GOOS=linux GOARCH=amd64 go build -o miner-orchestrator-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o miner-orchestrator-windows-amd64.exe .
echo "Build complete!"
EOF
chmod +x build.sh
./build.sh
```

---

### 2. Install the Agent (On Each Mining Server)

#### 🔹 Linux (as root)

```bash
sudo ./miner-agent install \
  --pool pool.supportxmr.com:443,backup-pool.example.com:443 \
  --wallet <YOUR_MONERO_WALLET_ADDRESS> \
  --worker linux-1 \
  --token $(openssl rand -hex 24)
```

#### 🔹 Windows (PowerShell as Administrator)

```powershell
# Open PowerShell as Administrator
.\miner-agent.exe install `
  --pool pool.supportxmr.com:443 `
  --wallet <YOUR_MONERO_WALLET_ADDRESS> `
  --worker win-1 `
  --token d2582d25864dea7816e84d49d1d3865d4034926ca57870d0
```

**What happens during installation?**

1. ✅ Validates your wallet address (checksum verification)
2. ✅ Copies the agent binary to the install directory (`C:\MinerAgent` or `/opt/miner-agent`)
3. ✅ Downloads the latest XMRig build from GitHub
4. ✅ Creates the agent configuration file
5. ✅ Tunes system settings (huge pages, MSR, etc.)
6. ✅ Registers and starts the service (systemd or Windows Service)
7. ✅ Prints server details for adding to the panel

**Parameter Reference:**

| Flag | Required | Description | Example |
|------|----------|-------------|---------|
| `--pool` | ✅ | Pool address(es), comma-separated for failover | `pool.supportxmr.com:443,backup:443` |
| `--wallet` | ✅ | Your Monero wallet address | `45zCPtw4...` (95 chars) |
| `--token` | ✅ | Random token for API auth (keep secret!) | Use `openssl rand -hex 24` |
| `--worker` | ❌ | Unique worker name (default: hostname) | `srv-nyc-01` |
| `--mode` | ❌ | Mining mode (default: `normal`) | `normal` \| `balanced` \| `max` |
| `--gpu` | ❌ | GPU backend (default: `auto`) | `auto` \| `cuda` \| `opencl` \| `none` |
| `--listen` | ❌ | Agent API address (default: `0.0.0.0:9090`) | `0.0.0.0:9090` |
| `--dir` | ❌ | Install directory (default: OS-appropriate) | `C:\MinerAgent` or `/opt/miner-agent` |
| `--tls` | ❌ | Use TLS for pool connections (default: `true`) | `true` \| `false` |
| `--donate` | ❌ | XMRig developer donation % (default: `1`) | `0` \| `1` \| `5` |
| `--skip-wallet-check` | ❌ | Skip wallet validation (not recommended) | — |

> 💡 **Generate a random token**: Use `openssl rand -hex 24` on Linux, or on Windows:
> ```powershell
> # PowerShell
> -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 48 | % {[char]$_})
> ```

---

### 3. Launch the Orchestrator Panel

On your central management server:

```bash
cd orchestrator

# Initialize authentication (one-time)
./miner-orchestrator --init-panel-auth \
  --panel-user admin \
  --panel-pass "your-strong-password"

# Start the panel
./miner-orchestrator --listen 0.0.0.0:8080
```

**Orchestrator Options:**

| Flag | Description |
|------|-------------|
| `--listen` | Address to bind the web panel (default: `0.0.0.0:8080`) |
| `--servers` | Path to servers.json file (default: `servers.json`) |
| `--panel-auth` | Path to panel-auth.json file (default: `panel-auth.json`) |
| `--panel-user` | Username for quick setup (used with `--panel-pass`) |
| `--panel-pass` | Password for quick setup (used with `--panel-user`) |
| `--init-panel-auth` | Create panel-auth.json and exit |
| `--allow-no-auth` | Run WITHOUT login protection (⚠️ DANGEROUS) |
| `--status` | Print status of all servers and exit |
| `--set-mode` | Change mode of servers (`normal`\|`balanced`\|`max`) |
| `--server` | Target server for `--set-mode` (`*` for all) |

Open `http://<ORCHESTRATOR_IP>:8080` in your browser and log in with the credentials you set.

> 🔒 **Security**: For production, run behind a VPN or use SSH tunneling. The panel uses HTTP by default — add a reverse proxy with HTTPS for external access.

---

### 4. Add Servers to the Panel

#### Option A: Via the Web Interface (Recommended)

In the web dashboard:

1. Click **"Add Server"**
2. Fill in the details from the agent installation output:

   ```
   ┌─────────────────────────────────────────────────┐
   │  name  : linux-1                               │
   │  addr  : 192.168.1.100:9090                    │
   │  token : a7f3e9c2d1b8f4a0e5c7d9b2f1a3e8c4d0f6 │
   │  os    : linux                                 │
   └─────────────────────────────────────────────────┘
   ```

3. Click **Save** — the server appears on the dashboard within seconds!

#### Option B: Via `servers.json` (Manual)

The orchestrator reads server definitions from `servers.json` (default) in its working directory. You can create this file manually:

```bash
cd orchestrator

# Copy the example file
cp servers.example.json servers.json

# Edit with your server details
nano servers.json
```

**`servers.json` structure:**

```json
{
  "servers": [
    {
      "name": "miner-nyc-01",
      "addr": "192.168.1.100:9090",
      "token": "a7f3e9c2d1b8f4a0e5c7d9b2f1a3e8c4d0f6b9a2c5e7d1f3a8b0c4d6e9f2a1b3",
      "os": "linux"
    },
    {
      "name": "miner-nyc-02",
      "addr": "192.168.1.101:9090",
      "token": "b8f4a0e5c7d9b2f1a3e8c4d0f6b9a2c5e7d1f3a8b0c4d6e9f2a1b3d2582d25",
      "os": "windows"
    }
  ]
}
```

**Field Reference:**

| Field | Description | Example |
|-------|-------------|---------|
| `name` | Human-readable server name | `miner-nyc-01` |
| `addr` | Agent IP + port (port 9090) | `192.168.1.100:9090` |
| `token` | Agent's authentication token (from install) | Long hex string |
| `os` | Operating system (`linux` or `windows`) | `linux` |

> 📝 **Note**: The `servers.json` file is auto-created when you add servers via the web interface. If you prefer manual configuration, you can create it from `servers.example.json`.

---

## ⚙️ Configuration

### Working Modes

MoneroMinerSuite supports three performance profiles that balance system resource usage against mining performance:

| Mode | Threads Used | Huge Pages | Priority | Yield | Use Case |
|------|--------------|------------|----------|-------|----------|
| **`normal`** (default) | ~50% cores | Off | Low (1) | True | Daily use with minimal system impact |
| **`balanced`** | ~85% cores | On | Medium (3) | True | Dedicated mining with good performance |
| **`max`** | 100% cores | 1GB pages + MSR | High (5) | False | Dedicated mining servers, maximum hashrate |

**Detailed Breakdown:**

- **Threads**: Percentage of CPU cores used. `normal` uses half, `balanced` uses 85%, `max` uses all.
- **Huge Pages**: Reduces memory translation overhead. Off in `normal`, on in others. `max` also enables 1GB pages.
- **Priority**: OS process priority. Higher priority gets more CPU time.
- **Yield**: Whether the miner yields CPU to other processes. `false` in `max` mode maximizes performance.
- **MSR Mod**: `max` mode applies MSR (Model-Specific Register) mods for better RandomX performance.

**Choose wisely:**
- **`normal`** — ideal for shared servers running other services (web servers, databases, etc.)
- **`balanced`** — good for dedicated mining where you still want some headroom
- **`max`** — only for dedicated mining rigs; can slow down other processes significantly

### Changing Settings After Installation

```bash
# Change mode to balanced
sudo ./miner-agent config --mode balanced --restart

# Change pool(s) — comma-separated for failover
sudo ./miner-agent config --pool "primary:443,backup:443" --restart

# Change wallet address
sudo ./miner-agent config --wallet <NEW_WALLET> --restart

# Change worker name
sudo ./miner-agent config --worker new-name --restart

# Change GPU setting
sudo ./miner-agent config --gpu cuda --restart

# Change API token
sudo ./miner-agent config --token <NEW_TOKEN> --restart

# Change without restarting (changes will apply on next restart)
sudo ./miner-agent config --mode balanced --restart=false
```

### Uninstalling

```bash
# Remove the service only (files remain in /opt/miner-agent or C:\MinerAgent)
sudo ./miner-agent uninstall

# Remove everything (including config, XMRig, logs, and the agent binary)
sudo ./miner-agent uninstall --purge

# Uninstall from a custom directory
sudo ./miner-agent uninstall --dir /custom/path --purge
```

---

## 📊 Dashboard Overview

### 🖥️ Main View

The dashboard is designed for at-a-glance monitoring of your entire fleet:

**Header Section:**
- **Total Hashrate** — aggregate across all online servers
- **Active Servers** — count of online vs. total servers
- **Earnings Estimate** — daily XMR and USD
- **Last Update** — when data was last refreshed

**Server Cards:**
Each server is displayed as a card with:

| Element | Description |
|---------|-------------|
| **Server Name** | As configured in `servers.json` |
| **Status Indicator** | 🟢 Online / 🔴 Offline / 🟡 Error |
| **Hashrate** | Current hashrate in KH/s or MH/s |
| **Temperature** | CPU temperature with color-coded gauge |
| **Mode** | normal / balanced / max |
| **Sparkline** | 1-hour hashrate history |
| **Actions** | Start / Stop / Change Mode buttons |

### 🔍 Detailed Views

**Live Logs:**
- Click the **"View Logs"** button on any server card
- Shows the last 200 lines of XMRig output
- Auto-refreshes every 5 seconds
- Useful for debugging connection issues or configuration errors

**Temperature Gauge:**
- Animated circular gauge
- Color coding:
  - 🟢 Green: < 60°C (safe)
  - 🟡 Yellow: 60-80°C (moderate)
  - 🔴 Red: > 80°C (critical)
- Helps identify cooling issues before hardware damage

**Earnings Breakdown:**
- Click any server to see its contribution to total earnings
- Hashrate distribution chart
- Daily XMR and USD per server

**Crash-loop Alerts:**
- Clear warning banner when a server enters crash loop
- Link to logs for troubleshooting
- Suggests checking configuration

### 🎮 Controls

| Action | Description |
|--------|-------------|
| **Start** | Start mining on the selected server |
| **Stop** | Stop mining gracefully |
| **Change Mode** | Switch between normal/balanced/max instantly |
| **Refresh** | Force a status update |
| **Add Server** | Add new mining server via web form |

---

## 🔐 Security Best Practices

### 1. Network Security

**Firewall Configuration:**

```bash
# Linux: Restrict agent port to orchestrator only
iptables -A INPUT -p tcp --dport 9090 -s <ORCHESTRATOR_IP> -j ACCEPT
iptables -A INPUT -p tcp --dport 9090 -j DROP

# Windows: Use Windows Firewall with Advanced Security
New-NetFirewallRule -DisplayName "MinerAgent" -Direction Inbound -LocalPort 9090 `
  -RemoteAddress <ORCHESTRATOR_IP> -Action Allow
New-NetFirewallRule -DisplayName "MinerAgent-Deny" -Direction Inbound -LocalPort 9090 `
  -Action Block
```

**VPN Recommendation:**
- Use WireGuard or Tailscale to create a private network
- Run orchestrator and agents on private IPs
- Only expose port 8080 if absolutely necessary (with HTTPS)

**HTTPS for Panel:**
```nginx
# Example nginx reverse proxy with Let's Encrypt
server {
    listen 443 ssl http2;
    server_name miner-panel.example.com;

    ssl_certificate /etc/letsencrypt/live/miner-panel.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/miner-panel.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 2. Authentication

**Panel Password Policy:**
- Minimum 14 characters
- Mix of uppercase, lowercase, numbers, and symbols
- Don't reuse passwords from other systems
- Change periodically

**Agent Tokens:**
- Use `openssl rand -hex 24` for each agent
- Store tokens securely (password manager)
- Never commit tokens to version control
- Rotate tokens if compromised (reinstall or use `config` command)

**Session Security:**
- Cookies are HttpOnly (not accessible via JavaScript)
- 7-day expiration
- Invalidated on logout
- Stored with SHA256 + salt

### 3. Operational Security

**Principle of Least Privilege:**
- Agents run as root/system only because they need to tune system settings
- For sensitive environments, consider running with `CAP_SYS_NICE` and `CAP_SYS_RAWIO` capabilities on Linux
- On Windows, the service runs as LocalSystem by default

**Audit Logs:**
- Agent logs (`agent.log`) record all actions
- XMRig logs are in memory (last 500 lines)
- Check logs regularly for suspicious activity

**No Telemetry:**
- MoneroMinerSuite phones home to GitHub ONLY for XMRig updates
- No user data is collected
- No cloud dependencies

### 4. Insecure Configurations (NEVER DO)

- ❌ `--allow-no-auth` on exposed networks
- ❌ `--skip-wallet-check` (unless on non-mainnet)
- ❌ Exposing port 9090 to the public internet
- ❌ Using weak panel passwords
- ❌ Running without firewall rules
- ❌ Sharing agent tokens

---

## 📁 File Structure Reference

### Agent Directory Structure (`/opt/miner-agent` or `C:\MinerAgent`)

```
/opt/miner-agent/
├── miner-agent              # Agent binary (or miner-agent.exe on Windows)
├── xmrig                    # XMRig binary (or xmrig.exe on Windows)
├── agent-config.json        # Agent configuration (auto-generated)
├── miner-run/
│   ├── agent.log            # Agent logs (auto-rotated at 20MB)
│   ├── agent.log.old        # Rotated log backup
│   └── xmrig-config.json    # XMRig configuration (auto-generated)
└── (service files on Linux) # /etc/systemd/system/miner-agent.service
```

### Orchestrator Directory Structure

```
orchestrator/
├── miner-orchestrator          # The binary
├── servers.json                # Server list (auto-generated via UI or manual)
├── servers.example.json        # Template file (provided in the repo)
├── panel-auth.json             # Hashed credentials (auto-generated)
└── web/
    └── index.html              # Web dashboard (embedded in binary)
```

### Key File Descriptions

| File | Purpose | Auto-generated? |
|------|---------|-----------------|
| `agent-config.json` | Agent settings (pools, wallet, mode, etc.) | ✅ Yes (on install) |
| `xmrig-config.json` | XMRig-specific configuration | ✅ Yes (on first run) |
| `agent.log` | Agent log output | ✅ Yes |
| `servers.json` | List of servers for orchestrator | ✅ Yes (via UI) |
| `servers.example.json` | Template for manual configuration | ❌ No (provided in repo) |
| `panel-auth.json` | Hashed admin credentials | ✅ Yes (via `--init-panel-auth`) |

### servers.example.json Template

```json
{
  "servers": [
    {
      "name": "example-server-1",
      "addr": "192.168.1.100:9090",
      "token": "your-agent-token-here",
      "os": "linux"
    },
    {
      "name": "example-server-2",
      "addr": "192.168.1.101:9090",
      "token": "another-agent-token",
      "os": "windows"
    }
  ]
}
```

---

## 🛠️ Troubleshooting

### Common Issues and Solutions

#### ❌ Agent fails to download XMRig

**Symptoms:**
```
2026/08/22 19:29:54 XMRig download failed: no matching asset found for windows/amd64 in release v6.26.0
```

**Solutions:**
1. Check internet connectivity:
   ```bash
   curl -I https://api.github.com/repos/xmrig/xmrig/releases/latest
   ```

2. If rate-limited, wait or use a GitHub token:
   ```bash
   export GITHUB_TOKEN=your_token
   ```

3. Update `download.go` to use `strings.Contains` instead of `strings.HasSuffix` for asset matching

4. Download XMRig manually and place it in the install directory, then run with `--xmrig` flag

---

#### ❌ Agent API is unreachable

**Symptoms:**
- Dashboard shows server as "Offline"
- `curl http://<AGENT_IP>:9090/healthz` fails

**Solutions:**
1. Check if the agent service is running:
   ```bash
   systemctl status miner-agent   # Linux
   sc query MinerAgent            # Windows
   ```

2. Check firewall:
   ```bash
   # Linux
   netstat -tlnp | grep 9090
   iptables -L -n | grep 9090
   
   # Windows
   netstat -ano | findstr 9090
   ```

3. Check agent logs:
   ```bash
   sudo cat /opt/miner-agent/miner-run/agent.log
   ```

4. Verify agent is listening on the correct address:
   ```bash
   # In agent-config.json
   cat /opt/miner-agent/agent-config.json | grep listen_addr
   ```

---

#### ❌ Crash-loop detected

**Symptoms:**
- Dashboard shows "CRASH LOOP" warning
- Agent logs show repeated "xmrig exited" messages

**Common causes:**
1. **Invalid wallet address** — recheck with `./miner-agent config --wallet <ADDRESS>`
2. **Invalid pool URL** — verify the pool is reachable and the URL is correct
3. **XMRig binary corrupted** — reinstall with `./miner-agent install --purge`
4. **Permission issues** — check that XMRigPath is executable
5. **Incompatible CPU** — RandomX requires AES-NI support (most modern CPUs have it)

**Solutions:**
1. Check the logs for specific errors:
   ```bash
   sudo cat /opt/miner-agent/miner-run/agent.log | grep -i error
   ```

2. Run the agent manually to see the actual error:
   ```bash
   sudo ./miner-agent run --config /opt/miner-agent/agent-config.json
   ```

3. Fix the configuration and restart:
   ```bash
   sudo ./miner-agent config --wallet <CORRECT_ADDRESS> --restart
   ```

4. If all else fails, uninstall and reinstall:
   ```bash
   sudo ./miner-agent uninstall --purge
   sudo ./miner-agent install --pool <POOL> --wallet <WALLET> --token <TOKEN>
   ```

---

#### ❌ Low hashrate

**Symptoms:**
- Hashrate is significantly lower than expected for your CPU

**Solutions:**
1. **Switch to max mode**:
   ```bash
   sudo ./miner-agent config --mode max --restart
   ```

2. **Check huge pages** (Linux):
   ```bash
   cat /proc/sys/vm/nr_hugepages
   # Should show 1280 or higher
   ```

3. **Check huge pages** (Windows):
   ```powershell
   # Check if Lock Pages in Memory is enabled
   whoami /priv | findstr "Lock"
   ```

4. **Verify no other processes are using CPU**:
   ```bash
   top
   # Look for other CPU-intensive processes
   ```

5. **Check CPU frequency scaling** (Linux):
   ```bash
   # Set to performance mode
   sudo cpupower frequency-set -g performance
   ```

---

#### ❌ Temperature not showing on Windows

**Symptoms:**
- Dashboard shows "N/A" or no temperature reading
- CPU temperature gauge is empty

**Explanation:**
Windows doesn't expose CPU temperature via WMI on most servers (especially VMs and cloud instances). This is expected behavior.

**Solutions:**
1. Use vendor-specific tools (iDRAC for Dell, iLO for HP, etc.)
2. Use third-party temperature monitoring software (Core Temp, HWMonitor)
3. On physical servers with proper ACPI support, it should work
4. Not critical for mining — temperature monitoring is informative, not required

---

#### ❌ GPU not being detected

**Symptoms:**
- Dashboard doesn't show GPU usage
- XMRig logs show no GPU devices

**Solutions:**
1. Check if drivers are installed:
   ```bash
   # NVIDIA
   nvidia-smi
   
   # AMD
   rocm-smi --showid
   
   # OpenCL
   clinfo | grep "Platform Vendor"
   ```

2. Force enable GPU:
   ```bash
   sudo ./miner-agent config --gpu cuda --restart   # For NVIDIA
   sudo ./miner-agent config --gpu opencl --restart # For AMD/Intel
   ```

3. On Windows, ensure Visual C++ Redistributable is installed

4. Note: RandomX is CPU-optimized; GPU contribution is modest (~5-10% of hashrate)

---

#### ❌ Pool connection issues

**Symptoms:**
- "Connection refused" or "Timeout" errors
- Hashrate drops to zero intermittently

**Solutions:**
1. Test connectivity to the pool:
   ```bash
   # Linux
   nc -zv pool.supportxmr.com 443
   
   # Windows
   Test-NetConnection pool.supportxmr.com -Port 443
   ```

2. Try a different pool:
   ```bash
   sudo ./miner-agent config --pool "backup-pool.com:443" --restart
   ```

3. Check if your ISP is blocking mining ports:
   - Try port 3333 (non-TLS) if 443 is blocked
   - Use a VPN if necessary

4. Configure failover pools:
   ```bash
   sudo ./miner-agent config --pool "pool1:443,pool2:443,pool3:443" --restart
   ```

---

#### ❌ Service won't start (Windows)

**Symptoms:**
- `sc query MinerAgent` shows STOPPED
- Error in Event Viewer

**Solutions:**
1. Check Event Viewer:
   ```powershell
   Get-WinEvent -LogName System | Where-Object { $_.ProviderName -eq "Service Control Manager" } | Select-Object -First 10
   ```

2. Ensure you're running as Administrator

3. Check the service path:
   ```powershell
   sc qc MinerAgent
   ```

4. Repair the service:
   ```powershell
   sc delete MinerAgent
   .\miner-agent.exe install --pool <POOL> --wallet <WALLET> --token <TOKEN>
   ```

---

#### ❌ Service won't start (Linux)

**Symptoms:**
- `systemctl status miner-agent` shows FAILED
- Error in journal logs

**Solutions:**
1. Check the unit file:
   ```bash
   cat /etc/systemd/system/miner-agent.service
   ```

2. Check logs:
   ```bash
   journalctl -u miner-agent -f
   ```

3. Ensure the binary is executable:
   ```bash
   sudo chmod +x /opt/miner-agent/miner-agent
   sudo chmod +x /opt/miner-agent/xmrig
   ```

4. Repair the service:
   ```bash
   systemctl stop miner-agent
   systemctl disable miner-agent
   rm /etc/systemd/system/miner-agent.service
   systemctl daemon-reload
   ./miner-agent install --pool <POOL> --wallet <WALLET> --token <TOKEN>
   ```

---

#### ❌ High CPU usage (non-mining)

**Symptoms:**
- System is slow even when mining is not running
- CPU usage is high in `top` or Task Manager

**Solutions:**
1. Switch to `normal` mode:
   ```bash
   sudo ./miner-agent config --mode normal --restart
   ```

2. Stop the agent temporarily:
   ```bash
   sudo ./miner-agent config --mode max --restart=false
   ```

3. Pause mining:
   - Use the dashboard's Stop button
   - Or run: `sudo systemctl stop miner-agent`

4. Consider reducing threads manually:
   - Edit `xmrig-config.json` and set `"max-threads-hint": 30`

---

## ❓ Frequently Asked Questions (FAQ)

### General Questions

#### Q1: What is MoneroMinerSuite?
**A:** MoneroMinerSuite is a complete, self-hosted platform for managing Monero (XMR) mining operations across multiple Linux and Windows servers. It includes auto-healing agents, a central web dashboard, automatic XMRig updates, crash-loop protection, and real-time monitoring.

---

#### Q2: Is MoneroMinerSuite free?
**A:** Yes! MoneroMinerSuite is open-source software released under the MIT License. You can use it for free, modify it, and deploy it on as many servers as you want.

---

#### Q3: What is a Monero wallet and why do I need one?
**A:** A Monero wallet is where your mined XMR coins are sent. You need a wallet address (a 95-character string starting with `4`) to receive mining rewards. We recommend [Cake Wallet](https://cakewallet.com/) for its excellent mobile and desktop support.

---

#### Q4: Do I need to register on the pool?
**A:** No! Pools like supportXMR.com don't require registration. Your wallet address serves as your identity. When you mine, the pool credits your wallet address, and when a block is found, the reward is sent directly to your wallet.

---

#### Q5: Can I mine Monero on a laptop or desktop?
**A:** Yes, but with caution. Mining generates heat and uses CPU resources. Use `normal` mode to limit impact, and monitor temperatures. Avoid mining on laptops without proper cooling.

---

#### Q6: Will mining damage my hardware?
**A:** Mining alone won't damage hardware if it's properly cooled and not overvolted. However, prolonged high-temperature operation can reduce component lifespan. Keep temperatures below 80°C for CPU and ensure adequate cooling.

---

### Technical Questions

#### Q7: What ports need to be open?
**A:** 
- Agent ports (9090): Only need to be accessible from the orchestrator. Use firewall rules to restrict.
- Orchestrator panel (8080): Accessible to your browser (can be protected with VPN/HTTPS).
- Outbound: Agents need internet access to download XMRig and connect to pools (usually port 443).

---

#### Q8: What is the "token" used for?
**A:** The token is a secret key that authenticates communication between the agent and orchestrator. Each agent has a unique token. It's like a password — keep it secret!

---

#### Q9: Can I run multiple agents on the same server?
**A:** Not recommended. Each agent manages one instance of XMRig. Running multiple agents on the same server would cause resource contention. Instead, use one agent and XMRig's built-in multi-threading.

---

#### Q10: Can I change the pool after installation?
**A:** Yes! Use the `config` command:
```bash
sudo ./miner-agent config --pool "newpool.com:443" --restart
```
You can specify multiple pools with commas for failover.

---

#### Q11: Can I change my wallet address after installation?
**A:** Yes! Use the `config` command:
```bash
sudo ./miner-agent config --wallet <NEW_ADDRESS> --restart
```
⚠️ **Warning**: If you change the wallet address, all future mining rewards will go to the new address. Previous earnings stay in the old wallet.

---

#### Q12: How do I stop mining temporarily?
**A:** Use the dashboard's Stop button for that server, or:
```bash
sudo systemctl stop miner-agent        # Linux
sc stop MinerAgent                     # Windows
```

---

#### Q13: How do I completely remove MoneroMinerSuite?
**A:** 
```bash
sudo ./miner-agent uninstall --purge   # Removes everything
```

---

#### Q14: What happens if a server goes offline?
**A:** The orchestrator will mark it as "Offline" in the dashboard. The agent will automatically try to reconnect when the server comes back up. No manual intervention is needed.

---

#### Q15: Can I mine with multiple pools for redundancy?
**A:** Yes! XMRig supports failover. Specify multiple pools with commas:
```bash
--pool "pool1.com:443,pool2.com:443,pool3.com:443"
```
If pool1 fails, XMRig switches to pool2, etc.

---

#### Q16: What is "crash-loop detection"?
**A:** If XMRig crashes 6 times within 3 minutes (each crash lasting less than 10 seconds), the agent stops trying to restart it. This prevents a configuration error from causing infinite restart loops. The dashboard shows a clear alert when this happens.

---

#### Q17: How does automatic XMRig updating work?
**A:** Every 24 hours, the agent checks GitHub for a new XMRig release. If a newer version exists, it downloads it, replaces the binary, and restarts XMRig gracefully. You can disable this with `--auto-update-xmrig=false` during installation.

---

#### Q18: Why is my hashrate lower than expected?
**A:** Several factors affect hashrate:
1. Mode: `normal` uses 50% cores, `max` uses 100%
2. Background processes: Other programs using CPU
3. Cooling: Thermal throttling reduces performance
4. Memory speed: RandomX is memory-intensive
5. Pool: Some pools show different hashrate calculations
6. CPU generation: Newer CPUs (Zen 3/4, Intel 12th+) are faster

---

#### Q19: Why is GPU mining not giving much hashrate?
**A:** RandomX is specifically designed to be CPU-friendly and ASIC/GPU-resistant. It uses memory-hard algorithms that favor CPU cache and memory latency. GPUs have higher latency memory, so they're less efficient. CPU always gives the best hashrate for Monero.

---

#### Q20: Can I use a cloud server (AWS/Google/Azure) for mining?
**A:** Technically yes, but it's usually **not cost-effective** due to cloud provider pricing. You'll likely mine less XMR than the cost of the instance. Cloud providers also prohibit or restrict cryptocurrency mining in their ToS.

---

#### Q21: Is mining Monero legal?
**A:** Mining is legal in most countries. However:
- Check your local regulations
- Ensure you have permission to use the servers
- Some jurisdictions may classify mining as a business activity requiring registration
- Cloud providers may prohibit mining in their terms of service

---

### Security Questions

#### Q22: Is MoneroMinerSuite secure?
**A:** Yes, when properly configured. Security features include:
- Token-based authentication between agent and orchestrator
- Session-based login for the web panel
- SHA256 + salt password hashing
- HttpOnly cookies
- No telemetry or cloud dependencies

---

#### Q23: What if someone finds my token?
**A:** If a token is compromised, anyone with it could control that agent. Immediately regenerate the token:
```bash
sudo ./miner-agent config --token <NEW_TOKEN> --restart
```
Then update the token in the orchestrator's `servers.json`.

---

#### Q24: Should I expose the dashboard to the internet?
**A:** **No!** The dashboard uses HTTP (not HTTPS) by default. If you need remote access:
1. Use a VPN (WireGuard, Tailscale)
2. Use SSH tunneling
3. Use a reverse proxy with HTTPS (nginx with Let's Encrypt)

---

#### Q25: Can I use the agent without the orchestrator?
**A:** Yes! The agent runs independently as a service. It will mine continuously without the orchestrator. The orchestrator is only for monitoring and control. The agent continues mining even if the orchestrator is offline.

---

#### Q26: How do I reset the panel password?
**A:** Delete `panel-auth.json` and reinitialize:
```bash
rm panel-auth.json
./miner-orchestrator --init-panel-auth --panel-user admin --panel-pass <NEW_PASSWORD>
```

---

#### Q27: Can I have multiple users for the panel?
**A:** Not currently. The panel supports a single admin user with a username and password. For multi-user setups, consider using a reverse proxy with basic auth.

---

#### Q28: Does the software collect any data?
**A:** No. MoneroMinerSuite does not collect any telemetry, analytics, or usage data. The only external connection is to GitHub for XMRig updates.

---

### Advanced Questions

#### Q29: Can I contribute to the project?
**A:** Absolutely! Contributions are welcome:
1. Fork the repository
2. Create a feature branch
3. Submit a Pull Request
4. Follow Go conventions and add tests when applicable

---

#### Q30: Can I run the orchestrator and agent on the same server?
**A:** Yes! You can run both on the same server. Just install the agent first, then run the orchestrator. Add the server to `servers.json` with address `127.0.0.1:9090` or the server's actual IP.

---

#### Q31: What about ARM support?
**A:** MoneroMinerSuite supports ARM64 (like Raspberry Pi 4/5, AWS Graviton). Build with:
```bash
GOOS=linux GOARCH=arm64 go build -o miner-agent .
```

---

#### Q32: Can I use this with a mining pool that requires registration?
**A:** Most Monero pools (supportXMR, moneroocean, etc.) don't require registration. If your pool requires registration, check their documentation. Usually, you can still use the pool URL, but you may need to set the username differently.

---

#### Q33: What's the difference between `normal`, `balanced`, and `max`?
**A:** See the [Working Modes](#working-modes) section. In short:
- `normal`: 50% CPU, minimal system impact
- `balanced`: 85% CPU, good performance with some headroom
- `max`: 100% CPU, maximum hashrate, dedicates the system to mining

---

#### Q34: How do I verify my wallet address is correct?
**A:** The agent validates wallet addresses during installation using the same CryptoNote base58 checksum as Monero wallets. If you get a checksum error, recheck your address. If you're sure it's correct (e.g., using a non-mainnet address), use `--skip-wallet-check`.

---

#### Q35: What is the "donate-level"?
**A:** XMRig has a developer donation feature. By default, it's set to 1% (meaning 1% of your hashrate is donated to the XMRig developers). You can change this with `--donate 0` during installation.

---

#### Q36: Why is the agent using so much memory?
**A:** RandomX requires about 2GB of memory for the dataset (scratchpad). This is normal and expected. The dataset is loaded once and shared across threads. Memory usage is around 2-2.5GB total.

---

#### Q37: Can I limit CPU cores used by XMRig?
**A:** Yes! Through the mode setting:
- `normal`: 50% of cores
- `balanced`: 85% of cores
- `max`: 100% of cores

For manual control, edit `xmrig-config.json` and adjust `"max-threads-hint"`.

---

#### Q38: What happens if the orchestrator goes down?
**A:** The agents continue mining independently. They don't depend on the orchestrator for mining — they only need it for monitoring and control. When the orchestrator comes back online, it will reconnect to the agents.

---

#### Q39: Can I mine other cryptocurrencies with this?
**A:** This suite is specifically designed for Monero (XMR) and RandomX. To mine other cryptocurrencies, you'd need to modify the agent to download different miners (like lolMiner for ETH).

---

#### Q40: What are the system requirements for the orchestrator?
**A:** Very minimal! Any Linux/Windows server with Go (for building) or just the binary. The orchestrator uses little CPU/memory — it's just a web server and API aggregator.

---

#### Q41: How do I check if my server is mining?
**A:** 
1. Check the dashboard — it shows hashrate
2. Check the pool's website — search for your wallet address
3. Check the agent logs:
   ```bash
   sudo cat /opt/miner-agent/miner-run/agent.log | grep "accepted"
   ```

---

#### Q42: Why does the agent need root/Administrator?
**A:** The agent needs privileges to:
1. Tune system settings (huge pages, MSR mod)
2. Install as a service (systemd/Windows Service)
3. Write to `/opt` or `C:\` directories

Without these, performance would be significantly lower and service management would be limited.

---

#### Q43: Can I use a custom XMRig build?
**A:** Yes! Place your custom `xmrig` binary in the install directory and run:
```bash
sudo ./miner-agent run --xmrig /custom/path/xmrig
```

---

#### Q44: How often does the dashboard update?
**A:**
- Hashrate: Every 5 seconds
- Temperature: Every 5 seconds
- History (sparkline): Every 30 seconds
- Earnings estimate: Every 10 minutes
- Logs: On-demand (click to view)

---

#### Q45: Can I add servers without restarting the orchestrator?
**A:** Yes! The orchestrator dynamically reloads `servers.json` on each API call. Add a server via the web UI or edit `servers.json` — no restart needed.

---

#### Q46: What's the difference between "worker" and "server name"?
**A:** 
- **Worker**: Name sent to the pool. Appears in pool stats.
- **Server name**: Display name in the dashboard. Can be different from worker.

Example: Worker `nyc-miner-01` in the pool, display name `Miner NYC #1` in the dashboard.

---

#### Q47: Can I mine with both CPU and GPU?
**A:** Yes! XMRig supports CPU and GPU mining simultaneously. The agent automatically enables GPU detection based on your system. You can force specific backends with `--gpu`.

---

#### Q48: What's the minimum CPU requirement?
**A:** RandomX requires AES-NI (Advanced Encryption Standard New Instructions) for optimal performance. Most CPUs from 2012 onward support it. On older CPUs without AES-NI, mining will be extremely slow.

Check for AES-NI:
```bash
# Linux
grep aes /proc/cpuinfo

# Windows
(Get-CimInstance -ClassName Win32_Processor).Caption
```

---

#### Q49: How do I backup my configuration?
**A:** Backup these files:
```bash
# Agent config
/opt/miner-agent/agent-config.json

# Orchestrator servers
servers.json

# Orchestrator auth
panel-auth.json

# XMRig config (auto-generated, but useful to keep)
/opt/miner-agent/miner-run/xmrig-config.json
```

---

#### Q50: What's coming in future releases?
**A:** Potential future features:
- WebSocket for real-time updates (instead of polling)
- Prometheus metrics export
- Email/Telegram alerts for downtime
- Mobile app for monitoring
- Support for more mining algorithms
- Integration with mining pool APIs
- Advanced scheduling (mine only at night, etc.)

---

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. **Fork** the repository
2. **Create a feature branch** (`git checkout -b feature/amazing-feature`)
3. **Commit your changes** (`git commit -m 'Add some amazing feature'`)
4. **Push** (`git push origin feature/amazing-feature`)
5. **Open a Pull Request**

### Development Guidelines

- **Go code**: Follow standard Go conventions (`gofmt`, `go vet`)
- **Documentation**: Update README and inline comments for new features
- **Testing**: Add tests for critical components (when applicable)
- **Security**: Report security issues privately (don't open public issues)
- **Style**: Keep code consistent with existing patterns

### Development Environment

```bash
# Install dependencies
cd agent
go mod download

# Run tests
go test ./...

# Build for your platform
go build -o miner-agent .

# Run with debug logging
./miner-agent run --config agent-config.json --silent=false
```

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

**MIT License Summary:**
- ✅ Commercial use
- ✅ Modification
- ✅ Distribution
- ✅ Private use
- ❌ Liability
- ❌ Warranty

---

## ⚠️ Disclaimer

**Use at your own risk.** Mining cryptocurrency may not be legal in your jurisdiction. This software is for educational and informational purposes only.

- 🚫 **Not financial advice**: Earnings estimates are for reference only; actual earnings may vary.
- 💡 **Electricity costs**: Consider your power costs before mining. Mining can significantly increase electricity bills.
- 🔐 **Wallet security**: Protect your private keys. The pool only needs your public address.
- 🌐 **Network compliance**: Ensure you have permission to use the servers you're mining on.
- 🖥️ **Hardware wear**: Mining generates heat and may reduce component lifespan. Ensure adequate cooling.
- 📝 **Tax implications**: Mining rewards may be taxable in your jurisdiction. Consult a tax professional.

---

## 🙏 Acknowledgements

- [XMRig](https://github.com/xmrig/xmrig) — the incredible mining software that makes this all possible
- [Monero](https://getmonero.org/) — the private, untraceable cryptocurrency
- [Cake Wallet](https://cakewallet.com/) — recommended mobile wallet
- All contributors and the open-source community

---

**Built with ❤️ for the Monero community**

---

## 📞 Support & Community

- **GitHub Issues**: [https://github.com/Batman-Priv/MoneroMinerSuite/issues](https://github.com/Batman-Priv/MoneroMinerSuite/issues)
- **Monero Community**: [https://monero.com/community](https://monero.com/community)
- **Cake Wallet Support**: [https://cakewallet.com/support](https://cakewallet.com/support)
- **Monero Reddit**: [r/Monero](https://www.reddit.com/r/Monero/)

---

### Quick Reference Card

```bash
# ────────────────────── INSTALL ──────────────────────
# Linux (as root)
sudo ./miner-agent install --pool pool.supportxmr.com:443 --wallet <WALLET> --worker srv1 --token $(openssl rand -hex 24)

# Windows (PowerShell Admin)
.\miner-agent.exe install --pool pool.supportxmr.com:443 --wallet <WALLET> --worker win1 --token <TOKEN>

# ────────────────────── ORCHESTRATOR ──────────────────────
# Initialize auth (one-time)
./miner-orchestrator --init-panel-auth --panel-user admin --panel-pass "strongpassword"

# Start panel
./miner-orchestrator --listen 0.0.0.0:8080

# ────────────────────── CONFIGURATION ──────────────────────
# Change mode
./miner-agent config --mode max --restart

# Change pool
./miner-agent config --pool "pool1.com:443,pool2.com:443" --restart

# Change wallet
./miner-agent config --wallet <NEW_WALLET> --restart

# ────────────────────── UNINSTALL ──────────────────────
# Remove service only
./miner-agent uninstall

# Remove everything
./miner-agent uninstall --purge

# ────────────────────── CLI COMMANDS ──────────────────────
# Check status of all servers
./miner-orchestrator --status

# Change mode from CLI
./miner-orchestrator --set-mode max --server "*"
```

**Happy mining! ⛏️**
