<!-- AI-DRAFT · Created via Claude Code · Model: claude-opus-4-8 · 2026-07-30 -->

# Reading the database

> 🔴 **PDPA:** the DB links a LINE user id to their diagnosis group — health-related
> personal data. Inspect only when needed, keep copies secure, prefer aggregate queries,
> and delete any exported file as soon as you're done.

## What's stored

The bot uses **SQLite** (a single file at `/app/data/users.db` inside the container,
persisted on the `chatbot_data` Docker volume). It holds one table that matters at runtime:

| Table | Purpose |
| --- | --- |
| `user_diagnosis` | LINE user id → chosen diagnosis (`d1`…`d5`), set when a user picks a DX |
| `faq_reply`, `faq_match_phrase`, `faq_quick_reply` | **legacy / unused** — FAQ now answers from `configs/faq_seed.yaml` in memory |

So the only live data is `user_diagnosis`. FAQ content lives in config, not the DB.

---

## Option A — DBeaver (snapshot)

DBeaver reads SQLite, but the DB is a **file** on the server (not a network service), so you
can't point DBeaver at it over an SSH tunnel. Pull a copy down and open it locally.

1. Pull a snapshot (from your machine):
   ```bash
   ./scripts/pull-db.sh root@<VPS_IP>
   # -> saves users-YYYYmmdd-HHMMSS.db in the current dir
   ```
   (or manually: `docker cp obgynrama-chatbot:/app/data/users.db /tmp/users.db` on the server,
   then `scp root@<VPS_IP>:/tmp/users.db .`)
2. DBeaver → **New Database Connection → SQLite** → set the file path to the downloaded
   `.db` → (download the driver if prompted) → **Connect**.
3. Open in **read-only** — it's a copy; edits don't reach prod, but this avoids confusion.
4. **Delete the copy when done:** `rm users-*.db` (and `rm /tmp/users.db` on the server if used).

> It's a snapshot at pull time, not live. Re-pull to refresh.

---

## Option B — sqlite3 on the server (quick, PII-safe)

For a quick look without downloading, query on the server. The runtime image has no `sqlite3`,
so copy the file out first and query the copy:

```bash
docker cp obgynrama-chatbot:/app/data/users.db /tmp/users.db
apt-get install -y sqlite3   # once

# aggregate only — no personal ids
sqlite3 /tmp/users.db "SELECT diagnosis, COUNT(*) FROM user_diagnosis GROUP BY diagnosis;"
sqlite3 /tmp/users.db "SELECT COUNT(*) AS total FROM user_diagnosis;"

rm /tmp/users.db
```

Avoid `SELECT line_user_id …` unless strictly necessary (it's personal data).

---

## Should we use SQLite at all?

For this service today — **yes, SQLite is a good fit**:

- one small table (`user_diagnosis`), single app instance, low traffic
- no separate DB server to run/patch/secure
- persists fine on the Docker volume

**Consider moving to PostgreSQL (managed) only if** one of these becomes true:

| Trigger | Why Postgres helps |
| --- | --- |
| Run **more than one** app instance / replicas | SQLite is single-file, single-writer — not shared across instances |
| **High concurrent** writes | SQLite serializes writes (`SetMaxOpenConns(1)`) |
| Need **easy remote access / BI / dashboards** | Postgres is a network DB — DBeaver/Metabase connect directly (over SSH tunnel) |
| Stronger **durability / PITR / managed backups** required by compliance | Managed Postgres gives automated backups & point-in-time recovery |

Until then, the pragmatic setup is **SQLite + a backup routine** (see below) — not a migration.

### The one gap to close: backups

The volume survives restarts/redeploys but there's no off-server backup yet. Recommended:
a nightly copy of `users.db` to off-server storage (encrypted). Ask to add `scripts/backup_db.sh`
+ a cron entry when ready.

---

## Files

- `scripts/pull-db.sh` — pull a snapshot to your machine for DBeaver
- DB path in container: `/app/data/users.db` (volume `chatbot_data`)
- `DB_PATH` is set in `.env.prod`
