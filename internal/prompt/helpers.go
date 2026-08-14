package prompt

import (
	"regexp"
	"strings"
	"unicode"

	"studio16/internal/model"
)

// ---- small predicates -------------------------------------------------------

// HasThai reports whether t contains any Thai codepoint (U+0E00..U+0E7F).
func HasThai(t string) bool {
	for _, r := range t {
		if r >= 0x0E00 && r <= 0x0E7F {
			return true
		}
	}
	return false
}

// EnOnly returns v trimmed, or fallback when v is blank or contains Thai.
func EnOnly(v, fallback string) string {
	t := strings.TrimSpace(v)
	if t == "" || HasThai(t) {
		return fallback
	}
	return t
}

// AudioMode ports audioMode(p). The HTML forceVoice flag does not exist on the
// Go model, so a blank AudioMode falls through to "talk".
func AudioMode(p model.Product) string {
	if p.AudioMode != "" {
		return p.AudioMode
	}
	return "talk"
}

// ---- tidy -------------------------------------------------------------------

var (
	reTidyWord         = regexp.MustCompile(`[A-Za-z0-9_]+`)
	reMultiSpace       = regexp.MustCompile(`\s{2,}`)
	reSpaceBeforePunct = regexp.MustCompile(`\s+([,.])`)
	reCommaComma       = regexp.MustCompile(`,\s*,`)
	reTrimEdges        = regexp.MustCompile(`^[\s,]+|[\s,]+$`)
)

func isAllSpace(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// dedupWords collapses runs of the same word separated only by whitespace,
// keeping the first copy. It replaces the JS regex /\b(\w+)(\s+\1\b)+/gi which
// RE2 cannot express (backreference).
func dedupWords(s string) string {
	locs := reTidyWord.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return s
	}
	var b strings.Builder
	prevEnd := 0
	i := 0
	for i < len(locs) {
		wStart, wEnd := locs[i][0], locs[i][1]
		word := s[wStart:wEnd]
		j := i
		lastEnd := wEnd
		for j+1 < len(locs) {
			gap := s[locs[j][1]:locs[j+1][0]]
			next := s[locs[j+1][0]:locs[j+1][1]]
			if isAllSpace(gap) && strings.EqualFold(next, word) {
				j++
				lastEnd = locs[j][1]
			} else {
				break
			}
		}
		b.WriteString(s[prevEnd:wStart])
		b.WriteString(word)
		prevEnd = lastEnd
		i = j + 1
	}
	b.WriteString(s[prevEnd:])
	return b.String()
}

// Tidy ports tidy(t).
func Tidy(t string) string {
	out := dedupWords(t)
	out = reMultiSpace.ReplaceAllString(out, " ")
	out = reSpaceBeforePunct.ReplaceAllString(out, "$1")
	out = reCommaComma.ReplaceAllString(out, ",")
	out = reTrimEdges.ReplaceAllString(out, "")
	return strings.TrimSpace(out)
}

// ---- sanitize (SAFE_SWAP) ---------------------------------------------------

var (
	reSpaghetti     = regexp.MustCompile(`(?i)spaghetti[\s\-_]*straps?`)
	reBraGroup      = regexp.MustCompile(`(?i)\b(bralettes?|bras?|camisoles?|lingerie|underwear|undergarments?)\b`)
	reChestPads     = regexp.MustCompile(`(?i)chest pads?|padded cups?|inner padding`)
	reCleavage      = regexp.MustCompile(`(?i)cleavage|bustline|\bbust\b|breasts?`)
	reChestEmphasis = regexp.MustCompile(`(?i)chest emphasis`)
	reUpperChest    = regexp.MustCompile(`(?i)upper chest|chest area|\bdécolletage\b|\bdecolletage\b`)
	reDeepNeck      = regexp.MustCompile(`(?i)deep(?:ly)?[\s\-]*(?:cut[\s\-]*)?(scoop|v[\s\-]*|square|round)?[\s\-]*neckline`)
	reLowCut        = regexp.MustCompile(`(?i)low[\s\-]*cut`)
	reCropTops      = regexp.MustCompile(`(?i)crop tops?`)
	reCroppedHems   = regexp.MustCompile(`(?i)cropped hems?`)
	reCropped       = regexp.MustCompile(`(?i)\bcropped\b`)
	reBraletteStyle = regexp.MustCompile(`(?i)bralette[\s\-]?style|bra[\s\-]?style|sports bra|bralettes?`)
	reMidriff       = regexp.MustCompile(`(?i)\bmidriff\b|bare waist`)
	reRevealing     = regexp.MustCompile(`(?i)revealing|seductive|sultry|sensual|alluring`)
	reBarelyThere   = regexp.MustCompile(`(?i)barely[\s\-]*there`)
	reSheer         = regexp.MustCompile(`(?i)\bsheer\b|see[\s\-]*through|semi[\s\-]*transparent|translucent fabric`)
	reBedrooms      = regexp.MustCompile(`(?i)\bbedrooms?\b`)

	reThaiBra    = regexp.MustCompile(`บรา|เสื้อชั้นใน|ชั้นใน`)
	reThaiSponge = regexp.MustCompile(`ฟองน้ำ`)
	reThaiChest  = regexp.MustCompile(`หน้าอก|ทรงอก|เต้า`)
	reThaiWaist  = regexp.MustCompile(`เอวลอย`)
	reThaiSexy   = regexp.MustCompile(`เซ็กซี่|ยั่ว|โป๊|วับวาม`)

	// crop word with JS negative lookahead (?!\s*(the|this|that)\b) emulated below.
	reCropWord  = regexp.MustCompile(`(?i)\bcrop\b`)
	reCropAfter = regexp.MustCompile(`(?i)^\s*(the|this|that)\b`)
)

