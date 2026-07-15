<!-- AI-DRAFT · Created via Claude Code · Model: claude-opus-4-8 · 2026-06-15 -->

# 🚀 START HERE — คู่มือขึ้นระบบทีละขั้น (สำหรับผู้ไม่เคย deploy)

ไล่จากบนลงล่าง ทำเสร็จข้อไหนกา `[x]` ไว้ จะได้ไม่หลงว่าทำถึงไหน
แต่ละเฟสมีลิงก์ไปเอกสารละเอียด

> ⚠️ ระบบนี้เกี่ยวกับข้อมูลสุขภาพคนไข้ — ก่อน go-live จริงกับคนไข้ ควรมีข้อความ **ขอความยินยอม (consent)**
> และคำเตือนว่าเป็นข้อมูลทั่วไปไม่ใช่คำวินิจฉัย และมีช่องทางติดต่อเจ้าหน้าที่เสมอ

---

## เฟส 0 — ทดสอบก่อนเสียเงิน (แนะนำมาก, ~30 นาที)

พิสูจน์ว่าบอตทำงานก่อนซื้อ server/domain

- [x] ติดตั้ง Go + Docker ที่เครื่องตัวเอง
- [x] สร้าง LINE Official Account + Messaging API channel (ได้ `LINE_CHANNEL_SECRET`, `LINE_CHANNEL_TOKEN`)
- [ ] รันบอตที่เครื่อง:
  ```bash
  ./scripts/migrate_db.sh data/users.db
  cp .env.example .env    # ใส่ SECRET/TOKEN, ตั้ง DB_PATH=data/users.db
  set -a && source .env && set +a
  go run ./cmd/server
  ```
- [ ] เปิด tunnel: `cloudflared tunnel --url http://localhost:8080` → ได้ URL `https://xxx.trycloudflare.com`
- [ ] ตั้ง Webhook URL = `https://xxx.trycloudflare.com/webhook` → กด Verify
- [ ] ทักบอตใน LINE แล้วมันตอบ → ✅ ผ่าน ไปเฟสจริงได้

รายละเอียด: [local-testing-guide.md](./local-testing-guide.md)

---

## เฟส 1 — ซื้อ domain + server (~1 ชม. + รอ DNS)

**คำแนะนำสำหรับมือใหม่ (เลือกอย่างละ 1):**

| อย่าง | แนะนำ | ราคาโดยประมาณ |
| --- | --- | --- |
| Domain | Cloudflare Registrar หรือ Namecheap (`.com`) | ~350 บาท/ปี |
| Server (VPS) | DigitalOcean (UI ง่ายสุด) หรือ Hetzner (ถูกสุด) | ~200 บาท/เดือน |
| สเปก | 1–2 vCPU / 2 GB RAM / Ubuntu 22.04 LTS | |

- [ ] ซื้อ domain
- [ ] สร้าง VPS (Ubuntu 22.04) จด **public IP** ไว้
- [ ] ตั้ง DNS: สร้าง **A record** `@` → public IP ของ server (รอ 5–30 นาที)

รายละเอียด + รายชื่อผู้ให้บริการ: [deployment-beginner-guide.md](./deployment-beginner-guide.md) ข้อ 1–4

---

## เฟส 2 — เตรียม server (~30 นาที)

- [ ] SSH เข้า server: `ssh root@public-ip`
- [ ] ติดตั้ง Docker + compose plugin (คำสั่งพร้อมคัดลอกในคู่มือ ข้อ 5)
- [ ] สร้างโฟลเดอร์ `/opt/obgynrama-chatbot` แล้ว copy ไฟล์เหล่านี้ขึ้นไป:
  `docker-compose.prod.yml`, `deploy/Caddyfile`, `scripts/`, `.env.prod.example`

รายละเอียด: [deployment-beginner-guide.md](./deployment-beginner-guide.md) ข้อ 5–6

---

## เฟส 3 — เอา image + deploy (~30 นาที)

- [ ] ให้ GitHub Actions build image ก่อน: ดูแท็บ **Actions** บน GitHub ว่าเขียว (build ทุกครั้งที่ push `main`)
- [ ] ทำให้ image ดึงได้: GitHub → repo → **Packages** → `rama-chatbot` → Package settings → **Change visibility → Public**
      (ไม่งั้นต้อง `docker login ghcr.io` ด้วย token)
