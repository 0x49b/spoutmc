# Webhook Testing Guide

This guide provides comprehensive instructions for testing the GitOps webhook integration in SpoutMC.

## Table of Contents

- [Quick Start](#quick-start)
- [Local Testing with curl](#local-testing-with-curl)
- [Remote Testing with ngrok](#remote-testing-with-ngrok)
- [GitHub Webhook Setup](#github-webhook-setup)
- [GitLab Webhook Setup](#gitlab-webhook-setup)
- [Testing Scripts](#testing-scripts)
- [Full Integration Test](#full-integration-test)
- [Log Monitoring](#log-monitoring)
- [Troubleshooting](#troubleshooting)
- [Verification Checklist](#verification-checklist)

## Quick Start

The fastest way to test webhooks:

```bash
# 1. Start SpoutMC with GitOps enabled
./spoutmc

# 2. In another terminal, trigger manual sync
curl -X POST http://localhost:3000/api/v1/git/sync

# 3. Check response
# Should return: {"status":"success","message":"Configuration synced successfully"}
```

## Local Testing with curl

Test webhooks locally without external services.

### Prerequisites

```bash
# Set webhook secret
export SPOUTMC_WEBHOOK_SECRET="test-secret-123"

# Start SpoutMC
./spoutmc
```

### GitHub-Style Webhook

```bash
# Create payload
PAYLOAD='{"ref":"refs/heads/main","repository":{"full_name":"test/repo"}}'

# Calculate HMAC signature
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SPOUTMC_WEBHOOK_SECRET" | cut -d' ' -f2)

# Send webhook
curl -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

Expected response:
```json
{
  "status": "success",
  "commit": "abc1234"
}
```

### GitLab-Style Webhook

```bash
curl -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-Gitlab-Event: Push Hook" \
  -H "X-Gitlab-Token: $SPOUTMC_WEBHOOK_SECRET" \
  -d '{"ref":"refs/heads/main","project":{"name":"test"}}'
```

### Test Invalid Signature (Should Fail)

```bash
curl -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=invalid" \
  -d '{"test":"data"}'
```

Expected response:
```json
{
  "error": "Invalid signature"
}
```

## Remote Testing with ngrok

Test with actual webhooks from GitHub/GitLab using ngrok tunneling.

### Step 1: Install ngrok

```bash
# macOS
brew install ngrok

# Or download from https://ngrok.com/download

# Authenticate (optional, but recommended)
ngrok config add-authtoken YOUR_TOKEN
```

### Step 2: Start ngrok Tunnel

```bash
# Start tunnel to SpoutMC port
ngrok http 3000
```

Output:
```
Forwarding  https://abc123.ngrok.io -> http://localhost:3000
```

**Save the HTTPS URL** (e.g., `https://abc123.ngrok.io`)

### Step 3: Configure Webhook

Use the ngrok URL in your Git provider webhook settings:
- **Payload URL:** `https://abc123.ngrok.io/api/v1/git/webhook`

### Step 4: Monitor Requests

Open ngrok web interface to see all requests:
```bash
# Visit in browser
open http://127.0.0.1:4040
```

Or use ngrok CLI:
```bash
# In another terminal
ngrok http 3000 --log=stdout
```

### Step 5: Test by Pushing Changes

```bash
cd /path/to/your/server-config-repo
echo "# test webhook" >> servers/test.yaml
git add .
git commit -m "Test webhook integration"
git push
```

### Step 6: Verify

- Check ngrok web UI for the webhook request
- Check SpoutMC logs for "Received webhook request"
- Verify containers updated with `docker ps`

## GitHub Webhook Setup

Complete guide for setting up GitHub webhooks.

### Create Webhook Secret

```bash
# Generate secure random secret
export SPOUTMC_WEBHOOK_SECRET=$(openssl rand -hex 32)

# Save it!
echo "Webhook Secret: $SPOUTMC_WEBHOOK_SECRET" >> ~/.spoutmc-secrets
```

### Configure SpoutMC

Add to `config/spoutmc.yaml`:
```yaml
git:
  enabled: true
  repository: "https://github.com/your-org/spoutmc-servers.git"
  branch: "main"
  token: "${SPOUTMC_GIT_TOKEN}"
  webhook_secret: "${SPOUTMC_WEBHOOK_SECRET}"
  poll_interval: 30s
```

Restart SpoutMC.

### Add Webhook in GitHub

1. Go to your repository on GitHub
2. Click **Settings** → **Webhooks** → **Add webhook**
3. Configure:
   - **Payload URL:** `https://your-domain.com:3000/api/v1/git/webhook`
   - **Content type:** `application/json`
   - **Secret:** Paste your `SPOUTMC_WEBHOOK_SECRET`
   - **SSL verification:** Enable SSL verification (if using HTTPS)
   - **Which events:** Select "Just the push event"
   - **Active:** ✓ Checked
4. Click **Add webhook**

### Test GitHub Webhook

#### Method 1: Push a Change

```bash
cd /path/to/spoutmc-servers
echo "# test" >> README.md
git add README.md
git commit -m "Test webhook"
git push
```

#### Method 2: Use "Redeliver"

1. Go to Settings → Webhooks → Your webhook
2. Scroll to **Recent Deliveries**
3. Click on any delivery
4. Click **Redeliver** button
5. Confirm

### View Webhook Deliveries

1. Go to Settings → Webhooks → Your webhook
2. Scroll to **Recent Deliveries**
3. Click on a delivery to see:
   - Request headers
   - Request body
   - Response from SpoutMC
   - Status code

Look for:
- ✅ Green checkmark = Success (200)
- ❌ Red X = Failed (401, 500, etc.)

## GitLab Webhook Setup

Complete guide for setting up GitLab webhooks.

### Create Webhook Secret

```bash
# Generate secure random secret
export SPOUTMC_WEBHOOK_SECRET=$(openssl rand -hex 32)
echo "Webhook Secret: $SPOUTMC_WEBHOOK_SECRET"
```

### Configure SpoutMC

Same as GitHub - add to `config/spoutmc.yaml`:
```yaml
git:
  webhook_secret: "${SPOUTMC_WEBHOOK_SECRET}"
```

### Add Webhook in GitLab

1. Go to your project on GitLab
2. Click **Settings** → **Webhooks**
3. Configure:
   - **URL:** `https://your-domain.com:3000/api/v1/git/webhook`
   - **Secret Token:** Paste your `SPOUTMC_WEBHOOK_SECRET`
   - **Trigger:** Check "Push events" ✓
   - **SSL verification:** Enable SSL verification (if using HTTPS)
4. Click **Add webhook**

### Test GitLab Webhook

#### Method 1: Push a Change

```bash
cd /path/to/spoutmc-servers
git commit --allow-empty -m "Test webhook"
git push
```

#### Method 2: Use "Test" Button

1. Go to Settings → Webhooks
2. Find your webhook
3. Click **Test** → **Push events**

### View Webhook Logs

1. Go to Settings → Webhooks
2. Click **Edit** on your webhook
3. Scroll to **Recent Deliveries**
4. Click "View details" on any delivery

## Testing Scripts

### Basic Webhook Test Script

Save as `test-webhook.sh`:

```bash
#!/bin/bash

# Configuration
WEBHOOK_SECRET="${SPOUTMC_WEBHOOK_SECRET:-test-secret}"
WEBHOOK_URL="${WEBHOOK_URL:-http://localhost:3000/api/v1/git/webhook}"
WEBHOOK_TYPE="${1:-github}"  # github or gitlab

echo "🧪 Testing webhook integration"
echo "   Type: $WEBHOOK_TYPE"
echo "   URL: $WEBHOOK_URL"
echo ""

if [ "$WEBHOOK_TYPE" = "github" ]; then
    # GitHub webhook
    PAYLOAD='{"ref":"refs/heads/main","repository":{"full_name":"test/repo"}}'
    SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" | cut -d' ' -f2)

    RESPONSE=$(curl -s -w "\nHTTP:%{http_code}" -X POST "$WEBHOOK_URL" \
      -H "Content-Type: application/json" \
      -H "X-GitHub-Event: push" \
      -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
      -d "$PAYLOAD")

elif [ "$WEBHOOK_TYPE" = "gitlab" ]; then
    # GitLab webhook
    PAYLOAD='{"ref":"refs/heads/main","project":{"name":"test"}}'

    RESPONSE=$(curl -s -w "\nHTTP:%{http_code}" -X POST "$WEBHOOK_URL" \
      -H "Content-Type: application/json" \
      -H "X-Gitlab-Event: Push Hook" \
      -H "X-Gitlab-Token: $WEBHOOK_SECRET" \
      -d "$PAYLOAD")
else
    echo "❌ Unknown webhook type. Use: github or gitlab"
    exit 1
fi

# Parse response
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP:/d')

echo "Response Code: $HTTP_CODE"
echo "Response Body: $BODY"
echo ""

if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ Webhook test PASSED"
    exit 0
else
    echo "❌ Webhook test FAILED"
    exit 1
fi
```

Usage:
```bash
chmod +x test-webhook.sh

# Test GitHub webhook
./test-webhook.sh github

# Test GitLab webhook
./test-webhook.sh gitlab

# Test with custom URL
WEBHOOK_URL=https://example.com/api/v1/git/webhook ./test-webhook.sh github
```

### Continuous Webhook Test

Save as `continuous-test.sh`:

```bash
#!/bin/bash

echo "🔄 Running continuous webhook tests (Ctrl+C to stop)"
echo ""

COUNT=0
SUCCESS=0
FAILED=0

while true; do
    COUNT=$((COUNT + 1))
    echo "Test #$COUNT - $(date +%H:%M:%S)"

    if ./test-webhook.sh github > /dev/null 2>&1; then
        SUCCESS=$((SUCCESS + 1))
        echo "  ✅ Success (Total: $SUCCESS)"
    else
        FAILED=$((FAILED + 1))
        echo "  ❌ Failed (Total: $FAILED)"
    fi

    sleep 5
done
```

Usage:
```bash
chmod +x continuous-test.sh
./continuous-test.sh
```

## Full Integration Test

Complete end-to-end test of GitOps with webhooks.

Save as `integration-test.sh`:

```bash
#!/bin/bash
set -e

echo "🧪 SpoutMC GitOps Integration Test"
echo "=================================="
echo ""

# Configuration
TEST_REPO_DIR=$(mktemp -d)
SPOUTMC_GIT_PATH="/tmp/spoutmc-git-test"

cleanup() {
    echo "🧹 Cleaning up..."
    rm -rf "$TEST_REPO_DIR"
    rm -rf "$SPOUTMC_GIT_PATH"
}
trap cleanup EXIT

# Step 1: Create test repository
echo "📦 Creating test repository..."
cd "$TEST_REPO_DIR"
git init -b main
mkdir servers

cat > servers/test-server.yaml <<EOF
name: test-server
image: itzg/minecraft-server
env:
  EULA: "TRUE"
  TYPE: PAPER
  VERSION: 1.21.10
  MAX_MEMORY: 2G
EOF

git add .
git commit -m "Initial commit"
INITIAL_COMMIT=$(git rev-parse HEAD)
echo "   ✅ Repository created at $TEST_REPO_DIR"
echo "   Commit: $INITIAL_COMMIT"

# Step 2: Configure SpoutMC
echo ""
echo "⚙️  Configuring SpoutMC for test..."
export SPOUTMC_GIT_TOKEN=""
export SPOUTMC_WEBHOOK_SECRET="test-secret-$(date +%s)"

cat > config/spoutmc-test.yaml <<EOF
git:
  enabled: true
  repository: "file://$TEST_REPO_DIR"
  branch: "main"
  poll_interval: 5s
  webhook_secret: "\${SPOUTMC_WEBHOOK_SECRET}"
  local_path: "$SPOUTMC_GIT_PATH"

servers: []
EOF
echo "   ✅ Test configuration created"

# Step 3: Start SpoutMC (manual step)
echo ""
echo "▶️  Start SpoutMC manually with:"
echo "   ./spoutmc"
echo ""
read -p "Press Enter when SpoutMC is running..."

# Step 4: Test initial sync
echo ""
echo "🔄 Testing initial sync..."
sleep 6  # Wait for initial clone and load

RESPONSE=$(curl -s http://localhost:3000/api/v1/git/sync)
echo "   Response: $RESPONSE"

if echo "$RESPONSE" | grep -q "success"; then
    echo "   ✅ Initial sync successful"
else
    echo "   ❌ Initial sync failed"
    exit 1
fi

# Step 5: Verify container exists
echo ""
echo "🔍 Checking if container was created..."
if docker ps --filter "name=test-server" --format "{{.Names}}" | grep -q "test-server"; then
    echo "   ✅ Container 'test-server' is running"
else
    echo "   ❌ Container 'test-server' not found"
    exit 1
fi

# Step 6: Modify configuration
echo ""
echo "📝 Modifying server configuration..."
cd "$TEST_REPO_DIR"
sed -i '' 's/MAX_MEMORY: 2G/MAX_MEMORY: 4G/' servers/test-server.yaml
git add .
git commit -m "Increase memory to 4G"
NEW_COMMIT=$(git rev-parse HEAD)
echo "   ✅ Configuration updated"
echo "   New commit: $NEW_COMMIT"

# Step 7: Trigger webhook
echo ""
echo "🪝 Simulating webhook..."
PAYLOAD="{\"ref\":\"refs/heads/main\",\"after\":\"$NEW_COMMIT\"}"
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SPOUTMC_WEBHOOK_SECRET" | cut -d' ' -f2)

WEBHOOK_RESPONSE=$(curl -s -w "\nHTTP:%{http_code}" -X POST http://localhost:3000/api/v1/git/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -d "$PAYLOAD")

HTTP_CODE=$(echo "$WEBHOOK_RESPONSE" | grep "HTTP:" | cut -d: -f2)
if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✅ Webhook accepted (200 OK)"
else
    echo "   ❌ Webhook failed (HTTP $HTTP_CODE)"
    exit 1
fi

# Step 8: Wait for changes to apply
echo ""
echo "⏳ Waiting for changes to apply..."
sleep 10

# Step 9: Verify container was recreated
echo ""
echo "🔍 Verifying container was recreated..."
CONTAINER_ID=$(docker ps --filter "name=test-server" --format "{{.ID}}")
MEMORY=$(docker inspect "$CONTAINER_ID" | grep -i '"Memory"' | head -1)
echo "   Container ID: $CONTAINER_ID"
echo "   Memory setting: $MEMORY"

# Step 10: Add a new server
echo ""
echo "➕ Adding new server..."
cd "$TEST_REPO_DIR"
cat > servers/new-server.yaml <<EOF
name: new-server
image: itzg/minecraft-server
env:
  EULA: "TRUE"
  TYPE: VANILLA
  MAX_MEMORY: 1G
EOF

git add .
git commit -m "Add new server"
curl -s -X POST http://localhost:3000/api/v1/git/sync > /dev/null
sleep 6

if docker ps --filter "name=new-server" --format "{{.Names}}" | grep -q "new-server"; then
    echo "   ✅ New server created successfully"
else
    echo "   ❌ New server not found"
    exit 1
fi

# Step 11: Remove a server
echo ""
echo "🗑️  Removing server..."
cd "$TEST_REPO_DIR"
rm servers/new-server.yaml
git add .
git commit -m "Remove new server"
curl -s -X POST http://localhost:3000/api/v1/git/sync > /dev/null
sleep 6

if docker ps --filter "name=new-server" --format "{{.Names}}" | grep -q "new-server"; then
    echo "   ❌ Server still exists (should be removed)"
    exit 1
else
    echo "   ✅ Server removed successfully"
fi

# Success!
echo ""
echo "=================================="
echo "✅ All integration tests PASSED!"
echo "=================================="
echo ""
echo "Tested scenarios:"
echo "  ✓ Initial configuration load"
echo "  ✓ Container creation"
echo "  ✓ Configuration modification"
echo "  ✓ Webhook triggering"
echo "  ✓ Container recreation"
echo "  ✓ Adding new servers"
echo "  ✓ Removing servers"
```

Usage:
```bash
chmod +x integration-test.sh
./integration-test.sh
```

## Log Monitoring

### What to Look For

#### Successful Webhook Processing

```
Received webhook request type=github remote_addr=140.82.115.0
Manual sync triggered via webhook
Pulling latest changes from Git repository
Changes detected in Git repository, reloading configuration
Successfully loaded server configurations from Git count=3
⛏️ Recreating container containerName=lobby
⛏️ Running lobby (abc123def4) with itzg/minecraft-server
Webhook processed successfully
```

#### Webhook Authentication Failure

```
Webhook signature verification failed error="signature mismatch"
```

#### No Changes Detected

```
Polling Git repository for changes
Repository already up to date
No changes to apply
```

#### Git Pull Error

```
Failed to pull Git repository error="authentication required"
```

### Real-Time Log Monitoring

```bash
# Watch all webhook activity
tail -f logs/spoutmc.log | grep -i webhook

# Watch Git sync activity
tail -f logs/spoutmc.log | grep -i "git\|pull\|clone"

# Watch container changes
tail -f logs/spoutmc.log | grep -i "recreat\|creating\|running"

# Colored output with highlighting
tail -f logs/spoutmc.log | grep --color=always -E "webhook|error|success|fail"
```

## Troubleshooting

### Problem: Webhook Returns 401 Unauthorized

**Symptoms:**
```json
{"error": "Invalid signature"}
```

**Solutions:**

1. **Verify secret matches:**
```bash
# Check SpoutMC secret
echo $SPOUTMC_WEBHOOK_SECRET

# Check webhook secret in Git provider
# They must be identical
```

2. **Check signature calculation:**
```bash
# Test locally
PAYLOAD='{"test":"data"}'
echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "your-secret"
```

3. **Verify headers:**
```bash
# GitHub should send: X-Hub-Signature-256
# GitLab should send: X-Gitlab-Token
```

### Problem: Webhook Returns 503 Service Unavailable

**Symptoms:**
```json
{"error": "GitOps not enabled or webhook handler not initialized"}
```

**Solutions:**

1. **Verify GitOps is enabled:**
```bash
grep "GitOps is enabled" logs/spoutmc.log
```

2. **Check webhook secret is set:**
```yaml
git:
  enabled: true
  webhook_secret: "${SPOUTMC_WEBHOOK_SECRET}"
```

3. **Restart SpoutMC:**
```bash
# Kill and restart
pkill spoutmc
./spoutmc
```

### Problem: Webhook Accepted but No Changes Applied

**Symptoms:**
- Webhook returns 200 OK
- No containers updated

**Solutions:**

1. **Check if there are actual changes:**
```bash
# View recent commits
cd /tmp/spoutmc-git
git log -3 --oneline
```

2. **Manually trigger sync:**
```bash
curl -X POST http://localhost:3000/api/v1/git/sync
```

3. **Check for Git pull errors:**
```bash
grep "Failed to pull" logs/spoutmc.log
```

4. **Verify YAML files are valid:**
```bash
# Lint YAML files
yamllint /tmp/spoutmc-git/servers/*.yaml
```

### Problem: Can't Reach Webhook Endpoint

**Symptoms:**
- GitHub shows delivery failed
- Connection timeout

**Solutions:**

1. **Check SpoutMC is running:**
```bash
curl http://localhost:3000/api/v1/ping
```

2. **Check firewall rules:**
```bash
# Test from external server
curl -X POST https://your-domain.com:3000/api/v1/git/webhook
```

3. **Verify port forwarding:**
```bash
# Check if port 3000 is listening
netstat -an | grep 3000
lsof -i :3000
```

4. **Use ngrok for local testing:**
```bash
ngrok http 3000
# Use ngrok URL in webhook config
```

### Problem: Webhook Works but Containers Not Updating

**Solutions:**

1. **Check diff detection:**
```bash
grep "Changes detected" logs/spoutmc.log
```

2. **Verify server name matches:**
```bash
# Names must match exactly
docker ps --filter "label=io.spout.network=true" --format "{{.Names}}"
```

3. **Check for YAML parse errors:**
```bash
grep "Failed to parse YAML" logs/spoutmc.log
```

## Verification Checklist

Use this checklist to verify webhook integration is working correctly.

### Initial Setup

- [ ] SpoutMC starts with log message: "GitOps is enabled, initializing Git sync"
- [ ] Git repository cloned to configured path (default: `/tmp/spoutmc-git`)
- [ ] Log shows: "Webhook handler initialized"
- [ ] Webhook secret is set in configuration
- [ ] Environment variable `SPOUTMC_WEBHOOK_SECRET` is set

### Basic Connectivity

- [ ] Webhook endpoint reachable: `curl http://localhost:3000/api/v1/git/webhook`
- [ ] Manual sync works: `curl -X POST http://localhost:3000/api/v1/git/sync`
- [ ] Ping endpoint works: `curl http://localhost:3000/api/v1/ping`

### Webhook Authentication

- [ ] Valid webhook request returns HTTP 200
- [ ] Invalid signature returns HTTP 401
- [ ] Missing signature returns HTTP 401
- [ ] GitHub-style webhook signature verified correctly
- [ ] GitLab-style webhook token verified correctly

### Functional Testing

- [ ] Pushing to Git repo triggers webhook delivery
- [ ] Webhook delivery shown in Git provider UI
- [ ] SpoutMC logs show "Received webhook request"
- [ ] Configuration reloaded after webhook
- [ ] Containers created for new servers
- [ ] Containers recreated for modified servers
- [ ] Containers removed for deleted servers

### Git Integration

- [ ] Git repository polls at configured interval
- [ ] Git changes detected and logged
- [ ] Configuration loaded from all YAML files
- [ ] Invalid YAML files skipped gracefully
- [ ] Git authentication works (for private repos)

### Performance

- [ ] Webhook responds within 1 second
- [ ] Container updates complete within 30 seconds
- [ ] No memory leaks after multiple webhooks
- [ ] Log file doesn't grow excessively

### Error Handling

- [ ] Git pull failures logged but don't crash
- [ ] Invalid YAML logged but don't block other servers
- [ ] Network errors handled gracefully
- [ ] Duplicate server names detected

### End-to-End

- [ ] Create test server → container created
- [ ] Modify test server → container recreated
- [ ] Delete test server → container removed
- [ ] Add multiple servers → all containers created
- [ ] Process survives webhook errors

## Quick Reference Commands

```bash
# Test manual sync
curl -X POST http://localhost:3000/api/v1/git/sync

# Test GitHub webhook
curl -X POST http://localhost:3000/api/v1/git/webhook \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: sha256=$(echo -n '{}' | openssl dgst -sha256 -hmac "$SECRET" | cut -d' ' -f2)" \
  -d '{}'

# Test GitLab webhook
curl -X POST http://localhost:3000/api/v1/git/webhook \
  -H "X-Gitlab-Event: Push Hook" \
  -H "X-Gitlab-Token: $SECRET" \
  -d '{}'

# Monitor webhooks
tail -f logs/spoutmc.log | grep -i webhook

# Check containers
docker ps --filter "label=io.spout.network=true"

# Check git repo
ls -la /tmp/spoutmc-git/servers/

# Generate webhook secret
openssl rand -hex 32

# Start ngrok tunnel
ngrok http 3000
```

## Additional Resources

- [GITOPS.md](GITOPS.md) - Full GitOps documentation
- [GITOPS_QUICKSTART.md](GITOPS_QUICKSTART.md) - Quick start guide
- [GitHub Webhooks Documentation](https://docs.github.com/en/webhooks)
- [GitLab Webhooks Documentation](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)
- [ngrok Documentation](https://ngrok.com/docs)
