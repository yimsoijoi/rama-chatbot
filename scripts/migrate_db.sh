#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-${DB_PATH:-data/users.db}}"

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "sqlite3 command not found. Please install sqlite3 first." >&2
  exit 1
fi

mkdir -p "$(dirname "$DB_PATH")"

sqlite3 "$DB_PATH" <<'SQL'
CREATE TABLE IF NOT EXISTS user_diagnosis (
  line_user_id TEXT PRIMARY KEY,
  diagnosis    TEXT NOT NULL,
  updated_at   DATETIME DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS faq_reply (
  diagnosis TEXT NOT NULL,
  faq_key   TEXT NOT NULL,
  question  TEXT NOT NULL DEFAULT '',
  category  TEXT NOT NULL DEFAULT '',
  answer    TEXT NOT NULL,
  PRIMARY KEY (diagnosis, faq_key)
);

CREATE TABLE IF NOT EXISTS faq_match_phrase (
  diagnosis TEXT NOT NULL,
  faq_key   TEXT NOT NULL,
  phrase    TEXT NOT NULL,
  PRIMARY KEY (diagnosis, faq_key, phrase)
);

CREATE TABLE IF NOT EXISTS faq_quick_reply (
  diagnosis   TEXT NOT NULL,
  faq_key     TEXT NOT NULL,
  quick_reply TEXT NOT NULL,
  PRIMARY KEY (diagnosis, faq_key, quick_reply)
);
SQL

question_col_exists="$(sqlite3 "$DB_PATH" "SELECT COUNT(1) FROM pragma_table_info('faq_reply') WHERE name='question';")"
if [[ "$question_col_exists" == "0" ]]; then
  sqlite3 "$DB_PATH" "ALTER TABLE faq_reply ADD COLUMN question TEXT NOT NULL DEFAULT '';"
fi

category_col_exists="$(sqlite3 "$DB_PATH" "SELECT COUNT(1) FROM pragma_table_info('faq_reply') WHERE name='category';")"
if [[ "$category_col_exists" == "0" ]]; then
  sqlite3 "$DB_PATH" "ALTER TABLE faq_reply ADD COLUMN category TEXT NOT NULL DEFAULT '';"
fi

echo "Migration complete: $DB_PATH"
