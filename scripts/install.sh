#!/usr/bin/env bash
# SpoutMC Linux Installer
# Installs Docker Engine, downloads SpoutMC, creates a systemd service,
# configures the firewall, and optionally sets up an nginx reverse proxy.
#
# Usage:
#   sudo bash install.sh [options]
#
# Options:
#   --install-path PATH     Installation directory        (default: /opt/spoutmc)
#   --data-path PATH        Server data directory         (default: /opt/spoutmc/data)
#   --port PORT             SpoutMC web UI port           (default: 3000)
#   --mc-port PORT          Minecraft server port         (default: 25565)
#   --memory MEM            Default Minecraft memory      (default: 2G)
#   --domain DOMAIN         Domain for nginx reverse proxy
#   --no-nginx              Skip nginx installation
#   --no-interactive        Use defaults for all prompts
#   -h, --help              Show this help

set -euo pipefail

# ─── Colours ────────────────────────────────────────────────────────────────
if [ -t 1 ] && command -v tput &>/dev/null && tput colors &>/dev/null; then
  RED=$(tput setaf 1); GREEN=$(tput setaf 2); YELLOW=$(tput setaf 3)
  CYAN=$(tput setaf 6); BOLD=$(tput bold); RESET=$(tput sgr0)
else
  RED=''; GREEN=''; YELLOW=''; CYAN=''; BOLD=''; RESET=''
fi

info()    { echo "${GREEN}[✓]${RESET} $*"; }
warn()    { echo "${YELLOW}[!]${RESET} $*"; }
error()   { echo "${RED}[✗]${RESET} $*" >&2; }
header()  { echo; echo "${BOLD}${CYAN}==> $*${RESET}"; }

# ─── Defaults ───────────────────────────────────────────────────────────────
INSTALL_PATH="${INSTALL_PATH:-/opt/spoutmc}"
DATA_PATH="${DATA_PATH:-/opt/spoutmc/data}"
SPOUTMC_PORT="${SPOUTMC_PORT:-3000}"
MC_PORT="${MC_PORT:-25565}"
MC_MEMORY="${MC_MEMORY:-2G}"
DOMAIN="${DOMAIN:-}"
INSTALL_NGINX="${INSTALL_NGINX:-}"    # empty = ask; "yes"/"no" to skip prompt
NON_INTERACTIVE="${NON_INTERACTIVE:-0}"

GITHUB_REPO="0x49b/spoutmc"
SERVICE_USER="spoutmc"

# ─── Argument parsing ────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-path)   INSTALL_PATH="$2";    shift 2 ;;
    --data-path)      DATA_PATH="$2";       shift 2 ;;
    --port)           SPOUTMC_PORT="$2";    shift 2 ;;
    --mc-port)        MC_PORT="$2";         shift 2 ;;
    --memory)         MC_MEMORY="$2";       shift 2 ;;
    --domain)         DOMAIN="$2"; INSTALL_NGINX="yes"; shift 2 ;;
    --no-nginx)       INSTALL_NGINX="no";   shift ;;
    --no-interactive) NON_INTERACTIVE=1;    shift ;;
    -h|--help)
      grep '^#' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      error "Unknown option: $1"
      exit 1
      ;;
  esac
done

# ─── Root check ──────────────────────────────────────────────────────────────
if [[ $EUID -ne 0 ]]; then
  error "This script must be run as root. Try: sudo bash $0"
  exit 1
fi

# ─── Detect OS ───────────────────────────────────────────────────────────────
detect_os() {
  if [[ -f /etc/os-release ]]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    OS_ID="${ID:-unknown}"
    OS_ID_LIKE="${ID_LIKE:-}"
    OS_CODENAME="${VERSION_CODENAME:-}"
  else
    error "Cannot detect OS. /etc/os-release not found."
    exit 1
  fi

  case "$OS_ID" in
    ubuntu|debian|linuxmint|pop)
      PKG_FAMILY="debian"
      ;;
    fedora|rhel|centos|almalinux|rocky)
      PKG_FAMILY="rhel"
      ;;
    *)
      if echo "$OS_ID_LIKE" | grep -qi "debian"; then
        PKG_FAMILY="debian"
      elif echo "$OS_ID_LIKE" | grep -qi "rhel\|fedora"; then
        PKG_FAMILY="rhel"
      else
        warn "Unsupported OS '$OS_ID'. Attempting to proceed as Debian-based."
        PKG_FAMILY="debian"
      fi
      ;;
  esac
}

