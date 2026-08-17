package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"studio16/internal/model"
)

// ScriptPrompt asks the model to write four short Thai spoken lines (2 per
// scene) for the review, focused entirely on the garment.
func ScriptPrompt(p model.Product) string {
	name := strings.TrimSpace(p.TypeTh)
	if name == "" {
		name = strings.TrimSpace(p.Type)
	}
	if name == "" {
		name = "เสื้อผ้าชิ้นนี้"
	}
	var b strings.Builder
	b.WriteString("คุณเขียนบทพูดสั้นๆ ภาษาไทย สำหรับคลิปรีวิวเสื้อผ้าลง TikTok/Shopee สไตล์คนจริงคุยกับเพื่อน ไม่ใช่พนักงานขาย น้ำเสียงอบอุ่นเป็นกันเอง\n\n")
	fmt.Fprintf(&b, "สินค้า: %s", name)
	if c := strings.TrimSpace(p.HeroColor); c != "" {
		fmt.Fprintf(&b, " สี %s", c)
	}
	b.WriteString("\n")
	sp := p.Spec
	add := func(label, v string) {
		if strings.TrimSpace(v) != "" {
			fmt.Fprintf(&b, "- %s: %s\n", label, v)
		}
	}
	add("เนื้อผ้า", sp.Fabric)
	add("ทรง/ฟิต", sp.Fit)
	add("คอเสื้อ", sp.Neckline)
	add("ชายเสื้อ", sp.Hem)
	if d := strings.TrimSpace(p.Desc); d != "" {
		fmt.Fprintf(&b, "คำบรรยายร้าน: %s\n", d)
	}
	b.WriteString("\nเขียนบทพูดเป็น \"เรื่องราวต่อเนื่องกัน\" 4 ประโยค ที่ต่อกันเหมือนคลิปเดียว มีโครงชัดเจน:\n")
	b.WriteString("1) ฮุค (hook) — ประโยคเปิดสะดุดหู ดึงให้คนหยุดดูใน 2 วินาทีแรก\n")
	b.WriteString("2) เนื้อเรื่อง (1) — เล่าจุดเด่นของชุด ทรง/เนื้อผ้า\n")
	b.WriteString("3) เนื้อเรื่อง (2) — เล่าการใส่จริง/แมทช์/ความรู้สึกตอนใส่\n")
	b.WriteString("4) ปิดการขาย (call to action) — ชวนซื้อ/กดลิงก์อย่างเป็นธรรมชาติ อบอุ่น ไม่ยัดเยียด\n\n")
	b.WriteString("แต่ละประโยคสั้นพูดจบใน 3-4 วินาที เป็นธรรมชาติ พูดถึง \"ตัวเสื้อผ้า\" เท่านั้น — ทรง เนื้อผ้า สี การใส่และการแมทช์ ห้ามพูดถึงสถานที่ อาหาร เครื่องดื่ม อากาศ หรือราคา (ยกเว้นประโยคปิดที่ชวนกดลิงก์ได้)\n\n")
	b.WriteString("ตอบเป็น JSON เท่านั้น ห้ามมีข้อความอื่น ห้ามมี markdown:\n")
	b.WriteString(`{"lines":["ฮุค","เนื้อเรื่อง1","เนื้อเรื่อง2","ปิดการขาย"]}`)
	return b.String()
}

// ExtractJSON pulls a JSON object out of a model reply, tolerating markdown
// fences and surrounding prose. Ported from STUDIO 16's extractJSON().
func ExtractJSON(t string) (string, error) {
	clean := strings.TrimSpace(strings.NewReplacer("```json", "", "```", "").Replace(t))
	var probe any
	if json.Unmarshal([]byte(clean), &probe) == nil {
		return clean, nil
	}
	a := strings.Index(clean, "{")
	b := strings.LastIndex(clean, "}")
	if a >= 0 && b > a {
		cand := clean[a : b+1]
		if json.Unmarshal([]byte(cand), &probe) == nil {
			return cand, nil
		}
	}
	if clean == "" {
		return "", fmt.Errorf("ไม่มีคำตอบกลับมา")
	}
	if len(clean) > 110 {
		clean = clean[:110]
	}
	return "", fmt.Errorf("ตอบกลับมาไม่ใช่ข้อมูล: %s", clean)
}

