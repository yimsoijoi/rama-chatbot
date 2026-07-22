#!/usr/bin/env bash
# One-shot server setup for a fresh Ubuntu VPS (run as root).
#   1. installs Docker (+ compose plugin) and git
#   2. clones this repo to /opt/rama-chatbot
#   3. creates .env.prod from the template (chmod 600)
# It does NOT deploy — you must fill secrets in .env.prod first (see output).
#
# Usage on the server:
#   curl -fsSL https://raw.githubusercontent.com/yimsoijoi/rama-chatbot/main/deploy/provision.sh | bash
#   # (or) scp this file up and: bash provision.sh
set -euo pipefail

REPO_URL="https://github.com/yimsoijoi/rama-chatbot.git"
APP_DIR="/opt/rama-chatbot"

echo "==> Installing prerequisites (git, curl)..."
apt-get update -y
apt-get install -y ca-certificates curl git

echo "==> Installing Docker (+ compose plugin)..."
if ! command -v docker >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com | sh || true
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "    get.docker.com unavailable for this Ubuntu; using distro packages..."
  apt-get install -y docker.io docker-compose-v2
  systemctl enable --now docker
fi
docker --version
docker compose version || true

echo "==> Fetching code into ${APP_DIR}..."
if [ -d "${APP_DIR}/.git" ]; then
  git -C "${APP_DIR}" pull --ff-only
else
  git clone "${REPO_URL}" "${APP_DIR}"
fi
cd "${APP_DIR}"

echo "==> Preparing .env.prod..."
if [ ! -f .env.prod ]; then
  cp .env.prod.example .env.prod
  chmod 600 .env.prod
  echo "    created .env.prod (from template)"
else
  echo "    .env.prod already exists — leaving it untouched"
fi

# Optional: open firewall if ufw is active
if command -v ufw >/dev/null 2>&1 && ufw status | grep -q "Status: active"; then
  ufw allow 22/tcp || true
  ufw allow 80/tcp || true
  ufw allow 443/tcp || true
fi

cat <<EOF

============================================================
 Server prepared. Next steps (do in this order):
============================================================
 0. DNS must already point bot.ppakzv.com -> this server IP.
    Check:  dig +short bot.ppakzv.com

 1. Fill in secrets (rotated LINE values):
      nano ${APP_DIR}/.env.prod
    Required: LINE_CHANNEL_SECRET, LINE_CHANNEL_TOKEN,
              DOMAIN=bot.ppakzv.com

 2. Make the container image pullable (one-time, pick one):
      - GitHub -> repo -> Packages -> rama-chatbot -> make PUBLIC   (easiest)
      - or:  docker login ghcr.io   (with a GitHub token)

 3. Deploy:
      cd ${APP_DIR} && ./scripts/deploy_with_rollback.sh latest

 4. Verify (Caddy issues TLS automatically once DNS resolves):
      curl -I https://bot.ppakzv.com/healthz     # expect: 200

 5. In LINE Developers Console:
      Webhook URL = https://bot.ppakzv.com/webhook  ->  Verify  ->  Use webhook = ON
============================================================
EOF