# ─── Detect architecture ─────────────────────────────────────────────────────
detect_arch() {
  local machine
  machine=$(uname -m)
  case "$machine" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *)
      error "Unsupported architecture: $machine"
      exit 1
      ;;
  esac
}

# ─── Interactive prompts ──────────────────────────────────────────────────────
ask() {
  local prompt="$1" default="$2" var_name="$3"
  if [[ $NON_INTERACTIVE -eq 1 ]]; then
    printf -v "$var_name" '%s' "$default"
    return
  fi
  local answer
  read -rp "${CYAN}${prompt}${RESET} [${default}]: " answer
  printf -v "$var_name" '%s' "${answer:-$default}"
}

ask_yn() {
  local prompt="$1" default="$2" var_name="$3"
  if [[ $NON_INTERACTIVE -eq 1 ]]; then
    printf -v "$var_name" '%s' "$default"
    return
  fi
  local hint answer
  [[ "$default" == "y" ]] && hint="Y/n" || hint="y/N"
  read -rp "${CYAN}${prompt}${RESET} [${hint}]: " answer
  answer="${answer:-$default}"
  [[ "$answer" =~ ^[Yy] ]] && printf -v "$var_name" 'yes' || printf -v "$var_name" 'no'
}

# ─── Docker installation ─────────────────────────────────────────────────────
install_docker_debian() {
  header "Installing Docker Engine (Debian/Ubuntu)"
  # Remove old packages silently
  apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true
  apt-get update -qq
  apt-get install -y ca-certificates curl gnupg

  install -m 0755 -d /etc/apt/keyrings
  if [[ "$OS_ID" == "ubuntu" || "$OS_ID_LIKE" == *"ubuntu"* ]]; then
    local docker_gpg_url="https://download.docker.com/linux/ubuntu/gpg"
    local docker_repo="https://download.docker.com/linux/ubuntu"
  else
    local docker_gpg_url="https://download.docker.com/linux/debian/gpg"
    local docker_repo="https://download.docker.com/linux/debian"
  fi

  curl -fsSL "$docker_gpg_url" | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg

  local codename
  # Try OS codename, fall back to lsb_release
  codename="${OS_CODENAME:-$(lsb_release -sc 2>/dev/null || echo stable)}"
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
${docker_repo} ${codename} stable" \
    > /etc/apt/sources.list.d/docker.list

  apt-get update -qq
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin
  info "Docker Engine installed."
}

install_docker_rhel() {
  header "Installing Docker Engine (RHEL/Fedora)"
  dnf install -y dnf-plugins-core
  dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
  dnf install -y docker-ce docker-ce-cli containerd.io
  info "Docker Engine installed."
}

ensure_docker() {
  if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    info "Docker is already installed and running — skipping."
    return
  fi

  if [[ "$PKG_FAMILY" == "debian" ]]; then
    install_docker_debian
  else
    install_docker_rhel
  fi

  systemctl enable --now docker
  info "Docker service started."
}

# ─── nginx installation ───────────────────────────────────────────────────────
install_nginx_debian() {
  apt-get install -y nginx certbot python3-certbot-nginx
}

install_nginx_rhel() {
  dnf install -y nginx certbot python3-certbot-nginx
}

configure_nginx() {
  header "Configuring nginx reverse proxy"

  if [[ "$PKG_FAMILY" == "debian" ]]; then
    install_nginx_debian
    NGINX_SITES_AVAILABLE="/etc/nginx/sites-available"
    NGINX_SITES_ENABLED="/etc/nginx/sites-enabled"
    mkdir -p "$NGINX_SITES_ENABLED"
  else
    install_nginx_rhel
    NGINX_SITES_AVAILABLE="/etc/nginx/conf.d"
    NGINX_SITES_ENABLED=""
  fi

  local conf_file="${NGINX_SITES_AVAILABLE}/spoutmc"

  cat > "$conf_file" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    location / {
        proxy_pass http://127.0.0.1:${SPOUTMC_PORT};
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_read_timeout 86400;
    }
}
EOF

  if [[ -n "$NGINX_SITES_ENABLED" ]]; then
    ln -sf "$conf_file" "${NGINX_SITES_ENABLED}/spoutmc"
  fi

  nginx -t
  systemctl enable --now nginx
  systemctl reload nginx
  info "nginx configured for ${DOMAIN} → http://127.0.0.1:${SPOUTMC_PORT}"

  echo
  warn "To add HTTPS with Let's Encrypt, run:"
  warn "  sudo certbot --nginx -d ${DOMAIN}"
}

