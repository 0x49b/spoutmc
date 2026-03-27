# Webhook Production Setup Guide

This guide explains how to run SpoutMC GitOps webhooks safely in production.

## Scope

- Endpoint: `POST /api/v1/git/webhook`
- Manual fallback: `POST /api/v1/git/sync`
- Applies to GitHub and GitLab webhook senders

## Production Architecture

Recommended flow:

1. Git provider sends webhook over HTTPS
2. Reverse proxy/load balancer terminates TLS
3. Proxy forwards request to SpoutMC (`:3000`)
4. SpoutMC validates webhook secret and triggers Git sync

Use HTTPS end-to-end where possible, and never expose an unauthenticated webhook endpoint publicly.

## Prerequisites

- SpoutMC running with GitOps enabled
- Reachable public hostname (for provider webhook callback)
- TLS certificate for webhook URL
- Secret management for:
  - `SPOUTMC_GIT_TOKEN` (private repositories)
  - `SPOUTMC_WEBHOOK_SECRET` (webhook verification)

## Required GitOps Configuration

Set `config/spoutmc.yaml`:

```yaml
git:
  enabled: true
  repository: "https://github.com/your-org/spoutmc-servers.git"
  branch: "main"
  token: "${SPOUTMC_GIT_TOKEN}"
  poll_interval: 60s
  webhook_secret: "${SPOUTMC_WEBHOOK_SECRET}"
  local_path: "/var/lib/spoutmc/gitops"
```

Notes:

- `poll_interval` remains a useful safety net if webhook delivery fails.
- `webhook_secret` must be set in production.
- `token` can be omitted for public repositories.

## Secret Generation and Storage

Generate a strong webhook secret:

```bash
openssl rand -hex 32
```

Export it in your runtime environment:

```bash
export SPOUTMC_WEBHOOK_SECRET="replace-with-generated-secret"
```

For private repositories:

```bash
export SPOUTMC_GIT_TOKEN="replace-with-read-only-token"
```

Security recommendations:

- Use a secret manager (Vault, 1Password Secrets Automation, AWS/GCP/Azure Secret Manager)
- Rotate webhook secret periodically
- Keep token scope minimal (`repo:read`/`read_repository`)

## Provider Setup

## GitHub

Repository -> `Settings` -> `Webhooks` -> `Add webhook`:

- Payload URL: `https://spoutmc.example.com/api/v1/git/webhook`
- Content type: `application/json`
- Secret: value of `SPOUTMC_WEBHOOK_SECRET`
- Events: `Just the push event`
- Active: enabled

GitHub request verification in SpoutMC:

- Event header: `X-GitHub-Event`
- Signature header: `X-Hub-Signature-256`
- Non-push events are ignored and do not trigger sync

## GitLab

Project -> `Settings` -> `Webhooks`:

- URL: `https://spoutmc.example.com/api/v1/git/webhook`
- Secret Token: value of `SPOUTMC_WEBHOOK_SECRET`
- Trigger: `Push events`
- Enable SSL verification

GitLab request verification in SpoutMC:

- Event header: `X-Gitlab-Event`
- Token header: `X-Gitlab-Token`
- Non-push events are ignored and do not trigger sync

## Reverse Proxy and Network Hardening

- Only expose HTTPS (443)
- Restrict direct access to SpoutMC port `3000` (internal network only)
- Add provider IP allowlisting where practical
- Apply request size limits at proxy (webhook payloads are small)
- Configure upstream timeout to cover sync latency safely

## Operational Checks

After deployment, validate:

1. `POST /api/v1/git/sync` returns HTTP `200`
2. Provider test delivery to `/api/v1/git/webhook` returns HTTP `200`
3. SpoutMC logs include:
   - `Received webhook request`
   - `Webhook processed successfully`

If GitOps/webhook is not initialized, webhook endpoint returns:

```json
{
  "error": "GitOps not enabled or webhook handler not initialized"
}
```

with HTTP `503`.

## Failure Modes and Response

- `401 Invalid signature`:
  - Secret mismatch or missing verification headers
  - Re-check provider secret and SpoutMC runtime env
- `500 Failed to sync configuration`:
  - Git pull/load failed
  - Check Git credentials, repository reachability, and config validity
- `503 GitOps not enabled...`:
  - `git.enabled` disabled or webhook handler unavailable

Keep polling enabled (`poll_interval`) so config still converges if webhook delivery is delayed.

## Production Checklist

- [ ] `git.enabled: true` is set
- [ ] `git.webhook_secret` is set from environment
- [ ] Webhook URL uses HTTPS and public DNS
- [ ] Reverse proxy blocks direct public access to port 3000
- [ ] Git token uses least privilege
- [ ] Provider webhook is push-only
- [ ] Logs and alerts cover 401/500/503 webhook failures
- [ ] Secret rotation procedure is documented

## Related Docs

- [GITOPS.md](GITOPS.md)
- [GITOPS_QUICKSTART.md](GITOPS_QUICKSTART.md)
- [WEBHOOK_TEST.md](WEBHOOK_TEST.md)
