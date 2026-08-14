# studio16-go

สร้างพรอมต์ + เจนวิดีโอรีวิวสินค้าเสื้อผ้าด้วย AI แล้วตรวจว่า "สินค้าตรงไหม" อัตโนมัติ
เขียนด้วย **Go** รันบน **Docker** — พอร์ตหัวใจพรอมต์เอนจินมาจากเว็บแอป STUDIO 16
และเสริม backend ที่เก็บ API key ฝั่งเซิร์ฟเวอร์ (ปลอดภัยกว่า) พร้อมรองรับ **OpenAI + Gemini (Veo)**

## ทำอะไรได้บ้าง

1. **คลังสินค้า** — เพิ่มสินค้า อัปรูป เก็บสเปก (เก็บลงไฟล์ JSON + โฟลเดอร์รูป)
2. **อ่านรูป → กรอกสเปกอัตโนมัติ** — vision AI (OpenAI หรือ Gemini) อ่านรูปเสื้อผ้าแล้วเติมคอเสื้อ/สาย/เนื้อผ้า/ชายเสื้อ ฯลฯ
3. **ปั้นพรอมต์วิดีโอ** — 3 ฟอร์แมต (ยืนหน้ากระจก / คาเฟ่ / จับคู่สีสองนางแบบ) + โหมดเสียง (พูด/หัวเราะ/เงียบ) + ตัวเตือนคำเสี่ยง
4. **เจนวิดีโอจริง** — ส่งพรอมต์ + รูปแรกไปให้ **Gemini Veo** (งาน async, poll จนเสร็จ, ดาวน์โหลดเก็บใน backend)
5. **รีพอทตรวจสินค้าตรง/ไม่ตรง** — AI ให้คะแนน 0–100 ต่อรูปและต่อคลิป (ดึงเฟรมด้วย ffmpeg) แล้วตัด **ผ่าน/ไม่ผ่าน** ตาม threshold

## รันด้วย Docker (แนะนำ)

```bash
cp .env.example .env
# แก้ .env ใส่ OPENAI_API_KEY และ GEMINI_API_KEY
docker compose up --build
```

เปิด **http://localhost:8080/** จะเจอหน้าเว็บ (UI) ให้ใช้งานได้เลย — สร้างสินค้า อัปรูป อ่านรูป AI สร้างพรอมต์ เจนวิดีโอ Veo และตรวจสินค้าตรง/ไม่ตรง
(ตรวจสถานะระบบได้ที่ http://localhost:8080/api/health)

> คีย์ทั้งหมดอยู่ฝั่ง server ในไฟล์ `.env` — หน้าเว็บไม่ต้องกรอกคีย์ ปุ่ม ⚙️ แค่โชว์สถานะว่าตั้งคีย์แล้วหรือยัง

## รันด้วย Go (ไม่ผ่าน Docker)

ต้องมี Go 1.22+ และ ffmpeg ในเครื่อง

```bash
go run ./cmd/server
```

## ENV ที่ใช้

| ตัวแปร | ค่าเริ่มต้น | ใช้ทำอะไร |
|--------|-----------|-----------|
| `OPENAI_API_KEY` | – | อ่านรูป + ให้คะแนน match |
| `OPENAI_MODEL` | `gpt-4o` | โมเดล vision ของ OpenAI |
| `GEMINI_API_KEY` | – | เจนวิดีโอ Veo + vision (ถ้าเลือก) |
| `VEO_MODEL` | `veo-3.1-generate-preview` | โมเดลวิดีโอ |
| `MATCH_THRESHOLD` | `75` | คะแนน ≥ นี้ = ผ่าน |
| `MATCH_PROVIDER` | `openai` | ใครให้คะแนน match (openai/gemini) |

## API หลัก

| Method & Path | ทำอะไร |
|---|---|
| `GET /api/health` | สถานะระบบ |
| `GET /api/products` | รายการสินค้า |
| `POST /api/products` | สร้างสินค้า (ค่าเริ่มต้นแบบ STUDIO 16) |
| `GET /api/products/{id}` | ดูสินค้า |
| `PATCH /api/products/{id}` | แก้ฟิลด์ (merge) |
| `DELETE /api/products/{id}` | ลบสินค้า + รูป |
| `POST /api/products/{id}/images` | อัปรูป (`{"dataUrl":"data:image/jpeg;base64,..."}`) |
| `DELETE /api/products/{id}/images/{imgId}` | ลบรูป |
| `POST /api/products/{id}/analyze?provider=openai\|gemini` | อ่านรูปเติมสเปก |
| `GET /api/products/{id}/prompt?format=mirror&audio=talk` | ปั้นพรอมต์ + คำเสี่ยง |
| `POST /api/products/{id}/generate` | สั่งเจนวิดีโอ Veo (คืน job) |
| `GET /api/products/{id}` | ดู `jobs[]` เพื่อ poll สถานะวิดีโอ |
| `POST /api/products/{id}/report` | สั่งตรวจ match ทั้งหมด |
| `GET /api/products/{id}/report` | ดูรีพอทล่าสุด |
| `GET /assets/...` | ไฟล์รูป/วิดีโอที่ผลิต |

## ตัวอย่างขั้นตอนใช้งาน

```bash
# 1) สร้างสินค้า
curl -X POST localhost:8080/api/products
# 2) อัปรูป (ใส่ base64 จริง)
curl -X POST localhost:8080/api/products/<id>/images -d '{"dataUrl":"data:image/jpeg;base64,..."}'
# 3) อ่านรูปเติมสเปก
curl -X POST "localhost:8080/api/products/<id>/analyze?provider=openai"
# 4) ปั้นพรอมต์
curl "localhost:8080/api/products/<id>/prompt?format=mirror&audio=talk"
# 5) เจนวิดีโอ
curl -X POST localhost:8080/api/products/<id>/generate -d '{"format":"mirror","audio":"talk"}'
# 6) ตรวจสินค้าตรง/ไม่ตรง
curl -X POST localhost:8080/api/products/<id>/report
```

## โครงสร้างโปรเจกต์

```
cmd/server        main
internal/config   อ่าน ENV
internal/model    โครงข้อมูลสินค้า (พอร์ตจาก STUDIO 16 blank())
internal/store    JSON-file store + โฟลเดอร์ assets
internal/prompt   พรอมต์เอนจิน (พอร์ต buildPrompt + 3 ฟอร์แมต) + prompt ตรวจรูป/match
internal/ai       interface กลาง + openai/ + gemini/ (Veo)
internal/video    คิวงานเจนวิดีโอ async
internal/match    รีพอทให้คะแนน match + ffmpeg ดึงเฟรม
internal/httpapi  REST API
```

## หมายเหตุเรื่อง Veo

รูปแบบ endpoint/โมเดลของ Veo บน Gemini API เปลี่ยนได้เรื่อย ๆ โค้ดแยก logic ไว้ที่
`internal/ai/gemini/gemini.go` ที่เดียว — ถ้า Google ปรับ contract แก้ไฟล์นี้ไฟล์เดียวจบ
ปรับโมเดลได้ผ่าน ENV `VEO_MODEL` โดยไม่ต้องแก้โค้ด
