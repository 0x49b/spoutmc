#!/bin/zsh
PAYLOAD='{"ref":"refs/heads/main","repository":{"full_name":"local/test"}}'
SIGNATURE=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SPOUTMC_WEBHOOK_SECRET" | cut -d' ' -f2)

curl -i -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -d "$PAYLOAD"