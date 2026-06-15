<!-- AI-DRAFT · Created via Claude Code · Model: claude-opus-4-8 · 2026-06-15 -->

# LINE Console Setup — Finish the "Choose DX" Flow

> **Status: AI-DRAFT** — review wording and clinical copy with the team before publishing.

This is the **console checklist** to make the self-service flow work: a new user opens the OA, is
prompted to choose their result group (DX1–DX5), the bot saves it and shows that DX's menu.

> Rich menus themselves are **created via the Messaging API**, not the console — see
> [line-rich-menu-dx1-guide.md](./line-rich-menu-dx1-guide.md). This doc covers only the console
> settings + server config that the choose-DX flow depends on.

## Target flow

```
Add OA as friend → Greeting message → DX selector (default rich menu)
   → tap "เลือก DX1" → bot saves dx=d1 → DX1 content menu appears
```

For each step to work, three layers must be set: **(1) Messaging API channel**, **(2) Official
Account Manager response settings**, **(3) server config (DB)**.

---

## 1. LINE Developers Console — Messaging API channel

developers.line.biz → your provider → your channel → **Messaging API** tab (and **Basic settings**).

| Setting | Value | Notes |
| --- | --- | --- |
| **Webhook URL** | `https://<your-domain>/webhook` | must be HTTPS; click **Verify** after saving |
| **Use webhook** | **Enabled** | without this, no taps/messages reach the bot |
| **Channel access token** (long-lived) | Issue → copy | → `LINE_CHANNEL_TOKEN` in `.env` |
| **Channel secret** (Basic settings) | copy | → `LINE_CHANNEL_SECRET` (signature check) |
| **Auto-reply / greeting (LINE-managed)** | leave off here | configured in OA Manager instead (below) |

Verify should return success against your deployed `/webhook`. If it fails, fix the URL/deploy
before continuing — nothing else will work.

---

## 2. LINE Official Account Manager — Response settings

manager.line.biz → your OA → **Settings → Response settings** (การตั้งค่าการตอบกลับ).

| Setting | Value | Why |
| --- | --- | --- |
| **Response mode** | **Bot** (แชทบอท) | lets the webhook drive replies |
| **Webhooks** | **On** | required for the bot to receive events |
| **Auto-response messages** | **Off** | otherwise LINE's canned replies compete with the bot |
| **Greeting message** | **On** | shown once when a user adds the OA — use it to prompt DX choice |

### Suggested greeting message (Thai)

Set under **Home → Greeting messages**:

```
สวัสดีค่ะ 💙 ยินดีต้อนรับสู่หน่วย Colposcopy
กรุณากดเลือก "กลุ่มผลตรวจ" ของคุณจากเมนูด้านล่าง เพื่อรับข้อมูลที่ตรงกับผลของคุณค่ะ
หากไม่แน่ใจ ให้ดูจากใบผลตรวจ หรือกด 💬 คุยกับเจ้าหน้าที่
```

> The greeting only shows text — the **DX selector buttons** come from the **default rich menu** (set
> via the API, Part A of the rich-menu guide). The greeting tells the user to use that menu.

---

## 3. Server config — DX choice persistence (required)

Saving the chosen DX writes to SQLite ([reply_usecase.go:45-49](../internal/usecase/reply_usecase.go#L45)).
If `DB_PATH` is **not** set, the bot replies *"ระบบยังไม่พร้อมบันทึกผลตรวจ"* and the DX is never saved
or its menu linked — the flow silently breaks.

```bash
./scripts/migrate_db.sh data/users.db     # create tables once
# run the server with:
#   DB_PATH=data/users.db
```

Confirm `DB_PATH` is present in your deployment env (`.env` / compose / Helm) before going live.

---

## 4. The default rich menu (DX selector) — pointer

The DX-selection buttons are an **API-created default rich menu** (`POST /v2/bot/user/all/richmenu`).
Build and set it per **Part A** of [line-rich-menu-dx1-guide.md](./line-rich-menu-dx1-guide.md).
After a user picks a DX, the bot links that DX's per-user menu, which overrides the default.

Button text the bot understands (sent as `message` actions): `เลือก DX1` … `เลือก DX5` (also accepts
`DX1`, `d1`). Mapping handled in
[parseDiagnosisSelection](../internal/usecase/reply_usecase.go#L159).

---

## 5. Verify the choose-DX flow

1. Add the OA as a **new** friend → the **greeting** appears and the **DX selector** menu shows.
2. Tap **เลือก DX1** → bot replies "บันทึกกลุ่มผลตรวจเรียบร้อยค่ะ…" and the menu switches to **DX1**.
3. Check server logs for `rich_menu_linked` (success) or `rich_menu_link_failed` (token/ID issue).
4. Tap a DX1 cell → correct answer, not `fallback_reply`.
5. Test on Android **and** iOS.

### Common console mistakes

| Symptom | Likely cause |
| --- | --- |
| Bot never replies | Webhook **Use webhook** off, or wrong URL, or Response mode not **Bot** |
| LINE sends generic canned replies | **Auto-response messages** still **On** |
| "ระบบยังไม่พร้อมบันทึกผลตรวจ" after tapping a DX | `DB_PATH` not set / DB not migrated |
| Menu never changes after choosing DX | `rich_menu_id` empty in `faq_seed.yaml`, or menu created in OA Manager (not API) |
| Signature errors in logs | `LINE_CHANNEL_SECRET` mismatch |

## Governance reminders

- No patient data in menu artwork, labels, or greeting.
- Keep 🚨 emergency + 💬 staff-handoff reachable from every menu.
- Clinical team reviews greeting copy and DX-selector labels before publishing (patients self-select
  their group — wrong choice = wrong info).

---

Sources:
- [Use rich menus — LINE Developers](https://developers.line.biz/en/docs/messaging-api/using-rich-menus/)
- [Use per-user rich menus — LINE Developers](https://developers.line.biz/en/docs/messaging-api/use-per-user-rich-menus/)