// AnalyzePrompt builds the vision instruction that reads product photos and
// returns the garment spec as JSON. Ported from STUDIO 16's readPhotos().
func AnalyzePrompt(shopDesc, focus string) string {
	var b strings.Builder
	if strings.TrimSpace(focus) != "" {
		b.WriteString("FOCUS: a previous reading of these photos left these points unclear or unusable — ")
		b.WriteString(focus)
		b.WriteString(". Give them your full attention and answer them precisely this time, while still filling in every other field.\n\n")
	}
	b.WriteString("Look at these product photos of a women's garment sold on Thai e-commerce.\n\n")
	if strings.TrimSpace(shopDesc) != "" {
		b.WriteString("The shop also wrote this description. Use it to resolve anything the photos leave ambiguous — it often states details the images cannot show clearly, such as strap width, built-in shaping, fabric content or sizing. Where the description and the photos disagree, trust the photos for shape and the description for materials and construction. Machine-translated shop copy often mislabels parts — it may call a shoulder strap a hip strap, or name the wrong neckline. Where a phrase plainly contradicts the photos, ignore that phrase:\n---\n")
		b.WriteString(Sanitize(shopDesc))
		b.WriteString("\n---\n\n")
	}
	b.WriteString("Zoom in mentally on the garment before answering. Look at the straps against the width of the shoulder, the top edge of the neckline against the collarbone, and the hem against the waistband. Fine details are small in these images and guessing them wrong ruins the result — read them, do not assume.\n\n")
	b.WriteString("Reply with ONLY a JSON object, no preamble, no markdown fences:\n")
	b.WriteString(`{
 "typeTh": "ประเภทสินค้าภาษาไทยสั้นๆ เช่น เสื้อกล้ามสายเดี่ยวซับในนุ่ม",
 "type": "the same garment category in plain English, e.g. sleeveless knit top",
 "spec": {
   "neckline": "Exact neckline shape and position. Name the shape plainly (straight across level, square, U-shaped scoop, V, sweetheart, round crew), how wide it runs between the straps, and how high or low the edge sits relative to the collarbone.",
   "straps": "Strap width two ways: an estimate in millimetres AND a comparison to a familiar object (thinner than a shoelace, as wide as a finger). How far apart they sit (outer shoulder, mid-shoulder, close to the neck). State hardware explicitly; if plain and fixed, write the words non-adjustable, no sliders, no hardware.",
   "armhole": "How deep and how wide the armhole is cut.",
   "fabric": "Knit or woven. If ribbed, state the gauge (ribs per 10cm, single rib width in mm, direction). Then finish (matte, sheen, brushed), stretch and drape.",
   "print": "Any pattern printed or woven in: motif, scale in cm, colours, where it appears. If plain, write the single word none.",
   "trim": "Every applied decoration with position, width and colour: lace, binding, bows, ruffles, embroidery, lettering, appliqués, buttons. Quote lettering exactly. If nothing applied, write none.",
   "hem": "Where the hem ends against a visible body landmark. Whether any skin shows between hem and waistband. How the hem is finished. Never use the words crop, cropped or short.",
   "fit": "How the garment fits through the body: close-fitting, skimming, relaxed, oversized.",
   "details": "Seams, panels, built-in shaping, linings and any other construction detail."
 },
 "garment": "One flowing English sentence combining the above into a single garment description suitable as an image-generation reference.",
 "heroColor": "the main colour of the garment in plain English"
}`)
	return b.String()
}

// MatchPrompt builds the instruction that scores a candidate image (an uploaded
// photo or a generated clip frame) against the reference product photos.
func MatchPrompt(numRefs int, specText string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are a strict quality inspector for a fashion e-commerce shop.\n\n")
	fmt.Fprintf(&b, "The FIRST %d image(s) are the REAL product reference photos. The LAST image is a CANDIDATE that may have been AI-generated or shot separately. Judge how faithfully the CANDIDATE shows the SAME garment as the references.\n\n", numRefs)
	if strings.TrimSpace(specText) != "" {
		b.WriteString("Written spec of the real garment (the candidate must match this):\n---\n")
		b.WriteString(specText)
		b.WriteString("\n---\n\n")
	}
	b.WriteString("Compare neckline shape and depth, strap width and spacing, hem position, fabric texture, colour, and any print or trim. A different neckline, thicker straps, a missing pattern, the wrong colour, or a bra/crop-top substitution are all serious mismatches.\n\n")
	b.WriteString("SEPARATELY, look for AI-generation defects in the candidate and list them under \"issues\": distorted hands or extra/missing fingers, an unnatural, twisted or garbled mouth (a bad lip-sync / speech look), a warped or melted face, garbled or gibberish text, or extra limbs. These are quality problems, not product mismatches.\n\n")
	b.WriteString("Reply with ONLY this JSON object, no prose, no markdown fences:\n")
	b.WriteString(`{
 "score": 0-100 integer — 100 means indistinguishable from the real product, 0 means a completely different garment,
 "verdict": "one short Thai sentence summarising whether the candidate matches the product",
 "mismatches": ["short Thai phrase per concrete product difference, empty array if none"],
 "issues": ["short Thai phrase per visual defect — hands/fingers, mouth/face, artifacts — empty array if none"]
}`)
	return b.String()
}
