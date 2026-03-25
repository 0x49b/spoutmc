# Development Guide

This document describes the development workflow for SpoutMC.

## Overview

SpoutMC uses a **separate frontend and backend** development setup for fast iteration:
- **Backend**: Go with hot-reload via Air (optional)
- **Frontend**: React with Vite dev server (hot module replacement)

This separation allows you to:
- ✅ Edit frontend code and see changes instantly (HMR)
- ✅ Edit backend code and restart quickly
- ✅ Use browser DevTools for frontend debugging
- ✅ Avoid rebuild/recompile cycles during development

## Prerequisites

### Required

- **Go 1.24.0+** - [Download](https://go.dev/dl/)
- **Node.js 20+** - [Download](https://nodejs.org/)
- **npm** - Comes with Node.js
- **Docker** - For running Minecraft servers
- **Git** - Version control

### Recommended

- **Air** - Live reload for Go applications
  ```bash
  go install github.com/air-verse/air@latest
  ```

### IDE Recommendations

- **GoLand** / **IntelliJ IDEA Ultimate** - Full Go and React support
- **VS Code** - Install extensions:
  - Go (golang.go)
  - React (dsznajder.es7-react-js-snippets)
  - TypeScript (ms-vscode.vscode-typescript-next)
  - Tailwind CSS IntelliSense

## Project Structure

```
spoutmc/
├── cmd/
│   └── spoutmc/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   ├── docker/                  # Docker API operations
│   ├── git/                     # GitOps functionality
│   ├── infrastructure/          # Infrastructure containers
│   ├── server/                  # Server lifecycle
│   ├── watchdog/                # Health monitoring
│   └── webserver/
│       ├── api/                 # API handlers
│       │   └── v1/
│       └── static/              # Embedded frontend (production only)
├── web/                         # React frontend
│   ├── src/
│   │   ├── components/          # React components
│   │   ├── service/             # API service layer
│   │   ├── store/               # Zustand state management
│   │   ├── types/               # TypeScript types
│   │   └── main.tsx             # React entry point
│   ├── package.json
│   └── vite.config.ts
├── config/
│   └── spoutmc.yaml             # Main configuration
├── build.sh                     # Unix wrapper for production build
├── build.mjs                    # Cross-platform production build logic
├── build.ps1                    # PowerShell build wrapper
├── build.cmd                    # Windows CMD build wrapper
└── docs/                        # Project documentation
    ├── BUILD.md                 # Build instructions
    ├── DEVELOPMENT.md           # This file
    ├── GITOPS.md                # GitOps reference
    └── AGENTS.md                # Architecture guidelines
```

## Development Workflow

### 1. Clone the Repository

```bash
git clone <repository-url>
cd spoutmc
```

### 2. Install Dependencies

**Backend dependencies:**
```bash
go mod download
```

**Frontend dependencies:**
```bash
cd web
npm install
cd ..
```

### 3. Set Up Configuration

Copy the example configuration:
```bash
cp config/spoutmc.example.yaml config/spoutmc.yaml
```

Edit `config/spoutmc.yaml` with your settings:
```yaml
storage:
  data_path: "/path/to/your/data"

servers:
  - name: lobby
    image: itzg/minecraft-server:latest
    lobby: true
    # ... more configuration
```

### 4. Start Development Servers

You'll run **two processes** in separate terminals:

#### Terminal 1: Backend Server

**Option A: Without live reload**
```bash
go run ./cmd/spoutmc
```

**Option B: With live reload (recommended)**
```bash
air
```

The backend will start on **http://localhost:3000**.

You should see:
```
⚠️ Frontend assets not embedded, running in API-only mode
🤵🏻‍♂️ webserver started on http://localhost:3000
```

**Note:** The warning is expected in development mode. The frontend runs separately.

#### Terminal 2: Frontend Dev Server

```bash
cd web
npm run dev
```

The frontend will start on **http://localhost:5173** with hot module replacement (HMR).

You should see:
```
VITE v5.x.x  ready in xxx ms

➜  Local:   http://localhost:5173/
➜  Network: use --host to expose
```

### 5. Open the Application

Open your browser to **http://localhost:5173** (Vite dev server).

The frontend will make API calls to **http://localhost:3000** (backend) via CORS.

## Making Changes

### Frontend Development

**Edit React components:**
```bash
cd web/src
# Edit any file in components/, pages/, etc.
```

**Changes are reflected immediately** thanks to Vite's HMR.

**Common frontend tasks:**

| Task | Command | Location |
|------|---------|----------|
| Start dev server | `npm run dev` | `web/` |
| Run linter | `npm run lint` | `web/` |
| Build for production | `npm run build` | `web/` |
| Preview production build | `npm run preview` | `web/` |

**Frontend stack:**
- React 19
- TypeScript
- Vite (build tool)
- TailwindCSS (styling)
- React Router v7 (routing)
- Zustand (state management)
- Axios (API calls)

### Backend Development

**Edit Go code:**
```bash
# Edit any file in internal/ or cmd/
```

**Without Air:** Restart manually with `Ctrl+C` and `go run ./cmd/spoutmc`

**With Air:** Changes trigger automatic rebuild and restart.

**Common backend tasks:**

| Task | Command | Location |
|------|---------|----------|
| Run server | `go run ./cmd/spoutmc` | Project root |
| Run with live reload | `air` | Project root |
| Run tests | `go test ./...` | Project root |
| Format code | `go fmt ./...` | Project root |
| Run linter | `golangci-lint run` | Project root |

**Backend stack:**
- Go 1.24
- Echo (web framework)
- Docker SDK (container management)
- Zap (logging)
- Viper (configuration)
- go-git (GitOps)

## API Development

### Adding a New API Endpoint

1. **Create handler** in `internal/webserver/api/v1/<domain>/`:
   ```go
   // internal/webserver/api/v1/example/example.go
   package example

   import (
       "github.com/labstack/echo/v4"
       "net/http"
   )

   func GetExample(c echo.Context) error {
       return c.JSON(http.StatusOK, map[string]string{
           "message": "Hello from example API",
       })
   }
   ```

2. **Register route** in the appropriate API file:
   ```go
   // internal/webserver/api/v1/example/routes.go
   package example

   import "github.com/labstack/echo/v4"

   func RegisterRoutes(g *echo.Group) {
       g.GET("/example", GetExample)
   }
   ```

3. **Call from API registration** in `internal/webserver/api/api.go`:
   ```go
   v1 := api.Group("/v1")
   example.RegisterRoutes(v1.Group("/example"))
   ```

### Testing API Endpoints

**Using curl:**
```bash
# GET request
curl http://localhost:3000/api/v1/server

# POST request
curl -X POST http://localhost:3000/api/v1/server \
  -H "Content-Type: application/json" \
  -d '{"name":"server1","image":"itzg/minecraft-server:latest"}'
```

## Frontend Development

### Adding a New Component

1. **Create component** in `web/src/components/<domain>/`:
   ```tsx
   // web/src/components/Example/ExampleComponent.tsx
   import React from 'react';

   export const ExampleComponent: React.FC = () => {
       return (
           <div className="p-4">
               <h1 className="text-2xl font-bold">Example Component</h1>
           </div>
       );
   };
   ```

2. **Add route** in `web/src/App.tsx`:
   ```tsx
   import { ExampleComponent } from './components/Example/ExampleComponent';

   const router = createBrowserRouter([
       // ... existing routes
       {
           path: "/example",
           element: <ExampleComponent />
       }
   ]);
   ```

### Making API Calls

**Create service function** in `web/src/service/`:
```typescript
// web/src/service/exampleService.ts
import api from './api';

export const exampleService = {
    getAll: async () => {
        const response = await api.get('/example');
        return response.data;
    },

    create: async (data: ExampleData) => {
        const response = await api.post('/example', data);
        return response.data;
    }
};
```

**Use in component:**
```tsx
import { exampleService } from '../../service/exampleService';

export const ExampleComponent: React.FC = () => {
    const [data, setData] = React.useState([]);

    React.useEffect(() => {
        exampleService.getAll().then(setData);
    }, []);

    return (
        <div>
            {data.map(item => <div key={item.id}>{item.name}</div>)}
        </div>
    );
};
```

### State Management with Zustand

**Create store** in `web/src/store/`:
```typescript
// web/src/store/exampleStore.ts
import { create } from 'zustand';

interface ExampleState {
    items: any[];
    setItems: (items: any[]) => void;
    addItem: (item: any) => void;
}

export const useExampleStore = create<ExampleState>((set) => ({
    items: [],
    setItems: (items) => set({ items }),
    addItem: (item) => set((state) => ({
        items: [...state.items, item]
    })),
}));
```

**Use in component:**
```tsx
import { useExampleStore } from '../../store/exampleStore';

export const ExampleComponent: React.FC = () => {
    const { items, addItem } = useExampleStore();

    return (
        <div>
            {items.map(item => <div key={item.id}>{item.name}</div>)}
            <button onClick={() => addItem({ id: 1, name: 'New' })}>
                Add Item
            </button>
        </div>
    );
};
```

## Architecture Guidelines

**IMPORTANT:** Follow the architecture guidelines in `AGENTS.md`.

### Key Principles

1. **API handlers stay thin** - No business logic in `internal/webserver/api/`
2. **Business logic in `/internal` packages** - Organized by domain
3. **Reusable functions** - Extract common operations
4. **Avoid over-engineering** - Only add complexity when needed
5. **Don't commit files unless necessary** - Prefer editing existing files

### Package Organization

```
/internal/
├── server/          # Server lifecycle & management
├── servercfg/       # Configuration persistence
├── docker/          # Docker operations
├── config/          # Configuration loading
├── infrastructure/  # Infrastructure containers
├── files/           # File system operations
└── webserver/
    └── api/         # Thin API handlers only
```

**Example - Good architecture:**
```go
// ✅ Thin API handler
func addServerHandler(c echo.Context) error {
    var req AddServerRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }

    // Call business logic
    server, err := serverpkg.CreateServer(req)
    if err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }

    return c.JSON(201, server)
}

// ✅ Business logic in separate package
// internal/server/lifecycle.go
func CreateServer(req AddServerRequest) (*Server, error) {
    // Validation, port assignment, env var merging, etc.
    // All business logic here
}
```

## Testing

### Backend Tests

**Run all tests:**
```bash
go test ./...
```

**Run specific package:**
```bash
go test ./internal/server
```

**Run with coverage:**
```bash
go test -cover ./...
```

**Test example:**
```go
// internal/server/validation_test.go
package server

import "testing"

func TestValidateServerName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid name", "server1", false},
        {"empty name", "", true},
        {"invalid chars", "server@123", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateServerName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateServerName() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Frontend Tests

Currently, no frontend tests are configured. To add testing:

**Install testing dependencies:**
```bash
cd web
npm install -D @testing-library/react @testing-library/jest-dom vitest
```

**Configure Vitest** in `vite.config.ts`:
```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
    plugins: [react()],
    test: {
        globals: true,
        environment: 'jsdom',
        setupFiles: './src/test/setup.ts',
    },
});
```

**Run tests:**
```bash
npm run test
```

## Debugging

### Backend Debugging

**Using Delve (Go debugger):**
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Start with debugger
dlv debug ./cmd/spoutmc
```

