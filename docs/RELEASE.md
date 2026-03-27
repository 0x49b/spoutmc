# Release Pipeline

This document describes the GitHub release workflow in `.github/workflows/release.yaml`.

## Trigger Modes

- Tag push: any tag matching `v*` (for example `v0.2.0`).
- Manual run (`workflow_dispatch`): requires a `tag` input.

## Build Order

The workflow runs jobs in this order:

1. `prepare` - resolves the release tag and semantic version.
2. `frontend` - installs frontend dependencies and builds the Vite app.
3. `plugins` - builds plugin jars using Gradle (`shadowJar`).
4. `go` - cross-compiles Go binaries using the resolved version.
5. `release` - collects all artifacts and publishes a GitHub Release.

## Versioning Rules

- Tag value is resolved from either:
  - pushed tag (`github.ref_name`), or
  - manual input (`github.event.inputs.tag`).
- Binary version passed to Go is tag without `v` prefix.
  - Example: `v1.4.3` -> `main.Version=1.4.3`.

## Produced Artifacts

### Go binaries

- `spoutmc-linux-amd64`
- `spoutmc-linux-arm64`
- `spoutmc-darwin-amd64`
- `spoutmc-darwin-arm64`
- `spoutmc-windows-amd64.exe`
- plus one `.sha256` file for each binary

### Plugin artifacts

- `velocity-players-bridge-*.jar`
- plus one `.sha256` file for each jar

All artifacts are attached to the same GitHub release tag.

## Important Notes

- Frontend is built once and reused by all Go matrix jobs via workflow artifacts.
- Plugin build currently targets `plugins/velocity-players-bridge`.
- Go builds run with `CGO_ENABLED=0`.
- Release publication uses `softprops/action-gh-release`.

## Runtime Self-Update (v1)

SpoutMC can check GitHub Releases at runtime and install new binaries in place.

### Required configuration

- `SPOUTMC_UPDATE_REPO`: GitHub repository in `owner/repo` format.
  - Example: `SPOUTMC_UPDATE_REPO=your-org/spoutmc`
- Optional for private repositories: `SPOUTMC_GITHUB_TOKEN` (token with `contents:read`).
- Optional scheduler override: `SPOUTMC_UPDATE_CHECK_INTERVAL` as Go duration.
  - Default is `24h` (once per day).
  - Example: `SPOUTMC_UPDATE_CHECK_INTERVAL=12h`

### Runtime behavior

- On startup, SpoutMC performs an initial release check shortly after boot.
- A recurring scheduler checks GitHub Releases every 24h by default.
- If a newer release is detected, admin users see an update banner in the UI.
- Starting an update downloads:
  - binary asset matching current OS/arch (`spoutmc-<os>-<arch>[.exe]`)
  - corresponding checksum file (`.sha256`)
- SpoutMC verifies SHA256 checksum before replacing the executable.

### Restart and supervision expectations

- After a successful install, SpoutMC requests graceful process termination.
- Production deployments should run SpoutMC under a supervisor that restarts it automatically, for example:
  - `systemd`
  - Docker restart policy
  - `launchd` (macOS)
- During restart, managed server network connectivity is interrupted and players may disconnect.

### Rollback path

- The previous executable is kept as a backup at `<spoutmc-binary>.bak`.
- If needed, stop SpoutMC and restore manually:
  - move backup file back to the original binary path
  - start SpoutMC again via your supervisor/service manager
