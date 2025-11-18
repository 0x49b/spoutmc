# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

SpoutMC is a Docker-based Minecraft Server Network manager written in Go with a React/TypeScript web frontend. It orchestrates multiple Minecraft server containers (proxy, lobby, game servers) using the Docker API and provides a web interface for monitoring and management.

## Important Notes

**DO NOT run `go build` automatically.** The developer will build the Go backend manually. Only suggest build commands when explicitly asked, but do not execute them.

**DO NOT run the frontend (`npm run dev`) automatically.** The developer will run the frontend manually when needed. Only suggest frontend commands when explicitly asked, but do not execute them.

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
2. **watchdog** - Starts container health monitoring (15s poll interval):
   - Monitors container status (stopped/dead)
   - Monitors container health (unhealthy)
   - Ensures all config servers are running
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
- `watchdog.go` - Monitors container health every 15s with three checks:
  1. **Stopped/Dead Containers**: Restarts containers in "exited" or "dead" state
  2. **Unhealthy Containers**: Restarts containers with health check status "unhealthy"
  3. **Missing Servers**: Creates/starts servers defined in config but not running
- `config_file_watcher.go` - Watches config file, triggers hot-reload on changes
- Can exclude specific containers from monitoring
- Automatically ensures config state matches running state

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
- Volume host paths auto-generated: `{storage.data_path}/{server.name}/{containerpath}`
- OS-specific path separators handled automatically via `filepath.Join`

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
storage:
  data_path: "/path/to/data"  # Base directory for server data

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
    volumes:                  # Volume bindings (hostpath auto-generated)
      - containerpath: "/server"   # → {data_path}/spoutproxy/server (default for proxy)

  - name: lobby
    image: itzg/minecraft-server:latest
    lobby: true
    ports:
      - hostPort: '25566'
        containerPort: '25566'
    env:
      TYPE: PAPER
      VERSION: "1.21.10"
    volumes:
      - containerpath: "/data"     # → {data_path}/lobby/data (default for lobby/game)
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

## Recent Changes & Important Notes

### Circular Import Fix
- **Issue**: Circular dependency between `internal/config` and `internal/docker`
- **Solution**: Removed `config` import from `internal/docker/mapper.go`
- `createHostPath()` now simply returns `{workingDir}/{containerName}` without accessing config

### Frontend Server Details Page
- **Removed features**: Replica Status, Pods Tab, Scale Deployment
- These were Kubernetes-oriented features not applicable to Docker-only setup
- Simplified to show: Server Information, Resource Usage (CPU/Memory), Description
- Updated skeleton components to match

### SSE & Loading States
- **SSE connection moved to ServerDetail component level** (was in OverviewTab)
- Stats are loaded once and persist across tab switches
- Skeleton only shows on initial page load, not when changing tabs
- Applies to both OverviewTab and ConsoleTab

### Console Scroll Fix
- Console "scroll to bottom" now scrolls the console container, not the entire page
- Changed from `scrollIntoView()` to direct `scrollTop` manipulation
- Uses `logsContainerRef` to target the scrollable div

### Add Server Modal & API
**Frontend (`web/src/components/Servers/AddServerModal.tsx`)**:
- Removed: IP Address, Location, Description, Plugins fields
- Added: Docker Image field, Environment Variables (key:value pairs with add/remove)
- Sends to backend: `{ name, image, port, env }`

**Backend (`internal/webserver/api/v1/server/server.go`)**:
- New endpoint: `POST /api/v1/server`
- Creates server configuration with Docker-specific fields
- **GitOps support**: Checks `config.IsGitOpsEnabled()`
  - If enabled: Creates `{name}.yaml` in git repo, commits, and pushes
  - If disabled: Appends to `config/spoutmc.yaml` and reloads
- Automatically starts the Docker container after creation

**Git Operations (`internal/git/repository.go`)**:
- Added `CommitAndPush()` method to Repository struct
- Added `CommitAndPushChanges()` convenience function
- Uses PAT authentication embedded in remote URL
- Format: `https://{token}@github.com/user/repo.git`

