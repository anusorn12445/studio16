package model

// Blank returns a new product pre-filled with the STUDIO 16 defaults (ported
// from the HTML blank() function). The id is assigned by the caller.
func Blank() *Product {
	return &Product{
		Name:      "สินค้าใหม่",
		Type:      "sleeveless knit top",
		TypeTh:    "เสื้อกล้ามสายเดี่ยว",
		HeroColor: "cream white",
		Colors:    []string{},
		ColorsEn:  []string{},
		Features:  []string{},
		Format:    "mirror",
		AudioMode: "laugh",
		Wm:        false,
		WmText:    "@go",
		WmPos:     "bottom-right",
		Bottom:    "high-waisted light blue wide-leg jeans that reach the ankle",
		BottomTh:  "ยีนส์ขากว้างยาว",
		Shoes:     "simple white low-top canvas sneakers",
		ShoesTh:   "ผ้าใบขาวทรงเตี้ย",
		Accessories:   "layered thin gold necklaces and small gold hoop earrings",
		AccessoriesTh: "สร้อยทองเส้นเล็กซ้อนชั้น ต่างหูห่วงเล็ก",
		CafeTh:   "กาแฟเย็น แก้วพลาสติกใส",
		CafeItem: "a tall clear plastic takeaway cup of iced coffee with a domed lid and a straw, pale caramel colour with ice visible through the cup",
		VoiceTh:  "เสียงเล็ก ใส น่าฟัง · 25 ปี",
		Voice:    "A young Thai woman in her mid-twenties. Light, clear, slightly high-pitched voice with a soft airy timbre and a gentle natural rasp on the breath. Warm and friendly, unhurried, speaking at a relaxed conversational pace as if chatting with a close friend across a room. Soft rising intonation at the end of phrases, no projection, no announcer energy, no sales tone.",
		Garment:  "sleeveless knit top, wide U-shaped neckline sitting moderately low across the collarbone, very thin pencil-width shoulder straps, deep armhole cut, smooth matte seamless fabric, neat finished neckline edge",
		Spec:     Spec{},
		Scripts:  []Script{},
		Pick:     0,
		Clips:    []Clip{},
		Images:   []Image{},
		Jobs:     []Job{},
	}
}
