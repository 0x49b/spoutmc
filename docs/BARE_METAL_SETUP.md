# Bare Metal Setup Guide

This guide walks through setting up SpoutMC on a bare metal server from scratch — from OS prerequisites to a production-ready, auto-starting service.

## Quick Install

One-line installers are available for Linux and Windows. They automate all steps in this guide (Docker, binary download, service registration, firewall).

### Linux (Ubuntu / Debian / RHEL / Fedora)

```bash
curl -fsSL https://raw.githubusercontent.com/0x49b/spoutmc/master/scripts/install.sh | sudo bash
```

Or download and review the script first:

```bash
curl -fsSL https://raw.githubusercontent.com/0x49b/spoutmc/master/scripts/install.sh -o install.sh
# review install.sh ...
sudo bash install.sh
```

**Options:**

| Flag | Description | Default |
|------|-------------|---------|
| `--install-path PATH` | Installation directory | `/opt/spoutmc` |
| `--data-path PATH` | Server data directory | `/opt/spoutmc/data` |
| `--port PORT` | SpoutMC web UI port | `3000` |
| `--mc-port PORT` | Minecraft server port | `25565` |
| `--memory MEM` | Default Minecraft memory | `2G` |
| `--domain DOMAIN` | Domain for nginx reverse proxy | — |
| `--no-nginx` | Skip nginx installation | — |
| `--no-interactive` | Use all defaults silently | — |

Example with nginx:

```bash
sudo bash install.sh --domain spoutmc.example.com --no-interactive
```

### Windows (PowerShell — run as Administrator)

```powershell
irm https://raw.githubusercontent.com/0x49b/spoutmc/master/scripts/install.ps1 | iex
```

Or download and review first:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/0x49b/spoutmc/master/scripts/install.ps1 -OutFile install.ps1
# review install.ps1 ...
.\install.ps1
```

**Parameters:**

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-InstallPath` | Installation directory | `C:\spoutmc` |
| `-DataPath` | Server data directory | `C:\spoutmc\data` |
| `-Port` | SpoutMC web UI port | `3000` |
| `-MinecraftPort` | Minecraft server port | `25565` |
| `-Memory` | Default Minecraft memory | `2G` |
| `-NonInteractive` | Use all defaults silently | — |

> **Note:** Docker Desktop requires WSL2. The script enables WSL2 automatically, but a **reboot may be required** on first run. Re-run the installer after rebooting.

---

The rest of this guide documents each step manually for those who prefer full control.

---

## System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| OS | Ubuntu 22.04 / Debian 12 | Ubuntu 24.04 LTS |
| Architecture | x86_64 (amd64) | x86_64 or ARM64 |
| RAM | 4 GB | 8 GB+ (each Minecraft server needs 1–4 GB) |
| Disk | 20 GB | 50 GB+ SSD |
| Network | Any | Static IP or DNS name |

**Notes for RHEL/Fedora/AlmaLinux:** Steps are the same; use `dnf` instead of `apt` where noted.

---

## Step 1: Install Docker Engine

SpoutMC manages Minecraft servers as Docker containers, so Docker Engine must be installed on the host.

### Ubuntu / Debian

```bash
# Remove old Docker installations (if any)
sudo apt-get remove docker docker-engine docker.io containerd runc

# Install dependencies
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg

# Add Docker's official GPG key
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Add the repository
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker Engine
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin
```

### RHEL / Fedora / AlmaLinux

```bash
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
sudo dnf install -y docker-ce docker-ce-cli containerd.io
```

### Start and Enable Docker

```bash
sudo systemctl enable --now docker
```

### Verify Docker Works

```bash
sudo docker run hello-world
```

You should see `Hello from Docker!` in the output.

---

## Step 2: Download SpoutMC

### Option A: Download a Pre-Built Release (Recommended)

