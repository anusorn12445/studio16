package prompt

import (
	"fmt"
	"strconv"
	"strings"

	"studio16/internal/model"
)

// physicsGuard is appended to EVERY video prompt (all formats) to stop the model
// from producing jumbled poses, a flickering/morphing environment, clipping, or
// physically impossible motion.
const physicsGuard = "PHYSICS: every pose and motion stays anatomically correct and physically possible — no contorted or impossible poses, no broken/backward joints, no extra or missing limbs, hands with five fingers. The background stays solid and stable — it never flickers, morphs or changes. Nothing clips or passes through anything else; feet stay planted on the ground — no floating, sliding or teleporting."

// physicsGuardImage is the still-image version of the same guard.
const physicsGuardImage = "PHYSICS: a natural, anatomically-correct, physically-possible pose — no contorted posture, no broken joints, no extra/missing limbs, five normal fingers; a solid coherent environment with correct perspective; nothing clips through anything and feet rest naturally on the ground with proper contact."

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
	if isSportFormat(p.Format) {
		timing = sportSceneAt(p.Format, o.Scene).action // each shot = a different athletic scene
		// Sport formats style the top with proper activewear, never jeans/casual trousers.
		bottom = "high-waisted black athletic leggings"
		shoes = " supportive running shoes on her feet."
		if p.Format == "hyrox" {
			shoes = " supportive training shoes on her feet."
		}
	}

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
	settingText := F.Setting
	camLine := "CAMERA: phone locked on a tripod, fixed 9:16 framing, eye/chest level. No drift, no dolly, no zoom; she stays the same size in frame; the background stays static.\n\n"
	if isSportFormat(p.Format) {
		s := sportSceneAt(p.Format, o.Scene)
		settingText = s.setting
		camLine = "CAMERA: " + s.camera + ", steady and clean; she stays a consistent size in frame; the background stays static.\n\n"
	}
	fmt.Fprintf(&b, "SETTING: %s\n\n", trimRunes(settingText, 200))
	b.WriteString(camLine)
	fmt.Fprintf(&b, "ACTION: %s\n\n", trimRunes(timing, 300))
	b.WriteString(consist)
	b.WriteString(physicsGuard + "\n\n")
	b.WriteString("Avoid: black bars, letterboxing, borders or padding, any English speech, camera drift or zoom, changing the garment, exposed waist, extra people, on-screen text or logos, distorted hands, robotic motion.")

	return trimRunes(b.String(), 2100)
}

