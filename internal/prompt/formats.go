package prompt

// Format is one review-video format, ported verbatim from the STUDIO 16
// FORMATS array. Every long English text block is copied character for
// character — these strings are the whole value of the app.
type Format struct {
	ID   string
	Th   string
	Tag  string
	Desc string

	Voice bool

	Props   string
	Setting string
	Light   string
	Camera  string
	Neg     string
	Clip    string
	Cont    string

	Pose1 string
	Pose2 string
	Lock1 string
	Lock2 string

	T1 []string
	T2 []string

	Amb     string
	Motion1 string
	Motion2 string

	TwoModels bool
	Label     bool
	Feet      bool

	NegCam   string
	CamLock1 string
	CamLock2 string
}

// Preset is a small th/en pairing used by the cafe, voice and bottom pickers.
type Preset struct {
	Th string
	En string
}

// FORMATS mirrors the HTML FORMATS array (mirror, cafe, pair).
var FORMATS = []Format{
	{
		ID:   "mirror",
		Th:   "ยืนหน้ากระจก",
		Tag:  "มีบทพูด",
		Desc: "ถือมือถือคุยกับคนดูตรงๆ ใช้ได้กับสินค้าทุกแบบ แปลงเป็นยอดขายดีที่สุด",

		Voice: true,

		Props:   `Nothing in her hands except the phone.`,
		Setting: `A bright modern Thai condo living room seen in a full-length mirror. Warm beige walls, light wood floor, grey linen sofa with cream cushions, one small potted plant, sheer white curtains on camera-left. A slight mirror edge is visible at the frame border. The layout is fixed: furniture and plant never move and no additional objects exist in the room.`,
		Light:   `Gentle diffused late-morning daylight around 10am, soft and bright, coming through the sheer curtain from camera-left. Gentle falloff, no harsh shadows. The colour temperature and intensity stay constant from the first frame to the last.`,
		Camera:  `Vertical 9:16, shot on a phone through a mirror, eye-level, held at a fixed distance on a locked framing. The phone is propped, not hand-carried: only micro-jitter of one or two pixels, no drift, no dolly, no push-in, no pull-back. Her body occupies the same proportion of the frame in the last second as in the first. Framed from the top of her head down to mid-thigh.`,
		Neg:     `no shoes, no sneakers, no feet in frame, no full-length shot to the floor, no walking,`,
		Clip:    `mirror-selfie video`,
		Cont:    `same mirror, same outfit, same light`,

		Pose1: `POSE — mirror selfie, front on: She stands facing a large full-length mirror, holding a black smartphone up beside her cheek with her right hand, the phone partially covering her jaw. Her left hand rests relaxed at her side. Weight shifted onto one hip. Posture upright and relaxed with shoulders back, standing straight, not leaning toward the mirror. Calm friendly expression, mouth slightly open mid-sentence as if chatting with a close friend, looking into the mirror.`,
		Pose2: `POSE — mirror selfie, angled: She stands turned about 25 degrees to one side in front of the mirror, still holding the black smartphone up beside her cheek with her right hand. Her left hand rests lightly on her waist, showing how the fabric falls along her side. Posture upright and relaxed with shoulders back, standing straight. She is looking down at the front of her top rather than at the lens, as if checking how it sits — a candid unposed moment, not a model pose.`,
		Lock1: `The phone stays in her right hand beside her cheek throughout. She stays standing in place. Body rotation must never exceed 30 degrees.`,
		Lock2: `The phone stays in her right hand beside her cheek. Body rotation must never exceed 30 degrees; she never shows her back.`,

		T1: []string{
			`0.0s-2.5s — Stands still facing the mirror, talking naturally with warm eye contact and gentle head movement. Only her face and shoulders move.`,
			`2.5s-4.5s — With her free left hand she tucks a strand of hair behind her ear and back over her shoulder, then lets the arm fall to her side.`,
			`4.5s-6.5s — Her free left hand lightly touches the thin shoulder strap at the collarbone, gives it a small tug and releases it. She glances down at the strap, then back to the mirror.`,
			`6.5s-8.0s — Lowers her hand, gives a relaxed confident nod, smiles softly into the mirror. The room behind her is completely unchanged.`,
		},
		T2: []string{
			`0.0s-3.0s — Holds the angled position talking to the mirror, then runs her free left hand down the side of her waist, glancing at how the fabric falls.`,
			`3.0s-5.0s — Rotates back to face the mirror front on and settles her posture. Hair stays behind both shoulders.`,
			`5.0s-6.5s — Looks down at the front of her top for a beat, then back up with a small honest smile. A candid unposed moment.`,
			`6.5s-8.0s — Raises her free left index finger and points clearly toward the bottom-left corner of the frame, holding the gesture. The room behind her is completely unchanged.`,
		},
	},
	{
		ID:   "cafe",
		Th:   "คาเฟ่ / กลางแจ้ง",
		Tag:  "ไม่มีบทพูด",
		Desc: "โพสถ่ายรูปสวยๆ ไม่พูด ใส่เพลงทับใน CapCut ยอดวิวดี ปากไม่ต้องตรงเสียง",

		Voice: false,

		Props:   `Her handbag is set down and resting on its own base on the seat of the empty chair beside her, standing upright and fully supported. It never hangs in the air, never rests on her arm or shoulder, and she never touches, lifts or adjusts it.`,
		Setting: `The outdoor terrace of a quiet Bangkok cafe in the late morning. A pale wooden table holding {CAFE}, a rattan chair, a whitewashed brick wall behind, leafy green plants along the edge of frame, pale grey stone floor tiles. Everything on the table is scenery only: it sits exactly where it is placed, nobody touches it, nothing is lifted, moved, opened, eaten or drunk, and no additional objects appear.`,
		Light:   `Bright neutral late-morning daylight in open shade, around 5500K, soft and even across her whole body. Cool open sky is the only light source, so the whole scene reads clean and neutral — the whitewashed wall stays white rather than sandy, and no warm bounce tints her skin. Dappled leaf shadow falls softly on the wall behind her, never across her face. No harsh sun, no blown highlights. The colour temperature and intensity stay constant from the first frame to the last.`,
		Camera:  `Vertical 9:16, shot on a phone held by a friend standing still, eye-level, locked framing at a fixed distance. Only gentle handheld micro-movement of a few pixels, no drift, no dolly, no push-in, no pull-back. She occupies the same proportion of the frame in the last second as in the first. Framed from the top of her head down to mid-thigh.`,
		Neg:     `no lip movement, no mouth opening, no visible speaking, no eating, no drinking, no sipping, no straw in the mouth, no touching the cup, no lifting the cup, no holding the cup, no hand on the plate, no picking up food, no cutlery in hand, no cup moving, no spilling, no traffic, no crowd, no passers-by,`,
		Clip:    `cafe portrait video`,
		Cont:    `same terrace, same outfit, same light`,

		Amb: `quiet outdoor cafe ambience — a light breeze through leaves, soft birdsong, the occasional faint clink of ceramics, a distant kitchen hum`,

		Pose1: `POSE — cafe portrait, seated: She sits at the pale wooden table in the rattan chair, angled slightly toward the camera, one forearm resting on the bare tabletop well clear of anything on it, hand open and relaxed. Her other hand rests in her lap. She is not holding or reaching for anything. Posture upright and relaxed with shoulders back, sitting straight. Calm soft expression with a small closed-lip smile, eyes toward the camera. Her lips stay closed, she is not speaking.`,
		Pose2: `POSE — cafe portrait, standing: She stands beside the table in front of the whitewashed brick wall, turned about 25 degrees to one side, one hand resting lightly on the back of the rattan chair. Posture upright and relaxed with shoulders back, standing straight. Her eyes are turned away from the lens, looking off to the side past the camera with a soft closed-lip expression — a candid unposed moment, not a model pose. Her lips stay closed, she is not speaking.`,
		Lock1: `She stays seated in place. Her lips stay closed for the entire clip and she never speaks. Body rotation must never exceed 30 degrees.`,
		Lock2: `She stays standing in place. Her lips stay closed for the entire clip and she never speaks. Body rotation must never exceed 30 degrees; she never shows her back.`,

		T1: []string{
			`0.0s-2.5s — Sits still looking softly toward the camera, blinking naturally, a light breeze moving a few strands of her hair. Lips closed.`,
			`2.5s-4.5s — Lifts her free hand from the tabletop and smooths it once down the front of her top from the collarbone toward the waist, then rests it back on the bare table. She never touches anything on the table.`,
			`4.5s-6.5s — Tucks a strand of hair behind her ear and back over her shoulder, then rests her hand on the table again.`,
			`6.5s-8.0s — Tilts her head slightly and gives a soft closed-lip smile toward the camera. The terrace behind her is completely unchanged.`,
		},
		T2: []string{
			`0.0s-2.0s — She shifts her weight from one foot to the other and settles, shoulders easing down. Her hair lifts and resettles in the breeze and she blinks naturally. Lips closed.`,
			`2.0s-4.0s — She lifts one hand and sweeps a strand of hair back behind her shoulder, turning her head slowly to follow something off to the side, then lets the hand fall and glances down at the front of her top.`,
			`4.0s-6.0s — She rotates gently back toward the camera over about a second, chin turning a beat after her shoulders so the motion reads as one continuous movement rather than a snap, and settles with a soft closed-lip smile.`,
			`6.0s-8.0s — She raises her free left index finger and points toward the bottom-left corner of the frame, holding it while her hair still settles. The terrace behind her is completely unchanged.`,
		},

		Motion2: `Movement is continuous and unhurried from the first frame to the last — she is never frozen. Weight shifts through the hips, the head turns a beat after the shoulders, hair carries momentum and settles naturally, and she blinks and breathes throughout. Every movement eases in and eases out; nothing snaps, teleports or holds rigid.`,
	},
	{
		ID:   "pair",
		Th:   "จับคู่สี สองนางแบบ",
		Tag:  "มีบทพูด · โชว์การแมทช์",
		Desc: "นางแบบสองคนใส่สินค้าตัวเดียวกันคนละสี ยืนคู่กันโชว์ว่าสีไหนแมทช์กับอะไร มีตัวหนังสือบอกคู่สี",

		Voice:     true,
		TwoModels: true,
		Label:     true,
		Feet:      true,

		Props:   `Each woman holds one small structured handbag at her side by its top handle, hanging still and fully supported. Neither ever touches, lifts or adjusts the other pieces.`,
		Setting: `A bright showroom corner. A tall cream wall with fine rectangular moulding panels fills the whole background edge to edge, and a pale polished marble floor with soft reflections runs to the bottom of the frame. There is nothing else in the room: no furniture, no plants, no rails, no doorways, no windows. The wall and floor are fixed and identical in every scene — only the clothes change, never the room.`,
		Light:   `Bright neutral daylight around 5500K, broad and soft from the front, flat and even across both women with no hard shadow on the wall behind them. The cream wall reads clean and neutral, never sandy or golden. The colour temperature and intensity stay constant from the first frame to the last and identical between the two scenes.`,
		Camera:  `Vertical 9:16, shot on a phone locked on a tripod at chest height, straight on and square to the wall, fixed framing at a fixed distance. Only micro-jitter of one or two pixels. Both women are framed full length from just above their heads down to their shoes, centred together with even space either side. Their size in frame never changes.`,
		Neg:     `no third person, no extra people, no models swapping sides, no one leaving frame, no one entering frame, no identical twins, no swapped faces, no walking out of frame, no bare feet,`,
		Clip:    `showroom pairing video`,
		Cont:    `same wall, same floor, same positions, same light`,

		Pose1: `POSE — colour pairing, front on: Both women stand side by side facing the camera, feet together, weight even, shoulders back, chins level, a small gap between them. Model A on the left rests one hand lightly at her side and holds her bag in the other. Model B on the right stands with one hand relaxed at her side, the other holding her bag. Both look into the lens with a warm natural smile, lips parted a little as if mid-sentence. Everyday relaxed posture, not a model pose.`,
		Pose2: `POSE — colour pairing, relaxed: Both women stand side by side facing the camera in the same positions. Model A has one hand resting at her waist above the belt, the other holding her bag. Model B stands with both hands relaxed, one holding her bag. Both look into the lens, Model A mid-sentence, Model B with a warm natural smile. Everyday relaxed posture.`,
		Lock1: `Model A stays on the left and Model B stays on the right; they never swap, never cross, never step forward or back. Both stay standing in place. Body rotation must never exceed 30 degrees.`,
		Lock2: `Model A stays on the left and Model B stays on the right; they never swap, never cross, never step forward or back. Body rotation must never exceed 30 degrees; neither shows her back.`,

		Motion1: `Movement is continuous and unhurried from the first frame to the last — neither woman is ever frozen. When one is speaking, the other stays alive in the shot: she blinks, breathes, shifts her weight and glances toward whoever is talking. Weight shifts through the hips, heads turn a beat after shoulders, hair carries momentum and settles. Every movement eases in and eases out.`,
		Motion2: `Movement is continuous and unhurried from the first frame to the last — neither woman is ever frozen. The one not speaking still blinks, breathes and shifts her weight rather than holding a pose. Heads turn a beat after shoulders, hair carries momentum and settles, and every movement eases in and eases out.`,

		T1: []string{
			`0.0s-3.0s — Model A on the left talks to the camera with a warm easy smile and a small open-palm gesture toward her own outfit. Model B listens, glancing across at her, nodding once.`,
			`3.0s-5.5s — Model A lowers her hand. Model B takes over, talking to the camera and lightly touching the belt at her waist to draw the eye down the outfit.`,
			`5.5s-7.0s — Both settle, look at each other for half a beat and share a small unforced laugh, then turn back to the lens.`,
			`7.0s-8.0s — Both stand relaxed with warm natural smiles into the camera. The wall behind them is completely unchanged.`,
		},
		T2: []string{
			`0.0s-3.0s — Model A talks to the camera, running one hand down the front of her outfit to show how the two colours sit together.`,
			`3.0s-5.5s — Model B answers, turning about 20 degrees to show her side then settling back square to the camera.`,
			`5.5s-7.0s — Both smile at the lens, Model A giving a small approving nod.`,
			`7.0s-8.0s — Model B raises her index finger and points clearly toward the bottom-left corner of the frame, holding the gesture. Model A keeps smiling beside her. The wall behind them is completely unchanged.`,
		},
	},
	{
		ID:   "hyrox",
		Th:   "กีฬา HYROX",
		Tag:  "มีบทพูด · วล็อกออกกำลัง HYROX",
		Desc: "วล็อกรีวิวชุดกีฬาแบบออกกำลังจริง ทำท่า HYROX จริง (วอลบอล/เบอร์ปี/สเลด) โชว์ว่าชุดเวิร์กตอนเล่นจริง",

		Voice: true,
		Feet:  true,

		Props:   `Nothing in her hands.`,
		Setting: `A bright indoor functional-fitness arena in the style of a HYROX race: black rubber flooring beside a lane of artificial turf, a weighted sled resting on the turf, a neat stack of wall balls, a rowing machine and a row of sandbags along the back wall, matte black rig frames. High industrial ceiling with even white lighting. The layout is fixed: the equipment never moves and no additional objects appear.`,
		Light:   `Bright even indoor sports-hall lighting around 5000K, clean and neutral, flat and soft across the whole body with no hard shadows. No coloured stage lights, no haze, no spotlights. The colour temperature and intensity stay constant from the first frame to the last.`,
		Camera:  `Vertical 9:16, shot on a phone locked on a tripod at chest height, straight on and square, fixed framing at a fixed distance. Only micro-jitter of one or two pixels, no drift, no dolly, no push-in, no pull-back. She occupies the same proportion of the frame in the last second as in the first. Framed full length from just above the top of her head down to her shoes.`,
		Neg:     `no other athletes, no crowd, no coach, no referee, no spectators, no barbell, no dropped weights, no chalk cloud, no coloured stage lighting, no haze, no smoke, no motion blur on the body,`,
		Clip:    `sportswear review video`,
		Cont:    `same arena, same outfit, same light`,

		Pose1: `POSE — mid wall-ball, front on: She stands in the lane holding a weighted wall ball at her chest with both hands in a quarter-squat, elbows tucked in, ready to throw, wearing the activewear. Athletic and focused, weight through the heels, mid-workout — showing how the top and leggings sit and hold under real load.`,
		Pose2: `POSE — top of a burpee, front on: She is standing tall at the top of a burpee, arms just coming down after the jump, chest lifted, breathing, wearing the activewear. Genuine effort, showing the top stayed down and in place through the movement, not a posed model shot.`,
		Lock1: `She trains in the same spot in front of the equipment. She may squat, hinge, lunge, jump and reach for each rep, but she always returns to the same upright framing and never walks or drifts out of frame. Body rotation stays within about 30 degrees.`,
		Lock2: `She trains in the same spot. She may squat, hinge and jump for each rep but returns to the same framing and never leaves the frame; body rotation stays within about 30 degrees and she never shows her back.`,

		Motion1: `Real athletic movement with genuine effort — she breathes, muscles work, and hair and fabric move with each rep — but every rep is controlled and returns to the same framing. Only she moves; the locked camera and the background never move. Nothing snaps, teleports or holds rigid.`,
		Motion2: `Real athletic movement with genuine effort — controlled reps, visible breathing, hair and fabric carrying momentum and settling — always returning to the same framing. Only she moves; the camera and background stay locked. Nothing snaps or freezes.`,

		Amb: `indoor sports-hall ambience — a faint echo, the soft thud of a wall ball, her own controlled breathing, distant footfalls, no music and no crowd noise`,

		T1: []string{
			`0.0s-3.0s — Performs two clean wall-ball reps: from a deep squat she drives up and throws the ball to a target overhead, catches it back at her chest and squats again. The top never rides up and no skin shows at the waist.`,
			`3.0s-5.0s — Holds the ball at her chest, stands, slightly out of breath, and talks to the camera about how the outfit moved with her through the reps.`,
			`5.0s-6.5s — Runs one hand down the side seam of the leggings to show they stayed put and squat-proof, still holding the ball in the other hand.`,
			`6.5s-8.0s — Stands tall, wipes her brow with the back of her wrist and gives a confident nod. The equipment behind her is unchanged.`,
		},
		T2: []string{
			`0.0s-3.0s — Performs one full burpee: squats down, hands to the floor, jumps her feet back to a plank, jumps them in and stands with a small hop. The top stays down and no skin shows at the waist.`,
			`3.0s-5.0s — Stands, hands on hips, breathing, and talks to the camera about the fabric being sweat-wicking and staying opaque even bent over.`,
			`5.0s-6.5s — Drops into one deep squat facing the camera, then rises, running a hand along the waistband to show it did not roll down.`,
			`6.5s-8.0s — Raises her free index finger and points clearly toward the bottom-left corner of the frame, holding it while she breathes easy. The arena behind her is unchanged.`,
		},
	},
}

