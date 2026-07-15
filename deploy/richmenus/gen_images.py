#!/usr/bin/env python3
"""Generate rich-menu background HTML (3x2 grid) for headless-Chrome capture.

Each cell maps 1:1 to the `areas` order in the matching *.json (row-major:
top-left, top-mid, top-right, bottom-left, bottom-mid, bottom-right).
The on-image label is short/friendly; the tap action text lives in the JSON.
"""
import os

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "preview")
os.makedirs(OUT, exist_ok=True)

# DX accent colors (dark enough for white text — AA large)
TEAL, BLUE, PURPLE, GREEN, AMBER = "#0E7490", "#1D4ED8", "#6D28D9", "#047857", "#B45309"
NAVY, RED = "#0D2B55", "#B91C1C"

STAFF = ("💬", "คุยกับเจ้าหน้าที่", NAVY)
EMERG = ("🚨", "อาการฉุกเฉิน", RED)

MENUS = {
    "dx-select": {
        "bar": "เลือกกลุ่มผลตรวจของคุณ",
        "cells": [
            ("🦠", "DX1 · Pap ปกติ พบเชื้อ HPV", TEAL),
            ("🔬", "DX2 · Pap ผิดปกติเล็กน้อย (LSIL)", BLUE),
            ("🧫", "DX3 · ตัดชิ้นเนื้อพบ CIN1", PURPLE),
            ("✅", "DX4 · ASCUS · HPV ลบ", GREEN),
            ("🔭", "DX5 · Pap ผิดปกติ ส่องกล้องปกติ", AMBER),
            STAFF,
        ],
    },
    "dx1": {"bar": "DX1 · เมนูช่วยเหลือ", "cells": [
        ("🩺", "Pap ปกติ แต่ HPV บวก?", TEAL),
        ("🌱", "HPV หายเองได้ไหม?", TEAL),
        ("📅", "ตรวจซ้ำเมื่อไหร่?", TEAL),
        ("⚠️", "ระหว่างรอ ระวังอะไร?", TEAL),
        STAFF, EMERG]},
    "dx2": {"bar": "DX2 · เมนูช่วยเหลือ", "cells": [
        ("🔬", "LSIL คืออะไร?", BLUE),
        ("🌱", "โอกาสหายเองมากแค่ไหน?", BLUE),
        ("🔭", "Colposcopy คืออะไร เจ็บไหม?", BLUE),
        ("📅", "ตรวจซ้ำเมื่อไหร่?", BLUE),
        STAFF, EMERG]},
    "dx3": {"bar": "DX3 · เมนูช่วยเหลือ", "cells": [
        ("🧫", "CIN1 เป็นมะเร็งไหม?", PURPLE),
        ("🌱", "โอกาสหายเอง กี่ปี?", PURPLE),
        ("🩹", "หลังตัดชิ้นเนื้อ เลือดออก ปกติไหม?", PURPLE),
        ("📅", "ตรวจซ้ำเมื่อไหร่?", PURPLE),
        STAFF, EMERG]},
    "dx4": {"bar": "DX4 · เมนูช่วยเหลือ", "cells": [
        ("🔬", "ASCUS คืออะไร?", GREEN),
        ("📊", "โอกาสเป็นมะเร็ง?", GREEN),
        ("🌱", "หายเองได้ไหม?", GREEN),
        ("📅", "ตรวจซ้ำเมื่อไหร่?", GREEN),
        STAFF, EMERG]},
    "dx5": {"bar": "DX5 · เมนูช่วยเหลือ", "cells": [
        ("✂️", "ทำไมไม่ตัดชิ้นเนื้อ?", AMBER),
        ("✅", "ส่องกล้องปกติ = ไม่มีรอยโรค?", AMBER),
        ("🔍", "Colposcopy พลาดได้ไหม?", AMBER),
        ("📅", "ตรวจซ้ำเมื่อไหร่?", AMBER),
        STAFF, EMERG]},
}

CELL = """  <div class="cell" style="background:{color}">
    <div class="ico">{icon}</div>
    <div class="lbl">{label}</div>
  </div>"""

PAGE = """<!doctype html><html lang="th"><head><meta charset="utf-8"><style>
  * {{ margin:0; padding:0; box-sizing:border-box; }}
  html,body {{ width:2500px; height:1686px; }}
  body {{ font-family:"Sarabun","Noto Sans Thai","Thonburi",sans-serif;
          background:#0b1f3a; display:grid;
          grid-template-columns:repeat(3,1fr); grid-template-rows:repeat(2,1fr);
          gap:10px; padding:10px; }}
  .cell {{ display:flex; flex-direction:column; align-items:center; justify-content:center;
           text-align:center; color:#fff; padding:40px; gap:28px; }}
  .ico {{ font-size:230px; line-height:1; }}
  .lbl {{ font-size:76px; font-weight:700; line-height:1.25; max-width:92%;
          text-shadow:0 2px 6px rgba(0,0,0,.25); }}
</style></head><body>
{cells}
</body></html>"""

for name, spec in MENUS.items():
    cells_html = "\n".join(
        CELL.format(icon=i, label=l, color=c) for (i, l, c) in spec["cells"]
    )
    html = PAGE.format(cells=cells_html)
    with open(os.path.join(OUT, name + ".html"), "w", encoding="utf-8") as f:
        f.write(html)
    print("wrote", name + ".html")
print("done ->", OUT)
