---
name: Docker Container Merge
overview: Consolidate `internal/container` into `internal/docker` as the single runtime package, then clean up adjacent Docker clutter (`model`, `image`, and `container_logs.go`) while preserving watchdog-aware lifecycle behavior and existing API contracts.
todos:
  - id: add-docker-watchdog-actions
    content: Port `internal/container` watchdog-aware lifecycle logic into `internal/docker` with explicit watchdog-aware start/stop/restart functions.
    status: completed
  - id: rewire-container-callsites
    content: Replace `internal/container` imports/usages in API handlers with new `internal/docker` watchdog-aware lifecycle functions.
    status: completed
  - id: remove-container-package
    content: Delete `internal/container` package after confirming no remaining references.
    status: completed
  - id: cleanup-docker-strays
    content: Remove or relocate unused `internal/docker/model`, `internal/docker/image`, and `internal/docker/container_logs.go` based on actual usage.
    status: completed
  - id: verify-runtime-merge
    content: Run lint/compile checks and validate lifecycle behavior parity, documenting any unrelated baseline failures.
    status: completed
isProject: false
---

# Merge `internal/container` Into `internal/docker`

## Goal

Make `internal/docker` the single canonical runtime package by folding the thin lifecycle wrapper from `internal/container` into it, then clean up adjacent Docker package clutter that increases navigation noise.

## Scope Confirmed

- Keep `internal/docker` as canonical package.
- Merge `internal/container` behavior into `internal/docker`.
- Also clean up:
  - unused/stray `internal/docker/model`
  - unused/stray `internal/docker/image`
  - `internal/docker/container_logs.go` layout issue

## Current High-Impact Callers

Primary lifecycle wrapper callers:

- `[/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/server/server.go](/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/server/server.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/infrastructure/infrastructure.go](/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/infrastructure/infrastructure.go)`

Watchdog integration source today:

- `[/Users/florianthievent/workspace/private/spoutmc/internal/container/actions.go](/Users/florianthievent/workspace/private/spoutmc/internal/container/actions.go)`

## Target Design

- Add watchdog-aware ID lifecycle functions directly in `internal/docker` (or `internal/docker/watchdog_actions.go`):
  - `StartContainerWithWatchdog(ctx, containerID)`
  - `StopContainerWithWatchdog(ctx, containerID)`
  - `RestartContainerWithWatchdog(ctx, containerID)`
- Keep existing low-level ID actions (`StartContainerById`, `StopContainerById`, `RestartContainerById`) unchanged.
- Update API handlers to call the new `docker` watchdog-aware functions directly.
- Remove `internal/container` package after migration.

## `model` / `image` / `container_logs` Cleanup

1. Validate no active imports for:
  - `internal/docker/model/*`
  - `internal/docker/image/*`
2. If unused, remove these files/directories.
3. Move `container_logs.go` out of library package path to avoid mixed concerns:
  - either relocate to dedicated command path (e.g. `cmd/container-logs/main.go`) if still needed,
  - or remove if dead and replaced by existing realtime/log streaming paths.

## Migration Steps

1. Implement watchdog-aware lifecycle helpers in `internal/docker` by porting logic from `internal/container/actions.go` (including `global.Watchdog` include/exclude semantics).
2. Update imports and callsites in server/infrastructure APIs from `containerpkg.*` to `docker.*` watchdog-aware actions.
3. Remove `internal/container/actions.go`.
4. Remove or relocate `docker/model`, `docker/image`, and `docker/container_logs.go` according to actual usage.
5. Run formatting and lint/build validation on touched runtime and API files.

## Validation Checklist

- No remaining imports of `spoutmc/internal/container`.
- Start/stop/restart API behavior unchanged (including watchdog include/exclude semantics).
- No import-cycle or package-main/package-library conflicts in `internal/docker`.
- Lints clean on touched files.
- Targeted compile/tests pass to baseline limits, documenting unrelated pre-existing failures.

## Expected Result

A single runtime ownership boundary under `internal/docker`, fewer package hops for lifecycle behavior, and lower maintenance overhead from removing dead/stray Docker subpackages.