// CAFE_ITEMS mirrors the HTML cafe table presets.
var CAFE_ITEMS = []Preset{
	{Th: "กาแฟเย็น แก้วพลาสติกใส", En: `a tall clear plastic takeaway cup of iced coffee with a domed lid and a straw, pale caramel colour with ice visible through the cup`},
	{Th: "ชาเขียวมัทฉะเย็น", En: `a tall clear plastic takeaway cup of iced matcha latte with a domed lid and a straw, layered jade green over milky white with ice visible`},
	{Th: "ชานมเย็น", En: `a tall clear plastic takeaway cup of iced Thai milk tea with a domed lid and a straw, warm orange-tan colour with ice visible`},
	{Th: "อเมริกาโน่เย็น", En: `a tall clear plastic takeaway cup of iced americano with a domed lid and a straw, deep clear brown with ice visible`},
	{Th: "เค้กหนึ่งชิ้น + กาแฟ", En: `a small white plate holding one neat slice of vanilla sponge cake with a fork resting beside it, and a tall clear plastic takeaway cup of iced coffee with a straw`},
	{Th: "จานผลไม้ + กาแฟ", En: `a small white plate of sliced fresh fruit — strawberry, orange and green grapes arranged neatly — and a tall clear plastic takeaway cup of iced coffee with a straw`},
}