// replaceCropWord replaces a standalone "crop" with "shorter length" unless it
// is followed by the/this/that (JS: /\bcrop\b(?!\s*(the|this|that)\b)/gi).
func replaceCropWord(s string) string {
	locs := reCropWord.FindAllStringIndex(s, -1)
	if locs == nil {
		return s
	}
	var b strings.Builder
	last := 0
	for _, loc := range locs {
		b.WriteString(s[last:loc[0]])
		if reCropAfter.MatchString(s[loc[1]:]) {
			b.WriteString(s[loc[0]:loc[1]])
		} else {
			b.WriteString("shorter length")
		}
		last = loc[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// Sanitize ports sanitize(t): apply SAFE_SWAP in order, then Tidy.
func Sanitize(t string) string {
	out := t
	out = reSpaghetti.ReplaceAllString(out, "thin shoulder straps")
	out = reBraGroup.ReplaceAllString(out, "sleeveless knit top")
	out = reChestPads.ReplaceAllString(out, "soft inner lining")
	out = reCleavage.ReplaceAllString(out, "neckline")
	out = reChestEmphasis.ReplaceAllString(out, "neckline detail")
	out = reUpperChest.ReplaceAllString(out, "collarbone area")
	out = reDeepNeck.ReplaceAllString(out, "wide scoop neckline")
	out = reLowCut.ReplaceAllString(out, "wide")
	out = reCropTops.ReplaceAllString(out, "shorter-length top")
	out = reCroppedHems.ReplaceAllString(out, "shorter hem")
	out = reCropped.ReplaceAllString(out, "shorter-length")
	out = replaceCropWord(out)
	out = reBraletteStyle.ReplaceAllString(out, "sleeveless knit top")
	out = reMidriff.ReplaceAllString(out, "waistline")
	out = reRevealing.ReplaceAllString(out, "everyday casual")
	out = reBarelyThere.ReplaceAllString(out, "minimal")
	out = reSheer.ReplaceAllString(out, "fine open-weave")
	out = reBedrooms.ReplaceAllString(out, "living room")
	out = reThaiBra.ReplaceAllString(out, "เสื้อตัวใน")
	out = reThaiSponge.ReplaceAllString(out, "ซับในนุ่ม")
	out = reThaiChest.ReplaceAllString(out, "ช่วงบน")
	out = reThaiWaist.ReplaceAllString(out, "ทรงเสื้อ")
	out = reThaiSexy.ReplaceAllString(out, "ดูดี")
	return Tidy(out)
}

// ---- ambience swap (AMB_SWAP) ----------------------------------------------

var (
	reAmb1 = regexp.MustCompile(`(?i)(a )?(low |soft |faint |distant )*(murmur|hum) of (conversation|chatter|voices|talking)[^,.]*`)
	reAmb2 = regexp.MustCompile(`(?i)(soft |low |distant |background )*(chatter|conversation|voices|talking|people talking)[^,.]*`)
	reAmb3 = regexp.MustCompile(`(?i)laughter|laughing`)
)

// AmbSafe ports ambSafe(t).
func AmbSafe(t string) string {
	out := t
	out = reAmb1.ReplaceAllString(out, "a faint distant kitchen hum")
	out = reAmb2.ReplaceAllString(out, "a faint distant hum")
	out = reAmb3.ReplaceAllString(out, "gentle background tone")
	return Tidy(out)
}

// ---- laughFix / talkFix -----------------------------------------------------

var (
	reLipsClosedClip  = regexp.MustCompile(`Her lips stay closed for the entire clip and she never speaks\.`)
	reLipsClosedShort = regexp.MustCompile(`Her lips stay closed, she is not speaking\.`)
	reLipsWholeScene  = regexp.MustCompile(`Her lips stay closed for the whole scene[^.]*\.`)
	reLipsClosed      = regexp.MustCompile(`\bLips closed\.`)
	reClosedLipSmile  = regexp.MustCompile(`closed-lip smile`)
	reClosedLipExpr   = regexp.MustCompile(`closed-lip expression`)
	reNoLipMovement   = regexp.MustCompile(`no lip movement, no mouth opening, no visible speaking, `)
	reNoTalking       = regexp.MustCompile(`no talking, `)
)

// LaughFix ports laughFix(t).
func LaughFix(t string) string {
	out := t
	out = reLipsClosedClip.ReplaceAllString(out, "She never says a word — her only sounds are a soft unforced laugh and quiet breathing.")
	out = reLipsClosedShort.ReplaceAllString(out, "She wears a warm natural smile, lips parted a little as if she has just laughed at something, but she is not saying any words.")
	out = reLipsWholeScene.ReplaceAllString(out, "She says no words at all, only a soft natural laugh.")
	out = reLipsClosed.ReplaceAllString(out, "Smiling softly.")
	out = reClosedLipSmile.ReplaceAllString(out, "warm natural smile")
	out = reClosedLipExpr.ReplaceAllString(out, "warm natural expression")
	out = reNoLipMovement.ReplaceAllString(out, "no spoken words, no dialogue, no mouthing words, no lip sync, ")
	return out
}

// TalkFix ports talkFix(t).
func TalkFix(t string) string {
	out := t
	out = reLipsClosedClip.ReplaceAllString(out, "She talks naturally to a friend just off camera throughout, with relaxed everyday mouth movement.")
	out = reLipsClosedShort.ReplaceAllString(out, "She is mid-sentence, talking naturally to a friend just off camera.")
	out = reLipsWholeScene.ReplaceAllString(out, "She talks naturally throughout the scene.")
	out = reLipsClosed.ReplaceAllString(out, "Talking naturally.")
	out = reClosedLipSmile.ReplaceAllString(out, "soft natural smile")
	out = reClosedLipExpr.ReplaceAllString(out, "soft natural expression")
	out = reNoLipMovement.ReplaceAllString(out, "")
	out = reNoTalking.ReplaceAllString(out, "")
	return out
}

// ---- activeScript -----------------------------------------------------------

// ActiveScript ports activeScript(p): the picked script's four lines when all
// four are non-empty, otherwise four blank lines.
func ActiveScript(p model.Product) []string {
	if p.Pick >= 0 && p.Pick < len(p.Scripts) {
		lines := p.Scripts[p.Pick].Lines
		if len(lines) >= 4 &&
			strings.TrimSpace(lines[0]) != "" &&
			strings.TrimSpace(lines[1]) != "" &&
			strings.TrimSpace(lines[2]) != "" &&
			strings.TrimSpace(lines[3]) != "" {
			return []string{lines[0], lines[1], lines[2], lines[3]}
		}
	}
	return []string{"", "", "", ""}
}

// ---- risk scanner (scanRisk / RISKY) ---------------------------------------

// RiskHit is one flagged risky phrase and its safe replacement.
type RiskHit struct {
	Found string `json:"found"`
	Good  string `json:"good"`
}

type riskRule struct {
	re   *regexp.Regexp
	good string
}

var riskyRules = []riskRule{
	{regexp.MustCompile(`บรา|เสื้อชั้นใน|ชั้นใน`), "เสื้อตัวใน"},
	{regexp.MustCompile(`ฟองน้ำ|เสริมฟองน้ำ`), "ซับในนุ่ม"},
	{regexp.MustCompile(`หน้าอก|ทรงอก|เต้า|เอวลอย`), "ช่วงบน หรือ ทรงเสื้อ"},
	{regexp.MustCompile(`เซ็กซี่|เผ็ด|ยั่ว|โชว์`), "ดูดี"},
	{regexp.MustCompile(`ไม่ต้องใส่บรา`), "ใส่ตัวเดียวจบ"},
	{regexp.MustCompile(`(?i)\bbra\b|bralette|underwear|lingerie|camisole`), "sleeveless knit top"},
	{regexp.MustCompile(`(?i)chest pad|padded cup|inner padding`), "soft inner lining"},
	{regexp.MustCompile(`(?i)cleavage|bust|chest emphasis|breast`), "neckline หรือ upper body"},
	{regexp.MustCompile(`(?i)revealing|seductive|sultry|sensual`), "everyday, casual"},
	{regexp.MustCompile(`(?i)crop top|cropped hem|midriff|bare waist`), "hem ending at the hip, fully tucked in"},
	{regexp.MustCompile(`(?i)spaghetti strap`), "thin shoulder strap"},
	{regexp.MustCompile(`(?i)\bbedroom\b`), "living room"},
	{regexp.MustCompile(`(?i)\bteen\b|schoolgirl|\b1[0-7][- ]year[- ]old\b`), "25 years old"},
}

// ScanRisk ports scanRisk(txt).
func ScanRisk(text string) []RiskHit {
	var hits []RiskHit
	for _, r := range riskyRules {
		m := r.re.FindAllString(text, -1)
		if len(m) == 0 {
			continue
		}
		seen := map[string]bool{}
		var uniq []string
		for _, x := range m {
			if !seen[x] {
				seen[x] = true
				uniq = append(uniq, x)
			}
		}
		hits = append(hits, RiskHit{Found: strings.Join(uniq, ", "), Good: r.good})
	}
	return hits
}
