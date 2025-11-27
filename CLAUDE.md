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

files:
  exclude_patterns:  # Files/folders to exclude from file browser
    - "*.jar"        # Exclude JAR files
    - "*.log"        # Exclude log files
    - "*.tmp"        # Exclude temporary files
    - ".git"         # Exclude git directory
    - "node_modules" # Exclude node_modules
    - "cache"        # Exclude cache directory
    - "*.class"      # Exclude compiled Java classes
    - "*.zip"        # Exclude zip archives

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

## Infrastructure Containers

SpoutMC supports infrastructure containers (databases, caches, etc.) that are managed separately from game servers.

### Configuration

Infrastructure containers can be configured in two ways:

**1. Local Config File (GitOps Disabled)**
- File: `config/infrastructure.yaml`
- Example: `config/infrastructure.example.yaml`

**2. GitOps Repository (GitOps Enabled)**
- Directory: `infrastructure/` in your GitOps repository
- Each `.yaml` file represents one infrastructure container

### Example Configuration

```yaml
infrastructure:
  - name: database
    image: mariadb:latest
    restart: always
    ports:
      - host: "3306"
        container: "3306"
    volumes:
      - containerpath: /var/lib/mysql
    env:
      MARIADB_ROOT_PASSWORD: changeme  # Auto-generated
      MARIADB_PASSWORD: changeme       # Auto-generated
      MARIADB_USER: spoutmc
      MARIADB_DATABASE: spoutmc
```

### Password Management

- Passwords with value `changeme` are automatically replaced with secure generated passwords
- Generated passwords are displayed in the console on first startup in a formatted box
- Users must save these passwords securely as they are not persisted to disk
- Passwords are regenerated on each startup if `changeme` placeholders are present

### Container Labels

Infrastructure containers are labeled with:
- `io.spout.network=true` - Part of spout network
- `io.spout.infrastructure=true` - Infrastructure container (excluded from Servers view)
- `io.spout.database=true` - Database type (or other type-specific labels)

### Frontend

- **Infrastructure page**: `/infrastructure` - Shows only infrastructure containers
- **Servers page**: `/servers` - Shows only game/lobby/proxy servers (excludes infrastructure)
- Infrastructure containers appear with database icon and type badge

### Watchdog Monitoring

- Infrastructure containers are monitored every 15 seconds
- Automatically restarted if stopped, dead, or unhealthy
- Uses Docker health checks when available

### API Endpoints

- `GET /api/v1/infrastructure` - List all infrastructure containers
- `GET /api/v1/infrastructure/:id` - Get single container details
- `GET /api/v1/infrastructure/debug/all` - Debug endpoint showing all containers with labels

### Key Files

- `/internal/infrastructure/database.go` - Infrastructure container management
- `/internal/infrastructure/passwords.go` - Password generation
- `/internal/infrastructure/config_loader.go` - Local config file loader
- `/internal/git/config_loader.go` - GitOps infrastructure loader
- `/cmd/spoutmc/main.go` - `startInfrastructure()` function

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

### File Browser Exclusions

The file browser supports excluding files and folders from being displayed to users. This is useful for:
- Hiding large binary files (JARs, archives)
- Preventing access to sensitive files
- Reducing clutter in the file tree
- Improving performance by not scanning unnecessary directories

**Configuration** (`config/spoutmc.yaml`):
```yaml
files:
  exclude_patterns:
    - "*.jar"        # Extension patterns (glob)
    - "*.log"        # Any file ending in .log
    - ".git"         # Exact folder/file name
    - "cache"        # Exact folder/file name
    - "temp*"        # Glob pattern (temp, tempfile, etc.)
```

**Pattern matching**:
- Uses Go's `filepath.Match` for glob patterns
- Supports wildcards: `*` (matches any sequence), `?` (matches single char)
- Exact string matching as fallback
- Applied to file/folder names (not full paths)
- Examples:
  - `*.jar` → matches `server.jar`, `plugin.jar`
  - `*.log` → matches `latest.log`, `debug.log`
  - `.git` → matches `.git` directory exactly
  - `cache*` → matches `cache`, `cache-old`, `cached`

**Default behavior**:
- If `files` config is not present, no files are excluded
- If `exclude_patterns` is empty, no files are excluded
- Excluded files/folders are skipped during tree building
- Children of excluded directories are not scanned

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

---

# Architecture Guidelines

This section defines the architecture and code organization principles for SpoutMC.

## Layer Separation

### API Layer (`internal/webserver/api/**`)

**Purpose:** Thin API handlers ONLY

API handlers should:
- Parse HTTP requests
- Validate input parameters
- Call business logic from `/internal` packages
- Return HTTP responses
- Handle HTTP-specific concerns (status codes, headers, etc.)

API handlers should **NOT**:
- Implement business logic
- Contain algorithms or complex operations
- Directly manipulate data structures
- Perform file operations
- Execute Docker operations

**Example:**
```go
// ✅ Good - Thin API handler
func addServerHandler(c echo.Context) error {
    var req AddServerRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
    }

    // Call business logic from internal package
    if err := serverpkg.ValidateProxyConstraint(); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
    }

    // More business logic calls...
    newServer := serverpkg.CreateServer(req)

    return c.JSON(http.StatusCreated, newServer)
}
```

