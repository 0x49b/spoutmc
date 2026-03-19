# SpoutMC GitOps Example Repository

This directory is an example GitOps repository layout for SpoutMC.

## Structure

```text
servers/
├── proxy.yaml               # Velocity proxy server (SpoutServer manifest)
├── lobby.yaml               # Lobby server (SpoutServer manifest)
└── skyblock.yaml            # Game server (SpoutServer manifest)

infrastructure/
└── database.yaml            # InfrastructureContainer manifest
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
  env:
    EULA: "TRUE"
    TYPE: PAPER
    VERSION: "1.21.10"
  volumes:
    - containerpath: "/data"
```

### InfrastructureContainer Example

```yaml
apiVersion: spoutmc.io/v1alpha1
kind: InfrastructureContainer
metadata:
  name: database
spec:
  name: database
  image: mariadb:latest
  restart: always
  ports:
    - host: "3306"
      container: "3306"
  env:
    MARIADB_ROOT_PASSWORD: changeme
```

## Notes

- In server volumes, use `containerpath` only.
- Host paths are generated automatically from SpoutMC `storage.data_path`.
- `metadata.name` should match `spec.name`.

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

- `../../docs/GITOPS.md`
- `../../docs/GITOPS_QUICKSTART.md`
- `../../docs/WEBHOOK_TEST.md`
