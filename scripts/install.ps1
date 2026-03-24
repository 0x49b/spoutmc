#Requires -Version 5.1
<#
.SYNOPSIS
    SpoutMC Windows Installer

.DESCRIPTION
    Installs Docker Desktop, downloads SpoutMC, registers it as a Windows Service,
    and configures Windows Firewall rules.

    Requirements:
    - Windows 10 version 2004+ or Windows 11 (for WSL2)
    - PowerShell 5.1+ (run as Administrator)
    - winget (App Installer) — included in Windows 10/11

.PARAMETER InstallPath
    Installation directory. Default: C:\spoutmc

.PARAMETER DataPath
    Server data directory. Default: C:\spoutmc\data

.PARAMETER Port
    SpoutMC web UI port. Default: 3000

.PARAMETER MinecraftPort
    Minecraft server port. Default: 25565

.PARAMETER Memory
    Default Minecraft server memory allocation. Default: 2G

.PARAMETER NonInteractive
    Skip all prompts and use defaults / provided parameters.

.EXAMPLE
    # Interactive install
    .\install.ps1

    # Non-interactive
    .\install.ps1 -NonInteractive -Port 8080 -Memory 4G
#>
[CmdletBinding()]
param(
    [string]$InstallPath   = "C:\spoutmc",
    [string]$DataPath      = "",
    [string]$Port          = "3000",
    [string]$MinecraftPort = "25565",
    [string]$Memory        = "2G",
    [switch]$NonInteractive
)

$ErrorActionPreference = "Stop"

# ─── Colours ─────────────────────────────────────────────────────────────────
function Write-Info   { param($Msg) Write-Host "[✓] $Msg" -ForegroundColor Green }
function Write-Warn   { param($Msg) Write-Host "[!] $Msg" -ForegroundColor Yellow }
function Write-Err    { param($Msg) Write-Host "[✗] $Msg" -ForegroundColor Red }
function Write-Header { param($Msg) Write-Host "`n==> $Msg" -ForegroundColor Cyan }

# ─── Helpers ─────────────────────────────────────────────────────────────────
function Prompt-Default {
    param([string]$PromptText, [string]$Default)
    if ($NonInteractive) { return $Default }
    $ans = Read-Host "$PromptText [$Default]"
    if ([string]::IsNullOrWhiteSpace($ans)) { return $Default }
    return $ans
}

function Prompt-YN {
    param([string]$PromptText, [bool]$DefaultYes = $false)
    if ($NonInteractive) { return $DefaultYes }
    $hint = if ($DefaultYes) { "Y/n" } else { "y/N" }
    $ans  = Read-Host "$PromptText [$hint]"
    if ([string]::IsNullOrWhiteSpace($ans)) { return $DefaultYes }
    return ($ans -match "^[Yy]")
}

function Test-CommandExists {
    param([string]$Command)
    return [bool](Get-Command $Command -ErrorAction SilentlyContinue)
}

# ─── Administrator check ──────────────────────────────────────────────────────
function Assert-Admin {
    $principal = [Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        Write-Err "This script must be run as Administrator."
        Write-Err "Right-click PowerShell and select 'Run as administrator', then run the script again."
        exit 1
    }
}

# ─── WSL2 check ───────────────────────────────────────────────────────────────
function Ensure-WSL2 {
    Write-Header "Checking WSL2"
    $wslFeature = Get-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux -ErrorAction SilentlyContinue
    $vmFeature  = Get-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform -ErrorAction SilentlyContinue

    $needsReboot = $false

    if ($wslFeature -and $wslFeature.State -ne "Enabled") {
        Write-Warn "Enabling Windows Subsystem for Linux..."
        Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux -NoRestart | Out-Null
        $needsReboot = $true
    }

    if ($vmFeature -and $vmFeature.State -ne "Enabled") {
        Write-Warn "Enabling Virtual Machine Platform..."
        Enable-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform -NoRestart | Out-Null
        $needsReboot = $true
    }

    if ($needsReboot) {
        Write-Warn "A reboot is required to finish enabling WSL2."
        Write-Warn "Please reboot and re-run this installer to continue."
        Write-Host ""
        Read-Host "Press Enter to reboot now, or Ctrl+C to cancel and reboot manually"
        Restart-Computer -Force
        exit 0
    }

    # Set WSL default version to 2
    if (Test-CommandExists "wsl") {
        wsl --set-default-version 2 2>$null | Out-Null
    }

    Write-Info "WSL2 is available."
}