### Server Model Structure
Servers now have:
- `name` - Container/server name
- `image` - Docker image (e.g., `itzg/minecraft-server:latest`)
- `env` - Map of environment variables
- `ports` - Array of port mappings (host:container)
- `volumes` - Array of volume bindings with only `containerpath`
  - Host path auto-generated: `{storage.data_path}/{server.name}/{containerpath}`
  - Default containerpath:
    - Proxy servers: `/server`
    - Lobby/Game servers: `/data`
  - Example: If `data_path=/data`, `name=lobby`, `containerpath=/data` → hostpath: `/data/lobby/data`
  - Multiple volumes supported: each containerpath gets its own subdirectory
  - OS-specific separators handled automatically

### Enhanced Add Server Feature (In Progress)

The Add Server modal has been significantly enhanced with the following features:

#### Server Type Selection
- **Radio button group** for server type selection:
  - **Proxy Server**: Main entry point (only one allowed)
  - **Lobby Server**: Central hub (only one allowed)
  - **Game Server**: Regular game servers (unlimited)
- Automatic Docker image selection based on server type:
  - Proxy: `itzg/mc-proxy:latest`
  - Lobby/Game: `itzg/minecraft-server:latest`
- **Disabled state** for proxy/lobby when one already exists in network
- Visual feedback with descriptions explaining each type

#### Validation & Constraints
- **Frontend validation**: Radio buttons disabled when proxy/lobby exists
- **Backend validation**: API endpoint validates and returns error if duplicate proxy/lobby
- **Auto-correction**: Switches to game server if selected type becomes unavailable
- Real-time validation with user-friendly error messages

#### Dynamic Port Assignment
- **Port field hidden** for lobby and game servers
- **Port field shown** only for proxy servers (user-defined, defaults to 25565)
- Backend automatically assigns ports starting from 25566 for lobby/game servers
- `findNextAvailablePort()` function scans existing ports and finds next available
- No port conflicts - system manages allocation automatically

#### System-Managed Environment Variables
**Proxy servers** get:
- `TYPE=VELOCITY`

**Lobby and game servers** get:
- `EULA=TRUE`
- `TYPE=PAPER`
- `ONLINE_MODE=FALSE`
- `GUI=FALSE`
- `CONSOLE=FALSE`
- `VERSION=<selected>`

**Frontend warnings**:
- Yellow border on env var inputs when user enters system-managed variable
- Warning message: "This variable is system-managed. Your value will override the default"
- Info box showing all system-managed variables for selected server type

**Backend merging**:
- Default env vars set based on server type
- User-provided values override defaults
- `mergeEnvVars()` function handles the merge logic

#### Version Selection
- **Dropdown/Select field** for Minecraft version (lobby/game servers only)
- Versions loaded from `config/spoutmc.yaml` via API endpoint `GET /api/v1/versions`
- 50+ PaperMC versions available (1.21.10 down to 1.8.8)
- Selected version automatically added as `VERSION` env var
- Listed as system-managed variable (can be overridden)
- Versions always loaded from local config, even with GitOps enabled

**GitOps integration**:
- `LoadConfigurationFromGit()` preserves versions from local config
- Servers come from Git, versions come from local `config/spoutmc.yaml`

#### Form Management
- **Auto-reset**: All fields clear when modal closes (any close method)
- **Smart defaults**: Version defaults to first in list, server type to game
- Clean state for next server creation

#### API Changes
**Request structure** (`POST /api/v1/server`):
```json
{
  "name": "server-name",
  "image": "itzg/minecraft-server:latest",
  "port": 25565,  // Optional - only for proxy
  "proxy": true,  // Optional
  "lobby": false, // Optional
  "env": {
    "CUSTOM_VAR": "value"
  }
}
```

**New endpoint** (`GET /api/v1/versions`):
Returns array of available Minecraft versions from config

#### Configuration Structure
```yaml
# config/spoutmc.yaml
versions:
  - "1.21.10"
  - "1.21.9"
  # ... more versions

servers:
  - name: example
    # ... server config
```

## TODO / Next Steps

### Future Server Creation Enhancements
- **Multiple ports**: Allow users to add multiple port mappings
- **Volume configuration**: Let users customize volume bindings beyond the default
- **Advanced options**: Network mode, restart policy, resource limits
- **Image validation**: Verify Docker image exists before deployment
