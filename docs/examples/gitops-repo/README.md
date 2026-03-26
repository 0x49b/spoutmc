# SpoutMC GitOps Example Repository

This directory is an example GitOps repository layout for SpoutMC.

## Structure

```text
servers/
├── proxy.yaml               # Velocity proxy server (SpoutServer manifest)
├── lobby.yaml               # Lobby server (SpoutServer manifest)
└── skyblock.yaml            # Game server (SpoutServer manifest)

infrastructure/              # SpoutMC uses SQLite; no database containers
└── README.md
```

## Manifest Format

SpoutMC supports:

- Manifest format (`apiVersion`, `kind`, `metadata`, `spec`) - recommended.
- Legacy flat YAML objects (still supported for compatibility).

### SpoutServer Example

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
    VERSION: "1.21.10"
  volumes:
    - containerpath: "/data"
```

### InfrastructureContainer Example

SpoutMC uses SQLite for its database. MySQL/MariaDB infrastructure containers are no longer supported.

## Notes

- In server volumes, use `containerpath` only.
- Host paths are generated automatically from SpoutMC `storage.data_path`.
- `metadata.name` should match `spec.name`.
- `spec.restartPolicy.container.policy` supports Docker values: `no`, `on-failure`, `always`, `unless-stopped`.
- `spec.restartPolicy.container.maxRetries` is only valid when policy is `on-failure`.
- `spec.restartPolicy.autoStartOnSpoutmcStart` defaults to `true` when omitted.
- When `spec.restartPolicy.container.policy` is omitted, Docker uses its default (`no` restart policy).

## Applying Changes

1. Add or edit YAML files in `servers/` or `infrastructure/`.
2. Commit and push to your GitOps repository.
3. SpoutMC will detect changes via polling and/or webhook:
   - create containers for added manifests
   - recreate containers for modified manifests
   - remove containers for deleted manifests

## Useful API Endpoints

```bash
# Trigger manual Git sync
curl -X POST http://your-spoutmc-host:3000/api/v1/git/sync

# Check GitOps sync status
curl http://your-spoutmc-host:3000/api/v1/git/status
```

## Related Docs

- `../../GITOPS.md`
- `../../GITOPS_QUICKSTART.md`
- `../../WEBHOOK_TEST.md`
