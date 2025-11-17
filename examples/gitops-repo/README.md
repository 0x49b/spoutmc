# SpoutMC GitOps Server Configurations

This repository contains server configurations for SpoutMC in a GitOps style.

## Structure

```
servers/
├── proxy.yaml      # Velocity proxy server
├── lobby.yaml      # Lobby server
├── skyblock.yaml   # Skyblock game server
└── ...             # Add more servers as needed
```

## Server Configuration Format

Each YAML file in the `servers/` directory represents a single Minecraft server configuration:

```yaml
name: servername           # Unique server name (required)
image: itzg/minecraft-server  # Docker image (required)
proxy: false               # Is this a proxy server? (optional)
lobby: false               # Is this a lobby server? (optional)

# Port mappings (optional)
ports:
  - hostPort: '25565'
    containerPort: '25565'

# Environment variables (optional)
env:
  EULA: "TRUE"
  TYPE: PAPER
  VERSION: 1.21.10
  MAX_MEMORY: 4G

# Volume mappings (optional)
volumes:
  - hostpath:
      - path
      - to
      - host
    containerpath: "/data"
```

## Making Changes

1. Add a new YAML file to `servers/` directory
2. Commit and push to the repository
3. SpoutMC will automatically:
   - Pull the changes (via polling or webhook)
   - Create new containers for added servers
   - Update existing containers if configuration changed
   - Remove containers for deleted server files

## Webhook Configuration

To enable real-time updates, configure a webhook in your Git provider:

### GitHub
1. Go to repository Settings → Webhooks → Add webhook
2. Payload URL: `http://your-spoutmc-host:3000/api/v1/git/webhook`
3. Content type: `application/json`
4. Secret: Same as `SPOUTMC_WEBHOOK_SECRET` environment variable
5. Events: Just the push event

### GitLab
1. Go to repository Settings → Webhooks → Add webhook
2. URL: `http://your-spoutmc-host:3000/api/v1/git/webhook`
3. Secret Token: Same as `SPOUTMC_WEBHOOK_SECRET` environment variable
4. Trigger: Push events

## Manual Sync

You can manually trigger a sync using the API:

```bash
curl -X POST http://your-spoutmc-host:3000/api/v1/git/sync
```
