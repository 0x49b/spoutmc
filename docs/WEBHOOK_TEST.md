# Webhook Local Testing Guide

This guide focuses on local verification of the GitOps webhook endpoint:

- `POST /api/v1/git/webhook`
- `POST /api/v1/git/sync` (manual fallback)

It matches the current webhook handler behavior in `internal/git/webhook.go`.

## Prerequisites

- SpoutMC running locally on `http://localhost:3000`
- GitOps configured (`git.enabled: true`)
- `SPOUTMC_WEBHOOK_SECRET` set in your runtime environment

Example:

```bash
export SPOUTMC_WEBHOOK_SECRET="test-secret-123"
./spoutmc
```

## Quick Health Check

Manual sync should work before webhook testing:

```bash
curl -i -X POST http://localhost:3000/api/v1/git/sync
```

Expected: HTTP `200` and JSON containing:

```json
{
  "status": "success",
  "message": "Configuration synced successfully"
}
```

## GitHub-Style Local Test

Build a signed payload and send with GitHub headers:

```bash
PAYLOAD='{"ref":"refs/heads/main","repository":{"full_name":"local/test"}}'
SIGNATURE=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SPOUTMC_WEBHOOK_SECRET" | cut -d' ' -f2)

curl -i -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

Expected: HTTP `200` and JSON:

```json
{
  "status": "success",
  "commit": "abc1234"
}
```

## GitLab-Style Local Test

Send GitLab headers with shared token:

```bash
curl -i -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-Gitlab-Event: Push Hook" \
  -H "X-Gitlab-Token: $SPOUTMC_WEBHOOK_SECRET" \
  -d '{"ref":"refs/heads/main","project":{"path_with_namespace":"local/test"}}'
```

Expected: HTTP `200` and JSON:

```json
{
  "status": "success",
  "commit": "abc1234"
}
```

## Non-Push Event Test

Webhook hardening ignores non-push events to avoid unnecessary syncs.

### GitHub non-push

```bash
PAYLOAD='{"action":"opened"}'
SIGNATURE=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SPOUTMC_WEBHOOK_SECRET" | cut -d' ' -f2)

curl -i -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: pull_request" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

Expected: HTTP `200` and JSON with:

```json
{
  "status": "ignored",
  "event": "pull_request"
}
```

### GitLab non-push

```bash
curl -i -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-Gitlab-Event: Merge Request Hook" \
  -H "X-Gitlab-Token: $SPOUTMC_WEBHOOK_SECRET" \
  -d '{"object_kind":"merge_request"}'
```

Expected: HTTP `200` and JSON with:

```json
{
  "status": "ignored",
  "event": "merge request hook"
}
```

## Negative Tests

## Invalid Signature / Token

### GitHub invalid signature

```bash
curl -i -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=deadbeef" \
  -d '{"ref":"refs/heads/main"}'
```

Expected: HTTP `401`:

```json
{
  "error": "Invalid signature"
}
```

### GitLab invalid token

```bash
curl -i -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-Gitlab-Event: Push Hook" \
  -H "X-Gitlab-Token: wrong-secret" \
  -d '{"ref":"refs/heads/main"}'
```

Expected: HTTP `401`:

```json
{
  "error": "Invalid signature"
}
```

## Missing Signature / Token

```bash
curl -i -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -d '{"ref":"refs/heads/main"}'
```

Expected: HTTP `401`:

```json
{
  "error": "Invalid signature"
}
```

## GitOps Disabled / Handler Missing (503)

When GitOps is disabled or webhook handler was not initialized:

```bash
curl -i -X POST http://localhost:3000/api/v1/git/webhook -d '{}'
```

Expected: HTTP `503`:

```json
{
  "error": "GitOps not enabled or webhook handler not initialized"
}
```

## Optional: Tunnel Testing (Real Provider -> Local)

Use ngrok (or cloudflared) so GitHub/GitLab can call your local machine.

```bash
ngrok http 3000
```

Then configure provider webhook URL:

`https://<your-tunnel-id>.ngrok.io/api/v1/git/webhook`

Validate deliveries in provider UI and local SpoutMC logs.

## Troubleshooting

- `401 Invalid signature`
  - Verify provider secret matches `SPOUTMC_WEBHOOK_SECRET`
  - Ensure GitHub signature is HMAC SHA-256 over raw payload
  - Ensure GitLab sends `X-Gitlab-Token`
- `500 Failed to sync configuration`
  - Validate repository token/access and branch
  - Check repository YAML/config correctness
- `503 GitOps not enabled...`
  - Confirm `git.enabled: true`
  - Ensure `InitializeGitOps` completed successfully at startup

## Related Docs

- [WEBHOOK_PRODUCTION.md](WEBHOOK_PRODUCTION.md)
- [GITOPS.md](GITOPS.md)
- [GITOPS_QUICKSTART.md](GITOPS_QUICKSTART.md)