# ─── Docker Desktop ───────────────────────────────────────────────────────────
function Ensure-Docker {
    Write-Header "Checking Docker Desktop"

    if (Test-CommandExists "docker") {
        try {
            docker info 2>$null | Out-Null
            Write-Info "Docker Desktop is already installed and running — skipping."
            return
        } catch { }
    }

    # Check winget
    if (-not (Test-CommandExists "winget")) {
        Write-Err "winget (App Installer) is not available."
        Write-Err "Install it from the Microsoft Store: https://aka.ms/getwinget"
        Write-Err "Then re-run this installer."
        exit 1
    }

    Write-Header "Installing Docker Desktop via winget"
    Write-Warn "This may take a few minutes. Docker Desktop requires a restart after installation."

    winget install --id Docker.DockerDesktop --accept-package-agreements --accept-source-agreements --silent
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Docker Desktop installation failed (exit code $LASTEXITCODE)."
        Write-Err "Install manually from https://www.docker.com/products/docker-desktop/ and re-run."
        exit 1
    }

    Write-Info "Docker Desktop installed."
    Write-Warn "Docker Desktop requires a reboot / manual start before first use."
    Write-Warn "After rebooting, start Docker Desktop, then re-run this installer to continue setup."
    Write-Warn "(Or if Docker Desktop is already running after install, you can continue now.)"

    if (-not $NonInteractive) {
        $continue = Prompt-YN "Continue without rebooting? (only if Docker Desktop started successfully)" $false
        if (-not $continue) {
            Write-Host "Please reboot, start Docker Desktop, then re-run this installer."
            exit 0
        }
    }
}

# ─── Wait for Docker to be ready ─────────────────────────────────────────────
function Wait-Docker {
    Write-Header "Waiting for Docker to be ready"
    $maxAttempts = 20
    for ($i = 1; $i -le $maxAttempts; $i++) {
        try {
            $result = & docker info 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Info "Docker is ready."
                return
            }
        } catch { }
        Write-Host "  Waiting for Docker... ($i/$maxAttempts)"
        Start-Sleep -Seconds 3
    }
    Write-Err "Docker did not become ready in time."
    Write-Err "Make sure Docker Desktop is running, then re-run this installer."
    exit 1
}

# ─── Download binary ─────────────────────────────────────────────────────────
function Download-SpoutMC {
    param([string]$Destination)

    Write-Header "Downloading SpoutMC binary"
    $url = "https://github.com/0x49b/spoutmc/releases/latest/download/spoutmc-windows-amd64.exe"

    Write-Host "  Downloading from $url ..."
    Invoke-WebRequest -Uri $url -OutFile $Destination -UseBasicParsing
    Write-Info "Downloaded to $Destination"
}

# ─── Windows Service ──────────────────────────────────────────────────────────
function Install-SpoutMCService {
    param([string]$BinaryPath, [string]$WorkDir)

    Write-Header "Installing Windows Service"
    $svcName = "SpoutMC"
    $svcDesc = "SpoutMC Minecraft Server Manager"

    # Remove existing service if present
    $existing = Get-Service -Name $svcName -ErrorAction SilentlyContinue
    if ($existing) {
        Write-Warn "Service '$svcName' already exists — replacing."
        Stop-Service -Name $svcName -Force -ErrorAction SilentlyContinue
        & sc.exe delete $svcName | Out-Null
        Start-Sleep -Seconds 2
    }

    # Create service
    & sc.exe create $svcName `
        binPath= "`"$BinaryPath`"" `
        start= auto `
        obj= "LocalSystem" `
        DisplayName= $svcDesc | Out-Null

    if ($LASTEXITCODE -ne 0) {
        Write-Err "Failed to create Windows service."
        exit 1
    }

    & sc.exe description $svcName $svcDesc | Out-Null
    & sc.exe failure $svcName reset= 60 actions= restart/10000/restart/30000/restart/60000 | Out-Null

    # Set working directory via registry (sc.exe doesn't support it natively)
    $regPath = "HKLM:\SYSTEM\CurrentControlSet\Services\$svcName"
    Set-ItemProperty -Path $regPath -Name "AppDirectory" -Value $WorkDir -ErrorAction SilentlyContinue

    Write-Info "Service '$svcName' created."
}