# ─── Firewall configuration ───────────────────────────────────────────────────
configure_firewall() {
  header "Configuring firewall"
  if command -v ufw &>/dev/null; then
    ufw allow "${SPOUTMC_PORT}/tcp" comment "SpoutMC Web UI" || true
    ufw allow "${MC_PORT}/tcp" comment "Minecraft" || true
    ufw allow "${MC_PORT}/udp" comment "Minecraft UDP" || true
    ufw reload || true
    info "UFW rules added for ports ${SPOUTMC_PORT} and ${MC_PORT}."
  elif command -v firewall-cmd &>/dev/null; then
    firewall-cmd --permanent --add-port="${SPOUTMC_PORT}/tcp" || true
    firewall-cmd --permanent --add-port="${MC_PORT}/tcp" || true
    firewall-cmd --permanent --add-port="${MC_PORT}/udp" || true
    firewall-cmd --reload || true
    info "firewalld rules added for ports ${SPOUTMC_PORT} and ${MC_PORT}."
  else
    warn "No recognised firewall (ufw/firewalld) found. Open ports ${SPOUTMC_PORT} and ${MC_PORT} manually."
  fi
}

configure_data_permissions() {
  header "Configuring data directory permissions"

  # Ensure the data root exists and is writable by the service user.
  mkdir -p "${DATA_PATH}"
  chown -R "${SERVICE_USER}:${SERVICE_USER}" "${DATA_PATH}"

  # Keep permissions compatible with SpoutMC writes and container-created content.
  find "${DATA_PATH}" -type d -exec chmod 775 {} \; 2>/dev/null || true
  find "${DATA_PATH}" -type f -exec chmod 664 {} \; 2>/dev/null || true

  # If ACL tools are available, keep write access even when files are created by other users.
  if command -v setfacl &>/dev/null; then
    setfacl -R -m "u:${SERVICE_USER}:rwx" "${DATA_PATH}" || true
    setfacl -R -d -m "u:${SERVICE_USER}:rwx" "${DATA_PATH}" || true
    info "Applied ACLs for '${SERVICE_USER}' on ${DATA_PATH}"
  else
    warn "setfacl not found; using owner/group permissions only for ${DATA_PATH}"
  fi
}

