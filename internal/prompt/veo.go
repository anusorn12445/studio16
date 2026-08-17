package prompt

import (
	"fmt"
	"strings"

	"studio16/internal/model"
)

// BuildVeo produces a COMPACT single-video prompt for a direct Veo API call.
//
// The full Build() output is an agent pipeline prompt (2 images + 2 videos,
// thousands of characters) meant to be pasted into an orchestrating agent.
// The Veo REST API instead wants one short video description, and rejects very
// long strings ("exceeds the length limit"). BuildVeo distills Scene 1 into a
// concise brief that stays well under that limit.
func BuildVeo(p model.Product) string {
	F := fmtFor(p.Format)
	base := ResolveBase(p)

	C := Sanitize(EnOnly(p.HeroColor, "cream white"))
	garment := Sanitize(EnOnly(p.Garment, "a fitted knit top"))
	bottom := Sanitize(EnOnly(p.Bottom, "high-waisted trousers"))
	shoes := ""
	if F.Feet {
		shoes = " " + Sanitize(EnOnly(p.Shoes, "plain white sneakers")) + " on her feet."
	}

	// Audio mode transforms the timing text (talk / laugh / silent).
	AM := "talk"
	if !F.Voice {
		AM = AudioMode(p)
	}
	T := func(x string) string { return x }
	if !F.Voice {
		if AM == "laugh" {
			T = LaughFix
		} else if AM == "talk" {
			T = TalkFix
		}
	}
	timing := T(strings.Join(F.T1, " "))

	// Give Veo an actual Thai line to speak (from the product script, else a
	// simple fallback) so it speaks Thai instead of improvising English.
	sc := ActiveScript(p)
	thaiLine := "ชิ้นนี้เนื้อผ้าดี ใส่สบาย ทรงสวยมากเลยค่ะ ลองดูนะคะ"
	if strings.TrimSpace(p.TypeTh) != "" {
		thaiLine = "ชิ้นนี้" + strings.TrimSpace(p.TypeTh) + " เนื้อผ้าดี ใส่สบาย ทรงสวยมากเลยค่ะ แนะนำเลย"
	}
	if strings.TrimSpace(sc[0]) != "" {
		thaiLine = strings.TrimSpace(sc[0])
	}
	audioLine := `She speaks THAI only (ภาษาไทย) — never English. Warm and casual, like chatting with a friend, she says: "` + thaiLine + `" Her voice only, quiet natural ambience, no music, no second speaker.`
	switch AM {
	case "laugh":
		audioLine = "No spoken words — only a soft natural laugh and quiet breathing; no music."
	case "silent":
		audioLine = "No dialogue and no music — picture only."
	}

	idBrief := firstSentence(base.Identity)

	var b strings.Builder
	fmt.Fprintf(&b, "Full-frame vertical 9:16 portrait, 8-second single continuous take, no cuts — %s. The scene fills the whole frame edge to edge: no black bars, no letterboxing, no borders, no padding.\n\n", strings.TrimSuffix(F.Clip, "."))
	fmt.Fprintf(&b, "SUBJECT: %s She wears %s in %s, with %s.%s\n\n", idBrief, garment, C, bottom, shoes)
	fmt.Fprintf(&b, "AUDIO: %s\n\n", audioLine)
	fmt.Fprintf(&b, "SETTING: %s\n\n", trimRunes(F.Setting, 200))
	b.WriteString("CAMERA: phone locked on a tripod, fixed 9:16 framing, eye/chest level. No drift, no dolly, no zoom; she stays the same size in frame; the background stays static.\n\n")
	fmt.Fprintf(&b, "ACTION: %s\n\n", trimRunes(timing, 340))
	fmt.Fprintf(&b, "CONSISTENCY: the garment's neckline, hem, fabric and %s colour stay identical every frame; the hem stays over the waistband, no skin between top and bottom; neutral daylight, no warm/yellow cast.\n\n", C)
	b.WriteString("Avoid: black bars, letterboxing, borders or padding, any English speech, camera drift or zoom, changing the garment, exposed waist, extra people, on-screen text or logos, distorted hands, robotic motion.")

	return trimRunes(b.String(), 1700)
}

// BuildVeoImage produces a COMPACT image prompt for the opening frame: the
// model wearing the garment in the scene (Pose 1). This image is generated
// first, then handed to Veo as the video's first frame.
func BuildVeoImage(p model.Product) string {
	F := fmtFor(p.Format)
	base := ResolveBase(p)

	C := Sanitize(EnOnly(p.HeroColor, "cream white"))
	garment := Sanitize(EnOnly(p.Garment, "a fitted knit top"))
	bottom := Sanitize(EnOnly(p.Bottom, "high-waisted trousers"))
	shoes := ""
	if F.Feet {
		shoes = " " + Sanitize(EnOnly(p.Shoes, "plain white sneakers")) + " on her feet."
	}
	idBrief := firstSentence(base.Identity)
	styleBrief := firstSentence(base.Style)

	var b strings.Builder
	fmt.Fprintf(&b, "A full-frame vertical 9:16 portrait photo that fills the entire frame edge to edge — no black bars, no letterboxing, no borders, no padding. %s She wears %s in %s, with %s.%s\n\n", idBrief, garment, C, bottom, shoes)
	fmt.Fprintf(&b, "%s\n\n", trimRunes(styleBrief, 200))
	fmt.Fprintf(&b, "SETTING: %s\n\n", trimRunes(F.Setting, 260))
	fmt.Fprintf(&b, "%s\n\n", trimRunes(F.Pose1, 320))
	b.WriteString("Use the attached reference photo as the exact source of truth for the garment — copy its neckline, sleeves, hem and fabric, changing only the colour to the stated one. Neutral daylight white balance, clean true colour, realistic skin and texture. Full-bleed 9:16 portrait, no black bars or borders, no text, no logos, no watermark.")

	return trimRunes(b.String(), 1600)
}

// firstSentence returns text up to (and including) the first sentence-ending period.
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ". "); i > 0 {
		return s[:i+1]
	}
	return s
}

// trimRunes caps a string to n runes, trimming back to the last space so it
// never cuts a word (or a multi-byte character) in half.
func trimRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	out := string(r[:n])
	if i := strings.LastIndex(out, " "); i > n/2 {
		out = out[:i]
	}
	return strings.TrimSpace(out)
}
