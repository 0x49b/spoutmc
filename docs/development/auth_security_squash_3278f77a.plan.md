---
name: Auth Security Squash
overview: Aggressively consolidate authentication, authorization, and permissions into one cohesive access-control area by merging `internal/auth`, `internal/authz`, `internal/security`, and `internal/permissions`, then rewiring middleware/guards/API callsites with compatibility-safe migration phases.
todos:
  - id: create-access-package
    content: Create `internal/access` and migrate JWT, password, authorization checks, effective permission, and permissions registry/seed/db helpers into it.
    status: completed
  - id: rewire-security-imports
    content: Update middleware, guards, API handlers, and storage bootstrap to import and use `internal/access` only.
    status: completed
  - id: preserve-security-semantics
    content: Verify DB-backed admin checks, claims-based plugin checks, and permission expansion behavior remain unchanged.
    status: completed
  - id: remove-legacy-security-packages
    content: Delete or fully deprecate old `internal/auth`, `internal/authz`, `internal/security`, and `internal/permissions` package files once unused.
    status: completed
  - id: validate-auth-security-refactor
    content: Run lint/compile/tests and confirm zero legacy imports plus unchanged API auth behavior.
    status: completed
isProject: false
---

# Squash Auth/Security Into One Access-Control Unit

## Goal

Merge fragmented security logic into a single cohesive module so ownership is obvious and cross-package hops disappear.

Current targets to merge:

- `[/Users/florianthievent/workspace/private/spoutmc/internal/auth/jwt.go](/Users/florianthievent/workspace/private/spoutmc/internal/auth/jwt.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/authz/check.go](/Users/florianthievent/workspace/private/spoutmc/internal/authz/check.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/authz/effective.go](/Users/florianthievent/workspace/private/spoutmc/internal/authz/effective.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/authz/userresponse.go](/Users/florianthievent/workspace/private/spoutmc/internal/authz/userresponse.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/security/password.go](/Users/florianthievent/workspace/private/spoutmc/internal/security/password.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/permissions/registry.go](/Users/florianthievent/workspace/private/spoutmc/internal/permissions/registry.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/permissions/db.go](/Users/florianthievent/workspace/private/spoutmc/internal/permissions/db.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/permissions/roleseed.go](/Users/florianthievent/workspace/private/spoutmc/internal/permissions/roleseed.go)`

## Target Package Shape

Create one package namespace: `internal/access` with clear files by concern:

- `jwt.go` (claims + token issue/verify)
- `password.go` (hash/verify)
- `checks.go` (DB and claims authorization checks)
- `effective.go` (effective permission resolution)
- `permissions_registry.go` (default permission definitions)
- `permissions_seed.go` (role-permission seed mapping)
- `permissions_db.go` (all-keys DB helpers)
- `userresponse.go` (user response projection, unless moved later)

## Architecture (Post-merge)

```mermaid
flowchart LR
  apiHandlers[ApiHandlers]
  webAuth[JwtMiddlewareAndGuards]
  accessCore[AccessPackage]
  dbLayer[StorageAndGorm]

  apiHandlers --> webAuth
  apiHandlers --> accessCore
  webAuth --> accessCore
  accessCore --> dbLayer
```



## Implementation Phases

### Phase 1: Add `internal/access` as compatibility layer

- Introduce `internal/access` with copied/adapted implementations from current auth/authz/security/permissions.
- Keep exported function names as close as possible (e.g. `GenerateToken`, `VerifyToken`, `ClaimsHasPermission`, `UserHasRole`, `Hash`, `Verify`, `Definitions`, `RolePermissionKeys`, `AllKeysFromDB`).
- Do not delete old packages yet.

### Phase 2: Rewire all imports to `internal/access`

Update all direct consumers, especially:

- `[/Users/florianthievent/workspace/private/spoutmc/internal/webserver/middleware/jwt.go](/Users/florianthievent/workspace/private/spoutmc/internal/webserver/middleware/jwt.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/webserver/guards/guards.go](/Users/florianthievent/workspace/private/spoutmc/internal/webserver/guards/guards.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/auth/auth.go](/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/auth/auth.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/user/user.go](/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/user/user.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/plugin/plugin.go](/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/plugin/plugin.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/setup/setup.go](/Users/florianthievent/workspace/private/spoutmc/internal/webserver/api/v1/setup/setup.go)`
- `[/Users/florianthievent/workspace/private/spoutmc/internal/storage/db.go](/Users/florianthievent/workspace/private/spoutmc/internal/storage/db.go)`

### Phase 3: Remove legacy package usage and preserve behavior

- Ensure there are zero imports of `internal/auth`, `internal/authz`, `internal/security`, `internal/permissions` from non-compat files.
- Preserve semantics:
  - JWT claims behavior (including stale-claims caveat in claim-based checks)
  - `RequireAdmin` remains DB-backed role check
  - admin “all permissions” still resolved from DB keys
  - setup/login/user password flows unchanged

### Phase 4: Delete legacy package files/directories

- Delete old files once no references remain.
- Optionally keep short-lived shims only if needed for staged rollout; otherwise remove in same PR for full squash.

## Risk Controls

- Avoid behavior changes while merging; this is a package-boundary refactor first.
- Keep constant names/strings stable (`admin`, `manager`, `plugins.manage`).
- Watch for import cycles (`access` must not import webserver API packages).

## Validation Checklist

- No remaining imports of `internal/auth`, `internal/authz`, `internal/security`, `internal/permissions` in active code.
- `internal/webserver/middleware/jwt.go` and `internal/webserver/guards/guards.go` compile and preserve auth behavior.
- Auth/login/setup/user/plugin/permission routes compile with unchanged contracts.
- Lints clean on touched files.
- `go test ./...` executed to baseline limits; document existing unrelated failures if present.

## Expected Result

A single `internal/access` ownership boundary for authn + authz + permissions, significantly lower cognitive load, and simpler future changes to security policy.