# ─── Main installation ────────────────────────────────────────────────────────
main() {
  echo
  echo "${BOLD}${CYAN}╔══════════════════════════════════════╗"
  echo "║       SpoutMC Linux Installer        ║"
  echo "╚══════════════════════════════════════╝${RESET}"
  echo

  detect_os
  detect_arch
  info "Detected OS: ${OS_ID} (${PKG_FAMILY}), arch: ${ARCH}"

  # ── Interactive config ──
  if [[ $NON_INTERACTIVE -eq 0 ]]; then
    header "Configuration"
    ask "Install path"    "$INSTALL_PATH"    INSTALL_PATH
    ask "Data path"       "$DATA_PATH"       DATA_PATH
    ask "SpoutMC UI port" "$SPOUTMC_PORT"    SPOUTMC_PORT
    ask "Minecraft port"  "$MC_PORT"         MC_PORT
    ask "Default Minecraft memory (e.g. 2G, 4G)" "$MC_MEMORY" MC_MEMORY

    if [[ -z "$INSTALL_NGINX" ]]; then
      ask_yn "Install nginx as a reverse proxy?" "n" INSTALL_NGINX
    fi

    if [[ "$INSTALL_NGINX" == "yes" && -z "$DOMAIN" ]]; then
      ask "Domain name for nginx (e.g. spoutmc.example.com)" "" DOMAIN
      if [[ -z "$DOMAIN" ]]; then
        warn "No domain provided — skipping nginx setup."
        INSTALL_NGINX="no"
      fi
    fi
  fi

  # ── Docker ──
  ensure_docker

  # ── Directory structure ──
  header "Creating directory structure"
  mkdir -p "${INSTALL_PATH}/config" "${DATA_PATH}"
  info "Created ${INSTALL_PATH}/{config} and ${DATA_PATH}"

  # ── Download binary ──
  header "Downloading SpoutMC binary"
  local binary_url="https://github.com/${GITHUB_REPO}/releases/latest/download/spoutmc-linux-${ARCH}"
  local binary_path="${INSTALL_PATH}/spoutmc"

  if [[ -f "$binary_path" ]]; then
    warn "Binary already exists at ${binary_path} — overwriting."
  fi

  curl -fsSL -o "$binary_path" "$binary_url"
  chmod +x "$binary_path"
  info "Downloaded to ${binary_path}"

  # ── Dedicated user ──
  header "Creating service user"
  if id "$SERVICE_USER" &>/dev/null; then
    info "User '${SERVICE_USER}' already exists — skipping."
  else
    useradd -r -s /bin/false -d "$INSTALL_PATH" "$SERVICE_USER"
    info "Created system user '${SERVICE_USER}'"
  fi

  # Ensure the user is in the docker group
  if ! id -nG "$SERVICE_USER" | grep -qw docker; then
    usermod -aG docker "$SERVICE_USER"
    info "Added '${SERVICE_USER}' to the docker group."
  fi

  # Set ownership
  chown -R "${SERVICE_USER}:${SERVICE_USER}" "$INSTALL_PATH"
  # DATA_PATH may be outside INSTALL_PATH.
  configure_data_permissions

  # ── Config file ──
  local config_file="${INSTALL_PATH}/config/spoutmc.yaml"
  if [[ -f "$config_file" ]]; then
    warn "Config file already exists at ${config_file} — skipping generation."
  else
    header "Writing default configuration"
    local today
    today=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    cat > "$config_file" <<EOF
# SpoutMC Configuration
# See https://github.com/${GITHUB_REPO} for full documentation

# EULA - you must accept the Minecraft EULA to run any servers
eula:
  accepted: true
  accepted_on: "${today}"

# Storage - where server data (worlds, plugins, etc.) is persisted on the host
storage:
  data_path: "${DATA_PATH}"

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
#   local_path: "${INSTALL_PATH}/git"

# Servers - define the Minecraft servers SpoutMC should manage
servers:
  - name: lobby
    image: itzg/minecraft-server:latest
    lobby: true
    proxy: false
    ports:
      - hostPort: "${MC_PORT}"
        containerPort: "25565"
    env:
      TYPE: PAPER
      VERSION: "1.21.4"
      EULA: "TRUE"
      MEMORY: "${MC_MEMORY}"
    volumes:
      - containerpath: "/data"
EOF
    chown "${SERVICE_USER}:${SERVICE_USER}" "$config_file"
    chmod 640 "$config_file"
    info "Config written to ${config_file}"
  fi

  # ── Systemd service ──
  header "Installing systemd service"
  cat > /etc/systemd/system/spoutmc.service <<EOF
[Unit]
Description=SpoutMC Minecraft Server Manager
Documentation=https://github.com/${GITHUB_REPO}
After=network-online.target docker.service
Wants=network-online.target
Requires=docker.service

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${INSTALL_PATH}
ExecStartPre=/usr/bin/mkdir -p ${DATA_PATH}
ExecStartPre=/usr/bin/chown -R ${SERVICE_USER}:${SERVICE_USER} ${DATA_PATH}
ExecStart=${INSTALL_PATH}/spoutmc
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=spoutmc
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable spoutmc
  systemctl restart spoutmc
  info "Service 'spoutmc' enabled and started."

  # ── Firewall ──
  configure_firewall

  # ── nginx (optional) ──
  if [[ "$INSTALL_NGINX" == "yes" && -n "$DOMAIN" ]]; then
    configure_nginx
  fi

  # ── Summary ──
  echo
  echo "${BOLD}${GREEN}╔══════════════════════════════════════════════╗"
  echo "║          SpoutMC Installation Complete!      ║"
  echo "╚══════════════════════════════════════════════╝${RESET}"
  echo
  echo "  Install path : ${INSTALL_PATH}"
  echo "  Data path    : ${DATA_PATH}"
  echo "  Config       : ${INSTALL_PATH}/config/spoutmc.yaml"
  echo
  if [[ "$INSTALL_NGINX" == "yes" && -n "$DOMAIN" ]]; then
    echo "  Web UI       : http://${DOMAIN}  (HTTPS via: sudo certbot --nginx -d ${DOMAIN})"
  else
    local host_ip
    host_ip=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "<server-ip>")
    echo "  Web UI       : http://${host_ip}:${SPOUTMC_PORT}"
  fi
  echo "  Minecraft    : <server-ip>:${MC_PORT}"
  echo
  echo "  Service commands:"
  echo "    sudo systemctl status spoutmc"
  echo "    sudo journalctl -u spoutmc -f"
  echo
  echo "  Open the Web UI and complete the setup wizard to create your first admin user."
  echo
}

main "$@"
