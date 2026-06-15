# OBGYNRama Chatbot Deployment Guide (Beginner)

This guide is for people with zero deployment background.

## 1) What you need first

- A server that is online 24/7 (VPS or cloud VM)
- A public HTTPS URL for LINE webhook
- Your LINE channel credentials
- Docker installed on server

For production, you should use your own domain.

## 2) Do I need a domain?

- Testing only: not required (you can use temporary tunnel URL)
- Real production: yes, strongly recommended

Why domain helps:

- Stable webhook URL for LINE
- Easier TLS/HTTPS setup
- Better trust and maintainability

## 3) Recommended providers (domain and server)

These are common and beginner-friendly options.

### Domain registrars

- Cloudflare Registrar: https://www.cloudflare.com/products/registrar/
- Namecheap: https://www.namecheap.com/
- Porkbun: https://porkbun.com/
- GoDaddy: https://www.godaddy.com/

### VPS/cloud server providers

- Hetzner Cloud: https://www.hetzner.com/cloud
- DigitalOcean: https://www.digitalocean.com/
- Vultr: https://www.vultr.com/
- Linode (Akamai): https://www.linode.com/
- AWS Lightsail: https://aws.amazon.com/lightsail/
- Google Cloud Compute Engine: https://cloud.google.com/compute

## 4) Recommended starter server spec

- CPU: 1 vCPU
- RAM: 1 GB
- Disk: 20 GB SSD
- OS: Ubuntu 22.04 LTS

Upgrade to 2 GB RAM if traffic increases.

## 5) Production architecture in this repository

- App container: `obgynrama-chatbot`
- Reverse proxy and TLS: Caddy
- Compose file: `docker-compose.prod.yml`
- TLS config: `deploy/Caddyfile`
- Rollback script: `scripts/deploy_with_rollback.sh`
- Production env template: `.env.prod.example`

## 6) Test locally first (recommended)

Before deploying to production, test the bot locally to ensure everything works.

See: [Local Testing Guide](./local-testing-guide.md)

Key checks:
- Health endpoint responds: `curl http://localhost:8080/healthz`
- Webhook accepts POST requests
- Bot replies to test messages
- Logs show no errors

Time needed: 15 minutes

## 7) Step-by-step production deployment

### Step 1: Buy domain

Buy a domain from one registrar above.

### Step 2: Create server

Create Ubuntu server from provider dashboard.

### Step 3: Point DNS to your server

At your domain DNS settings:

- Create `A` record: host `@` -> your server public IP
- Optional `A` record: host `www` -> your server public IP

Wait for DNS propagation (5 to 30 minutes usually).

### Step 4: SSH into server

```bash
ssh root@YOUR_SERVER_IP
```

### Step 5: Install Docker + Compose plugin

```bash
apt update
apt install -y ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo $VERSION_CODENAME) stable" \
  > /etc/apt/sources.list.d/docker.list
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

### Step 6: Prepare deployment directory

```bash
mkdir -p /opt/obgynrama-chatbot
cd /opt/obgynrama-chatbot
```

Copy these files from your repo to server:

- `docker-compose.prod.yml`
- `deploy/Caddyfile`
- `scripts/deploy_with_rollback.sh`
- `.env.prod.example`

Then create `.env.prod`:

```bash
cp .env.prod.example .env.prod
```

### Step 7: Fill `.env.prod`

Set at least:

- `LINE_CHANNEL_SECRET`
- `LINE_CHANNEL_TOKEN`
- `IMAGE_REGISTRY=ghcr.io`
- `IMAGE_REPOSITORY=owner/obgynrama-chatbot`
- `IMAGE_TAG=latest`
- `DOMAIN=your-domain.com`
- `ENABLE_PPROF=false`

### Step 8: Login to container registry (if private image)

```bash
docker login ghcr.io
```

### Step 9: First deploy

```bash
chmod +x scripts/deploy_with_rollback.sh
IMAGE_REGISTRY=ghcr.io IMAGE_REPOSITORY=owner/obgynrama-chatbot ./scripts/deploy_with_rollback.sh latest
```

### Step 10: Configure LINE webhook

In LINE Developers console:

- Set webhook URL to `https://your-domain.com/webhook`
- Enable webhook
- Verify connection

### Step 11: Verify service

```bash
curl -I https://your-domain.com/healthz
curl -I https://your-domain.com/metrics
```

## 8) Rollback usage

If a new version fails, run previous tag:

```bash
IMAGE_REGISTRY=ghcr.io IMAGE_REPOSITORY=owner/obgynrama-chatbot ./scripts/deploy_with_rollback.sh PREVIOUS_TAG
```

The script also attempts automatic rollback if health check fails.

## 9) What to ask a provider/shop if you need support

Use this checklist when contacting hosting support:

- Do you support Ubuntu 22.04 VPS with root SSH?
- Can I open ports 80 and 443?
- Is there fixed public IPv4?
- Any outbound firewall restrictions for API calls to LINE?
- Can reverse DNS be configured if needed?
- What is backup/snapshot policy and price?

## 10) Fast test option (no domain yet)

For temporary test only:

- Run app locally or on server
- Use tunnel tool (e.g. ngrok/cloudflared) to get temporary HTTPS URL
- Put temporary URL in LINE webhook

Do not use temporary tunnel URL for long-term production.

## 11) Security basics before go-live

- Keep `ENABLE_PPROF=false` in production
- Rotate LINE secrets if leaked
- Restrict server SSH (key-only login)
- Enable server auto security updates
- Monitor logs and metrics in Grafana
