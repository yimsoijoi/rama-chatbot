<!-- AI-DRAFT · Created via Claude Code · Model: claude-opus-4-8 · 2026-07-22 -->

# Deploy Runbook — bot.ppakzv.com (OVH + Cloudflare)

> **Status: AI-DRAFT** — ทำตามลำดับจากบนลงล่าง กา `[x]` ไปเรื่อยๆ
> ทำจาก **คอมส่วนตัว + เน็ตส่วนตัว** (เป็นโปรเจกต์ส่วนตัว ไม่ควรใช้เครื่อง/เน็ตบริษัท)

## สิ่งที่มีแล้ว / ต้องเตรียม
- ✅ โค้ด + rich menu + config อยู่บน branch `main`
- ✅ VPS ที่ OVH (Ubuntu) — ต้องมี **public IPv4** (จากอีเมล OVH หรือ OVH panel)
- ✅ Domain `ppakzv.com` ที่ Cloudflare
- ✅ Rotate LINE secret/token แล้ว — เตรียมค่าใหม่ไว้กรอก
- 🎯 เป้าหมาย: บอตออนไลน์ที่ `https://bot.ppakzv.com`

หา IP ของ VPS: อีเมล OVH / OVH panel → VPS / หรือรันบน server `curl -4 ifconfig.me`

---

## ขั้น 1 — ตั้ง DNS ที่ Cloudflare ⚠️ จุดพลาดบ่อยสุด

- [ ] dash.cloudflare.com → เลือก **ppakzv.com** → **DNS → Records → Add record**
- [ ] กรอก:
  - **Type:** `A`
  - **Name:** `bot`  → ได้ `bot.ppakzv.com`
  - **IPv4 address:** `<VPS_IP>` (public IPv4 ของ VPS)
  - **Proxy status:** **DNS only (เมฆสีเทา)** ⚠️ ห้ามเป็นเมฆส้ม!
  - **TTL:** Auto
- [ ] Save

> 🔴 **ต้องเป็นเมฆเทา (DNS only)** — ถ้าเป็นเมฆส้ม (Proxied) Caddy จะขอ TLS cert ไม่ได้ → deploy พัง
> เพราะเราให้ Caddy บน server จัดการ HTTPS เอง

ตรวจว่า DNS ชี้ถูก (รอ 2–5 นาที):
```bash
dig +short bot.ppakzv.com     # ต้องได้ <VPS_IP>
```

---

## ขั้น 2 — SSH เข้า VPS

```bash
ssh root@<VPS_IP>
```
- [ ] ใส่ password จากอีเมล OVH (พิมพ์แล้วจอไม่ขึ้นตัวอักษร = ปกติ)
- [ ] ครั้งแรกถาม fingerprint → พิมพ์ `yes`
- [ ] ถ้า OVH บังคับเปลี่ยน password → ตั้งใหม่แล้วจำไว้

> ถ้า user ในอีเมลเป็น `ubuntu` (ไม่ใช่ root) → `ssh ubuntu@<VPS_IP>` และเติม `sudo` หน้าคำสั่งในขั้น 3

---

## ขั้น 3 — ติดตั้ง Docker + clone repo (คำสั่งเดียว)

```bash
curl -fsSL https://raw.githubusercontent.com/yimsoijoi/rama-chatbot/main/deploy/provision.sh | bash
```
สคริปต์จะ: ติดตั้ง Docker (+ compose) + git → clone ไป `/opt/rama-chatbot` → สร้าง `.env.prod`
(มี fallback ติดตั้ง Docker จาก package Ubuntu ถ้า get.docker.com ไม่รองรับ 26.04)

- [ ] เช็ค: `docker --version && docker compose version` ขึ้นเวอร์ชันทั้งคู่

---

## ขั้น 4 — กรอก secret ลง .env.prod

