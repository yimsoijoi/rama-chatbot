#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-${DB_PATH:-data/users.db}}"
SEED_CONFIG_PATH="${2:-configs/faq_seed.yaml}"

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "sqlite3 command not found. Please install sqlite3 first." >&2
  exit 1
fi

"$(dirname "$0")/migrate_db.sh" "$DB_PATH"

sqlite3 "$DB_PATH" <<'SQL'
INSERT INTO user_diagnosis (line_user_id, diagnosis, updated_at)
VALUES ('U11111111111111111111111111111111', 'd1', datetime('now'))
ON CONFLICT(line_user_id) DO UPDATE SET
  diagnosis  = excluded.diagnosis,
  updated_at = excluded.updated_at;

INSERT INTO user_diagnosis (line_user_id, diagnosis, updated_at)
VALUES ('U22222222222222222222222222222222', 'd2', datetime('now'))
ON CONFLICT(line_user_id) DO UPDATE SET
  diagnosis  = excluded.diagnosis,
  updated_at = excluded.updated_at;
SQL

go run ./scripts/cmd/seed-faq-from-config "$DB_PATH" "$SEED_CONFIG_PATH"

echo "Seed complete: $DB_PATH (FAQ imported from $SEED_CONFIG_PATH)"