// VOICES mirrors the HTML voice presets.
var VOICES = []Preset{
	{Th: "เสียงเล็ก ใส น่าฟัง · 25 ปี", En: `A young Thai woman in her mid-twenties. Light, clear, slightly high-pitched voice with a soft airy timbre and a gentle natural rasp on the breath. Warm and friendly, unhurried, speaking at a relaxed conversational pace as if chatting with a close friend across a room. Soft rising intonation at the end of phrases, no projection, no announcer energy, no sales tone.`},
	{Th: "เสียงนุ่ม ต่ำ สุขุม · 27 ปี", En: `A Thai woman in her late twenties. Soft mid-low voice with a smooth velvety timbre and calm steady delivery. Speaks slowly and thoughtfully, warm and reassuring, like someone giving honest advice to a friend. Gentle downward intonation, no brightness, no perkiness, no sales tone.`},
	{Th: "เสียงสดใส ร่าเริง · 22 ปี", En: `A Thai woman in her early twenties. Bright, light, energetic voice with a youthful lift and a soft giggle in the tone. Speaks quickly but clearly, playful and excited, like sharing a good find with a best friend. Bouncy intonation, natural laughter breaks, no shouting, no announcer energy, no sales tone.`},
	{Th: "เสียงกระซิบนุ่ม ASMR · 25 ปี", En: `A Thai woman in her mid-twenties speaking in a soft close-mic near-whisper. Breathy, intimate, very gentle, low volume with clear diction. Slow relaxed pacing, warm and calming, as if talking quietly beside a friend. No projection, no brightness, no sales tone.`},
	{Th: "เสียงธรรมชาติ พูดเร็วนิด · 24 ปี", En: `A Thai woman in her mid-twenties. Natural everyday speaking voice, medium pitch, slightly quick and casual with unpolished real-person delivery including small natural pauses and breaths. Friendly and matter-of-fact, like a quick voice note to a friend. No announcer energy, no sales tone.`},
}

