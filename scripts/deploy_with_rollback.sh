#!/usr/bin/env sh
set -eu

COMPOSE_FILE="docker-compose.prod.yml"
ENV_FILE=".env.prod"
APP_CONTAINER="obgynrama-chatbot"
TARGET_TAG="${1:-latest}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Missing ${ENV_FILE} in current directory"
  exit 1
fi

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "Missing ${COMPOSE_FILE} in current directory"
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "./${ENV_FILE}"
set +a

PREVIOUS_IMAGE=""
if docker ps -a --format '{{.Names}}' | grep -q "^${APP_CONTAINER}$"; then
  PREVIOUS_IMAGE="$(docker inspect --format='{{.Config.Image}}' "${APP_CONTAINER}" 2>/dev/null || true)"
fi

echo "Deploying tag ${TARGET_TAG}"
export IMAGE_TAG="${TARGET_TAG}"

docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" pull obgynrama-chatbot
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d

if wget -qO- http://127.0.0.1/healthz >/dev/null 2>&1; then
  echo "Deploy succeeded"
  exit 0
fi

echo "Health check failed"
docker logs --tail 120 "$APP_CONTAINER" || true

if [ -z "$PREVIOUS_IMAGE" ]; then
  echo "No previous image found. Cannot rollback automatically"
  exit 1
fi

PREVIOUS_TAG="${PREVIOUS_IMAGE##*:}"
echo "Rolling back to ${PREVIOUS_TAG}"
export IMAGE_TAG="$PREVIOUS_TAG"

docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d

if wget -qO- http://127.0.0.1/healthz >/dev/null 2>&1; then
  echo "Rollback succeeded"
  exit 1
fi

echo "Rollback failed. Manual intervention required"
docker logs --tail 120 "$APP_CONTAINER" || true
exit 1
