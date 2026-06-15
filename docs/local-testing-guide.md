# Local Testing Guide for obgynrama-chatbot

Complete guide to test your chatbot locally before deploying to production.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start (5 minutes)](#quick-start-5-minutes)
3. [Running the App](#running-the-app)
4. [Testing the Bot Locally](#testing-the-bot-locally)
5. [Testing LINE Integration](#testing-line-integration)
6. [Viewing Logs](#viewing-logs)
7. [Checking Metrics](#checking-metrics)
8. [Debugging Tips](#debugging-tips)

---

## Prerequisites

You need:
- **Docker** & **Docker Compose** installed
- **Go 1.25+** (optional, only if testing without Docker)
- **curl** or **Postman** (for HTTP testing)
- **.env.local** file configured (see Quick Start)

Verify installations:
```bash
docker --version
docker compose --version
```

---

## Quick Start (5 minutes)

### 1. Copy environment template
```bash
cp .env.local.example .env.local
```

### 2. Start the app
```bash
docker compose -f docker-compose.yml up
```

You should see:
```
obgynrama-chatbot-1  | {"level":"info","msg":"HTTP server listening","addr":"0.0.0.0:8080"}
```

### 3. Test it works
```bash
curl http://localhost:8080/healthz
```

Expected response: `{"status":"ok"}`

**Done!** Your bot is running locally. Now test the functionality below.

---

## Running the App

### Option 1: Docker Compose (Recommended)

```bash
# Start all services (app + Prometheus)
docker compose -f docker-compose.yml up

# Or run in background
docker compose -f docker-compose.yml up -d

# View logs
docker compose logs -f obgynrama-chatbot

# Stop services
docker compose down
```

**Services running:**
- App: http://localhost:8080
- Prometheus: http://localhost:9090 (if compose includes it)

### Option 2: Run Go app directly (advanced)

```bash
# Install dependencies
go mod download

# Run server
go run cmd/server/main.go

# App listens on http://localhost:8080
```

---

## Testing the Bot Locally

### 1. Test Health Endpoint

```bash
curl -X GET http://localhost:8080/healthz
```

Expected:
```json
{"status":"ok"}
```

### 2. Test Webhook Endpoint (Without LINE)

Send a mock webhook request:

```bash
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-Line-Request-Id: test-request-123" \
  -d '{
    "events": [
      {
        "type": "message",
        "replyToken": "test-reply-token",
        "source": {
          "userId": "U1234567890abcdef1234567890abcdef"
        },
        "timestamp": 1462629479859,
        "message": {
          "type": "text",
          "id": "1234567890",
          "text": "help"
        }
      }
    ]
}'
```

**Note:** Without valid LINE signature, you'll get 401. To test the core logic, see [Testing without LINE validation](#testing-without-line-validation-optional) below.

### 3. Check App Logs

```bash
# If using Docker
docker compose logs -f obgynrama-chatbot

# If running Go directly, check console output
```

Look for:
```json
{"level":"info","webhook_event_id":"...","method":"POST","path":"/webhook"}
```

---

## Testing LINE Integration

### Option 1: Use LINE Simulator (Easiest for Beginners)

1. Go to [LINE Developers Console](https://developers.line.biz/)
2. Select your channel → **Bot** → **Bot Designer**
3. In **Bot Designer**, use the **Simulator** on the right
4. Type a message → It will send to your webhook

**Requirements:**
- Your webhook must be publicly accessible (not localhost)
- For local testing, use a tunnel service (see below)

### Option 2: Use ngrok Tunnel (for Local Testing)

Expose your local app to the internet temporarily:

```bash
# Install ngrok (one-time)
brew install ngrok  # macOS
# or download from https://ngrok.com/download

# Start tunnel
ngrok http 8080
```

You'll see:
```
Forwarding                    https://abc123.ngrok.io -> http://localhost:8080
```

**Then update LINE webhook URL:**
1. Go to [LINE Developers Console](https://developers.line.biz/)
2. Select your channel → **Messaging API** → **Webhook URL**
3. Set: `https://abc123.ngrok.io/webhook`
4. Click **Verify** (should show "Success")

**Now test:**
1. Go to **Bot Designer** → **Simulator**
2. Type a message
3. Check app logs: `docker compose logs -f obgynrama-chatbot`

### Option 3: Use webhook.site (for Testing Only)

Good for verifying webhook signature format:

1. Go to https://webhook.site/
2. Copy the unique URL
3. Send a test request:

```bash
curl -X POST https://webhook.site/your-unique-id \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "type": "message",
        "replyToken": "test",
        "source": {"userId": "U123"},
        "message": {"type": "text", "text": "hello"}
      }
    ]
  }'
```

You'll see the request logged at webhook.site.

---

## Viewing Logs

### Docker Compose Logs

```bash
# View all logs
docker compose logs

# Follow logs (real-time)
docker compose logs -f

# Only app logs
docker compose logs -f obgynrama-chatbot

# Last 50 lines
docker compose logs --tail=50
```

### Log Format

Each request produces JSON:
```json
{
  "level": "info",
  "ts": 1717419200.123,
  "logger": "http",
  "msg": "HTTP request completed",
  "request_id": "req-abc123",
  "method": "POST",
  "path": "/webhook",
  "status": 200,
  "latency_ms": 45
}
```

### Filter Logs

```bash
# Show only errors
docker compose logs | grep '"level":"error"'

# Show only webhook requests
docker compose logs | grep '"path":"/webhook"'

# Save logs to file
docker compose logs > app.log 2>&1
```

---

## Checking Metrics

### View Prometheus Metrics (Raw)

```bash
curl http://localhost:8000/metrics
```

You'll see:
```
# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="POST",path="/webhook",status="200"} 15
```

### Import Grafana Dashboard

If using docker-compose with Prometheus/Grafana:

1. Open Grafana: http://localhost:3000
2. **Data Sources** → Add Prometheus → URL: `http://prometheus:9090`
3. **Dashboards** → **Import** → Upload `monitoring/grafana-dashboard-obgynrama-chatbot.json`

You'll see graphs for:
- Request rate (RPS)
- Latency (p50, p95)
- Error rates
- Requests by path

---

## Debugging Tips

### Issue: Health check fails

```bash
# Check if app is running
docker ps | grep obgynrama-chatbot

# Check logs for startup errors
docker compose logs obgynrama-chatbot

# Verify port is open
netstat -an | grep 8080  # macOS/Linux
```

### Issue: Webhook returns 401 Unauthorized

**Cause:** Invalid LINE signature header.

**Solution for local testing:** 
- Use ngrok tunnel (see Option 2 above)
- Or disable signature check (dev mode only, not production):
  - Set `SKIP_LINE_SIGNATURE_CHECK=true` in `.env.local`
  - Restart app: `docker compose restart obgynrama-chatbot`

### Issue: Bot doesn't reply

Check:
1. **Webhook received?** Look for `"msg":"HTTP request completed"` in logs
2. **Which FAQ matched?** Look for `"matched_faq_id":"D1-Q1"` in logs
3. **Config loaded?** Check for `"msg":"Config loaded"` at startup

### Issue: Metrics not showing

```bash
# Check metrics endpoint
curl http://localhost:8000/metrics | head -20

# If empty, restart app
docker compose restart obgynrama-chatbot

# Wait 10 seconds, then check again
sleep 10 && curl http://localhost:8000/metrics | grep http_requests
```

### Issue: Docker image fails to build

```bash
# Clean up and rebuild
docker compose down
docker compose build --no-cache
docker compose up
```

---

## Checklist Before Production

- [ ] Health endpoint responds: `curl http://localhost:8080/healthz`
- [ ] Webhook accepts POST requests
- [ ] Logs show structured JSON (no plain text errors)
- [ ] Metrics endpoint works: `curl http://localhost:8000/metrics`
- [ ] FAQ replies work in simulator (if testing LINE)
- [ ] No errors in logs after 5 test messages
- [ ] Rollback script works: `bash scripts/deploy_with_rollback.sh <IMAGE_TAG>`

---

## Next Steps

1. **Ready for production?** Follow [deployment-beginner-guide.md](./deployment-beginner-guide.md)
2. **Want to modify bot replies?** Edit `configs/bot.yaml`
3. **Need to debug?** Check [Debugging Tips](#debugging-tips) above

---

**Questions?** Check app logs first:
```bash
docker compose logs -f obgynrama-chatbot | grep error
```
