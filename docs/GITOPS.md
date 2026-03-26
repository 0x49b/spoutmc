# GitOps Configuration for SpoutMC

SpoutMC now supports GitOps-style configuration management, allowing you to manage your server fleet using Git repositories with individual YAML files per server.

## Overview

When GitOps is enabled, SpoutMC will:
1. Clone a Git repository containing server configurations
2. Read server manifests from the repository (each representing a server)
3. Create an in-memory configuration
4. Monitor the repository for changes (via polling and/or webhooks)
5. Automatically apply changes when the repository is updated

## Configuration

### Enable GitOps in spoutmc.yaml

Add a `git` section to your `config/spoutmc.yaml`:

```yaml
git:
  enabled: true
  repository: "https://github.com/your-org/spoutmc-servers.git"
  branch: "main"
  token: "${SPOUTMC_GIT_TOKEN}"
  poll_interval: 30s
  webhook_secret: "${SPOUTMC_WEBHOOK_SECRET}"
  local_path: "/tmp/spoutmc-git"

# The servers section is ignored when GitOps is enabled
servers: []
```

### Configuration Fields

| Field | Required | Description | Default |
|-------|----------|-------------|---------|
| `enabled` | Yes | Enable/disable GitOps mode | `false` |
| `repository` | Yes | Git repository URL (HTTPS) | - |
| `branch` | No | Git branch to track | `main` |
| `token` | No | Personal Access Token for private repos | - |
| `poll_interval` | No | How often to check for changes | `30s` |
| `webhook_secret` | No | Secret for webhook signature verification | - |
| `local_path` | No | Local directory for git clone | `/tmp/spoutmc-git` |

### Environment Variables

Use environment variables for sensitive data:

```bash
export SPOUTMC_GIT_TOKEN="ghp_xxxxxxxxxxxx"
export SPOUTMC_WEBHOOK_SECRET="your-secret-key"
```

Then reference them in the config:
```yaml
git:
  token: "${SPOUTMC_GIT_TOKEN}"
  webhook_secret: "${SPOUTMC_WEBHOOK_SECRET}"
```

## Git Repository Structure

Create a Git repository with the following structure:

```
spoutmc-servers/
├── servers/
│   ├── proxy.yaml
│   ├── lobby.yaml
│   ├── skyblock.yaml
│   └── bedwars.yaml
├── .gitignore
└── README.md
```

**Important:** SpoutMC looks for server files in `servers/` and infrastructure files in `infrastructure/`.
Legacy flat YAML files are still supported, but new setups should use manifest format.

### Example Server Configuration

`servers/lobby.yaml`:
```yaml
apiVersion: spoutmc.io/v1alpha1
kind: SpoutServer
metadata:
  name: lobby
spec:
  name: lobby
  image: itzg/minecraft-server
  lobby: true
  restartPolicy:
    container:
      policy: unless-stopped
    autoStartOnSpoutmcStart: true
  env:
    EULA: "TRUE"
    TYPE: PAPER
    VERSION: 1.21.10
    MAX_MEMORY: 4G
  volumes:
    - containerpath: "/data"
```

### Restart Policy Fields

`spec.restartPolicy` controls two independent restart behaviors:

- Docker restart policy (`spec.restartPolicy.container`)
- SpoutMC startup behavior (`spec.restartPolicy.autoStartOnSpoutmcStart`)

Example:

```yaml
spec:
  name: skyblock
  image: itzg/minecraft-server
  restartPolicy:
    container:
      policy: on-failure
      maxRetries: 3
    autoStartOnSpoutmcStart: true
```

Docker policy values:

- `no`
- `on-failure` (supports optional `maxRetries`, must be >= 1 when provided)
- `always`
- `unless-stopped`
- omitted `container`/`policy`: Docker default behavior (`no` restart policy)

Startup behavior:

- `autoStartOnSpoutmcStart: true` (or omitted): server is started/recreated during SpoutMC startup.
- `autoStartOnSpoutmcStart: false`: server is skipped during SpoutMC startup.

### Infrastructure

SpoutMC uses SQLite for its database. MySQL/MariaDB infrastructure containers are no longer supported.

## Authentication

### Public Repositories

For public repositories, no authentication is needed:

```yaml
git:
  enabled: true
  repository: "https://github.com/public-org/spoutmc-servers.git"
```

### Private Repositories (Personal Access Token)

1. Create a Personal Access Token (PAT) in your Git provider:
   - **GitHub:** Settings → Developer settings → Personal access tokens → Tokens (classic)
   - **GitLab:** Settings → Access Tokens
   - Required permissions: `repo` (read)

2. Set the token as an environment variable:
   ```bash
   export SPOUTMC_GIT_TOKEN="ghp_xxxxxxxxxxxx"
   ```

3. Configure SpoutMC:
   ```yaml
   git:
     token: "${SPOUTMC_GIT_TOKEN}"
   ```

## Change Detection Methods

SpoutMC supports two methods for detecting Git repository changes:

### 1. Polling (Automatic)

SpoutMC periodically checks the repository for new commits:

```yaml
git:
  poll_interval: 30s  # Check every 30 seconds
```

**Pros:** Simple, no external configuration needed
**Cons:** Delayed updates (up to poll_interval)

### 2. Webhooks (Real-time)

Configure your Git provider to send webhooks on push events:

#### GitHub Webhook Setup

