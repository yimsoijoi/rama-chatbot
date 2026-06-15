<!-- AI-DRAFT · Created via Claude Code · Model: claude-opus-4-8 · 2026-06-15 -->

# DX1 Rich Menu & Self-Service DX Selection (LINE Messaging API)

> **Status: AI-DRAFT** — review wording and clinical copy with the team before publishing.

This guide finishes the **self-service DX1 flow**: a new user picks their result group from a
**default menu**, the bot saves it and swaps them to the **DX1 content menu** (FAQ tiles), and taps
on that menu return answers.

## The flow

```
New friend ──> DEFAULT menu (เลือก DX1..DX5) ──tap "เลือก DX1"──> bot saves dx=d1,
            links DX1 content menu (per-user, overrides default) ──> taps return FAQ answers
```

- The **default** menu is shown to everyone who hasn't picked a DX yet.
- Tapping a DX button sends a text the bot already understands — `parseDiagnosisSelection`
  ([reply_usecase.go:159](../internal/usecase/reply_usecase.go#L159)) accepts `เลือก DX1`, `DX1`,
  `d1`, etc.
- The bot saves the choice, then `LinkUserRichMenu`
  ([line_handler.go:163](../internal/interface/http/line_handler.go#L163)) links the DX1 content menu
  to that user. A **per-user** menu overrides the default and appears immediately.

## ⚠️ Two hard requirements before any of this works

1. **Create rich menus via the Messaging API, not the OA Manager console.** LINE keeps them
   separate: a console-created menu is *"retrievable and editable only through the Official Account
   Manager"* and is invisible to the Messaging API, so its ID can never be used by `LinkUserRichMenu`.
2. **`DB_PATH` must be set when running the server.** Saving the user's DX choice writes to SQLite
   ([reply_usecase.go:45-49](../internal/usecase/reply_usecase.go#L45)); if `DB_PATH` is empty the bot
   replies *"ระบบยังไม่พร้อมบันทึกผลตรวจ"* and the DX is never saved or linked. Run the schema first:
   ```bash
   ./scripts/migrate_db.sh data/users.db   # creates tables
   # then run the server with DB_PATH=data/users.db
   ```

> **Clinical-safety note:** patients self-selecting a diagnosis group can pick the wrong one. Label
> the default-menu buttons with the plain-language result the patient was given (and/or "ดูจากใบผล
> ตรวจของคุณ"), keep the 🚨 emergency + 💬 staff options reachable, and have the clinical team approve
> the labels.

---

## Console & channel settings (do these once)

**LINE Developers Console → your channel → Messaging API tab**

| Setting | Value |
| --- | --- |
| Webhook URL | `https://<your-domain>/webhook` (then click **Verify**) |
| Use webhook | **Enabled** |
| Channel access token (long-lived) | issue → `LINE_CHANNEL_TOKEN` |
| Channel secret (Basic settings) | → `LINE_CHANNEL_SECRET` |

**LINE Official Account Manager → Settings → Response settings**

| Setting | Value |
| --- | --- |
| Response mode | **Bot** |
| Webhooks | **On** |
| Auto-response messages | **Off** (so canned replies don't fight the bot) |
| Greeting message | Optional — e.g. "กดเลือกกลุ่มผลตรวจของคุณจากเมนูด้านล่างค่ะ" |

---

## Prerequisites for the API calls

```bash
export LINE_CHANNEL_TOKEN="<long-lived channel access token>"   # secret — never paste into chat/tickets/logs
```

`curl` + `jq` installed. Two background images ready: `dx-select.png` and `dx1-richmenu.png`.

### Image requirements

| Item | Value |
| --- | --- |
| Recommended size | **2500 × 1686 px** (large, 3×2 grid) |
| Width / height | 800–2500 px wide, ≥ 250 px tall, ratio width÷height ≥ 1.45 |
| Format / size | JPEG or PNG, ≤ 1 MB |

Tap-area `bounds` below assume a 2500 × 1686 canvas (cols at x=0/833/1666, rows at y=0/843).

---

## Part A — Default menu: DX self-selection

### A1. Define `dx-select.json`

Five DX buttons + a help tile. Each DX button sends `เลือก DXn` (the bot maps it to the diagnosis).

```json
{
  "size": { "width": 2500, "height": 1686 },
  "selected": true,
  "name": "DX Selector (default)",
  "chatBarText": "เลือกกลุ่มผลตรวจ",
  "areas": [
    { "bounds": { "x": 0,    "y": 0,   "width": 833, "height": 843 },
      "action": { "type": "message", "text": "เลือก DX1" } },
    { "bounds": { "x": 833,  "y": 0,   "width": 833, "height": 843 },
      "action": { "type": "message", "text": "เลือก DX2" } },
    { "bounds": { "x": 1666, "y": 0,   "width": 834, "height": 843 },
      "action": { "type": "message", "text": "เลือก DX3" } },
    { "bounds": { "x": 0,    "y": 843, "width": 833, "height": 843 },
      "action": { "type": "message", "text": "เลือก DX4" } },
    { "bounds": { "x": 833,  "y": 843, "width": 833, "height": 843 },
      "action": { "type": "message", "text": "เลือก DX5" } },
    { "bounds": { "x": 1666, "y": 843, "width": 834, "height": 843 },
      "action": { "type": "message", "text": "ต้องการคุยกับเจ้าหน้าที่โดยตรง" } }
  ]
}
```

Label each cell on the artwork in plain Thai, e.g. DX1 = "Pap ปกติ · พบ HPV", DX2 = "LSIL จากแปป",
DX3 = "CIN1 จากชิ้นเนื้อ", DX4 = "ASCUS · HPV ลบ", DX5 = "ผิดปกติ · ส่องกล้องปกติ".

### A2. Create it, upload image, set as **default**

```bash
SELECT_ID=$(curl -s -X POST https://api.line.me/v2/bot/richmenu \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" -H "Content-Type: application/json" \
  -d @dx-select.json | jq -r .richMenuId)
echo "$SELECT_ID"

curl -s -X POST "https://api-data.line.me/v2/bot/richmenu/$SELECT_ID/content" \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" -H "Content-Type: image/png" \
  --data-binary @dx-select.png

# Make it the account-wide default (shown until a user picks a DX)
curl -s -X POST "https://api.line.me/v2/bot/user/all/richmenu/$SELECT_ID" \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN"
```

---

## Part B — DX1 content menu (FAQ tiles)

Each tap sends a **full question** — the matcher is a case-insensitive substring check against each
FAQ's `match_phrases` (code + full question) in
[configs/faq_seed.yaml](../configs/faq_seed.yaml). Use the full question (codes like `D1-Q1` collide
with `D1-Q10`).

| Cell | Label (short) | `message` text (full question) | FAQ |
| --- | --- | --- | --- |
| 1 | Pap ปกติ แต่ HPV บวก? | `ผล Pap ปกติ แต่ทำไมแพทย์บอกว่ามี HPV?` | D1-Q1 |
| 2 | HPV หายเองได้ไหม? | `ถ้า Pap ปกติ แล้ว HPV จะหายเองได้ไหม?` | D1-Q3 |
| 3 | ตรวจซ้ำเมื่อไหร่? | `ต้องมาตรวจซ้ำเมื่อไหร่ และตรวจอะไรบ้าง?` | D1-Q5 |
| 4 | ระหว่างรอระวังอะไร? | `ระหว่างรอตรวจซ้ำ ต้องระวังอะไรบ้าง?` | D1-Q6 |
| 5 | 💬 คุยกับเจ้าหน้าที่ | `ต้องการคุยกับเจ้าหน้าที่โดยตรง` | S6 |
| 6 | 🚨 อาการฉุกเฉิน | `อาการแบบไหนที่ต้องรีบมาพบแพทย์ฉุกเฉิน? (ทุก Diagnosis)` | S1 |

### B1. Define `dx1-richmenu.json`

```json
{
  "size": { "width": 2500, "height": 1686 },
  "selected": false,
  "name": "DX1 - NILM + hrHPV Other High Risk",
  "chatBarText": "เมนูช่วยเหลือ",
  "areas": [
    { "bounds": { "x": 0,    "y": 0,   "width": 833, "height": 843 },
      "action": { "type": "message", "text": "ผล Pap ปกติ แต่ทำไมแพทย์บอกว่ามี HPV?" } },
    { "bounds": { "x": 833,  "y": 0,   "width": 833, "height": 843 },
      "action": { "type": "message", "text": "ถ้า Pap ปกติ แล้ว HPV จะหายเองได้ไหม?" } },
    { "bounds": { "x": 1666, "y": 0,   "width": 834, "height": 843 },
      "action": { "type": "message", "text": "ต้องมาตรวจซ้ำเมื่อไหร่ และตรวจอะไรบ้าง?" } },
    { "bounds": { "x": 0,    "y": 843, "width": 833, "height": 843 },
      "action": { "type": "message", "text": "ระหว่างรอตรวจซ้ำ ต้องระวังอะไรบ้าง?" } },
    { "bounds": { "x": 833,  "y": 843, "width": 833, "height": 843 },
      "action": { "type": "message", "text": "ต้องการคุยกับเจ้าหน้าที่โดยตรง" } },
    { "bounds": { "x": 1666, "y": 843, "width": 834, "height": 843 },
      "action": { "type": "message", "text": "อาการแบบไหนที่ต้องรีบมาพบแพทย์ฉุกเฉิน? (ทุก Diagnosis)" } }
  ]
}
```

### B2. Create it and upload its image

```bash
DX1_ID=$(curl -s -X POST https://api.line.me/v2/bot/richmenu \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" -H "Content-Type: application/json" \
  -d @dx1-richmenu.json | jq -r .richMenuId)
echo "$DX1_ID"

curl -s -X POST "https://api-data.line.me/v2/bot/richmenu/$DX1_ID/content" \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" -H "Content-Type: image/png" \
  --data-binary @dx1-richmenu.png
```

> Do **not** set this one as default — it's linked per user by the bot when they pick DX1.

### B3. Wire the ID into the repo

Paste `$DX1_ID` into [configs/faq_seed.yaml](../configs/faq_seed.yaml):

```yaml
diagnoses:
  d1:
    name: NILM + hrHPV Other High Risk Positive
    rich_menu_id: "richmenu-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"   # = $DX1_ID
```

Redeploy with `DB_PATH` set.

---

## Verify the full flow

1. As a fresh test user, open the OA → you should see the **DX selector** default menu.
2. Tap **เลือก DX1** → bot replies "บันทึกกลุ่มผลตรวจเรียบร้อยค่ะ…" and the menu switches to the
   **DX1** content menu (per-user link). Check logs for `rich_menu_linked`.
3. Tap each DX1 cell → confirm the matching answer (not `fallback_reply`).
4. Type an urgent word (`เลือดออก`, `ปวดมาก`, `ไข้สูง`) → confirm the escalation reply fires.
5. Test on Android **and** iOS.

**Switching DX later:** the per-user DX1 menu hides the default selector, so to change groups a user
re-sends `เลือก DX2` (etc.) — consider noting this in the staff-handoff reply, or unlink with
`DELETE /v2/bot/user/{userId}/richmenu` to show the selector again.

---

## Repeat for DX2–DX5

Duplicate `dx1-richmenu.json`, swap `name`, `chatBarText`, and the four DX-specific questions
(DX2 → `D2-Q*`, etc.), keep cells 5–6, run B2–B3 for each, and paste every ID into the matching
`diagnoses.dX.rich_menu_id`. The default selector (Part A) is created only once.

## Managing rich menus

```bash
# List Messaging-API rich menus (console-created ones never appear here)
curl -s https://api.line.me/v2/bot/richmenu/list \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" | jq '.richmenus[] | {richMenuId, name}'

# Delete by ID
curl -s -X DELETE "https://api.line.me/v2/bot/richmenu/$DX1_ID" \
  -H "Authorization: Bearer $LINE_CHANNEL_TOKEN"
```

## Governance reminders

- No patient data in menu artwork or labels.
- Keep staff-handoff + emergency options on every menu.
- Clinical team reviews all Thai copy and DX-selector labels before publishing.

---

Sources:
- [Use rich menus — LINE Developers](https://developers.line.biz/en/docs/messaging-api/using-rich-menus/)
- [Use per-user rich menus — LINE Developers](https://developers.line.biz/en/docs/messaging-api/use-per-user-rich-menus/)
- [Rich menus overview — LINE Developers](https://developers.line.biz/en/docs/messaging-api/rich-menus-overview/)
