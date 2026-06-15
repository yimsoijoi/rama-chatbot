#!/usr/bin/env sh
set -eu

APP_NAME="obgynrama-chatbot"
IMAGE_TAG="${1:-latest}"
IMAGE_REGISTRY="${IMAGE_REGISTRY:-ghcr.io}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-owner/obgynrama-chatbot}"
IMAGE="${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}:${IMAGE_TAG}"

if [ ! -f .env ]; then
  echo "Missing .env in current directory"
  exit 1
fi

echo "Pulling image: ${IMAGE}"
docker pull "${IMAGE}"

echo "Stopping old container if exists"
docker rm -f "${APP_NAME}" >/dev/null 2>&1 || true

echo "Starting new container"
docker run -d \
  --name "${APP_NAME}" \
  --restart unless-stopped \
  --env-file .env \
  -p 8080:8080 \
  "${IMAGE}"

echo "Waiting for health endpoint"
for i in 1 2 3 4 5 6 7 8 9 10; do
  if wget -qO- http://127.0.0.1:8080/healthz >/dev/null 2>&1; then
    echo "Deploy succeeded"
    exit 0
  fi
  sleep 2
done

echo "Health check failed. Printing logs"
docker logs --tail 100 "${APP_NAME}" || true
exit 1
