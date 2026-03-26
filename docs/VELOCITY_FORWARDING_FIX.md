# Velocity Player Forwarding Fix

## What Was Fixed

There were TWO critical issues preventing Velocity player forwarding:

1. **Velocity proxy never received the forwarding secret file**
2. **Paper servers' `paper-global.yml` was not configured with Velocity settings**

### Changes Made

#### 1. Added Forwarding Secret Volume Mount (`internal/docker/docker.go`)
- Proxy containers now mount `forwarding.secret` from host to `/config/forwarding.secret` (read-only)
- The itzg/mc-proxy image copies files from `/config` to `/server` on startup
- This ensures Velocity has access to the forwarding secret

#### 2. Added paper-global.yml Configuration (`internal/docker/paper.go` + `docker.go`)
- **NEW FILE**: `internal/docker/paper.go` with `CreatePaperGlobalConfig()` function
- Generates `paper-global.yml` with proper Velocity configuration:
  - `proxies.velocity.enabled: true`
  - `proxies.velocity.online-mode: true`
  - `proxies.velocity.secret: <actual secret>`
- Paper servers mount this file to `/config/config/paper-global.yml`
- The itzg/minecraft-server image syncs it to `/data/config/paper-global.yml`

#### 3. Added Debug Logging (`internal/server/lifecycle.go`)
- Paper server environment variable setup now logs the secret preview
- Helps verify that the same secret is being used across all containers
- Only logs first 8 characters for security

## How It Works

### Secret Generation Flow

1. **Startup**: `CreateOrUpdateVelocityToml()` is called
2. **Secret Creation**: `ensureForwardingSecret()` generates or reads existing secret
3. **Secret Caching**: Secret is cached in memory for reuse
4. **Proxy Starts**: Velocity container mounts the secret file via volume
5. **Paper Starts**: Paper servers receive secret via `CFG_VELOCITY_SECRET` environment variable

### Configuration Details

**Velocity Proxy (`itzg/mc-proxy`):**
- Volume mount: `{dataPath}/{proxyName}/server/forwarding.secret` → `/config/forwarding.secret:ro`
- velocity.toml settings:
  - `player-info-forwarding-mode = "modern"`
  - `forwarding-secret-file = "forwarding.secret"`
  - `online-mode = true`
- SpoutMC player chat ingest also uses this same secret via `X-Spout-Chat-Ingest`
  - Backend reads from `{dataPath}/{proxyName}/server/forwarding.secret`
  - Velocity players bridge resolves `forwarding-secret-file` from `velocity.toml`

**Paper Servers (`itzg/minecraft-server`):**
- Environment variables:
  - `ONLINE_MODE=FALSE`
  - `REPLACE_ENV_VARIABLES=TRUE`
  - `ENV_VARIABLE_PREFIX=CFG_`
  - `CFG_VELOCITY_ENABLED=true`
  - `CFG_VELOCITY_ONLINE_MODE=true`
  - `CFG_VELOCITY_SECRET=<base64-encoded-secret>`

## Verification Steps

### 1. Check Secret File on Host
```bash
cat /path/to/data/spoutproxy/server/forwarding.secret
```
Expected: Base64-encoded string (44 characters for 32 bytes)

### 2. Check Secret in Velocity Container
```bash
docker exec spoutproxy cat /server/forwarding.secret
```
Expected: Same base64 string as host file

### 3. Check velocity.toml
```bash
docker exec spoutproxy cat /server/velocity.toml | grep -E "forwarding|online-mode"
```
Expected output:
```toml
online-mode = true
force-key-authentication = false
player-info-forwarding-mode = "modern"
forwarding-secret-file = "forwarding.secret"
```

### 4. Check Paper Server Environment
```bash
docker exec lobby env | grep CFG_VELOCITY
```
Expected output:
```
CFG_VELOCITY_ENABLED=true
CFG_VELOCITY_ONLINE_MODE=true
CFG_VELOCITY_SECRET=<same-base64-string>
```

### 5. Check Paper Configuration
```bash
docker exec lobby cat /data/config/paper-global.yml | grep -A 4 "velocity:"
```
Expected output:
```yaml
velocity:
  enabled: true
  online-mode: true
  secret: <same-base64-string>
```

### 6. Check Logs for Secret Preview
Look for log messages in SpoutMC output:
```
Configuring Paper server with Velocity forwarding secret_preview=AbCdEfGh... secret_length=44
Mounting Velocity forwarding secret source=/data/spoutproxy/server/forwarding.secret target=/config/forwarding.secret
```

### 7. Test Player Connection
1. Start all servers
2. Connect to proxy: `localhost:25565`
3. Should successfully authenticate and join lobby
4. Check Velocity logs for "modern forwarding" confirmation
5. Check Paper logs - should NOT show "Unable to forward player"

## Troubleshooting

### Issue: "Unable to forward player to lobby"

**Cause**: Secret mismatch between Velocity and Paper

**Solution**:
1. Verify both containers have the same secret (steps 2 and 4 above)
2. Restart all containers to ensure they pick up the latest secret
3. Check logs for secret preview - should match across all servers

### Issue: "forwarding.secret: No such file or directory"

**Cause**: Secret file not created before Velocity starts

**Solution**:
1. Check startup order in `cmd/spoutmc/main.go`
2. Ensure `CreateOrUpdateVelocityToml()` runs before `startProxyContainer()`
3. Verify file exists on host before starting proxy

### Issue: "Read-only file system"

**Cause**: Mounting directly to `/server/forwarding.secret` instead of `/config`

**Solution**:
- The fix uses `/config/forwarding.secret` mount
- itzg/mc-proxy copies files from `/config` to `/server` on startup
- This avoids read-only filesystem errors

### Issue: Players kicked with "Invalid signature"

**Cause**: Paper server has `ONLINE_MODE=TRUE` or wrong secret

**Solution**:
1. Ensure Paper has `ONLINE_MODE=FALSE` (proxy handles auth)
2. Verify `CFG_VELOCITY_SECRET` matches Velocity's secret
3. Check `paper-global.yml` has correct velocity settings

## Technical Notes

### Why `/config` instead of `/server`?

The itzg/mc-proxy Docker image:
- Copies files from `/config` to `/server` on startup
- This allows read-only mounts to `/config` without permission errors
- Standard pattern for this image (see official documentation)

### Secret Generation

The secret is:
- 32 random bytes
- Base64-encoded (results in 44 characters)
- Generated once and cached in memory
- Reused across all Paper servers
- Written to `{dataPath}/{proxyName}/server/forwarding.secret`

### Docker Network

All containers must be on the same Docker network (`spoutnetwork`):
- Velocity connects to Paper using container names as hostnames
- Example: `lobby:25565`, `game1:25565`
- Port 25565 is the internal container port, not host port
- Already configured in your velocity.toml

## Related Documentation

- [PaperMC: Configuring Player Information Forwarding](https://docs.papermc.io/velocity/player-information-forwarding/)
- [itzg/docker-mc-proxy GitHub](https://github.com/itzg/docker-mc-proxy)
- [itzg/docker-minecraft-server Docs](https://docker-minecraft-server.readthedocs.io/)
- [PaperMC Forums: Docker Velocity Setup](https://forums.papermc.io/threads/docker-compose-setup-velocity-proxy-refuses-to-connect-to-paper-hub-world-on-same-machine.493/)
