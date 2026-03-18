# SpoutMC Velocity Players Bridge

This Velocity plugin provides a real-time player API for SpoutMC.

It tracks:
- player name
- last login time
- last logout time
- current server
- banned state and reason
- status (`online`, `offline`, `banned`)
- avatar URL (Crafatar, by UUID)

It exposes:
- `GET /healthz`
- `GET /players`
- `GET /players/stream` (SSE)
- `POST /players/{name}/message` body: `{ "message": "..." }`
- `POST /players/{name}/kick` body: `{ "reason": "..." }`
- `POST /players/{name}/ban` body: `{ "reason": "..." }`
- `POST /players/{name}/unban`

## Build

From this directory:

```bash
gradle shadowJar
```

Output jar:

`build/libs/velocity-players-bridge-0.1.0.jar`

## Install

1. Copy the jar into your Velocity `plugins/` directory.
2. Start Velocity once to generate config.
3. Edit:
   `plugins/spoutmc-players/config.properties`
4. Restart Velocity.

## Config

File: `config.properties`

```properties
bindHost=127.0.0.1
port=19132
token=
```

- `bindHost`: HTTP bind address
- `port`: HTTP port
- `token`: optional bearer token (if set, all endpoints except `/healthz` require `Authorization: Bearer <token>`)

## Notes

- Ban enforcement is global at proxy login via plugin-managed ban state.
- Current implementation stores state in `plugins/spoutmc-players/state.json`.
- Endpoint paths are intentionally simple for easy integration with the Go backend.