```go
// ❌ Bad - Business logic in API handler
func addServerHandler(c echo.Context) error {
    var req AddServerRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
    }

    // ❌ Business logic should not be here
    existingConfig := config.All()
    usedPorts := make(map[int]bool)
    for _, server := range existingConfig.Servers {
        for _, portMapping := range server.Ports {
            var port int
            fmt.Sscanf(portMapping.HostPort, "%d", &port)
            usedPorts[port] = true
        }
    }
    // ... more business logic
}
```

### Business Logic Layer (`/internal`)

**Purpose:** All implementation and business logic

The `/internal` directory contains all business logic, organized by **logical domains**.

#### Package Organization Principles

1. **Group by domain**, not by function
2. **Logical cohesion** - Related functionality stays together
3. **Reusability** - Can be called from multiple API endpoints
4. **Single responsibility** - Each package has one clear purpose

#### Examples of Good Package Structure

```
/internal/server/          # Server lifecycle & management
  - lifecycle.go           # Start, stop, restart, create
  - validation.go          # Proxy/lobby constraints, input validation
  - ports.go              # Port assignment & availability checking
  - environment.go        # Environment variable handling & merging

/internal/servercfg/       # Configuration management
  - operations.go         # High-level add/update/remove operations
  - gitops.go            # Git-specific operations (commit, push)
  - local.go             # Local file operations (write YAML)

/internal/files/           # File system operations
  - tree.go              # Tree building & traversal
  - filters.go           # Pattern matching & exclusions
  - operations.go        # Read, write, backup operations

/internal/sse/             # Server-Sent Events utilities
  - event.go             # SSE event structure and marshaling

/internal/docker/          # Docker operations
  - docker.go            # Container lifecycle
  - network.go           # Network management
  - velocity.go          # Velocity proxy configuration
  - mapper.go            # Model to Docker API mapping
```

#### Examples of Bad Package Structure

```
❌ Too granular (function-per-package):
/internal/validateproxy/
/internal/findport/
/internal/mergeenv/
/internal/checklobby/

❌ Too broad (everything in one place):
/internal/utils/
  - everything.go        # 5000 lines of unrelated code

❌ Wrong layer (business logic in API):
/internal/webserver/api/v1/server/
  - server.go            # 2456 lines with business logic mixed in
```

## Refactored Package Structure

### `/internal/sse`
Server-Sent Events utilities for real-time data streaming.
- Event structure
- SSE marshaling

### `/internal/files`
File system operations for server file management.
- File tree building with exclusion patterns
- Recursive directory traversal
- Pattern matching

### `/internal/container`
**Shared container action functions** (DRY principle).
- `StartContainer()` - Starts container and includes in watchdog
- `StopContainer()` - Excludes from watchdog before stopping
- `RestartContainer()` - Restarts and ensures watchdog inclusion
- Used by both server and infrastructure API endpoints

### `/internal/server`
Server lifecycle and business logic.
- Server type determination
- Port assignment and availability
- Environment variable management
- Proxy/lobby constraint validation
- Default configuration generation

### `/internal/servercfg`
Configuration persistence (GitOps and local).
- Add/update/remove servers in Git repositories
- Add/update/remove servers in local YAML files
- Configuration file management

### `/internal/infrastructure`
Infrastructure container management (databases, etc.).
- Password generation and management
- Database container creation
- Configuration loading (Git and local)

### `/internal/docker`
Docker container and network operations.
- Container lifecycle (create, start, stop, remove)
- Network management
- Velocity proxy configuration
- Paper server configuration (paper-global.yml)
- Image pulling and management

### `/internal/config`
Configuration loading and state management.
- YAML configuration parsing
- In-memory configuration state
- Configuration diffing and hot-reload

### `/internal/git`
Git repository operations for GitOps.
- Repository cloning and pulling
- Configuration file reading from Git
- Commit and push operations
- Webhook handling

## Design Patterns

### Dependency Flow

```
API Layer (webserver/api/**)
    ↓ calls
Business Logic (/internal/server, /internal/servercfg, etc.)
    ↓ calls
Core Services (/internal/docker, /internal/config, /internal/git)
```

### Error Handling

- Business logic returns errors
- API handlers convert errors to HTTP responses
- Use structured logging (zap) throughout

### Configuration

- Configuration is loaded once and cached in memory
- Hot-reload triggers re-read from disk/git
- Business logic reads from `config.All()`

## Adding New Features

When adding new functionality:

1. **Identify the domain** - Where does this belong logically?
2. **Check existing packages** - Can this fit in an existing package?
3. **Create new package if needed** - Only if it's a new logical domain
4. **Keep API handlers thin** - All logic goes to `/internal`
5. **Write reusable functions** - Think about other use cases

### Example: Adding Server Backup Feature

```
❌ Bad approach:
- Create /internal/backup/
- Create /internal/compress/
- Create /internal/upload/
- Implement logic in API handler

✅ Good approach:
- Add to existing /internal/server/ package:
  - backup.go (backup operations, compression, upload)
- API handler just calls: serverpkg.BackupServer(id)
```

## Testing

- Test business logic in `/internal` packages independently
- Mock dependencies where needed
- API handlers should have minimal logic to test

## Future Considerations

- Keep packages focused on single responsibilities
- Avoid circular dependencies between packages
- Use interfaces for testability and flexibility
- Document public functions and complex logic