1. Navigate to repository → Settings → Webhooks → Add webhook
2. Configure:
   - **Payload URL:** `http://your-spoutmc-host:3000/api/v1/git/webhook`
   - **Content type:** `application/json`
   - **Secret:** Your webhook secret (same as `SPOUTMC_WEBHOOK_SECRET`)
   - **Events:** Just the push event
   - **Active:** ✓ (checked)
3. Click "Add webhook"

#### GitLab Webhook Setup

1. Navigate to repository → Settings → Webhooks
2. Configure:
   - **URL:** `http://your-spoutmc-host:3000/api/v1/git/webhook`
   - **Secret Token:** Your webhook secret
   - **Trigger:** Push events ✓
   - **SSL verification:** Enable if using HTTPS
3. Click "Add webhook"

**Pros:** Instant updates when you push
**Cons:** Requires external network access to SpoutMC

### Combined Approach (Recommended)

Enable both for best results:

```yaml
git:
  poll_interval: 60s  # Fallback polling every minute
  webhook_secret: "${SPOUTMC_WEBHOOK_SECRET}"  # Real-time updates
```

## How Changes Are Applied

When SpoutMC detects changes in the Git repository:

1. **Added servers** (new YAML files):
   - Pulls Docker image
   - Creates container
   - Starts container

2. **Modified servers** (changed YAML files):
   - Stops existing container
   - Removes old container (volumes preserved)
   - Creates new container with updated config
   - Starts new container

3. **Removed servers** (deleted YAML files):
   - Stops container
   - Removes container
   - Removes volumes

## API Endpoints

### Manual Sync

Trigger a manual Git sync:

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

### Webhook Endpoint

Receives webhooks from Git providers:

```
POST /api/v1/git/webhook
```

This endpoint is automatically configured and secured with HMAC-SHA256 signature verification.

## Troubleshooting

### GitOps Not Starting

Check logs for:
```
GitOps is enabled, initializing Git sync
```

If you see "GitOps is disabled", verify:
- `git.enabled: true` in config
- Config file is being read correctly

### Authentication Failures

```
failed to clone repository: authentication required
```

Solutions:
- Verify token is correct
- Check token permissions (needs read access)
- Ensure token environment variable is set
- For GitHub, use Personal Access Token (classic), not fine-grained

### Webhook Not Triggering

1. Check webhook delivery in Git provider UI
2. Verify webhook secret matches configuration
3. Test with manual sync: `curl -X POST http://localhost:3000/api/v1/git/sync`
4. Check SpoutMC logs for webhook errors

### Invalid Server Configuration

```
Failed to parse YAML file as SpoutServer, skipping
```

SpoutMC will skip invalid files and continue. Check:
- `apiVersion` is set for manifest files
- `kind` is `SpoutServer` or `InfrastructureContainer`
- `metadata.name` matches `spec.name` (if both are set)
- `spec.image` is set
- YAML syntax is correct

### No Servers Loaded

```
no valid server configurations found in repository
```

Verify:
- Repository contains `.yaml` or `.yml` files in `servers/`
- Files have valid server manifests/configuration

## Migration from File-Based Config

### Step 1: Create Git Repository

1. Create a new Git repository
2. Create a `servers/` directory
3. Convert your existing `spoutmc.yaml` servers section

**From:**
```yaml
servers:
  - name: lobby
    image: itzg/minecraft-server
    env:
      EULA: "TRUE"
```

**To:** `servers/lobby.yaml`
```yaml
apiVersion: spoutmc.io/v1alpha1
kind: SpoutServer
metadata:
  name: lobby
spec:
  name: lobby
  image: itzg/minecraft-server
  env:
    EULA: "TRUE"
```

### Step 2: Update spoutmc.yaml

```yaml
git:
  enabled: true
  repository: "https://github.com/your-org/spoutmc-servers.git"
  branch: "main"
  token: "${SPOUTMC_GIT_TOKEN}"
  poll_interval: 30s

servers: []  # Empty when GitOps enabled
```

### Step 3: Restart SpoutMC

SpoutMC will:
1. Clone the Git repository
2. Load configurations from Git
3. Recreate containers with Git-based configs

## Best Practices

1. **Use branches for testing:**
   ```yaml
   git:
     branch: "staging"  # Test changes before merging to main
   ```

2. **Keep sensitive data in environment variables:**
   - Don't commit tokens to Git
   - Use `.env` files locally
   - Use secrets management in production

3. **Validate YAML before committing:**
   ```bash
   yamllint servers/*.yaml
   ```

4. **Use descriptive commit messages:**
   ```
   git commit -m "Add new skyblock server with 8GB RAM"
   ```

5. **Review changes in PR/MR:**
   - Peer review server configurations
   - Test in staging branch first

6. **Monitor SpoutMC logs after changes:**
   ```bash
   tail -f logs/spoutmc.log
   ```

## Security Considerations

1. **Private repositories:** Always use private repositories for server configurations
2. **Token permissions:** Use read-only tokens with minimal scope
3. **Webhook secrets:** Use strong, random webhook secrets
4. **Network security:** Restrict webhook endpoint access via firewall
5. **HTTPS:** Use HTTPS for webhook URLs in production

## Examples

See the `../examples/gitops-repo/` directory for:
- Example server configurations
- Repository structure
- README template

## Troubleshooting Commands

```bash
# Check if GitOps is enabled
curl http://localhost:3000/api/v1/ping

# Manually trigger sync
curl -X POST http://localhost:3000/api/v1/git/sync

# View last commit
ls -la /tmp/spoutmc-git/.git

# Check loaded configuration
docker ps --filter "label=io.spout.network=true"
```