Download the latest binary from the [GitHub Releases page](https://github.com/0x49b/spoutmc/releases).

```bash
# Create installation directory
sudo mkdir -p /opt/spoutmc/config /opt/spoutmc/data

# Download the binary (replace <version> with the latest release)
sudo curl -L -o /opt/spoutmc/spoutmc \
  https://github.com/0x49b/spoutmc/releases/latest/download/spoutmc-linux-amd64

# For ARM64 servers, use:
# sudo curl -L -o /opt/spoutmc/spoutmc \
#   https://github.com/0x49b/spoutmc/releases/latest/download/spoutmc-linux-arm64

sudo chmod +x /opt/spoutmc/spoutmc
```

### Option B: Build from Source

If you prefer to build from source, see [`BUILD.md`](BUILD.md) for full instructions. The resulting binary is at `build/spoutmc-linux-amd64` — copy it to `/opt/spoutmc/spoutmc`.

---

## Step 3: Create a Dedicated System User

Running SpoutMC as root is not recommended. Create a dedicated user instead.

```bash
# Create system user (no home dir, no login shell)
sudo useradd -r -s /bin/false -d /opt/spoutmc spoutmc

# Add to the docker group so it can manage containers
sudo usermod -aG docker spoutmc

# Set ownership of the installation directory
sudo chown -R spoutmc:spoutmc /opt/spoutmc
```

---

## Step 4: Create the Configuration File

SpoutMC requires a configuration file at `config/spoutmc.yaml` relative to its working directory.

```bash
sudo nano /opt/spoutmc/config/spoutmc.yaml
```

Paste and adjust the following example:

```yaml
# EULA - you must accept the Minecraft EULA to run any servers
eula:
  accepted: true
  accepted_on: "2025-01-01T00:00:00Z"

# Storage - where server data (worlds, plugins, etc.) is persisted on the host
storage:
  data_path: "/opt/spoutmc/data"

# Optional: files to exclude from GitOps synchronisation
files:
  exclude_patterns:
    - "*.jar"
    - "world*"
    - ".DS_Store"
    - "*.env"

# Optional: GitOps - sync server configs from a git repository
# git:
#   enabled: false
#   repository: "https://github.com/youruser/your-servers-repo"
#   branch: "main"
#   poll_interval: 5m
#   webhook_secret: "your-secret"
#   local_path: "/opt/spoutmc/git"

# Servers - define the Minecraft servers SpoutMC should manage
servers:
  - name: lobby
    image: itzg/minecraft-server:latest
    lobby: true    # marks this server as the network entry point
    proxy: false
    ports:
      - hostPort: "25565"
        containerPort: "25565"
    env:
      TYPE: PAPER
      VERSION: "1.21.4"
      EULA: "TRUE"
      MEMORY: "2G"
    volumes:
      - containerpath: "/data"
```

Set correct ownership:

```bash
sudo chown spoutmc:spoutmc /opt/spoutmc/config/spoutmc.yaml
sudo chmod 640 /opt/spoutmc/config/spoutmc.yaml
```

---

## Step 5: Test the First Launch

Before setting up the service, verify SpoutMC starts correctly by running it manually:

```bash
sudo -u spoutmc /opt/spoutmc/spoutmc
```

Expected output:

```
Serving embedded frontend from binary
webserver started on http://localhost:3000
```

Open a browser and navigate to `http://<your-server-ip>:3000/`. You should see the SpoutMC setup wizard — follow it to create the first admin user.

Once confirmed, stop the process with `Ctrl+C`.

---

## Step 6: Create a Systemd Service

Create the service unit file so SpoutMC starts automatically on boot and restarts on failure.

```bash
sudo nano /etc/systemd/system/spoutmc.service
```

```ini
[Unit]
Description=SpoutMC Minecraft Server Manager
Documentation=https://github.com/0x49b/spoutmc
After=network-online.target docker.service
Wants=network-online.target
Requires=docker.service

[Service]
Type=simple
User=spoutmc
Group=spoutmc
WorkingDirectory=/opt/spoutmc
ExecStart=/opt/spoutmc/spoutmc
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=spoutmc

# Prevent the process from gaining new privileges
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable spoutmc
sudo systemctl start spoutmc
```

Check the status:

```bash
sudo systemctl status spoutmc
```

View live logs:

```bash
sudo journalctl -u spoutmc -f
```

---

## Step 7: Configure the Firewall

Open the required ports on the host firewall.

### UFW (Ubuntu / Debian)

```bash
# SpoutMC web UI
sudo ufw allow 3000/tcp comment "SpoutMC Web UI"

# Minecraft server (adjust if using a different port)
sudo ufw allow 25565/tcp comment "Minecraft"
sudo ufw allow 25565/udp comment "Minecraft UDP"

sudo ufw reload
sudo ufw status
```

### firewalld (RHEL / Fedora)

```bash
sudo firewall-cmd --permanent --add-port=3000/tcp
sudo firewall-cmd --permanent --add-port=25565/tcp
sudo firewall-cmd --permanent --add-port=25565/udp
sudo firewall-cmd --reload
```

**Behind NAT:** If your server is behind a router, also configure port forwarding for ports 3000 and 25565 to the server's local IP in your router's settings.

---

## Step 8: Optional — Reverse Proxy with HTTPS

To serve the SpoutMC UI over HTTPS (recommended for production), set up nginx as a reverse proxy.

### Install nginx

```bash
sudo apt-get install -y nginx certbot python3-certbot-nginx
```

### Configure nginx

Create `/etc/nginx/sites-available/spoutmc`:

```nginx
server {
    listen 80;
    server_name spoutmc.example.com;  # replace with your domain

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 86400;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/spoutmc /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Add SSL with Let's Encrypt

```bash
sudo certbot --nginx -d spoutmc.example.com
```

Certbot automatically modifies the nginx config and renews certificates.

---

## Verify the Full Installation

```bash
# 1. Service is running
sudo systemctl status spoutmc

# 2. Docker containers started
docker ps

# 3. Web UI is accessible
curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/
# Expected: 200

# 4. Connect a Minecraft client to <server-ip>:25565
```

---

## Upgrading SpoutMC

1. Download the new binary
2. Stop the service
3. Replace the binary
4. Start the service

```bash
sudo systemctl stop spoutmc
sudo curl -L -o /opt/spoutmc/spoutmc \
  https://github.com/0x49b/spoutmc/releases/latest/download/spoutmc-linux-amd64
sudo chmod +x /opt/spoutmc/spoutmc
sudo chown spoutmc:spoutmc /opt/spoutmc/spoutmc
sudo systemctl start spoutmc
```

Configuration files and server data in `/opt/spoutmc/data` are preserved across upgrades.

---

## Troubleshooting

### Service fails to start

```bash
sudo journalctl -u spoutmc -n 50 --no-pager
```

Common causes:
- Missing or invalid `config/spoutmc.yaml`
- Docker socket not accessible (check group membership)
- Port 3000 already in use

### Port already in use

```bash
sudo lsof -i :3000
```

Stop the conflicting process or change SpoutMC's port in the config.

### Docker permission denied

```bash
# Verify the spoutmc user is in the docker group
groups spoutmc

# If not, add it and restart the service
sudo usermod -aG docker spoutmc
sudo systemctl restart spoutmc
```

### permission denied updating paper-global.yml

If logs show errors like:

- `failed to write paper-global.yml: ... permission denied`
- `Failed to create backup of paper-global.yml: ... permission denied`

the `spoutmc` system user cannot write files under `/opt/spoutmc/data`.

Fix ownership and permissions, then restart:

```bash
sudo chown -R spoutmc:spoutmc /opt/spoutmc/data
sudo find /opt/spoutmc/data -type d -exec chmod 775 {} \;
sudo find /opt/spoutmc/data -type f -exec chmod 664 {} \;
sudo systemctl restart spoutmc
```

If shutdown still times out, increase systemd stop timeout:

```ini
# /etc/systemd/system/spoutmc.service
[Service]
TimeoutStopSec=120
```

Then apply and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart spoutmc
```

### EULA not accepted

SpoutMC refuses to start Minecraft servers if the EULA is not accepted. Ensure `config/spoutmc.yaml` contains:

```yaml
eula:
  accepted: true
```

### Containers not starting

```bash
# Check Docker logs for a specific container
docker logs <container-name>

# Check available disk space
df -h
```

---

## Additional Resources

- **Build from Source**: [`BUILD.md`](BUILD.md)
- **Development Workflow**: [`DEVELOPMENT.md`](DEVELOPMENT.md)
- **GitOps Configuration**: [`GITOPS.md`](GITOPS.md)
- **GitOps Quickstart**: [`GITOPS_QUICKSTART.md`](GITOPS_QUICKSTART.md)
- **Release Notes**: [`RELEASE.md`](RELEASE.md)