```bash
nano /opt/rama-chatbot/.env.prod
```
แก้ 3 ค่า (ที่เหลือถูกตั้งไว้ให้แล้ว):
```
LINE_CHANNEL_SECRET=<secret ที่ rotate แล้ว>
LINE_CHANNEL_TOKEN=<token ที่ rotate แล้ว>
DOMAIN=bot.ppakzv.com
```
- [ ] เซฟใน nano: `Ctrl+O` → `Enter` → `Ctrl+X`

> `.env.prod` ถูก gitignore + chmod 600 แล้ว — จะไม่หลุดขึ้น git

---

## ขั้น 5 — เปิด container image ให้ดึงได้ (ครั้งเดียว)

- [ ] GitHub → repo `yimsoijoi/rama-chatbot` → **Packages** → package **rama-chatbot**
      → **Package settings** → **Change visibility → Public**

> ถ้าไม่เจอ package = CI ยัง build ไม่เสร็จ → ดูแท็บ **Actions** ให้เขียวก่อน
> (ทางเลือก: ไม่เปิด public ก็ได้ แต่ต้อง `docker login ghcr.io` ด้วย GitHub token บน server)

---

## ขั้น 6 — Deploy

```bash
cd /opt/rama-chatbot
./scripts/deploy_with_rollback.sh latest
```
- [ ] รอ Caddy ขอ TLS cert (สักครู่) แล้วเช็ค:
```bash
curl -I https://bot.ppakzv.com/healthz     # ต้องได้ HTTP/2 200
```

---

## ขั้น 7 — ต่อ LINE webhook

LINE Developers Console → channel → **Messaging API**:
- [ ] **Webhook URL** = `https://bot.ppakzv.com/webhook` → Update → **Verify** (ต้อง Success)
- [ ] **Use webhook** = ON
- [ ] OA Manager → Response settings: **Bot mode**, **Auto-response = Off**

---

## ขั้น 8 — ทดสอบ flow จริง

- [ ] แอด/เปิดแชท OBGYN RAMA → เห็นเมนู selector
- [ ] กด "เลือก DX1" → บอตตอบยืนยัน + เมนูสลับเป็น DX1
- [ ] กดปุ่มคำถามใน DX1 → ได้คำตอบ (ไม่ใช่ fallback)
- [ ] พิมพ์คำฉุกเฉิน (เลือดออก / ปวดมาก) → ขึ้นข้อความ escalation

---

## ขั้น 9 — ความปลอดภัย (แนะนำ ทำหลัง deploy สำเร็จ)

- [ ] เปลี่ยนไปใช้ SSH key + ปิด password login
- [ ] เปิด backup (OVH snapshot และ/หรือ backup DB — คุยเรื่องนี้ทีหลัง)
- [ ] ตรวจว่า `ENABLE_PPROF=false` (ตั้งไว้แล้วใน template)

---

## แก้ปัญหาที่เจอบ่อย

| อาการ | สาเหตุ / วิธีแก้ |
| --- | --- |
| `curl healthz` ไม่ได้ 200 / TLS error | Cloudflare ยังเป็นเมฆส้ม → เปลี่ยนเป็น DNS only; หรือ DNS ยังไม่ propagate (`dig` เช็ค) |
| `docker pull` failed / denied | ยังไม่เปิด package เป็น Public (ขั้น 5) หรือยังไม่ `docker login` |
| Webhook Verify ไม่ผ่าน | server ยังไม่ขึ้น / healthz ไม่ 200 / URL พิมพ์ผิด |
| กดเมนูแล้วบอตเงียบ | Use webhook ยังไม่ ON / Auto-response ยังเปิด / DB_PATH ไม่ได้ตั้ง |
| เลือก DX แล้วเมนูไม่สลับ | `rich_menu_id` ใน faq_seed.yaml ว่าง หรือ DB_PATH ไม่ได้ตั้ง |

## อ้างอิง
- Checklist ภาพรวม: [START-HERE-deploy-checklist.md](./START-HERE-deploy-checklist.md)
- ตั้งค่า console/choose-DX: [line-console-choose-dx-setup.md](./line-console-choose-dx-setup.md)
- Rich menu: [line-rich-menu-dx1-guide.md](./line-rich-menu-dx1-guide.md)
