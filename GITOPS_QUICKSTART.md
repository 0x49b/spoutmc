# GitOps Quick Start Guide

This guide will help you quickly set up and test GitOps configuration for SpoutMC.

## Prerequisites

- Git installed
- GitHub or GitLab account
- SpoutMC installed and configured

## Step 1: Create a Git Repository for Server Configs

### Option A: Using GitHub

1. Create a new repository on GitHub (e.g., `spoutmc-servers`)
2. Clone it locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/spoutmc-servers.git
   cd spoutmc-servers
   ```

### Option B: Using GitLab

1. Create a new project on GitLab
2. Clone it locally:
   ```bash
   git clone https://gitlab.com/YOUR-USERNAME/spoutmc-servers.git
   cd spoutmc-servers
   ```

## Step 2: Create Server Configuration Files

Create a directory structure and add server configs:

```bash
mkdir servers
```

Create `servers/proxy.yaml`:
```yaml
name: spoutproxy
image: itzg/mc-proxy
proxy: true
ports:
  - hostPort: '25565'
    containerPort: '25565'
env:
  TYPE: VELOCITY
  MAX_MEMORY: 1G
volumes:
  - hostpath:
      - testservers
      - data
      - spoutproxy
    containerpath: "/server"
```

Create `servers/lobby.yaml`:
```yaml
name: lobby
image: itzg/minecraft-server
lobby: true
env:
  EULA: "TRUE"
  TYPE: PAPER
  VERSION: 1.21.10
  MAX_MEMORY: 4G
  ONLINE_MODE: 'FALSE'
volumes:
  - hostpath:
      - testservers
      - data
      - lobby
    containerpath: "/data"
```

## Step 3: Push to Git

```bash
git add servers/
git commit -m "Initial server configurations"
git push origin main
```

## Step 4: Configure SpoutMC

### For Public Repository

Edit `config/spoutmc.yaml`:

```yaml
git:
  enabled: true
  repository: "https://github.com/YOUR-USERNAME/spoutmc-servers.git"
  branch: "main"
  poll_interval: 30s
  local_path: "/tmp/spoutmc-git"

servers: []  # Empty when using GitOps
```

### For Private Repository

1. Create a Personal Access Token:
   - **GitHub:** Settings → Developer settings → Personal access tokens → Generate new token (classic)
   - **GitLab:** Settings → Access Tokens → Add new token
   - Scope: `repo` (read) or `read_repository`

2. Set environment variable:
   ```bash
   export SPOUTMC_GIT_TOKEN="your-token-here"
   ```

3. Edit `config/spoutmc.yaml`:
   ```yaml
   git:
     enabled: true
     repository: "https://github.com/YOUR-USERNAME/spoutmc-servers.git"
     branch: "main"
     token: "${SPOUTMC_GIT_TOKEN}"
     poll_interval: 30s
     local_path: "/tmp/spoutmc-git"

   servers: []
   ```

## Step 5: Start SpoutMC

```bash
./spoutmc
```

You should see in the logs:
```
⚔️ starting: gitSync
GitOps is enabled, initializing Git sync
Cloning Git repository repository=https://github.com/...
Repository cloned successfully commit=abc1234
Successfully loaded server configurations from Git count=2
Git poller started interval=30s
```

## Step 6: Test Auto-Update

### Test 1: Add a New Server

1. Create `servers/skyblock.yaml` in your Git repo:
   ```yaml
   name: skyblock
   image: itzg/minecraft-server
   env:
     EULA: "TRUE"
     TYPE: PAPER
     VERSION: 1.21.10
     MAX_MEMORY: 8G
   volumes:
     - hostpath:
         - testservers
         - data
         - skyblock
       containerpath: "/data"
   ```

2. Commit and push:
   ```bash
   git add servers/skyblock.yaml
   git commit -m "Add skyblock server"
   git push
   ```

3. Wait up to 30 seconds (or trigger webhook)

4. Check SpoutMC logs:
   ```
   Changes detected in Git repository, reloading configuration
   ⛏️ Running skyblock (abc123def4) with itzg/minecraft-server
   ```

### Test 2: Modify a Server

1. Edit `servers/lobby.yaml` - change MAX_MEMORY:
   ```yaml
   MAX_MEMORY: 6G  # Changed from 4G
   ```

2. Commit and push:
   ```bash
   git add servers/lobby.yaml
   git commit -m "Increase lobby memory to 6G"
   git push
   ```

3. SpoutMC will recreate the lobby container with new settings

### Test 3: Remove a Server

1. Delete `servers/skyblock.yaml`:
   ```bash
   git rm servers/skyblock.yaml
   git commit -m "Remove skyblock server"
   git push
   ```

2. SpoutMC will stop and remove the skyblock container

## Step 7: Enable Webhooks (Optional)

### For GitHub:

1. Generate a webhook secret:
   ```bash
   export SPOUTMC_WEBHOOK_SECRET=$(openssl rand -hex 32)
   echo "Save this secret: $SPOUTMC_WEBHOOK_SECRET"
   ```

2. Update `config/spoutmc.yaml`:
   ```yaml
   git:
     webhook_secret: "${SPOUTMC_WEBHOOK_SECRET}"
   ```

3. Restart SpoutMC

4. Go to GitHub repo → Settings → Webhooks → Add webhook:
   - Payload URL: `http://YOUR-HOST:3000/api/v1/git/webhook`
   - Content type: `application/json`
   - Secret: Your webhook secret
   - Events: Just the push event

5. Test by pushing a change - it should update immediately!

### For GitLab:

1. Set webhook secret (same as above)

2. Go to GitLab repo → Settings → Webhooks:
   - URL: `http://YOUR-HOST:3000/api/v1/git/webhook`
   - Secret Token: Your webhook secret
   - Trigger: Push events

## Testing Manual Sync

You can manually trigger a sync via API:

```bash
curl -X POST http://localhost:3000/api/v1/git/sync
```

Response:
```json
{
  "status": "success",
  "message": "Configuration synced successfully"
}
```

## Verification Commands

```bash
# Check running containers
docker ps --filter "label=io.spout.network=true"

# Check Git repository
ls -la /tmp/spoutmc-git/servers/

# View SpoutMC logs
tail -f logs/spoutmc.log

# Check webhook deliveries (GitHub)
# Go to: Settings → Webhooks → Recent Deliveries
```

## Troubleshooting

### GitOps not starting?
```bash
# Check config syntax
cat config/spoutmc.yaml | grep -A 10 "git:"

# Verify environment variable
echo $SPOUTMC_GIT_TOKEN
```

### Can't clone repository?
```bash
# Test Git credentials manually
git clone https://${SPOUTMC_GIT_TOKEN}@github.com/YOUR-USERNAME/spoutmc-servers.git /tmp/test
```

### Changes not applying?
```bash
# Check polling is working
grep "Polling Git repository" logs/spoutmc.log

# Force manual sync
curl -X POST http://localhost:3000/api/v1/git/sync
```

## Next Steps

- Set up separate staging/production branches
- Add CI/CD validation for YAML files
- Configure monitoring for webhook failures
- Document your server configurations in the Git repo README

## Example Repository

See `examples/gitops-repo/` in the SpoutMC repository for a complete example.
