# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

SpoutMC is a Docker-based Minecraft Server Network manager written in Go with a React/TypeScript web frontend. It orchestrates multiple Minecraft server containers (proxy, lobby, game servers) using the Docker API and provides a web interface for monitoring and management.

## Important Notes

**DO NOT run `go build` automatically.** The developer will build the Go backend manually. Only suggest build commands when explicitly asked, but do not execute them.

## Development Commands

### Go Backend

```bash
# Build the application
go build -o spoutmc ./cmd/spoutmc

# Run the application (requires config/spoutmc.yaml)
./spoutmc

# Run with live reload (requires Air)
air

# Generate Swagger documentation
swag init -g cmd/spoutmc/main.go

# Run Go tests (if any exist)
go test ./...
```

### Web Frontend

```bash
# Navigate to web directory first
cd web

# Install dependencies
npm install

# Run development server (localhost:5173)
npm run dev

# Build for production
npm run build

# Lint code
npm run lint

# Preview production build
npm run preview
```

## Architecture

### Startup Sequence

The application starts components in this order (see cmd/spoutmc/main.go:71-77):
1. **spoutmc** - Initializes Docker network and containers
2. **watchdog** - Starts container health monitoring (15s poll interval)
3. **fileWatcher** - Monitors config/spoutmc.yaml for changes
4. **webserver** - Starts Echo HTTP server on port 3000

### Core Components

**Docker Management (internal/docker/)**
- `docker.go` - Container lifecycle (create, start, stop, remove)
- `network.go` - Docker network management (creates "spoutnetwork")
- `client.go` - Docker client initialization
- `mapper.go` - Maps config models to Docker API types
- Uses label `io.spout.network=true` to identify managed containers
- Supports labels: `io.spout.proxy`, `io.spout.lobby`, `io.spout.servername`

**Configuration (internal/config/)**
- Loads from `config/spoutmc.yaml` or `config/spoutmc.yml`
- `config_loader.go` - Reads YAML into package-level state
- `config_diff.go` - Compares old/new configs for hot-reload
- File watcher applies changes dynamically without restart

**Watchdog (internal/watchdog/)**
- `watchdog.go` - Monitors container health every 15s, restarts exited/dead containers
- `config_file_watcher.go` - Watches config file, triggers hot-reload on changes
- Can exclude specific containers from monitoring

**Web Server (internal/webserver/)**
- Echo framework on port 3000
- API routes under `/api/v1/` (server, user, host endpoints)
- Swagger docs at `/swagger/*`
- CORS enabled for localhost:3000 and localhost:5173
- Rate limiting: 20 requests per store window

**Models (internal/models/)**
- `spoutmodel.go` - Core config types (SpoutConfiguration, SpoutServer, etc.)
- Servers can be proxy, lobby, or regular game servers
- Each server has: name, image, env vars, ports, volumes

### Container Lifecycle

1. **Container Creation Flow**:
   - Config loaded from spoutmc.yaml
   - Image pulled if not cached
   - Container created with labels, network, volumes, ports
   - Connected to "spoutnetwork" bridge network
   - Started and monitored by watchdog

2. **Hot Reload**:
   - File watcher detects config changes
   - Config diff identifies added/removed/modified servers
   - Modified servers: stop → remove → recreate
   - Removed servers: stop → remove (with volume cleanup)
   - Added servers: create → start

3. **Shutdown**:
   - Stops in reverse order: fileWatcher, watchdog, containers, webserver
   - Graceful shutdown with 30s timeout context
   - All spout network containers stopped cleanly

### Frontend Architecture (web/)

- React 19 + TypeScript + Vite
- TailwindCSS 4 for styling
- React Router v7 for routing
- Zustand for state management
- Axios for API calls
- Framer Motion for animations
- Lucide React for icons
- JWT authentication with jwt-decode

## GitOps Support

SpoutMC now supports GitOps-style configuration with individual YAML files per server in a Git repository.

**Key Files:**
- `internal/git/` - Complete GitOps implementation
  - `repository.go` - Git clone/pull with PAT authentication
  - `config_loader.go` - Multi-file YAML reader
  - `poller.go` - Background polling for changes
  - `webhook.go` - Webhook handler with signature verification
  - `git_sync.go` - Main GitOps orchestration

**How it works:**
1. Enable GitOps in `config/spoutmc.yaml` with `git.enabled: true`
2. SpoutMC clones the configured repository
3. Reads all `.yaml`/`.yml` files as individual server configs
4. Merges them into a SpoutConfiguration in memory
5. Monitors for changes via polling (default 30s) and/or webhooks
6. Automatically applies changes using existing config diff system

**Startup with GitOps:**
- `gitSync` runs first (before `spoutmc`)
- If GitOps enabled: loads config from Git, starts poller, disables file watcher
- If GitOps disabled: uses traditional file-based config with file watcher

**Webhook endpoints:**
- `POST /api/v1/git/webhook` - Receives GitHub/GitLab webhooks
- `POST /api/v1/git/sync` - Manual sync trigger

**Authentication:**
- Supports private repos via HTTPS with Personal Access Token
- Token added to repository URL: `https://token@github.com/user/repo.git`
- Webhook verification using HMAC-SHA256 (GitHub) or token comparison (GitLab)

See `GITOPS.md` for detailed documentation.

## Configuration

The `config/spoutmc.yaml` defines the server network:

```yaml
servers:
  - name: spoutproxy          # Container name
    image: itzg/mc-proxy      # Docker image
    proxy: true               # Marks as proxy server
    ports:                    # Port mappings
      - hostPort: '25565'
        containerPort: '25565'
    env:                      # Environment variables
      TYPE: VELOCITY
      MAX_MEMORY: 1G
    volumes:                  # Volume bindings
      - hostpath: [testservers, data, spoutproxy]
        containerpath: "/server"
```

## Important Notes

- The application uses Go 1.24.0
- All Docker operations use background context (docker.go:25)
- Containers are labeled for identification and filtering
- The watchdog excludes containers via ID, not name
- Web server writes routes.json on startup for debugging
- Database integration exists but is not actively used in startup
- Proxy config file path: `{proxyVolumeMount}/velocity.toml`

## Testing

Currently, no test files exist in the codebase. When adding tests:
- Place in `*_test.go` files alongside source
- Use standard Go testing package
- Test container lifecycle operations with Docker test containers if needed