// BOTTOMS mirrors the HTML bottom presets (the risk field is UI-only and dropped).
var BOTTOMS = []Preset{
	{Th: "ยีนส์ขากว้างยาว", En: `high-waisted light blue wide-leg jeans that reach the ankle`},
	{Th: "กระโปรงผ้าฝ้ายยาวขาว", En: `a long white cotton skirt with a soft crinkled texture and relaxed drape, hemline at the ankle`},
	{Th: "กระโปรงระบายชั้นครีมยาว", En: `a cream tiered skirt with fine lace trim on each tier, soft cotton, hemline below the knee`},
	{Th: "กางเกงผ้าลื่นขายาวครีม", En: `high-waisted wide-leg trousers in cream satin-finish fabric, full length, soft flowing drape`},
	{Th: "กางเกงลินินขายาวเบจ", En: `high-waisted tailored linen trousers in warm beige, full length, soft drape`},
	{Th: "กระโปรงยาวผ้าบางสีนวล", En: `a high-waisted flowing maxi skirt in soft ivory chiffon, hemline at the ankle`},
	{Th: "กระโปรงคลุมเข่า", En: `a high-waisted A-line midi skirt in soft cream, hemline below the knee`},
	{Th: "กระโปรงน้ำตาลเข้มคลุมเข่า", En: `a high-waisted A-line skirt in dark chocolate brown matte woven fabric, hemline below the knee`},
	{Th: "กางเกงวอร์มขายาว", En: `high-waisted soft-jersey joggers in heather grey, full length, relaxed fit`},
	{Th: "กระโปรงทรงเอเหนือเข่า", En: `a high-waisted A-line skirt in soft cream, hemline just above the knee, gently flared`},
	{Th: "กระโปรงจีบพาสเทล", En: `a high-waisted pleated skirt in pastel butter yellow, crisp knife pleats, hemline just above the knee`},
	{Th: "กระโปรงระบายชั้นครีมสั้น", En: `a cream tiered ruffle mini skirt with fine lace trim on each tier, hemline at mid-thigh`},
	{Th: "กระโปรงสั้นน้ำตาลเข้ม", En: `a high-waisted dark chocolate brown A-line mini skirt in matte woven fabric, hemline at mid-thigh`},
	{Th: "กางเกงลินินขาสั้น", En: `high-waisted tailored linen shorts in warm beige, hemline at mid-thigh, soft drape`},
	{Th: "ยีนส์ขาสั้น", En: `high-waisted light blue denim shorts with a neat turned-up hem, hemline at mid-thigh`},
}

// fmtFor picks the format for a product id, defaulting to FORMATS[0] (mirror).
func fmtFor(id string) Format {
	if id == "" {
		id = "mirror"
	}
	for _, f := range FORMATS {
		if f.ID == id {
			return f
		}
	}
	return FORMATS[0]
}