- [ ] บน server สร้าง `.env.prod` (จาก `.env.prod.example`) ใส่:
  ```
  DOMAIN=your-domain.com
  LINE_CHANNEL_SECRET=...
  LINE_CHANNEL_TOKEN=...
  IMAGE_REGISTRY=ghcr.io
  IMAGE_REPOSITORY=yimsoijoi/rama-chatbot
  IMAGE_TAG=latest
  DB_PATH=/app/data/users.db
  ENABLE_PPROF=false
  ```
- [ ] deploy:
  ```bash
  chmod +x scripts/deploy_with_rollback.sh
  IMAGE_REGISTRY=ghcr.io IMAGE_REPOSITORY=yimsoijoi/rama-chatbot ./scripts/deploy_with_rollback.sh latest
  ```
- [ ] เช็ค: `curl -I https://your-domain.com/healthz` ได้ `200` (Caddy จะขอ HTTPS cert ให้เอง)

รายละเอียด: [deployment-beginner-guide.md](./deployment-beginner-guide.md) ข้อ 7–9

---

## เฟส 4 — ตั้งค่า LINE console (~20 นาที)

- [ ] Developers Console: Webhook URL = `https://your-domain.com/webhook`, Use webhook = **On**, กด Verify
- [ ] OA Manager → Response settings: Response mode = **Bot**, Auto-response = **Off**, Greeting = **On**
- [ ] ใส่ข้อความ greeting ให้ผู้ใช้กดเลือกกลุ่มผลตรวจ

รายละเอียด + ข้อความ greeting + ตารางแก้ปัญหา: [line-console-choose-dx-setup.md](./line-console-choose-dx-setup.md)

---

## เฟส 5 — สร้าง Rich Menu + ผูก ID (~40 นาที)

- [ ] สร้างเมนู **DX selector** (default) ผ่าน API — Part A
- [ ] สร้างเมนู **DX1** (แล้วทำ DX2–DX5) ผ่าน API — Part B
- [ ] เอา `richMenuId` แต่ละอันใส่ `configs/faq_seed.yaml` ช่อง `rich_menu_id` (ตอนนี้ยังว่างทั้ง 5)
- [ ] commit + push `main` → รอ CI → deploy ซ้ำ (`./scripts/deploy_with_rollback.sh latest`)

รายละเอียด + คำสั่ง curl + ไฟล์ JSON: [line-rich-menu-dx1-guide.md](./line-rich-menu-dx1-guide.md)

---

## เฟส 6 — ทดสอบ flow จริง + go-live

- [ ] เพิ่มบอทเป็นเพื่อน (บัญชีใหม่) → เห็น greeting + เมนูเลือก DX
- [ ] กด "เลือก DX1" → บอตตอบยืนยัน + เมนูเปลี่ยนเป็น DX1
- [ ] กดปุ่มในเมนู DX1 → ได้คำตอบถูก (ไม่ใช่ fallback)
- [ ] พิมพ์คำฉุกเฉิน (เลือดออก / ปวดมาก) → ขึ้นข้อความ escalation
- [ ] ทดสอบทั้ง Android และ iOS

---

## เฟส 7 — ความปลอดภัยก่อนใช้จริง

- [ ] `ENABLE_PPROF=false` ใน prod (ปิด profiler สาธารณะ)
- [ ] SSH เข้า server แบบ key-only (ปิด password login)
- [ ] เปิด auto security update บน Ubuntu
- [ ] มีข้อความ consent / disclaimer ทางการแพทย์ในบอต
- [ ] เก็บ `.env.prod` เป็นความลับ (อย่า commit ขึ้น git)
- [ ] ถ้า token/secret หลุด → rotate ใน LINE console ทันที

---

## ตอนนี้อยู่ตรงไหน?

- โค้ด + เอกสาร: ✅ เสร็จ (อยู่บน branch `main`)
- `rich_menu_id` 5 ช่อง: ⬜ ยังว่าง (เติมในเฟส 5)
- **ขั้นถัดไปที่ควรทำ: เฟส 0** (ทดสอบด้วย tunnel ก่อนเสียเงิน)