// BuildVeoImage produces a COMPACT image prompt for the opening frame: the
// model wearing the garment in the scene (Pose 1). This image is generated
// first, then handed to Veo as the video's first frame.
func BuildVeoImage(p model.Product, o VeoOpts) string {
	F := fmtFor(p.Format)
	base := ResolveBase(p)

	C := Sanitize(EnOnly(p.HeroColor, "cream white"))
	garment := Sanitize(EnOnly(p.Garment, "a fitted knit top"))
	bottom := Sanitize(EnOnly(p.Bottom, "high-waisted trousers"))
	shoes := ""
	if F.Feet {
		shoes = " " + Sanitize(EnOnly(p.Shoes, "plain white sneakers")) + " on her feet."
	}
	if isSportFormat(p.Format) {
		bottom = "high-waisted black athletic leggings"
		shoes = " supportive running shoes on her feet."
		if p.Format == "hyrox" {
			shoes = " supportive training shoes on her feet."
		}
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

	pose := F.Pose1
	settingText := F.Setting
	opening := fmt.Sprintf("A full-frame vertical 9:16 portrait photo that fills the entire frame edge to edge — no black bars, no letterboxing, no borders, no padding. %s %s, with %s.%s", idBrief, wearLine, bottom, shoes)
	garmentNote := " Use the attached photo for the GARMENT only, not for any person in it — the woman is the one described above."
	if isSportFormat(p.Format) {
		s := sportSceneAt(p.Format, o.Scene)
		settingText = s.setting
		pose = "POSE — she is " + s.pose + ", athletic and focused, in genuine effort. CAMERA: " + s.camera + "."
		// Lead with the scene + shot number so the model treats each scene as a
		// distinct photograph instead of collapsing to one pose.
		series, subject := "HYROX", "mid-"+s.name+", clearly using the "+s.name+" equipment — this exercise and its equipment are the main subject"
		if p.Format == "run" {
			series, subject = "RUNNING", "mid-"+s.name+" — this running scene and her running form are the main subject"
		}
		opening = "A full-frame vertical 9:16 ACTION photo — this is SHOT #" + strconv.Itoa(o.Scene+1) + " of a " + series + " review series, and it MUST look completely different from the other shots: a different setting, a different scene and a different camera angle. It shows a Thai woman " + subject + " and must fill the frame. " + idBrief + " " + wearLine + ", with " + bottom + "." + shoes
		garmentNote = " The attached photo shows ONLY the shirt's design, print and colour — copy the shirt's graphic exactly, but do NOT copy its pose, background, cropping or composition; dress the athlete in this new action scene."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", opening)
	fmt.Fprintf(&b, "%s\n\n", trimRunes(styleBrief, 200))
	fmt.Fprintf(&b, "SETTING: %s\n\n", trimRunes(settingText, 300))
	fmt.Fprintf(&b, "%s\n\n", trimRunes(pose, 360))
	b.WriteString(physicsGuardImage + " ")
	b.WriteString(authLine + garmentNote + " Full-bleed 9:16 portrait, no black bars or borders, no on-screen caption or watermark overlay (but keep the shirt's own printed graphic/logo exactly).")

	return trimRunes(b.String(), 2000)
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
	Scene int    // 0-based scene index — for hyrox, picks a different exercise per shot
}

// hyroxScene is one HYROX exercise with its own pose, action, setting and camera
// so every shot is a visibly different scene, not the same frame re-posed.
type hyroxScene struct{ name, pose, action, setting, camera string }

var hyroxScenes = []hyroxScene{
	{
		name:    "wall balls",
		pose:    "holding a weighted wall ball at her chest in a quarter-squat, ready to throw it to the target overhead",
		action:  "She does wall-ball reps: from a deep squat she drives up and throws the ball to the wall target overhead, catches it and squats again, talking to camera between reps. The top never rides up.",
		setting: "At the WALL-BALL station of a HYROX arena: a tall round target mark high on a matte wall in front of her, a stack of wall balls beside her, black turf underfoot, other lanes blurred behind.",
		camera:  "front on, slightly low angle, framed full length from head to shoes",
	},
	{
		name:    "sled push",
		pose:    "leaning low into a loaded HYROX sled with both arms extended on the posts, ready to drive it forward",
		action:  "She drives the loaded HYROX sled forward across the turf, pushing hard and low through her legs, then straightens and talks to camera. The activewear stays put under load.",
		setting: "On the SLED-PUSH lane: a long black artificial-turf strip with white boundary lines and a loaded HYROX sled right in front of her, the arena stretching out behind.",
		camera:  "a three-quarter side angle from the front-left, low to the ground, full length",
	},
	{
		name:    "battle ropes",
		pose:    "in a low athletic half-squat gripping the ends of two heavy battle ropes anchored to a rig",
		action:  "She slams two heavy battle ropes in fast alternating waves from a low athletic stance, then eases off and talks to camera. The top stays put and opaque.",
		setting: "At the RIG: two heavy battle ropes anchored to a black steel rig, rubber flooring, a rack of kettlebells behind, spectators blurred far back.",
		camera:  "straight-on side profile, mid-low angle, full length",
	},
	{
		name:    "farmers carry",
		pose:    "standing tall holding a heavy kettlebell in each hand at her sides, ready to walk",
		action:  "She takes controlled steps carrying a heavy kettlebell in each hand (farmers carry), shoulders braced and tall, then sets them down and talks to camera. The leggings stay put.",
		setting: "On the CARRY lane: an open marked turf lane with a rack of heavy kettlebells at the start line and lane lines running away into the arena.",
		camera:  "a front three-quarter angle as she walks toward the camera, full length",
	},
	{
		name:    "sandbag lunges",
		pose:    "a heavy sandbag hugged across the front of her shoulders, ready to lunge",
		action:  "She performs walking lunges with a sandbag across her shoulders, dropping into deep lunges, then stands and talks to camera. The fabric stretches but stays opaque and in place.",
		setting: "On the LUNGE lane: a turf strip with a row of sandbags on a low rack beside her and the HYROX rig in the background.",
		camera:  "side profile at eye level, full length",
	},
	{
		name:    "burpees",
		pose:    "standing tall at the top of a burpee, arms coming down after the jump, breathing",
		action:  "She does a full burpee — squat, hands to the floor, jump back to a plank, jump in and stand with a hop — then talks to camera, breathing. The top stays down.",
		setting: "On a clear turf square in the middle of the HYROX arena, other stations and blurred athletes visible far behind.",
		camera:  "front on, slightly high angle, full length",
	},
}

func hyroxSceneAt(i int) hyroxScene {
	n := len(hyroxScenes)
	if n == 0 {
		return hyroxScene{}
	}
	return hyroxScenes[((i%n)+n)%n]
}

// runScenes are the running-review shots — each a different running scene so every
// shot looks distinct (jog, sprint, hill, treadmill, trail, cool-down).
var runScenes = []hyroxScene{
	{
		name:    "warm-up jog",
		pose:    "jogging at an easy pace, arms relaxed and swinging, one foot mid-stride, warming up",
		action:  "She jogs at an easy warm-up pace toward the camera, relaxed and smiling, talking between breaths. The top stays put and no skin shows at the waist.",
		setting: "On an outdoor rubber running track at a stadium: red-brown track with white lane lines curving away, low empty stands and floodlight towers softly blurred behind, soft morning light.",
		camera:  "a front three-quarter angle as she jogs toward the camera, framed full length",
	},
	{
		name:    "sprint",
		pose:    "in a powerful full sprint, driving one knee high and pumping her arms, leaning forward with speed",
		action:  "She sprints hard down the straight, driving her knees and pumping her arms, then eases off and talks to camera, breathing. The activewear holds under speed.",
		setting: "On the straight of an outdoor running track, crisp white lane lines running straight away into the distance, blurred stadium behind.",
		camera:  "a side tracking angle level with her as she sprints, full length",
	},
	{
		name:    "uphill run",
		pose:    "running up a steep grassy slope, leaning into the incline, driving hard with her legs",
		action:  "She runs up a steep hill, leaning into the climb and driving with her legs, then reaches a flatter spot and talks to camera. The leggings stay put and squat-proof.",
		setting: "On a steep grassy hillside path in a green park, the slope rising ahead, trees and open sky behind.",
		camera:  "a low front angle looking up the hill, full length",
	},
	{
		name:    "treadmill run",
		pose:    "running on a gym treadmill at a steady pace, relaxed upright form",
		action:  "She runs on a treadmill at a steady pace with relaxed upright form, glancing to camera and talking. The top stays down through every step.",
		setting: "On a treadmill in a bright modern gym, rows of cardio machines and mirrored walls softly blurred behind, clean even lighting.",
		camera:  "front on at chest height, full length",
	},
	{
		name:    "trail run",
		pose:    "running along an outdoor dirt trail, mid-stride over the path",
		action:  "She runs along a forest trail, mid-stride over the dirt path, then slows and talks to camera. The activewear moves with her and stays opaque.",
		setting: "On a dirt trail path winding through a green park/forest, dappled daylight, trees lining both sides of the path.",
		camera:  "a side three-quarter tracking angle along the trail, full length",
	},
	{
		name:    "cool-down",
		pose:    "slowing from a run to a walk, hands on her hips, catching her breath",
		action:  "She slows from a run to a walk, hands on her hips, catching her breath and smiling at camera for a warm sincere close. The top stays in place.",
		setting: "Back on the running track at the finish, lane lines underfoot, the stadium softly blurred behind, warm late-afternoon light.",
		camera:  "front on, slightly low angle, full length",
	},
}

func runSceneAt(i int) hyroxScene {
	n := len(runScenes)
	if n == 0 {
		return hyroxScene{}
	}
	return runScenes[((i%n)+n)%n]
}

// isSportFormat reports whether a format uses per-shot athletic scenes.
func isSportFormat(format string) bool { return format == "hyrox" || format == "run" }

// sportSceneAt returns the per-shot scene for hyrox or run formats.
func sportSceneAt(format string, i int) hyroxScene {
	if format == "run" {
		return runSceneAt(i)
	}
	return hyroxSceneAt(i)
}

// SceneLabel is a short human label for scene i (used in the per-scene prompt UI).
func SceneLabel(format string, i int) string {
	if isSportFormat(format) {
		return sportSceneAt(format, i).name
	}
	return ""
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
