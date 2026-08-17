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

	audioLine := "She talks naturally in Thai about this garment only — its fit, fabric and colour; her voice only, quiet ambience, no music."
	switch AM {
	case "laugh":
		audioLine = "No spoken words — only a soft natural laugh and quiet breathing; no music."
	case "silent":
		audioLine = "No dialogue and no music — picture only."
	}

	idBrief := firstSentence(base.Identity)

	var b strings.Builder
	fmt.Fprintf(&b, "Vertical 9:16, 8-second single continuous take, no cuts — %s.\n\n", strings.TrimSuffix(F.Clip, "."))
	fmt.Fprintf(&b, "SUBJECT: %s She wears %s in %s, with %s.%s\n\n", idBrief, garment, C, bottom, shoes)
	fmt.Fprintf(&b, "SETTING: %s\n\n", trimRunes(F.Setting, 240))
	b.WriteString("CAMERA: phone locked on a tripod, fixed 9:16 framing, eye/chest level. No drift, no dolly, no push-in, no zoom; she stays the same size in frame; the background stays completely static.\n\n")
	fmt.Fprintf(&b, "ACTION: %s\n\n", timing)
	fmt.Fprintf(&b, "CONSISTENCY: the garment's neckline, hem, fabric and %s colour stay identical in every frame; the hem stays over the waistband with no skin between top and bottom; neutral daylight white balance, no warm or yellow cast.\n\n", C)
	fmt.Fprintf(&b, "AUDIO: %s\n\n", audioLine)
	b.WriteString("Avoid: camera drift or zoom, changing the garment, exposed waist, extra people, on-screen text or logos, distorted hands, robotic motion.")

	return trimRunes(b.String(), 1500)
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
	fmt.Fprintf(&b, "A vertical 9:16 photo. %s She wears %s in %s, with %s.%s\n\n", idBrief, garment, C, bottom, shoes)
	fmt.Fprintf(&b, "%s\n\n", trimRunes(styleBrief, 200))
	fmt.Fprintf(&b, "SETTING: %s\n\n", trimRunes(F.Setting, 260))
	fmt.Fprintf(&b, "%s\n\n", trimRunes(F.Pose1, 320))
	b.WriteString("Use the attached reference photo as the exact source of truth for the garment — copy its neckline, sleeves, hem and fabric, changing only the colour to the stated one. Neutral daylight white balance, clean true colour, realistic skin and texture. No text, no logos, no watermark.")

	return trimRunes(b.String(), 1500)
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