# ─── Firewall rules ───────────────────────────────────────────────────────────
function Configure-Firewall {
    param([string]$SpoutMCPort, [string]$MCPort)

    Write-Header "Configuring Windows Firewall"

    $rules = @(
        @{ Name = "SpoutMC Web UI (TCP $SpoutMCPort)";  Port = $SpoutMCPort; Proto = "TCP" },
        @{ Name = "Minecraft Server (TCP $MCPort)";      Port = $MCPort;      Proto = "TCP" },
        @{ Name = "Minecraft Server (UDP $MCPort)";      Port = $MCPort;      Proto = "UDP" }
    )

    foreach ($rule in $rules) {
        $existing = Get-NetFirewallRule -DisplayName $rule.Name -ErrorAction SilentlyContinue
        if ($existing) {
            Write-Info "Firewall rule '$($rule.Name)' already exists — skipping."
        } else {
            New-NetFirewallRule `
                -DisplayName $rule.Name `
                -Direction Inbound `
                -Protocol $rule.Proto `
                -LocalPort $rule.Port `
                -Action Allow | Out-Null
            Write-Info "Added firewall rule: $($rule.Name)"
        }
    }
}

# ─── Main ─────────────────────────────────────────────────────────────────────
function Main {
    Write-Host ""
    Write-Host "╔══════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║      SpoutMC Windows Installer       ║" -ForegroundColor Cyan
    Write-Host "╚══════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""

    Assert-Admin

    # ── Interactive config ──
    if (-not $NonInteractive) {
        Write-Header "Configuration"
        $InstallPath   = Prompt-Default "Install path"              $InstallPath
        $Port          = Prompt-Default "SpoutMC UI port"           $Port
        $MinecraftPort = Prompt-Default "Minecraft port"            $MinecraftPort
        $Memory        = Prompt-Default "Default Minecraft memory"  $Memory
    }

    if ([string]::IsNullOrWhiteSpace($DataPath)) {
        $DataPath = Join-Path $InstallPath "data"
    }

    $ConfigDir  = Join-Path $InstallPath "config"
    $BinaryPath = Join-Path $InstallPath "spoutmc.exe"
    $ConfigFile = Join-Path $ConfigDir "spoutmc.yaml"

    # ── WSL2 + Docker ──
    Ensure-WSL2
    Ensure-Docker
    Wait-Docker

    # ── Directories ──
    Write-Header "Creating directory structure"
    foreach ($dir in @($InstallPath, $ConfigDir, $DataPath)) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
            Write-Info "Created $dir"
        } else {
            Write-Info "$dir already exists — skipping."
        }
    }

    # ── Download binary ──
    if (Test-Path $BinaryPath) {
        Write-Warn "Binary already exists at $BinaryPath — overwriting."
    }
    Download-SpoutMC -Destination $BinaryPath

    # ── Config file ──
    if (Test-Path $ConfigFile) {
        Write-Warn "Config already exists at $ConfigFile — skipping generation."
    } else {
        Write-Header "Writing default configuration"
        $today = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
        $dataPathFwd = $DataPath -replace '\\', '/'
        $config = @"
# SpoutMC Configuration
# See https://github.com/0x49b/spoutmc for full documentation

# EULA - you must accept the Minecraft EULA to run any servers
eula:
  accepted: true
  accepted_on: "$today"

# Storage - where server data (worlds, plugins, etc.) is persisted on the host
storage:
  data_path: "$dataPathFwd"

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

# Servers - define the Minecraft servers SpoutMC should manage
servers:
  - name: lobby
    image: itzg/minecraft-server:latest
    lobby: true
    proxy: false
    ports:
      - hostPort: "$MinecraftPort"
        containerPort: "25565"
    env:
      TYPE: PAPER
      VERSION: "1.21.4"
      EULA: "TRUE"
      MEMORY: "$Memory"
    volumes:
      - containerpath: "/data"
"@
        Set-Content -Path $ConfigFile -Value $config -Encoding UTF8
        Write-Info "Config written to $ConfigFile"
    }

    # ── Windows Service ──
    Install-SpoutMCService -BinaryPath $BinaryPath -WorkDir $InstallPath

    # Start the service
    Write-Header "Starting SpoutMC service"
    try {
        Start-Service -Name "SpoutMC"
        Start-Sleep -Seconds 3
        $svc = Get-Service -Name "SpoutMC"
        if ($svc.Status -eq "Running") {
            Write-Info "SpoutMC service is running."
        } else {
            Write-Warn "SpoutMC service status: $($svc.Status). Check logs if it doesn't start."
        }
    } catch {
        Write-Warn "Could not start service automatically: $_"
        Write-Warn "Start it manually: Start-Service SpoutMC"
    }

    # ── Firewall ──
    Configure-Firewall -SpoutMCPort $Port -MCPort $MinecraftPort

    # ── Reverse proxy note ──
    Write-Host ""
    Write-Warn "Note: nginx is not typically used on Windows."
    Write-Warn "If you need a reverse proxy (HTTPS), consider:"
    Write-Warn "  - Caddy  : https://caddyserver.com  (easiest, auto-TLS)"
    Write-Warn "  - IIS    : built into Windows Server"

    # ── Summary ──
    Write-Host ""
    Write-Host "╔══════════════════════════════════════════════╗" -ForegroundColor Green
    Write-Host "║       SpoutMC Installation Complete!         ║" -ForegroundColor Green
    Write-Host "╚══════════════════════════════════════════════╝" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Install path : $InstallPath"
    Write-Host "  Data path    : $DataPath"
    Write-Host "  Config       : $ConfigFile"
    Write-Host ""
    Write-Host "  Web UI       : http://localhost:$Port"
    Write-Host "  Minecraft    : <server-ip>:$MinecraftPort"
    Write-Host ""
    Write-Host "  Service commands (run as Administrator):"
    Write-Host "    Start-Service SpoutMC"
    Write-Host "    Stop-Service  SpoutMC"
    Write-Host "    Get-Service   SpoutMC"
    Write-Host ""
    Write-Host "  Open http://localhost:$Port and complete the setup wizard"
    Write-Host "  to create your first admin user."
    Write-Host ""
}

Main
