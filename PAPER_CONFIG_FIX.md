# Paper Configuration Fix for Velocity Forwarding

## The Problem

Paper servers need their `paper-global.yml` file configured with:
```yaml
proxies:
  velocity:
    enabled: true
    online-mode: true
    secret: "<same secret as Velocity>"
```

The file exists at `/data/config/paper-global.yml` inside the container after Paper generates it on first startup. We needed to modify the existing file, not create a new one.

## The Solution

### Approach: Modify After First Start

1. Let Paper generate its default `paper-global.yml` on first startup
2. Watchdog detects the file exists
3. Modify the `proxies.velocity` section with correct settings
4. Container continues running with updated config

### Implementation

**New File: `internal/docker/paper.go`**

Contains three functions:

1. **`EnsurePaperVelocityConfig(serverDataPath, velocitySecret)`**
   - Checks if `paper-global.yml` exists
   - Reads and parses the existing YAML file
   - Modifies only the `proxies.velocity` section
   - Preserves all other Paper settings
   - Creates a backup before modifying
   - Writes the updated file back

2. **`CheckAndConfigurePaperServers(dataPath, velocitySecret)`**
   - Scans all server directories in data path
   - Skips proxy servers
   - Finds Paper servers (those with `/data/config` directory)
   - Calls `EnsurePaperVelocityConfig` for each Paper server
   - Logs configuration status

**Modified: `internal/watchdog/watchdog.go`**

Added `ensurePaperVelocityConfig()` to the watchdog's check cycle:
- Runs every 15 seconds (same as container health checks)
- Gets the Velocity secret
- Calls `CheckAndConfigurePaperServers()`
- Silently succeeds if files don't exist yet (first start)
- Configures Paper servers as soon as `paper-global.yml` appears

## How It Works

### Timeline

**First Server Start:**
1. Container starts
2. Paper generates default `paper-global.yml`
3. Watchdog tick (within 15 seconds)
4. Detects `paper-global.yml` exists
5. Modifies file with Velocity settings
6. Paper reads updated config
7. Velocity forwarding works!

**Subsequent Starts:**
1. Watchdog checks if config is correct
2. If `proxies.velocity.enabled=true` and secret matches → no action
3. If config is wrong or missing → updates it

### File Locations

**On Host:**
```
{dataPath}/
  {serverName}/
    data/
      config/
        paper-global.yml        ← Modified by SpoutMC
        paper-global.yml.backup ← Backup of original
```

**In Container:**
```
/data/
  config/
    paper-global.yml ← Paper reads from here
```

## Verification Steps

### 1. Check Logs

Look for these messages in SpoutMC logs:
```
Configuring paper-global.yml with Velocity settings path=.../lobby/data/config/paper-global.yml
✅ Successfully configured paper-global.yml for Velocity path=... velocity_enabled=true
Configured Paper servers for Velocity forwarding count=2
```

### 2. Check File on Host

```bash
cat testservers/data/lobby/data/config/paper-global.yml | grep -A 4 "velocity:"
```

Expected output:
```yaml
velocity:
  enabled: true
  online-mode: true
  secret: <base64-secret-string>
```

### 3. Check File in Container

```bash
docker exec lobby cat /data/config/paper-global.yml | grep -A 4 "velocity:"
```

Should match the host file.

### 4. Test Connection

```bash
# Connect to proxy
# Player should successfully join lobby without errors
```

Velocity logs should show:
```
[connected player] username -> Lobby has connected
```

Paper logs should show:
```
username joined the game
```

No more "did not send a forwarding request" errors!

## Technical Details

### YAML Preservation

The code uses `gopkg.in/yaml.v3` to:
- Parse the existing YAML file generically
- Preserve ALL existing Paper settings
- Only modify the `proxies.velocity` section
- Maintain proper YAML formatting

### Backup Strategy

Before modifying any file:
- Creates `paper-global.yml.backup` with original content
- Allows manual rollback if needed
- Doesn't overwrite existing backups

### Error Handling

- If file doesn't exist yet → silently skip (will retry on next watchdog tick)
- If file can't be read → log error, don't crash
- If YAML parsing fails → log error, don't modify file
- If write fails → log error, backup is safe

### Watchdog Integration

- Runs every 15 seconds (same as container health checks)
- Low overhead (only checks if file exists, skips if already configured)
- Automatically handles new servers added to the network
- No manual intervention required

## Advantages of This Approach

✅ **No custom Docker images** - Uses official itzg images
✅ **No volume mount complexity** - Modifies files directly on host
✅ **Preserves Paper defaults** - Only changes what's needed
✅ **Automatic recovery** - Watchdog fixes config if it gets reset
✅ **Works with restarts** - Config persists across container restarts
✅ **Handles timing** - Doesn't matter when Paper generates the file

## Related Files

- `internal/docker/paper.go` - Paper configuration functions
- `internal/docker/velocity.go` - Velocity secret generation
- `internal/watchdog/watchdog.go` - Automated configuration check
- `internal/docker/docker.go` - Velocity secret mounting for proxy

## Sources

- [PaperMC: Configuring Player Information Forwarding](https://docs.papermc.io/velocity/player-information-forwarding/)
- [PaperMC Global Configuration](https://docs.papermc.io/paper/reference/global-configuration/)
- [itzg/docker-minecraft-server Documentation](https://docker-minecraft-server.readthedocs.io/)
