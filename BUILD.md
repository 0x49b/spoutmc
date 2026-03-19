# Build Instructions

This document describes how to build SpoutMC for production deployment.

## Overview

SpoutMC uses a single cross-platform build script (`build.mjs`) with thin platform wrappers:
- Unix/macOS/Linux: `build.sh`
- Windows PowerShell: `build.ps1`
- Windows CMD: `build.cmd`

The build process:
1. Builds the React frontend with Vite
2. Embeds the frontend into the Go binary using Go's native `embed` package
3. Compiles multi-architecture Go binaries for production deployment

The result is a **single, standalone binary** for each platform that contains both the backend and frontend.

## Prerequisites

### Required

- **Go 1.24.0+** - [Download](https://go.dev/dl/)
- **Node.js 20+** - [Download](https://nodejs.org/)
- **npm** - Comes with Node.js
- **No Bash required on Windows** - Native wrappers are provided for PowerShell/CMD

### Optional

- **swag** - For generating Swagger API documentation
  ```bash
  go install github.com/swaggo/swag/cmd/swag@latest
  ```

## Quick Start

### 1. Make the Unix build wrapper executable (first time only)

```bash
chmod +x build.sh
```

### 2. Run the build

```bash
./build.sh
```

Or on Windows:

```powershell
.\build.ps1
```

```cmd
build.cmd
```

That's it! The script handles everything automatically.

## Build Process

The build script performs the following steps:

### Step 1: Clean Previous Builds
- Removes `build/` directory
- Removes `internal/webserver/static/dist/` directory

### Step 2: Install Frontend Dependencies
- Checks if `web/node_modules/` exists
- Runs `npm install` if dependencies are missing
- Skips if already installed

### Step 3: Build Frontend
- Runs `npm run build` in `web/` directory
- Produces optimized production build in `web/dist/`
- Validates that build succeeded

### Step 4: Copy Frontend to Embed Location
- Copies `web/dist/*` to `internal/webserver/static/dist/`
- This directory is embedded into the Go binary via `//go:embed` directive

### Step 5: Generate Swagger Documentation (Optional)
- Runs `swag init` if `swag` command is available
- Skips if `swag` is not installed (optional)

### Step 6: Build Go Binaries
- Compiles Go binary for each target architecture:
  - `linux/amd64` - Linux servers
  - `darwin/amd64` - macOS Intel
  - `darwin/arm64` - macOS Apple Silicon
  - `windows/amd64` - Windows
- Uses `-ldflags="-s -w"` to strip debug symbols (smaller binaries)
- Embeds version number: `-X main.Version=0.0.1`

## Build Output

After a successful build, you'll find binaries in the `build/` directory:

```
build/
├── spoutmc-linux-amd64         (~25-30MB)
├── spoutmc-darwin-amd64        (~28-33MB)
├── spoutmc-darwin-arm64        (~26-31MB)
└── spoutmc-windows-amd64.exe   (~27-32MB)
```

**Note:** Binary sizes include the embedded frontend (~2-5MB) plus Go runtime and dependencies.

## Running the Binary

### Linux

```bash
./build/spoutmc-linux-amd64
```

### macOS (Intel)

```bash
./build/spoutmc-darwin-amd64
```

### macOS (Apple Silicon)

```bash
./build/spoutmc-darwin-arm64
```

### Windows

```powershell
.\build\spoutmc-windows-amd64.exe
```

## Verifying the Build

After starting the binary, you should see:

```
🎨 Serving embedded frontend from binary
🤵🏻‍♂️ webserver started on http://localhost:3000
```

Then open your browser to:
- **Frontend**: http://localhost:3000/
- **API**: http://localhost:3000/api/v1/
- **Swagger**: http://localhost:3000/swagger/index.html

## Configuration

The binary requires a configuration file to run:

```bash
config/spoutmc.yaml  # Must be present in the working directory
```

**Minimal configuration:**

```yaml
storage:
  data_path: "/path/to/data"

servers:
  - name: lobby
    image: itzg/minecraft-server:latest
    lobby: true
    ports:
      - hostPort: '25565'
        containerPort: '25565'
    env:
      TYPE: PAPER
      VERSION: "1.21.10"
      EULA: "TRUE"
    volumes:
      - containerpath: "/data"
```

See `config/spoutmc.yaml` for full configuration examples.

## Build Targets

The build script supports the following architectures by default:

| Platform | GOOS    | GOARCH | Binary Name                   |
|----------|---------|--------|-------------------------------|
| Linux    | linux   | amd64  | spoutmc-linux-amd64           |
| macOS    | darwin  | amd64  | spoutmc-darwin-amd64          |
| macOS    | darwin  | arm64  | spoutmc-darwin-arm64          |
| Windows  | windows | amd64  | spoutmc-windows-amd64.exe     |

### Customizing Build Targets

Edit the `TARGETS` array in `build.mjs`:

```bash
TARGETS=(
    "linux/amd64"
    "linux/arm64"     # Add ARM64 Linux
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)
```

## Troubleshooting

### Build Script Fails: "Frontend build failed - dist directory not found"

**Cause:** `npm run build` failed to create `web/dist/`

**Solution:**
```bash
cd web
npm install
npm run build
```

Check for TypeScript errors or missing dependencies.

### Build Script Fails: Permission denied

**Cause:** Build script is not executable

**Solution:**
```bash
chmod +x build.sh
```

### Binary Shows: "Frontend assets not embedded, running in API-only mode"

**Cause:** Go binary was built without running `build.sh`

**Solution:** Always use the build wrapper (`./build.sh`, `.\build.ps1`, or `build.cmd`) instead of `go build` directly. The build script ensures the frontend is embedded before Go compilation.

### Binary Fails to Start: "failed to bind to port"

**Cause:** Port 3000 is already in use

**Solution:**
```bash
# Find process using port 3000
lsof -i :3000

# Kill the process or stop the conflicting service
```

### Frontend Loads but API Calls Fail

**Cause:** API routes may be conflicting with catch-all route

**Solution:** This should not happen due to Echo's route priority, but verify:
1. Check browser DevTools Network tab
2. Ensure API calls go to `/api/v1/*` paths
3. Check server logs for route registration

### Swagger Documentation Not Generated

**Cause:** `swag` command not installed

**Solution (Optional):**
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

This is optional - the build will succeed without Swagger docs.

## Advanced Build Options

### Building for a Single Architecture

To build only for Linux (for example):

```bash
# Build frontend first
cd web
npm run build
cd ..

# Copy to embed location
mkdir -p internal/webserver/static/dist
cp -r web/dist/* internal/webserver/static/dist/

# Build Go binary
GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o spoutmc-linux-amd64 \
    ./cmd/spoutmc
```

### Development Build (No Optimization)

For faster builds during development (without stripping debug symbols):

```bash
go build -o spoutmc ./cmd/spoutmc
```

**Note:** This still requires frontend to be embedded. For active development, use the development workflow instead (see `DEVELOPMENT.md`).

### Custom Version Number

Edit `build.mjs` and change:

```bash
VERSION="0.0.1"  # Change to your version
```

The version can be displayed in the application (if implemented in `main.go`).

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build SpoutMC

on:
  push:
    branches: [ master ]
  release:
    types: [ created ]

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.24.0'

    - name: Set up Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '20'

    - name: Install swag
      run: go install github.com/swaggo/swag/cmd/swag@latest

    - name: Build binaries
      run: |
        chmod +x build.sh
        ./build.sh

    - name: Upload artifacts
      uses: actions/upload-artifact@v4
      with:
        name: spoutmc-binaries
        path: build/*
```

### GitLab CI Example

```yaml
build:
  image: golang:1.24
  before_script:
    - apt-get update && apt-get install -y nodejs npm
    - go install github.com/swaggo/swag/cmd/swag@latest
  script:
    - chmod +x build.sh
    - ./build.sh
  artifacts:
    paths:
      - build/*
```

## Deployment

### Single Server Deployment

1. Copy the appropriate binary to your server:
   ```bash
   scp build/spoutmc-linux-amd64 user@server:/opt/spoutmc/
   ```

2. Copy the configuration file:
   ```bash
   scp config/spoutmc.yaml user@server:/opt/spoutmc/config/
   ```

3. Run the binary:
   ```bash
   ssh user@server
   cd /opt/spoutmc
   ./spoutmc-linux-amd64
   ```

### Docker Deployment

If using Docker, the build script output can be used in a Docker image:

```dockerfile
FROM alpine:latest
WORKDIR /app
COPY build/spoutmc-linux-amd64 ./spoutmc
COPY config/ ./config/
RUN chmod +x spoutmc
CMD ["./spoutmc"]
```

### Systemd Service (Linux)

Create `/etc/systemd/system/spoutmc.service`:

```ini
[Unit]
Description=SpoutMC Minecraft Server Manager
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
User=spoutmc
WorkingDirectory=/opt/spoutmc
ExecStart=/opt/spoutmc/spoutmc-linux-amd64
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable spoutmc
sudo systemctl start spoutmc
```

## Additional Resources

- **Development Workflow**: See `DEVELOPMENT.md`
- **Project Architecture**: See `CLAUDE.md`
- **GitOps Configuration**: See `GITOPS.md`
- **API Documentation**: http://localhost:3000/swagger/ (when running)

## Support

If you encounter issues:
1. Check the troubleshooting section above
2. Review logs from the binary
3. Report issues at: https://github.com/your-repo/spoutmc/issues
