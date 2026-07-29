#!/usr/bin/env bash
# Pull a snapshot of the prod SQLite DB to inspect locally (e.g. in DBeaver).
# Run from your own machine.
#
# Usage:  ./scripts/pull-db.sh root@<VPS_IP> [container_name]
#
# PDPA: the file contains patient data (LINE id <-> diagnosis). Open read-only
# and delete the local copy when done.
set -euo pipefail

HOST="${1:?usage: pull-db.sh user@host [container_name]}"
CONTAINER="${2:-obgynrama-chatbot}"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT="users-${STAMP}.db"

echo "==> copying DB out of the container on ${HOST}..."
ssh "$HOST" "docker cp ${CONTAINER}:/app/data/users.db /tmp/_dbpull.db"

echo "==> downloading snapshot..."
scp "${HOST}:/tmp/_dbpull.db" "./${OUT}"
ssh "$HOST" "rm -f /tmp/_dbpull.db"

echo
echo "Saved snapshot: ./${OUT}"
echo "Open it in DBeaver (SQLite driver) — read-only."
echo "PDPA: delete when done ->  rm ./${OUT}"
