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
func BuildVeo(p model.Product, o VeoOpts) string {
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
	if strings.TrimSpace(o.Line) != "" {
		thaiLine = strings.TrimSpace(o.Line)
	}
	audioLine := `She speaks THAI only (ภาษาไทย) — never English. Warm and casual, like chatting with a friend, she says: "` + thaiLine + `" Her voice only, quiet natural ambience, no music, no second speaker.`
	switch AM {
	case "laugh":
		audioLine = "No spoken words — only a soft natural laugh and quiet breathing; no music."
	case "silent":
		audioLine = "No dialogue and no music — picture only."
	}

	idBrief := firstSentence(base.Identity)

	roleNote := ""
	switch o.Role {
	case "hook":
		roleNote = "ROLE: this is the OPENING HOOK — she opens with an attention-grabbing beat and bright, energetic delivery so the viewer stops scrolling."
	case "story":
		roleNote = "ROLE: this is the MIDDLE of the review — she explains the garment naturally and shows the details."
	case "close":
		roleNote = "ROLE: this is the CLOSING — she looks straight into the camera, gives a warm sincere recommendation and invites the viewer to check it out or tap the link, a soft friendly sales close."
	}
	contLine := ""
	if o.Total > 1 {
		contLine = fmt.Sprintf("CONTINUITY: part %d of %d of one continuous review — same woman, same face, same outfit, same place and light as the other parts.\n\n", o.Part, o.Total)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Full-frame vertical 9:16 portrait, 8-second single continuous take, no cuts — %s. The scene fills the whole frame edge to edge: no black bars, no letterboxing, no borders, no padding.\n\n", strings.TrimSuffix(F.Clip, "."))
	b.WriteString(contLine)
	hasRef := len(p.Images) > 0
	subjGarment := garment + " in " + C
	consist := fmt.Sprintf("CONSISTENCY: the garment's neckline, hem, fabric and %s colour stay identical every frame; the hem stays over the waistband, no skin between top and bottom; neutral daylight, no warm/yellow cast.\n\n", C)
	if hasRef {
		subjGarment = "the same garment as in the first frame (do not restyle or recolour it)"
		consist = "CONSISTENCY: keep the garment identical to the first frame — same neckline, sleeves, hem, fabric and colour — in every frame; the hem stays over the waistband, no skin between top and bottom; neutral daylight, no warm/yellow cast.\n\n"
	}
	fmt.Fprintf(&b, "SUBJECT: %s She wears %s, with %s.%s\n\n", idBrief, subjGarment, bottom, shoes)
	fmt.Fprintf(&b, "AUDIO: %s\n\n", audioLine)
	if roleNote != "" {
		b.WriteString(roleNote + "\n\n")
	}
	fmt.Fprintf(&b, "SETTING: %s\n\n", trimRunes(F.Setting, 190))
	b.WriteString("CAMERA: phone locked on a tripod, fixed 9:16 framing, eye/chest level. No drift, no dolly, no zoom; she stays the same size in frame; the background stays static.\n\n")
	fmt.Fprintf(&b, "ACTION: %s\n\n", trimRunes(timing, 300))
	b.WriteString(consist)
	b.WriteString("Avoid: black bars, letterboxing, borders or padding, any English speech, camera drift or zoom, changing the garment, exposed waist, extra people, on-screen text or logos, distorted hands, robotic motion.")

	return trimRunes(b.String(), 1850)
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

	// When the product has reference photos, the PHOTO is the source of truth —
	// do not let the (possibly default/unanalyzed) text spec fight it.
	hasRef := len(p.Images) > 0
	wearLine := "She wears " + garment + " in " + C
	authLine := "Neutral daylight white balance, clean true colour, realistic skin and texture."
	if hasRef {
		wearLine = "She wears the exact garment shown in the attached reference photo — the same type, cut, neckline, sleeves, hem, fabric and colour"
		authLine = "The attached reference photo is the exact source of truth for the garment: reproduce it faithfully — do not restyle it, do not change the neckline, sleeves or length, and do not change its colour. Keep the model's face and styling as described. Neutral daylight, realistic skin and texture."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "A full-frame vertical 9:16 portrait photo that fills the entire frame edge to edge — no black bars, no letterboxing, no borders, no padding. %s %s, with %s.%s\n\n", idBrief, wearLine, bottom, shoes)
	fmt.Fprintf(&b, "%s\n\n", trimRunes(styleBrief, 200))
	fmt.Fprintf(&b, "SETTING: %s\n\n", trimRunes(F.Setting, 240))
	fmt.Fprintf(&b, "%s\n\n", trimRunes(F.Pose1, 300))
	b.WriteString(authLine + " Full-bleed 9:16 portrait, no black bars or borders, no text, no logos, no watermark.")

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

// VeoOpts carries per-shot narrative context so a batch of shots tells one
// connected story (hook → body → close) instead of identical clips.
type VeoOpts struct {
	Line  string // the Thai line this shot speaks
	Role  string // "hook" | "story" | "close" | ""
	Part  int    // 1-based part index
	Total int    // total parts in this story
}

// Beat is one planned shot: which line to speak and its narrative role.
type Beat struct {
	Line string
	Role string
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// PlanBeats maps the product's 4 script lines (hook, body1, body2, close) onto
// `shots` connected beats so the clips form one story arc.
func PlanBeats(p model.Product, shots int) []Beat {
	sc := ActiveScript(p)
	hook, b1, b2, cl := strings.TrimSpace(sc[0]), strings.TrimSpace(sc[1]), strings.TrimSpace(sc[2]), strings.TrimSpace(sc[3])
	if shots < 1 {
		shots = 1
	}
	switch shots {
	case 1:
		return []Beat{{Line: firstNonEmpty(hook, b1, cl), Role: "hook"}}
	case 2:
		return []Beat{{Line: firstNonEmpty(hook, b1), Role: "hook"}, {Line: firstNonEmpty(cl, b2), Role: "close"}}
	case 3:
		return []Beat{{Line: firstNonEmpty(hook, b1), Role: "hook"}, {Line: firstNonEmpty(b1, b2), Role: "story"}, {Line: firstNonEmpty(cl, b2), Role: "close"}}
	default:
		return []Beat{{Line: hook, Role: "hook"}, {Line: b1, Role: "story"}, {Line: b2, Role: "story"}, {Line: cl, Role: "close"}}
	}
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
