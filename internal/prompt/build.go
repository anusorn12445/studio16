package prompt

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"studio16/internal/model"
)

// Regexes used only by Build (translated to RE2 from the HTML buildPrompt).
var (
	reTypeStrap = regexp.MustCompile(`(?i)\b(thin |adjustable |very thin )?shoulder straps?\b`)
	reTypeNeck  = regexp.MustCompile(`(?i)\b(wide |scoop |square |v-)?neckline\b`)

	reIsOpen       = regexp.MustCompile(`(?i)lace|open-weave|mesh|eyelet|crochet`)
	reBuiltIn      = regexp.MustCompile(`(?i)built[\s\-]?in|inner layer|inner lining|soft inner|shaping`)
	reStraightNeck = regexp.MustCompile(`(?i)\b(straight|horizontal|square|bandeau|band|straight[\s\-]?across)\b`)
	reShortHem     = regexp.MustCompile(`(?i)\b(above the waist|at the waist|natural waist|navel|shorter length|shorter-length)\b`)
	reMM           = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*mm`)
	reFineStrap    = regexp.MustCompile(`(?i)\b(hair[\s\-]?thin|string strap|thread[\s\-]?thin|pencil[\s\-]?width|very thin|ultra[\s\-]?thin)\b`)
	reWhite        = regexp.MustCompile(`(?i)\b(pure white|bright white|optic white|crisp white|snow white)\b`)
	reOnlyWhite    = regexp.MustCompile(`(?i)^\s*white\s*$`)
	reNone         = regexp.MustCompile(`(?i)^\s*(none|no|ไม่มี)\b`)
	reNoHW         = regexp.MustCompile(`(?i)\b(non[\s\-]?adjustable|not adjustable|fixed strap|no slider|without slider|no hardware|no adjuster)\b`)
	reHasHW        = regexp.MustCompile(`(?i)\b(slider|adjuster|adjustable|buckle|clasp|metal ring|hardware)\b`)
	reBag          = regexp.MustCompile(`(?i)\bbag\b|tote|clutch|purse`)
	reBagPhrase    = regexp.MustCompile(`(?i)[^,]*\b(bag|tote|clutch|purse)\b[^,]*,?`)
)

// Build ports buildPrompt(p) from the STUDIO 16 HTML app.
func Build(p model.Product) string {
	const NL = "\n"
	F := fmtFor(p.Format)
	base := ResolveBase(p)

	sc := ActiveScript(p)
	L := []string{Sanitize(sc[0]), Sanitize(sc[1]), Sanitize(sc[2]), Sanitize(sc[3])}

	C := Sanitize(EnOnly(p.HeroColor, "cream white"))
	typeSrc := Sanitize(EnOnly(p.Type, "sleeveless knit top"))
	typeSrc = reTypeStrap.ReplaceAllString(typeSrc, "")
	typeSrc = reTypeNeck.ReplaceAllString(typeSrc, "")
	TYPE := Tidy(typeSrc)

	GARMENT := Sanitize(EnOnly(p.Garment, "sleeveless knit top, wide U-shaped neckline sitting moderately low across the collarbone, very thin shoulder straps, smooth matte fabric"))
	BOTTOM := Sanitize(EnOnly(p.Bottom, "high-waisted light blue wide-leg jeans that reach the ankle"))
	SHOES := Sanitize(EnOnly(p.Shoes, "simple white low-top canvas sneakers"))
	ACC := Sanitize(EnOnly(p.Accessories, "layered thin gold necklaces and small gold hoop earrings"))

	shoeLine := ""
	if F.Feet {
		shoeLine = " On her feet, " + SHOES + "."
	}

	wmTxt := strings.TrimSpace(p.WmText)
	wmOn := p.Wm && wmTxt != ""
	wmWhere := map[string]string{
		"bottom-right": "lower right corner",
		"bottom-left":  "lower left corner",
		"top-right":    "upper right corner",
		"top-left":     "upper left corner",
		"center":       "centre of the frame",
	}[p.WmPos]
	if wmWhere == "" {
		wmWhere = "lower right corner"
	}

	wmBlock := ""
	if wmOn {
		var spelled []string
		for _, ch := range wmTxt {
			if ch == ' ' {
				spelled = append(spelled, "space")
			} else {
				spelled = append(spelled, string(ch))
			}
		}
		wmBlock = "=====================================================================" + NL +
			"WATERMARK" + NL +
			"=====================================================================" + NL +
			`Overlay the text "` + wmTxt + `" onto the frame as a light watermark.` + NL +
			"Place it in the " + wmWhere + ", inset from the edge by roughly one" + NL +
			"twentieth of the frame width so it never touches the border. Render it" + NL +
			"small — about one eighteenth of the frame width in cap height — in a" + NL +
			"clean simple sans-serif, plain white at roughly 25 percent opacity, no" + NL +
			"shadow, no outline, no box behind it, no logo mark." + NL + NL +
			"It appears exactly once in the whole frame. There is one watermark and" + NL +
			"no other text anywhere — not a second copy, not a faint repeat, not a" + NL +
			"tiled pattern." + NL + NL +
			"The watermark is a flat overlay, not part of the scene: it stays in" + NL +
			"exactly the same screen position at exactly the same size and opacity" + NL +
			"for every frame, it never moves with the camera or the subject, it never" + NL +
			"wraps onto clothing or furniture, and it is never lit or shadowed by the" + NL +
			"room. It must never cover her face or the garment being sold." + NL +
			"Spell it exactly as written, character for character: " +
			strings.Join(spelled, " then ") + "." + NL +
			"It is exactly " + strconv.Itoa(utf8.RuneCountInString(wmTxt)) + " characters long. Do not add a letter," + NL +
			"do not drop one, do not swap one for a similar shape." + NL + NL
	}

	two := F.TwoModels
	labelNeg := "no text overlay, "
	if two && F.Label {
		labelNeg = ""
	}
	wmNeg := ""
	if wmOn {
		wmNeg = "no second watermark, no repeated watermark, no two watermarks, no tiled watermark, no duplicated text, no text anywhere else in the frame, no misspelled watermark, no extra letters in the watermark, no dropped letters, no altered characters, no moving watermark, no watermark on her face, no watermark on the garment, no logo, no subtitles,"
	} else {
		wmNeg = labelNeg + "no watermark, no subtitles, no logos,"
	}

	cafeItem := Sanitize(EnOnly(p.CafeItem, CAFE_ITEMS[0].En))
	SETTING := strings.Replace(F.Setting, "{CAFE}", cafeItem, 1)

	SP := p.Spec
	specPairs := [][2]string{
		{"Neckline shape and depth", SP.Neckline},
		{"Strap width and how far apart they sit", SP.Straps},
		{"Armhole depth", SP.Armhole},
		{"Fabric, texture and finish", SP.Fabric},
		{"Pattern printed or woven into the fabric", SP.Print},
		{"Applied trims, lace, bows, embroidery and lettering", SP.Trim},
		{"Where the hem ends", SP.Hem},
		{"Fit through the body", SP.Fit},
		{"Seams, panels and other details", SP.Details},
	}
	var specRows []string
	for _, r := range specPairs {
		if strings.TrimSpace(r[1]) != "" {
			specRows = append(specRows, "  - "+r[0]+": "+Sanitize(r[1]))
		}
	}
	specBlock := ""
	if len(specRows) > 0 {
		specBlock = "CHECK EACH OF THESE AGAINST THE REFERENCE BEFORE YOU RENDER. If the text" + NL +
			"and the reference ever disagree, the reference wins:" + NL + strings.Join(specRows, NL)
	} else {
		specBlock = "Read every construction detail directly off the reference image."
	}

	hemTxt := strings.TrimSpace(SP.Hem)
	hemLine := ""
	if hemTxt != "" {
		hemLine = " The hem sits exactly where the reference shows it: " + Sanitize(hemTxt) + "."
	}
	blobLine := ""
	if len(specRows) > 0 {
		blobLine = strings.TrimSpace("Hold every one of those against the picture, not against memory." + hemLine)
	} else {
		blobLine = strings.TrimSpace("Written reading of the reference: " + GARMENT + "." + hemLine)
	}

	isOpen := reIsOpen.MatchString(GARMENT + " " + TYPE + " " + SP.Fabric)
	builtIn := reBuiltIn.MatchString(SP.Details + " " + GARMENT)
	builtInLine := ""
	if builtIn {
		builtInLine = " The garment has its own soft shaping sewn inside, so the front panel reads as one smooth unbroken surface with a gentle even curve and no seams, edges or outlines showing through from underneath. Nothing is worn under it and nothing shows above the neckline."
	}

	neckTxt := SP.Neckline + " " + GARMENT
	straightNeck := reStraightNeck.MatchString(neckTxt)
	shortHem := reShortHem.MatchString(hemTxt)

	var shapeNegParts []string
	if !straightNeck {
		shapeNegParts = append(shapeNegParts, "no strapless band top, no bandeau, no tube top, no elasticated band across the upper body,")
	}
	if !shortHem {
		shapeNegParts = append(shapeNegParts, "no shortened top, no top ending above the waist, no hem raised toward the ribs,")
	}
	shapeNeg := strings.Join(shapeNegParts, " ")

	var shapeLockParts []string
	if straightNeck {
		shapeLockParts = append(shapeLockParts, "The neckline runs straight and level across, exactly as the reference shows — do not curve it into a scoop or a V, and do not raise it.")
	}
	if shortHem {
		shapeLockParts = append(shapeLockParts, "This is a shorter-length top by design: the hem sits where the reference shows it and is not lengthened. Coverage comes from pulling the high-waisted lower garment up to meet it, never from stretching the top downward.")
	}
	shapeLock := strings.Join(shapeLockParts, " ")

	strapTxt := strings.TrimSpace(SP.Straps)
	if strapTxt == "" {
		strapTxt = GARMENT
	}
	mmStr := ""
	if m := reMM.FindStringSubmatch(strapTxt); m != nil {
		mmStr = m[1]
	}
	fineStrap := reFineStrap.MatchString(strapTxt)
	if !fineStrap && mmStr != "" {
		if v, err := strconv.ParseFloat(mmStr, 64); err == nil && v <= 6 {
			fineStrap = true
		}
	}
	strapFidelity := ""
	if fineStrap {
		strapFidelity = "STRAP FIDELITY — the straps on this garment are string-thin and this is the" + NL +
			"single detail that fails most often. Render them as fine lines, not as" + NL +
			"bands: each strap is roughly one twentieth of the shoulder width, about as" + NL +
			"thick as a shoelace, and it should read as a delicate cord lying on the" + NL +
			"shoulder. If a strap looks like a strip of fabric with two visible edges" + NL +
			"and a surface between them, it is far too thick — halve it and render" + NL +
			"again. Keep both straps identical in width along their whole length, and" + NL +
			"keep them exactly as far apart as the reference shows."
	}

	pureWhite := reWhite.MatchString(C) || reOnlyWhite.MatchString(C)
	whiteLine := ""
	whiteNeg := ""
	if pureWhite {
		whiteLine = " This is a true clean white, the same white as fresh paper — not cream, not off-white, not ivory, not ecru, not bone and not beige. Judge it against the whitest thing in the frame and match that, not the warm surfaces around it."
		whiteNeg = "no cream garment, no off-white garment, no ivory garment, no ecru, no bone colour, no beige garment,"
	}

	printTxt := strings.TrimSpace(SP.Print)
	trimTxt := strings.TrimSpace(SP.Trim)
	hasPrint := printTxt != "" && !reNone.MatchString(printTxt)
	hasTrim := trimTxt != "" && !reNone.MatchString(trimTxt)

	detailBlock := ""
	if hasPrint || hasTrim {
		detailBlock = "DECORATION FIDELITY — this garment is not plain, and generators default to" + NL +
			"plain. Every element below must appear in the render, in the right place," + NL +
			"at the right scale. A version without them is a different product:" + NL
		if hasPrint {
			detailBlock += "  - Pattern: " + Sanitize(printTxt) + NL
		}
		if hasTrim {
			detailBlock += "  - Applied trim: " + Sanitize(trimTxt) + NL
		}
		detailBlock += "Keep the pattern at the same scale as the reference — do not enlarge the" + NL +
			"motif or wash it out into a solid colour. Keep every trim exactly where it" + NL +
			"sits, at the same width, in the same colour. Reproduce any lettering" + NL +
			"character for character. Do not simplify, do not omit, do not relocate." + NL + NL
	}
	detailNeg := ""
	if hasPrint || hasTrim {
		detailNeg = "no plain solid version of the garment, no missing pattern, no missing lace, no missing trim, no missing bow, no missing embroidery, no simplified decoration, no enlarged motif, no relocated trim,"
	}
	strapNeg := ""
	if fineStrap {
		strapNeg = "no wide straps, no broad shoulder straps, no vest straps, no tank-style straps, no strap widened into a band, no straps thicker than a cord,"
	}

	noHW := reNoHW.MatchString(strapTxt)
	hasHW := !noHW && reHasHW.MatchString(strapTxt)
	var strapLock string
	switch {
	case noHW:
		strapLock = "the straps left completely plain along their whole length, with no slider, adjuster, buckle, ring or hardware of any kind"
	case hasHW:
		strapLock = "the strap hardware exactly as the reference shows it"
	default:
		strapLock = "any strap hardware exactly as the reference shows it, and none at all where the reference shows none"
	}
	hwNeg := "no invented strap hardware,"
	if noHW {
		hwNeg = "no strap sliders, no strap adjusters, no buckles on the straps, no rings on the straps, no clasps, no hardware on the straps, no adjustable straps, no knots or ties on the straps,"
	}

	cropLine := " COVERAGE — non-negotiable: the bottom edge of the top and the top edge of the lower garment meet and overlap by several centimetres. There is no gap between them and no skin is visible between the two garments at any moment, in any frame, in any pose. If the top is a shorter length, the high-waisted lower garment is pulled up to meet it. If they would not otherwise meet, lengthen the lower garment upward — never shorten the top."

	liningLine := ""
	if isOpen {
		liningLine = " The top is fully lined throughout with a smooth opaque under-layer in a matching colour, so the garment reads as solid and nothing shows through at any point."
	}

	ACC2 := ACC
	if reBag.MatchString(F.Props) {
		ACC2 = Tidy(reBagPhrase.ReplaceAllString(ACC, ""))
	}

	negCam := F.NegCam
	if negCam == "" {
		negCam = "no camera drift, no dolly in, no dolly out, no push-in, no pull-back, no zoom, no framing change, no subject scale change, no shrinking subject, no widening shot,"
	}
	camLock1 := F.CamLock1
	if camLock1 == "" {
		camLock1 = "The framing is fixed for all 8 seconds. The filming phone is locked in place — only micro-jitter of one or two pixels. Her head stays at the same height in frame and her body stays exactly the same size from the first frame to the last. No drift, no dolly, no push-in, no pull-back, no reframing."
	}
	camLock2 := F.CamLock2
	if camLock2 == "" {
		camLock2 = camLock1
	}
	motionDefault := "Movement is continuous and unhurried from the first frame to the last — she is never frozen. Weight shifts subtly, the head turns a beat after the shoulders, hair carries momentum and settles, and she blinks and breathes throughout. Every movement eases in and eases out; nothing snaps or holds rigid."
	motion1 := F.Motion1
	if motion1 == "" {
		motion1 = motionDefault
	}
	motion2 := F.Motion2
	if motion2 == "" {
		motion2 = motionDefault
	}

	AM := "talk"
	if !F.Voice {
		AM = AudioMode(p)
	}

	cA := Sanitize(EnOnly(p.PairA, C))
	cB := Sanitize(EnOnly(p.PairB, C))
	cC := Sanitize(EnOnly(p.PairC, cA))
	cD := Sanitize(EnOnly(p.PairD, cB))
	sA := Sanitize(EnOnly(p.StyleA, "high-waisted wide-leg tailored trousers in cream, a slim leather belt, plain low-heeled shoes"))
	sB := Sanitize(EnOnly(p.StyleB, "a high-waisted long skirt in soft ivory, a slim leather belt, plain low-heeled shoes"))
	sC := Sanitize(EnOnly(p.StyleC, sA))
	sD := Sanitize(EnOnly(p.StyleD, sB))

	identityB := ""
	if two {
		identityB = "=====================================================================" + NL +
			"IDENTITY_BLOCK_B (verbatim, never reword)" + NL +
			"=====================================================================" + NL +
			"Model B — a Thai woman, 26 years old, clearly a different person from" + NL +
			"Model A. Slightly longer face with a defined V-line jaw, straight brows," + NL +
			"narrower almond eyes and thinner lips. Very dark brown hair, almost" + NL +
			"black, worn down loose with a centre part and no bangs, straight past" + NL +
			"the shoulders. Fair porcelain skin with cool pink undertones. Slim" + NL +
			"natural build. She stands on the RIGHT of the frame for the entire clip" + NL +
			"and never swaps sides." + NL + NL +
			"Model A stands on the LEFT for the entire clip and never swaps sides." + NL +
			"Both women are the same height and stand at the same distance from the" + NL +
			"camera, shoulder to shoulder with a small gap between them, so neither" + NL +
			"appears larger or closer than the other." + NL + NL
	}

	labelBlock := ""
	if two && F.Label {
		labelBlock = "=====================================================================" + NL +
			"LABEL" + NL +
			"=====================================================================" + NL +
			"Render a caption across the top of the frame, inset from the top edge by" + NL +
			`about one tenth of the frame height, centred: the word "` + cA + `" on the` + NL +
			`left, a small four-pointed sparkle in the middle, and the word "` + cB + `"` + NL +
			"on the right. Each word sits directly above the woman wearing that colour." + NL + NL +
			"Set it in a clean simple sans-serif, plain white, about one fourteenth of" + NL +
			"the frame width in cap height, with a soft dark drop shadow so it stays" + NL +
			"readable against the wall. It is a flat overlay pinned to the screen: it" + NL +
			"never moves, never changes size, never follows the camera or the subjects," + NL +
			"and never wraps onto the wall or the clothing." + NL + NL +
			"Spell both words exactly as written, character for character. The caption" + NL +
			"appears exactly once — no second copy, no repeat, no other text anywhere." + NL +
			`In Scene 2 the caption reads "` + cC + `" and "` + cD + `" instead, in the` + NL +
			"same position, same size and same style." + NL + NL
	}

	twinLock := ""
	if two {
		twinLock = "The two tops must be recognisably the same garment. If a viewer paused the" + NL +
			"frame and compared them, every measurement would match: the neckline sits" + NL +
			"at the same depth and width on both, the straps are the same thickness and" + NL +
			"the same distance apart, the hem ends at the same point, and any pattern or" + NL +
			"trim appears at the same scale in the same place. A difference in" + NL +
			"construction between the two is a failure — only the colour may differ." + NL + NL
	}
	twinNeg := ""
	if two {
		twinNeg = "no different garment on the second model, no mismatched necklines, no mismatched strap width, no mismatched hem, no restyled second top, no pattern on one and not the other, no trim missing from one of the two,"
	}

	speaks := AM == "talk"
	laughs := AM == "laugh"
	T := func(x string) string { return x }
	if !F.Voice {
		if laughs {
			T = LaughFix
		} else if speaks {
			T = TalkFix
		}
	}
	POSE1, POSE2 := T(F.Pose1), T(F.Pose2)
	LOCK1, LOCK2 := T(F.Lock1), T(F.Lock2)

	mapT := func(in []string) []string {
		out := make([]string, len(in))
		for i, s := range in {
			out[i] = T(s)
		}
		return out
	}
	TIM1 := mapT(F.T1)
	TIM2 := mapT(F.T2)
	if laughs {
		beat := "ANOTHER BEAT — somewhere around 3.0s she breaks into a small unforced laugh, eyes crinkling and shoulders lifting slightly for about half a second, then settles back into a warm smile. It reads as something genuinely amusing off camera, not a performance, and she says nothing."
		TIM1 = append(TIM1, beat)
		TIM2 = append(TIM2, beat)
	}
	FNEG := T(F.Neg)

	pipeline := "RUN THIS NOW, IN ORDER, WITHOUT PAUSING:" + NL +
		"  Step 1 — Generate ONE image from IMAGE_PROMPT_1 using the attached" + NL +
		"           clothing reference." + NL +
		"  Step 2 — Generate ONE image from IMAGE_PROMPT_2 using TWO references:" + NL +
		"           the clothing reference for the top, AND the Step 1 image as" + NL +
		"           the face and styling reference. Same person, not a lookalike." + NL +
		"  Step 3 — Using the Step 1 image as the first frame, generate ONE video" + NL +
		"           from VIDEO_PROMPT_1, 8 seconds." + NL +
		"  Step 4 — Using the Step 2 image as the first frame, generate ONE video" + NL +
		"           from VIDEO_PROMPT_2, 8 seconds." + NL +
		"  Step 5 — Present all four assets together, labelled by scene."

	quantity := "OUTPUT QUANTITY — strict:" + NL +
		"Exactly 2 images and 2 videos in total. One image and one video per" + NL +
		"scene. Never generate variants, alternates, or extra takes."

	voiceBlock := ""
	if speaks {
		voiceLine := p.Voice
		if voiceLine == "" {
			voiceLine = "A Thai woman in her mid-twenties, light clear voice, warm and conversational."
		}
		voiceBlock = "=====================================================================" + NL +
			"VOICE_BLOCK (use for the speaking voice in both videos)" + NL +
			"=====================================================================" + NL +
			voiceLine + NL +
			"She speaks Thai. Her voice is recorded on a phone in a small furnished space: close, dry, slightly soft at the edges, with faint natural ambience underneath. No studio polish, no reverb, no echo, no music, no second voice." + NL + NL
	}

	audioSpeak := "AUDIO: [VOICE_BLOCK]. Her voice only, quiet natural ambience, no background music, no echo, no reverb, no second speaker."
	ambSrc := F.Amb
	if ambSrc == "" {
		ambSrc = "quiet natural location ambience, mixed low"
	}
	ambience := AmbSafe(ambSrc)

	laughAudio := "SOUND — this is a live take with its own natural sound, recorded on the phone" + NL +
		"as the shot happens. What the microphone picks up: " + ambience + ", and around" + NL +
		"the three second mark her own quiet unforced laugh, then a small contented" + NL +
		"breath near the end. Everything sits low and natural, the way a real phone" + NL +
		"clip sounds. She says nothing at all — her voice appears only as that soft" + NL +
		"laugh and her breathing, never as words, never as a sentence, and her mouth" + NL +
		"never shapes speech. Keep it free of music."

	var dlg1 string
	if speaks {
		if two {
			dlg1 = "DIALOGUE (Thai, two friends showing outfits to camera, never salespeople):" + NL +
				`0.0s-3.0s [Model A, on the left]: "` + L[0] + `"` + NL +
				`3.5s-8.0s [Model B, on the right]: "` + L[1] + `"` + NL + NL +
				"SOUND — a live take with its own natural sound: quiet showroom ambience, a faint room tone, the soft sound of fabric moving, and their two voices speaking the lines above. Nothing else. No music."
		} else {
			dlg1 = "DIALOGUE (Thai, casual warm tone, a real customer talking to a close friend, never a salesperson):" + NL +
				`0.0s-3.0s: "` + L[0] + `"` + NL +
				`3.5s-8.0s: "` + L[1] + `"` + NL + NL + audioSpeak
		}
	} else if laughs {
		dlg1 = laughAudio
	} else {
		dlg1 = "Her lips stay closed for the whole scene — this is a picture-only clip."
	}

	var dlg2 string
	if speaks {
		if two {
			dlg2 = "DIALOGUE (Thai, two friends, sincere recommendation tone):" + NL +
				`0.0s-4.0s [Model A, on the left]: "` + L[2] + `"` + NL +
				`4.5s-8.0s [Model B, on the right]: "` + L[3] + `"` + NL + NL +
				"SOUND — a live take with its own natural sound: quiet showroom ambience with their two voices only. No music."
		} else {
			dlg2 = "DIALOGUE (Thai, sincere recommendation tone, talking to a close friend):" + NL +
				`0.0s-4.0s: "` + L[2] + `"` + NL +
				`4.5s-8.0s: "` + L[3] + `"` + NL + NL + audioSpeak
		}
	} else if laughs {
		dlg2 = laughAudio
	} else {
		dlg2 = "Her lips stay closed for the whole scene — this is a picture-only clip."
	}

	// ---- interpolation fragments ----
	wmApply := ""
	if wmOn {
		wmApply = "\n- Apply the WATERMARK block to every image and every video, identically."
	}
	audioRule := ""
	if AM == "laugh" {
		audioRule = "- This clip has no dialogue but it is not silent. Generate the vocal track" + NL +
			"  described in each scene: soft non-verbal sounds only, laid over the" + NL +
			"  location ambience. Never generate words."
	} else {
		audioRule = "- These clips carry no dialogue, so do not request, describe or generate" + NL +
			"  any soundtrack for them. Produce picture only and leave the audio track" + NL +
			"  empty. Sound is added later in editing."
	}

	var outfit string
	if two {
		outfit = "OUTFIT — Both women wear the SAME product, the " + TYPE + " from the clothing" + NL +
			"reference, differing only in colourway: Model A on the left wears it in" + NL +
			cA + ", Model B on the right wears it in " + cB + ". Both tops are identical in" + NL +
			"every construction detail. Only the dye colour differs between them." + NL + NL +
			"Each is then styled with her own matching pieces, chosen to flatter that" + NL +
			"colourway: Model A wears " + sA + ". Model B wears " + sB + ". Every matching piece" + NL +
			"sits high on the waist so the waistline stays covered along its full width" + NL +
			"at all times, and stays plain and tonal with no pattern and no logo." + NL +
			"The two tops are what the eye reads first in every frame." + NL + NL +
			"In Scene 2 the same two women wear the same product again in the next" + NL +
			"pairing: Model A in " + cC + " styled with " + sC + ", and Model B in " + cD + " styled" + NL +
			"with " + sD + ". Nothing else in the room or about the women changes."
	} else {
		outfit = "OUTFIT — A " + C + " " + TYPE + " worn as the main piece, exactly as the reference" + NL +
			"shows it, no jacket or cardigan over it. Styled with" + NL +
			BOTTOM + ". The lower garment sits high on the waist so the waistline" + NL +
			"stays covered along its full width at all times."
	}

	garmentTruth := "this garment"
	if two {
		garmentTruth = "the top worn by BOTH women"
	}
	garmentCopy := " in " + C + ", changing nothing but the colour"
	if two {
		garmentCopy = " on each of them, changing nothing but the colour: " + cA + " on the left and " + cB + " on the right"
	}

	twinCheck := ""
	if two {
		twinCheck = NL +
			"  6. Twin check. Put the two tops side by side in your own render. Same" + NL +
			"     neckline, same straps, same hem, same trim, same pattern scale. If" + NL +
			"     one is plainer or shaped differently from the other, it is wrong —" + NL +
			"     make them match and render again."
	}

	idbImg := ""
	if two {
		idbImg = "\n[IDENTITY_BLOCK_B]"
	}
	labelImg1 := ""
	labelImg2 := ""
	if two && F.Label {
		labelImg1 = "\n[LABEL]"
		labelImg2 = "\n[LABEL — with the Scene 2 colours]"
	}
	secondPersonNeg := "no second person in frame, "
	if two {
		secondPersonNeg = ""
	}

	// ---- assemble the ROLE template ----
	var b strings.Builder
	w := func(s string) { b.WriteString(s) }

	w("ROLE\n" + base.Role + "\n\n" +
		"MODEL SETTINGS\n" + base.ModelSettings + "\n\n")
	w(quantity)
	w("\n\n")
	w(pipeline)
	w("\n\n")
	w("BAG AND OBJECT PHYSICS:\n" +
		"Every object obeys gravity and is clearly supported. A bag is either set\n" +
		"down resting on its base on a solid surface, or hanging by a strap that\n" +
		"is plainly over a shoulder — never floating, never mid-air, never\n" +
		"attached to nothing, never half-merged with a chair or a table. Straps\n" +
		"are continuous and traceable from end to end. If a bag cannot be shown\n" +
		"clearly supported, leave it out of the frame entirely rather than\n" +
		"rendering it ambiguously.\n\n" +
		"PRODUCT FOCUS — the single most important rule:\n" +
		"The top is the product being sold and must stay the clear subject of\n" +
		"every frame. Compose so it sits in the centre of attention, fully lit,\n" +
		"fully visible and never obscured by hair, hands, a bag, an arm or any\n" +
		"other object. Every other element — the lower garment, the shoes, the\n" +
		"accessories, the bag, the background — stays plainer and quieter than the\n" +
		"top. Her hands never cover it and never fidget with anything else. Where\n" +
		"the timing calls for a gesture, that gesture points attention back at the\n" +
		"top, never away from it.\n\n" +
		"CONTEXT\n" +
		"This is an everyday clothing catalogue shoot. The subject is fully\n" +
		"dressed in ordinary casual daywear throughout, worn the way anyone would\n" +
		"wear it to go out. Treat it as ordinary apparel photography. Describe\n" +
		"clothing by its fabric, seams, stitching and drape, never by the body\n" +
		"underneath it. Keep every pose ordinary and relaxed: shoulders back,\n" +
		"standing straight, no leaning toward the camera.\n\n" +
		"RULES\n" +
		"- Append GLOBAL_NEGATIVE to every image and every video.")
	w(wmApply)
	w("\n- Copy IDENTITY_BLOCK verbatim into every prompt. Never reword it.\n" +
		"- Use the clothing reference for the top's construction only — ignore its\n" +
		"  background, lighting, colour grading, text and any people in it.\n" +
		"- The clothing reference outranks every written description of the\n" +
		"  garment, in every step. Where they disagree, copy what you can see.\n" +
		"  Before you finish an image, compare the neckline shape, the strap\n" +
		"  spacing, the hem position, the knit gauge and the seam lines against the\n" +
		"  reference and correct any drift. The hem position is the one that fails\n" +
		"  most often: match the landmark named in the spec exactly, neither higher\n" +
		"  nor lower. A shorter top stays short and a longer top stays long.\n" +
		"- In Step 2, the Step 1 image outranks the text wherever they differ on\n" +
		"  face, hair, skin tone or location.\n" +
		"- Never mention any price in dialogue.\n" +
		"- CONTENT OF SPEECH — every spoken line is about the top being sold and\n" +
		"  nothing else: its look, cut, neckline, straps, fabric, colour, fit and\n" +
		"  how to wear or style it. She never talks about the location, the room,\n" +
		"  the cafe, the coffee or any drink, the food, the weather, her day, the\n" +
		"  camera or anything unrelated to the garment. Keep 100 percent of the\n" +
		"  dialogue on this one piece of clothing.\n" +
		"- Framing is fixed in both videos. The camera never drifts, dollies,\n" +
		"  pushes in or pulls back, and the subject never changes size in frame.\n" +
		"- Nothing may enter or leave the frame during a clip.\n" +
		"- If a generation is refused, retry once with the top in black and the\n" +
		"  framing lengthened to show more of the lower garment, then continue the\n" +
		"  run and note it at the end.\n" +
		"- If a call fails, report the raw error text only and continue with the\n" +
		"  remaining steps.\n")
	w(audioRule)
	w("\n- If a video call still fails with an audio error, the picture itself is\n" +
		"  fine and only the soundtrack failed. Retry that same clip once more,\n" +
		"  again requesting picture only. If it fails a second time, deliver that\n" +
		"  scene as its still image and say so — never abandon the run and never\n" +
		"  skip Scene 2.\n" +
		"- Scene 2 must be delivered even if Scene 1 needed a retry. Treat the two\n" +
		"  scenes as independent: a failure in one never cancels the other.\n\n" +
		"=====================================================================\n" +
		"IDENTITY_BLOCK (verbatim, never reword)\n" +
		"=====================================================================\n" +
		base.Identity + "\n\n")
	w(identityB)
	w(voiceBlock)
	w("=====================================================================\n" +
		"STYLE_BLOCK (apply to both scenes)\n" +
		"=====================================================================\n" +
		base.Style + "\n\n")
	w(outfit)
	w(" The lower garment is\n" +
		"a quiet supporting piece: its colour, pattern and volume stay simpler\n" +
		"than the top's.")
	w(builtInLine)
	w(cropLine)
	w(liningLine)
	w(shoeLine)
	w(" Accessories: ")
	w(ACC2)
	w(" — nothing\n" +
		"larger or brighter than the top, and nothing is added or removed at any\n" +
		"point. ")
	w(F.Props)
	w("\n\nSETTING — ")
	w(SETTING)
	w("\n\nLIGHT — ")
	w(F.Light)
	w("\n\nCAMERA — ")
	w(F.Camera)
	w("\n\nWHITE BALANCE — neutral daylight at 5500K, the single most important\n" +
		"grading rule. Judge it against the scene: anything white in frame must\n" +
		"read as clean white, not cream, and pale walls stay neutral rather than\n" +
		"turning sandy. Her skin keeps its cool pink undertone under this light.\n" +
		"No warm filter, no amber wash, no golden-hour treatment, no tungsten\n" +
		"tint, no sepia, no teal-and-orange grade. If the frame looks even\n" +
		"slightly yellow overall, the white balance is wrong — cool it back to\n" +
		"neutral and render again.\n\n" +
		"LOOK — True-to-life colour with a clean neutral daylight balance, bright\n" +
		"and airy, milky highlights, low contrast, realistic skin rendering, no\n" +
		"smoothing.\n\n" +
		"GARMENT — REFERENCE IS THE AUTHORITY\n" +
		"The clothing reference image is the single source of truth for ")
	w(garmentTruth)
	w(". Reproduce it as an exact copy")
	w(garmentCopy)
	w(".\n\n")
	w(twinLock)
	w(" The written notes below are a reading aid only — they are there\n" +
		"to stop you drifting, never to override what you can see.\n\n")
	w(specBlock)
	w("\n\n")
	w(blobLine)
	w("\n\n")
	w(shapeLock)
	w("\n\n")
	w(strapFidelity)
	w("\n\n")
	w(detailBlock)
	w("FABRIC FIDELITY — match the knit gauge exactly as it appears in the\n" +
		"reference: the width of each rib, how closely the ribs are set, and which\n" +
		"way they run. Fine close-set ribbing must stay fine and close-set — do\n" +
		"not coarsen it into wide chunky ribbing, and do not flatten it into plain\n" +
		"jersey. Reproduce the fabric's matte or sheen level as seen, not as\n" +
		"imagined.\n\n" +
		"COLOUR FIDELITY — reproduce the colour as ")
	w(C)
	w(" exactly.")
	w(whiteLine)
	w(" Do not\n" +
		"warm it, do not push it toward beige, tan or ivory, and do not let the\n" +
		"room's light or any nearby surface tint the garment. The colour reads the\n" +
		"same in every frame.\n\n" +
		"VERIFY BEFORE YOU FINISH — put your render side by side with the\n" +
		"reference and check these four in order. They are the four that drift\n" +
		"most often, and any one of them wrong makes the product unrecognisable:\n" +
		"  1. Hem position. Hold it against the same landmark named above. If it\n" +
		"     sits higher or lower than the reference, correct it in that direction.\n" +
		"  2. Neckline shape. A U has a rounded bottom, a V comes to a point, and a\n" +
		"     straight neckline runs level across between the straps. Confirm which\n" +
		"     one the reference has and match it — never substitute one for another.\n" +
		"  3. Strap width and spacing. Very fine string straps must stay hair-thin —\n" +
		"     rendering them as broader flat bands is a common failure. Measure how\n" +
		"     far apart they sit relative to the shoulder width, not by eye alone.\n" +
		"  4. Knit gauge. Count the ribs across the width. If your render has\n" +
		"     visibly fewer, thicker ribs than the reference, redo it finer.\n" +
		"  5. Decoration. Every pattern, lace edge, bow, ruffle and piece of\n" +
		"     lettering named in the spec must be present, in place and at the\n" +
		"     right scale. If the render came out plainer than the reference,\n" +
		"     it is wrong — add them and render again.")
	w(twinCheck)
	w("\n\nDo not restyle, redesign, simplify or \"improve\" the garment. Do not\n" +
		"thicken or move the straps. Do not change the neckline shape — a U is not\n" +
		"a V and a square is not a scoop. Do not move the strap spacing inward or\n" +
		"outward. Do not add, remove or relocate a single seam. Do not add texture\n" +
		"the reference does not have, and do not smooth away texture it does have.\n" +
		"Do not change where the hem ends. If any detail is unclear in the\n" +
		"reference, copy the closest thing you can see rather than inventing.\n\n")
	w(labelBlock)
	w(wmBlock)
	w("=====================================================================\n" +
		"GLOBAL_NEGATIVE (append to every image and every video)\n" +
		"=====================================================================\n")
	w(negCam)
	w(" no new objects appearing, no props materialising, no\n" +
		"furniture moving, no plant moving, no background shift, no layout change,\n" +
		"no colour temperature shift, no warming light, no warm grade, no amber\n" +
		"cast, no golden hour, no orange tint, no sepia, no tungsten light, no\n" +
		"yellow cast on skin, no yellow cast on white surfaces, no sandy walls, no\n" +
		"olive skin tone, no golden skin, no bronzed skin, no terracotta blush, no\n" +
		"peach blush, no coral lips, no warm shadows on the face, no hair falling forward\n" +
		"over the shoulders, no hair covering the straps, no exposed waistline, no\n" +
		"altered neckline shape, no V-neck substituted for a scoop, no straps\n" +
		"moved closer to the neck, no straps moved onto the shoulder edge,\n")
	w(hwNeg)
	w(" ")
	w(strapNeg)
	w(" ")
	w(detailNeg)
	w(" ")
	w(twinNeg)
	w(" no visible seams showing\n" +
		"through the front panel, no\n" +
		"outline showing under the fabric, no missing seam, no invented seam, no\n" +
		"changed hem position, no\n" +
		"changed knit gauge, no coarsened ribbing, no chunky rib, no widened rib\n" +
		"spacing, no flattened texture, no invented texture, no colour drift, no\n" +
		"beige cast on the garment, no warmed garment colour, ")
	w(whiteNeg)
	w(" no gap between the\n" +
		"hem and the waistband, no visible skin between the two garments,\n")
	w(shapeNeg)
	w(" no tanned skin, no olive\n" +
		"skin tone, no bronzed skin, no dull uneven skin tone, no yellow cast, no\n" +
		"heavy contouring, no bronzer, no flat blown-out face, no body rotation\n" +
		"beyond 30 degrees, no full turn, no back view, no spinning, no runway\n" +
		"walk, no posing like a model, no large gestures, ")
	w(FNEG)
	w(" no jacket, no\n" +
		"cardigan, no layer covering the top, no restyled garment, no thickened\n" +
		"straps, no raised neckline, no added seams, no shiny satin, no\n" +
		"transparent unlined panels, no garment morphing, no face change, no different person, no\n" +
		"lookalike, no hair tied up, no ponytail, no bun, no short hair, no\n" +
		"outfit change, no lower garment change, no changed hemline, no patterned\n" +
		"lower garment, no bright clashing lower garment, no changed shoes, no\n" +
		"added accessories, no extra jewellery, no oversized jewellery, no\n" +
		"accessory change, no touching the bag, no holding the bag strap, no hand\n" +
		"on the bag, no adjusting the bag, no re-slinging the bag, no swinging\n" +
		"bag, no floating bag, no bag hanging in mid-air, no unsupported bag, no\n" +
		"bag merging into furniture, no broken strap, no strap disappearing, no\n" +
		"bag in front of the top, no hand covering the top, no arm across\n" +
		"the chest area, no makeup change, no\n" +
		"lighting change, ")
	w(wmNeg)
	w(" no price tags, no cuts, no slow motion, no cinematic\n" +
		"colour grading, no studio lighting, no ring light, no flash, no golden\n" +
		"hour, no plastic shiny skin, no airbrushed poreless face, no doll-like\n" +
		"face, no heavy beauty filter, no distorted hands, no extra fingers, no\n" +
		"two phones, ")
	w(secondPersonNeg)
	w("no low camera angle, no\n" +
		"exaggerated proportions, no frozen subject, no mannequin stillness, no\n" +
		"snapping movement, no teleporting limbs, no stiff robotic motion.\n\n" +
		"=====================================================================\n" +
		"IMAGE_PROMPT_1 — Scene 1 opening frame\n" +
		"=====================================================================\n" +
		"[Reference = clothing only, and it is the authority for the garment.\n" +
		"Copy its neckline shape, strap width and spacing, armhole cut, fabric\n" +
		"texture, hem position and every seam exactly. Ignore its background,\n" +
		"lighting, colour grading, text and any people in it.]\n\n" +
		"[IDENTITY_BLOCK]")
	w(idbImg)
	w("\n[STYLE_BLOCK]")
	w(labelImg1)
	w("\n\n")
	w(POSE1)
	w("\n\n[GLOBAL_NEGATIVE]\n\n" +
		"=====================================================================\n" +
		"IMAGE_PROMPT_2 — Scene 2 opening frame\n" +
		"=====================================================================\n" +
		"[Reference 1 = the Step 1 image. Authoritative for her face, features,\n" +
		"makeup, hair, skin tone, jewellery, the lower garment, the location, the\n" +
		"object placement and the lighting. She must be the exact same person —\n" +
		"not a lookalike. Where this image and the text differ, this image wins.]\n\n" +
		"[Reference 2 = clothing reference, and it stays the authority for the\n" +
		"garment itself. The Step 1 image governs her face, hair, styling and the\n" +
		"location; this reference governs the top's construction. Copy the\n" +
		"neckline shape, strap spacing, hem position and seams from it exactly.]\n\n" +
		"CONTINUITY: The same person in the same place, moments later in the same\n" +
		"take. Only her body angle and hand position change from the Step 1 image.\n" +
		"The framing, the subject size in frame, the object placement and the\n" +
		"lighting stay identical.\n\n" +
		"[IDENTITY_BLOCK]")
	w(idbImg)
	w("\n[STYLE_BLOCK]")
	w(labelImg2)
	w("\n\n")
	w(POSE2)
	w("\n\n[GLOBAL_NEGATIVE]\n\n" +
		"=====================================================================\n" +
		"VIDEO_PROMPT_1 — Scene 1 — 8s, 9:16\n" +
		"=====================================================================\n" +
		"Animate this image into an 8-second vertical 9:16 ")
	w(F.Clip)
	w(".\nContinuous single take, no cuts.\n\n" +
		"CAMERA LOCK: ")
	w(camLock1)
	w("\n\nSCENE LOCK: The background is completely static. Every object named in\n" +
		"the SETTING block stays exactly where it is, and no new object enters the\n" +
		"frame at any point.\n\n" +
		"SUBJECT LOCK: Keep everything from the source frame identical at every\n" +
		"moment — her face, the glass-skin glow, the long loose hair behind both\n" +
		"shoulders, the top's neckline shape and depth, strap width and how far\n" +
		"apart the straps sit, ")
	w(strapLock)
	w(", every seam line, the knit gauge and\n" +
		"fabric texture, the fit, and above all where the hem ends. The hem stays\n" +
		"level with the waistband for all 8 seconds and never rides up — no skin\n" +
		"appears between the top and the lower garment at any moment. The lower\n" +
		"garment, the colour of the top and the lighting colour and intensity stay\n" +
		"identical too. Nothing may morph, and the garment must never redesign\n" +
		"itself between frames.\n")
	w(LOCK1)
	w("\n\nMOTION QUALITY: ")
	w(motion1)
	w("\n\nTIMING:\n")
	w(strings.Join(TIM1, "\n"))
	w("\n\n")
	w(dlg1)
	w("\n\n[GLOBAL_NEGATIVE]\n\n\n" +
		"=====================================================================\n" +
		"VIDEO_PROMPT_2 — Scene 2 — 8s, 9:16\n" +
		"=====================================================================\n" +
		"Animate this image into an 8-second vertical 9:16 ")
	w(F.Clip)
	w(".\nContinuous single take, no cuts. A direct continuation of Scene 1 — same\n" +
		"woman, same face, same glow, same hair, ")
	w(F.Cont)
	w(".\n\nCAMERA LOCK: ")
	w(camLock2)
	w("\n\nSCENE LOCK: The background is completely static. Every object named in\n" +
		"the SETTING block stays exactly where it is, and no new object enters the\n" +
		"frame at any point.\n\n" +
		"SUBJECT LOCK: Keep everything from the source frame identical at every\n" +
		"moment — her face, the top's neckline shape and depth, strap width and\n" +
		"how far apart the straps sit, ")
	w(strapLock)
	w(", every seam line, the knit\n" +
		"gauge and fabric texture, the fit, and above all where the hem ends. The\n" +
		"hem stays level with the waistband for all 8 seconds and never rides up —\n" +
		"no skin appears between the top and the lower garment at any moment. The\n" +
		"lower garment, the colour of the top and the lighting colour and\n" +
		"intensity stay identical too. Nothing may morph, and the garment must\n" +
		"never redesign itself between frames. ")
	w(LOCK2)
	w("\n\nMOTION QUALITY: ")
	w(motion2)
	w("\n\nTIMING:\n")
	w(strings.Join(TIM2, "\n"))
	w("\n\n")
	w(dlg2)
	w("\n\n[GLOBAL_NEGATIVE]")

	return b.String()
}
