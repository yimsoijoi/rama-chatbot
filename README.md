# LINE Chatbot (Go + Clean Architecture)

Config-driven LINE chatbot for colposcopy Q&A.

- Language: Go
- Architecture: Clean Architecture
- Storage: No database (YAML config only)

## Project Structure

- `cmd/server/main.go` - app entrypoint and dependency wiring
- `internal/domain` - entities and repository contracts
- `internal/usecase` - business logic for reply building
- `internal/infrastructure/config` - YAML config loader + repository implementation
- `internal/interface/http` - LINE webhook HTTP handler
- `configs/faq_seed.yaml` - single source of truth: runtime config (routing, escalation, fallback, per-DX rich menu) + FAQ knowledge base (`items`)

## Run

1. Copy env file and set credentials

```bash
cp .env.example .env
```

2. Export environment variables

```bash
export $(cat .env | xargs)
```

3. Install dependencies and run

```bash
go mod tidy
go run ./cmd/server
```

## Run with Docker

1. Create environment file

```bash
cp .env.example .env
```

2. Build image

```bash
docker build -t obgynrama-chatbot:local .
```

3. Run container

```bash
docker run --rm -p 8080:8080 --env-file .env obgynrama-chatbot:local
```

Or use compose:

```bash
docker compose up -d --build
```

For local env template:

```bash
cp .env.local.example .env
```

## Production Docker Compose (Reverse Proxy + TLS)

Production stack files:

- `docker-compose.prod.yml`
- `deploy/Caddyfile`
- `.env.prod.example`

1. Prepare production env

```bash
cp .env.prod.example .env.prod
```

2. Set required values in `.env.prod`:

- `LINE_CHANNEL_SECRET`
- `LINE_CHANNEL_TOKEN`
- `IMAGE_REGISTRY`
- `IMAGE_REPOSITORY`
- `IMAGE_TAG`
- `DOMAIN`

3. Run production stack

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

This stack uses Caddy as reverse proxy and automatic TLS certificate manager.

Server endpoints:

- `GET /healthz`
- `POST /webhook`
- `GET /metrics` (Prometheus metrics)
- `GET /debug/pprof/*` (Go profiler, controlled by `ENABLE_PPROF`)

## Observability and Monitoring

The service now emits structured JSON logs with request IDs and Prometheus metrics.

- Request tracing: every request has `X-Request-ID`
- Error tracing: startup, config, webhook parse/reply, and panic recovery logs include descriptive error context
- Performance profiling: Go pprof endpoints for CPU, heap, and execution trace
- Event dedup: `webhookEventId` is cached in memory to skip duplicate deliveries within TTL

Main metrics:

- `chatbot_http_requests_total{path,method,status}`
- `chatbot_http_request_duration_seconds{path,method}`

Environment:

- `EVENT_DEDUP_TTL` duration format (example: `24h`, `90m`, `30s`)

## CI/CD

Workflow file:

- `.github/workflows/ci-cd.yml`

What it does:

1. CI on pull request and main push: `go build` and `go test`
2. Build and push Docker image to GHCR on `main`
3. Optional deploy over SSH if deploy secrets are configured

Required GitHub secrets for deploy job:

- `DEPLOY_HOST`
- `DEPLOY_USER`
- `DEPLOY_SSH_KEY`
- `DEPLOY_PATH`

Server preparation:

1. Install Docker on server
2. Create deploy directory (same value as `DEPLOY_PATH`)
3. Put `.env` in deploy directory
4. Ensure server can pull from GHCR (docker login if needed)

Manual deploy command on server:

```bash
cd /path/to/deploy
IMAGE_REGISTRY=ghcr.io IMAGE_REPOSITORY=owner/repo ./scripts/deploy.sh latest
```

Rollback-capable deploy command:

```bash
IMAGE_REGISTRY=ghcr.io IMAGE_REPOSITORY=owner/repo ./scripts/deploy_with_rollback.sh latest
```

If health check fails, it will redeploy previous image tag automatically.

## Self-hosted Free Stack (Prometheus + Grafana)

1. Run chatbot on your server
2. Run Prometheus to scrape `http://<chatbot-host>:8080/metrics`
3. Run Grafana and connect Prometheus as data source

Minimal `prometheus.yml` scrape config:

```yaml
global:
	scrape_interval: 15s

scrape_configs:
	- job_name: obgynrama-chatbot
		metrics_path: /metrics
		static_configs:
			- targets: ["localhost:8080"]
```

Useful pprof commands:

```bash
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
go tool pprof http://localhost:8080/debug/pprof/heap
curl http://localhost:8080/debug/pprof/trace?seconds=5 -o trace.out
```

Starter Grafana dashboard JSON:

- `monitoring/grafana-dashboard-obgynrama-chatbot.json`

Import path in Grafana:

1. Dashboards
2. New
3. Import
4. Upload `monitoring/grafana-dashboard-obgynrama-chatbot.json`

## LINE Messaging API Guidelines (Practical Summary)

Based on LINE development guidelines:

- Validate webhook signatures (already done via `ParseRequest`)
- Save logs for incoming webhooks and outgoing Messaging API calls
- Save `x-line-request-id` for API calls (now logged as `line_request_id`)
- Handle future non-breaking additions safely (ignore unknown event fields/types)
- Respect unsend intent (now logging unsend events for downstream deletion policies)
- Do not load test through LINE Platform
- Do not rely on LINE webhook IP allowlists as security control; use signature validation

### What this project now does

- Structured JSON access/error logs with `request_id`
- Logs `client_ip`, method, path, status, and latency for each request
- Logs webhook metadata such as `webhook_event_id` and `is_redelivery`
- Uses timeout-bound reply calls and logs LINE API request IDs

## Designing chatbot UI/flow (Bot Designer question)

`LINE Bot Designer` is no longer maintained by LINE.

Recommended approach:

1. Design Flex messages with Flex Message Simulator
2. Define conversation flows as YAML intents/FAQ in `configs/faq_seed.yaml`
3. Validate real behavior in a LINE test channel
4. Measure outcomes in Grafana (fallback rate, latency, error rate)
5. Iterate copy and UX from metrics + user feedback

Suggested design workflow for your project:

1. Draft intents from your diagnosis scripts (DX1..DX5 + shared)
2. Map each intent to `match_phrases`, `answer`, and `quick_reply`
3. Keep messages short and friendly; use quick replies to reduce ambiguity
4. Reserve escalation keywords for urgent symptom messages
5. Add a monthly review loop for unmatched/fallback phrases

Detailed LINE UI/logo workflow:

- `docs/line-ui-logo-guide.md`

Beginner deployment step-by-step:

- `docs/deployment-beginner-guide.md`

## How it works

1. LINE sends events to `/webhook`.
2. Use case checks escalation keywords first.
3. Bot resolves user diagnosis from config (`user_diagnosis`), or uses `default_diagnosis`.
4. Bot searches diagnosis FAQ first, then shared FAQ.
5. If no match, fallback reply is returned.

## Config-first workflow

You can expand all 53 scripts from your HTML into `configs/faq_seed.yaml` without changing Go code.

For each question, add:

- `answer`
- `quick_reply`
- `match_phrases`

This keeps logic in code and medical content in config.
