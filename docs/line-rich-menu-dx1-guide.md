<!-- AI-DRAFT · Created via Claude Code · Model: claude-opus-4-8 · 2026-06-15 -->

# Creating the DX1 Rich Menu in the LINE Console

> **Status: AI-DRAFT** — review wording and clinical copy with the team before publishing.

This guide walks through building the **DX1 (NILM + hrHPV Other High Risk Positive)** rich menu
in the LINE Console, then wiring its ID back into this repo so the bot can link it per user.

## How the rich menu connects to this bot

The webhook handler ([internal/interface/http/line_handler.go](../internal/interface/http/line_handler.go))
only understands **text messages**. It matches the incoming text against each FAQ's `match_phrases`
in [configs/faq_seed.yaml](../configs/faq_seed.yaml), replies with the answer + quick replies, and —
when a DX is resolved — links that DX's rich menu via `LinkUserRichMenu(userID, rich_menu_id)`.

**Implication:** every tappable area in the DX1 rich menu must use a **Text** action whose text
matches one of an FAQ's `match_phrases`. In `faq_seed.yaml`, each item's match phrases are its
**code** (e.g. `D1-Q1`) and its **full question**. The matcher is a case-insensitive substring
check ([knowledge_repository.go:86](../internal/infrastructure/config/knowledge_repository.go#L86)),
so **use the full question text** as the action payload — short codes like `D1-Q1` can substring-
collide with `D1-Q10`. If the text matches nothing, the user gets the `fallback_reply`.

### DX1 questions the bot can answer

All DX1 items from the clinical script are loaded (`D1-Q1` … `D1-Q10`). Pick the most useful for the
menu. Suggested 6-tile layout (4 DX1 + shared handoff + emergency):

| Tile label (Thai, on artwork) | Text action sends (full question)                       | FAQ   |
| ----------------------------- | ------------------------------------------------------- | ----- |
| Pap ปกติ แต่ทำไม HPV บวก?      | `ผล Pap ปกติ แต่ทำไมแพทย์บอกว่ามี HPV?`                  | D1-Q1 |
| HPV หายเองได้ไหม?            | `ถ้า Pap ปกติ แล้ว HPV จะหายเองได้ไหม?`                  | D1-Q3 |
| ต้องตรวจซ้ำเมื่อไหร่?         | `ต้องมาตรวจซ้ำเมื่อไหร่ และตรวจอะไรบ้าง?`                | D1-Q5 |
| ระหว่างรอต้องระวังอะไร?       | `ระหว่างรอตรวจซ้ำ ต้องระวังอะไรบ้าง?`                    | D1-Q6 |
| 💬 คุยกับเจ้าหน้าที่           | `ต้องการคุยกับเจ้าหน้าที่โดยตรง`                          | S6    |
| 🚨 อาการฉุกเฉิน               | `อาการแบบไหนที่ต้องรีบมาพบแพทย์ฉุกเฉิน? (ทุก Diagnosis)` | S1    |

> The **menu bar/cell label** on your background image can be short and friendly; the **Text action
> payload** must be the full question above. Shared items (`S1`…`S6`) work on every DX menu, so reuse
> the handoff + emergency tiles across DX2–DX5.
>
> Separately, urgent free-text from users (`เลือดออก`, `ปวดมาก`, `ไข้สูง`, `ตกขาวผิดปกติ`, `กลัวมาก`)
> triggers the `escalation` reply regardless of the menu — see `escalation.keywords` in `faq_seed.yaml`.

---

## Prerequisites

- A LINE Official Account with the **Messaging API enabled** (the channel whose
  `LINE_CHANNEL_SECRET` / `LINE_CHANNEL_TOKEN` you set in `.env`).
- Admin access to **LINE Official Account Manager** → <https://manager.line.biz/>.
- A rich menu **background image** prepared (see specs below).
- The bot deployed and reachable so you can test taps end to end.

### Background image specs

| Item          | Value                                                         |
| ------------- | ------------------------------------------------------------- |
| Large size    | 2500 × 1686 px (recommended for a 6-tile 3×2 layout)          |
| Compact size  | 2500 × 843 px (good for a 3-tile single row)                  |
| Format        | JPEG or PNG                                                   |
| Max file size | 1 MB                                                          |
| Design notes  | Thai-readable, high contrast, one primary + one accent color  |

Keep the 🚨 emergency tile visually distinct (e.g. red) and always visible, per the healthcare UX
checklist in [line-ui-logo-guide.md](./line-ui-logo-guide.md).

---

## Step-by-step in the LINE Console (Official Account Manager)

### Step 1 — Open Rich Menus

1. Go to <https://manager.line.biz/> and select your Official Account.
2. Left sidebar → **Home** → **Rich menus** (เมนูริชเมนู).
3. Click **Create** (สร้างใหม่).

### Step 2 — Title and display settings

1. **Title** (internal reference, not shown to users): `DX1 - NILM + hrHPV Other High Risk`.
2. **Display period**: set a start date; leave end open (or per your campaign rules).
3. **Menu bar text** (label on the collapsed bar): e.g. `เมนูช่วยเหลือ`.
4. **Default behavior**: **Shown** so the menu opens by default.

### Step 3 — Choose layout (template)

1. Under **Content settings** → **Select a template**.
2. Pick the **Large** template with **6 areas (3×2)** to fit the 6 tiles above, or **2×2 (4 areas)**
   for core questions + handoff + emergency only.
3. Click **Select**.

### Step 4 — Upload the background image

1. Click **Upload image** (or **Create a background image** in the built-in editor).
2. Upload your PNG/JPEG (≤ 1 MB) sized for the chosen template.
3. Confirm each cell lines up with your labels.

### Step 5 — Set the action for each area (the critical part)

For **every** area set **Action type = Text** (ข้อความ), then paste the **full question** from the
table above. LINE sends that string as if the user typed it, and the webhook matches it against
`match_phrases`.

| Area | Action type | Text to enter                                            |
| ---- | ----------- | -------------------------------------------------------- |
| 1    | Text        | `ผล Pap ปกติ แต่ทำไมแพทย์บอกว่ามี HPV?`                   |
| 2    | Text        | `ถ้า Pap ปกติ แล้ว HPV จะหายเองได้ไหม?`                   |
| 3    | Text        | `ต้องมาตรวจซ้ำเมื่อไหร่ และตรวจอะไรบ้าง?`                 |
| 4    | Text        | `ระหว่างรอตรวจซ้ำ ต้องระวังอะไรบ้าง?`                     |
| 5    | Text        | `ต้องการคุยกับเจ้าหน้าที่โดยตรง`                           |
| 6    | Text        | `อาการแบบไหนที่ต้องรีบมาพบแพทย์ฉุกเฉิน? (ทุก Diagnosis)`  |

> ⚠️ Copy the question text **exactly** (including punctuation and the `(ทุก Diagnosis)` suffix on
> shared items) so the substring match succeeds.

### Step 6 — Swap in a different DX1 question

Want a different tile? Any `D1-Q*` question in `faq_seed.yaml` works — just paste that item's full
`question` value as the Text action. No code or config change needed (all items are already loaded).

### Step 7 — Save and publish

1. Click **Save** (บันทึก).
2. Open the OA in the LINE app and tap each tile to confirm it returns the right answer (not the
   fallback).

---

## Step 8 — Get the rich menu ID and wire it into the repo

The console does **not** display the rich menu ID, but the bot's per-user linking
(`rich_menu_id` in `faq_seed.yaml`) needs it. Fetch it via the Messaging API.

1. List rich menus on the channel (same token as `LINE_CHANNEL_TOKEN`):

   ```bash
   curl -s -X GET https://api.line.me/v2/bot/richmenu/list \
     -H "Authorization: Bearer $LINE_CHANNEL_TOKEN" | jq .
   ```

   > Do not paste the token into chat, tickets, or shared logs — treat it as a secret.

2. Find the entry whose `name`/`chatBarText` matches the DX1 menu and copy its `richMenuId`
   (looks like `richmenu-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`).

3. Paste it into [configs/faq_seed.yaml](../configs/faq_seed.yaml) under the `diagnoses` block:

   ```yaml
   diagnoses:
     d1:
       name: NILM + hrHPV Other High Risk Positive
       rich_menu_id: "richmenu-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"   # DX1 menu
   ```

4. Redeploy. When a DX1 user is resolved, the handler links this menu to that user automatically.

---

## Step 9 — Verify end to end

1. In the LINE app, open the OA chat as a DX1 test user.
2. Confirm the DX1 rich menu shows.
3. Tap each tile and confirm the bot returns the matching answer (not `fallback_reply`).
4. Check the bot logs for `rich_menu_linked` (success) or `rich_menu_link_failed` (investigate
   token/ID).
5. Test on both Android and iOS LINE clients.

---

## Repeat for DX2–DX5

Each DX has its own `rich_menu_id: ""` placeholder in the `diagnoses` block of `faq_seed.yaml`.
Repeat steps 1–8 per diagnosis, mapping tiles to that DX's question text (e.g. DX2 uses `D2-Q*`
questions). Reuse the same shared tiles (`ต้องการคุยกับเจ้าหน้าที่โดยตรง`,
`อาการแบบไหนที่ต้องรีบมาพบแพทย์ฉุกเฉิน? (ทุก Diagnosis)`) on every menu.

## Governance reminders

- Do not embed patient data in menu artwork or labels.
- Keep a staff-handoff and emergency option on every menu (healthcare UX requirement).
- Have the clinical team review all Thai copy before publishing.