**In GoLand/VS Code:**
- Set breakpoints in your IDE
- Use "Debug" run configuration
- Step through code, inspect variables

**Logging:**
```go
import "go.uber.org/zap"

logger.Debug("Debug message", zap.String("key", "value"))
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message", zap.Error(err))
```

### Frontend Debugging

**Browser DevTools:**
- Open DevTools (F12)
- **Console tab** - View logs, errors
- **Network tab** - Inspect API calls
- **React DevTools** - Inspect component state/props
- **Sources tab** - Set breakpoints in TypeScript code

**Logging:**
```typescript
console.log("Debug:", data);
console.warn("Warning:", message);
console.error("Error:", error);
```

**React DevTools Extension:**
- Chrome: [React Developer Tools](https://chrome.google.com/webstore/detail/react-developer-tools/fmkadmapgofadopljbjfkapdkoienihi)
- Firefox: [React Developer Tools](https://addons.mozilla.org/en-US/firefox/addon/react-devtools/)

## Common Development Tasks

### Format Code

**Go:**
```bash
go fmt ./...
goimports -w .
```

**TypeScript/React:**
```bash
cd web
npm run lint
```

### Update Dependencies

**Go modules:**
```bash
go get -u ./...
go mod tidy
```

**npm packages:**
```bash
cd web
npm update
npm audit fix
```

### Clean Build Artifacts

```bash
# Remove built binaries
rm -rf build/

# Remove embedded frontend
rm -rf internal/webserver/static/dist/

# Remove frontend build
rm -rf web/dist/

# Remove node_modules (rare)
rm -rf web/node_modules/
```

## Environment Variables

### Backend

The backend doesn't require environment variables for development, but you can set:

```bash
# Optional: Set log level
export LOG_LEVEL=debug

# Optional: Set environment mode
export SPOUTMC_ENV=development
```

### Frontend

Vite uses `.env` files for environment variables:

**Create `web/.env.development`:**
```env
VITE_API_URL=http://localhost:3000
```

**Access in code:**
```typescript
const apiUrl = import.meta.env.VITE_API_URL;
```

## Git Workflow

### Branch Strategy

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Make changes and commit
git add .
git commit -m "Add: description of changes"

# Push to remote
git push origin feature/your-feature-name

# Create pull request on GitHub/GitLab
```

### Commit Message Convention

```
Type: Brief description

Detailed explanation (optional)

Examples:
- Add: new server creation API
- Fix: SPA routing issue on refresh
- Update: improve error handling in docker package
- Refactor: extract validation logic to separate package
- Docs: update development guide
```

## Production Build

When you're ready to create a production build:

```bash
./build.sh
```

See `BUILD.md` for detailed build instructions.

## Troubleshooting

### Port 3000 Already in Use

```bash
# Find process
lsof -i :3000

# Kill process
kill -9 <PID>
```

### Port 5173 Already in Use

```bash
# Find process
lsof -i :5173

# Kill process or change Vite port in vite.config.ts
```

### CORS Errors in Browser

The backend CORS configuration should allow `localhost:5173`. If you see CORS errors:

1. Check `internal/webserver/webserver.go` line 34:
   ```go
   AllowOrigins: []string{"http://localhost:3000", "http://localhost:5173"}
   ```

2. Restart the backend server

### Frontend Can't Connect to API

1. Verify backend is running on port 3000
2. Check frontend API service configuration in `web/src/service/api.ts`
3. Open browser DevTools Network tab and check for failed requests

### Air Not Working

```bash
# Install/reinstall Air
go install github.com/air-verse/air@latest

# Verify installation
which air

# Create default config if missing
air init
```

### Docker Containers Not Starting

1. Verify Docker is running:
   ```bash
   docker ps
   ```

2. Check Docker network exists:
   ```bash
   docker network ls | grep spoutnetwork
   ```

3. View container logs:
   ```bash
   docker logs <container-name>
   ```

## Player Moderation (UUID) and Velocity Bridge Workflow

This feature persists player moderation/chat data using the canonical Minecraft UUID (stable across gamertag changes).

### 1) Required `spoutmc.yaml` additions

Add the predefined timed-ban options for the frontend ban dropdown:

```yaml
player-bans:
  ban-durations:
    - key: "1h"
      label: "1 hour"
      duration: "1h"
    - key: "5h"
      label: "5 hours"
      duration: "5h"
    - key: "1d"
      label: "1 day"
      duration: "24h"
    - key: "2d"
      label: "2 days"
      duration: "48h"
    - key: "2w"
      label: "2 weeks"
      duration: "336h"
```

If `player-bans` / `ban-durations` is missing, the backend falls back to these same defaults.

### 2) Updating the Velocity players bridge

SpoutMC downloads the Velocity bridge JAR from the compile-time system registry in:

- `internal/plugins/system.go`

Whenever you change the bridge behavior that the backend depends on (for example: UUID in `/players` snapshots, UUID-keyed ban checks, and the optional `playerUuid` field in chat ingest), you must:

1. Build a new bridge JAR from source (`plugins/velocity-players-bridge`).
2. Publish/upload that JAR to the URL referenced by `internal/plugins/system.go`.
3. Recreate (restart) the affected containers so the updated JAR is downloaded.

Notes:
- SpoutMC only injects/updates plugin `PLUGINS` on container creation (or recreate). For running containers, you need a recreate to pick up a new bridge JAR URL.
- The URL in `internal/plugins/system.go` is versioned and tied to SpoutMC releases; keep it in sync with the bridge changes you ship.

## Additional Resources

- **Build Instructions**: `BUILD.md`
- **Architecture Guidelines**: `AGENTS.md`
- **GitOps Documentation**: `GITOPS.md`
- **Release Pipeline**: `RELEASE.md`
- **Go Documentation**: https://go.dev/doc/
- **Echo Framework**: https://echo.labstack.com/
- **React Documentation**: https://react.dev/
- **Vite Documentation**: https://vitejs.dev/

## Getting Help

- **Project Issues**: https://github.com/your-repo/spoutmc/issues
- **Go Community**: https://golang.org/help/
- **React Community**: https://react.dev/community
- **Docker Documentation**: https://docs.docker.com/

## Contributing

1. Fork the repository
2. Create a feature branch
3. Follow the architecture guidelines in `AGENTS.md`
4. Write tests for new functionality
5. Ensure all tests pass
6. Submit a pull request with a clear description

Happy coding! 🚀
