package prompt

import (
	"strings"

	"studio16/internal/model"
)

// The default base blocks — copied verbatim from the STUDIO 16 buildPrompt so
// the "default" preset reproduces the original output byte-for-byte.

const defaultRole = `You are my production agent for TikTok Shop / Shopee fashion review
videos. Execute the entire pipeline below in one continuous run. Do not
stop, do not ask for approval, do not wait between steps.`

const defaultModelSettings = `Do not specify, request, or pass any model name. I have already
configured the models in Agent Settings:
- Images: Nano Banana Pro, 9:16, x1 output
- Video: Veo 3.1 - Lite [Lower Priority], 9:16, x1 output
Always use the project default models. Never append UI badges such as
"[Lower Priority]" to a model call, and never report on my subscription
tier — treat it as Ultra.
Video duration: 8 seconds. Aspect ratio: 9:16 vertical.`

const defaultIdentity = `A Thai woman, 25 years old. Oval face with a soft rounded jawline, high
smooth forehead, small straight nose with a rounded tip, gently upturned
almond eyes with a subtle inner fold and dark brown irises, thick natural
lower lashes, full lips with a defined cupid's bow, softly arched
cheekbones. Fair porcelain skin with cool pink undertones, bright and
even in tone. Slim natural build.`

const defaultStyle = `SKIN — fair porcelain complexion with cool pink undertones, the pale
bright tone seen in Korean beauty photography. Bright and even with no
tan, no olive cast and no dullness. Korean glass skin: luminous,
translucent and dewy with a wet-look sheen across the cheekbones, nose
bridge, brow bones and collarbones, yet keeping visible fine pores and
real skin texture beneath the glow — luminous, never plastic. Keep
natural dimension: soft shadow under the cheekbones and jaw, never flat
or blown out. The shadows on her skin fall cool and neutral grey, never
warm brown or golden.

MAKEUP — a soft pink Korean daytime look, light and fresh. Cool rosy-pink
blush swept high across the cheeks and lightly over the nose bridge. Pink
pearlescent shimmer under the lower lashline. Soft mauve-pink eyeshadow
blended sheer, thin brown liner close to the lashes, straight
softly-drawn brows, inner-corner highlight. Glossy cool pink gradient
lips with a wet shine. Every pink stays cool-toned throughout — no peach,
no coral, no terracotta, no warm orange anywhere on the face.

HAIR — Long dark brown hair with a warm chestnut tint, worn down loose
with soft face-framing layers and see-through wispy bangs, gentle natural
waves. Swept behind both shoulders so the neckline edge and both shoulder
straps stay clearly visible for the entire duration.`

// A second look — natural, minimal makeup, warmer everyday styling.
const naturalIdentity = `A Thai woman, 24 years old. Soft heart-shaped face with a gently rounded
jaw, natural straight brows, warm dark brown almond eyes, a small nose
and relaxed full lips. Healthy neutral-warm skin with a light natural
glow. Slim natural build, an approachable girl-next-door look.`

const naturalStyle = `SKIN — healthy natural skin with a soft satin finish, not glassy and not
matte. Even tone with visible real texture and fine pores kept, a light
inner glow across the cheeks. No heavy highlight, no wet-look sheen, no
airbrushing. Soft natural dimension with gentle shadow under the
cheekbones.

MAKEUP — a bare-faced "no-makeup" everyday look. Skin-tone tinted balm,
softly groomed natural brows, a thin wash of neutral brown on the lids,
no visible liner, and a sheer my-lips-but-better rosy tint with a soft
satin finish. Nothing bold, nothing shimmery.

HAIR — Long dark brown hair worn down loose with a soft centre part and
gentle natural movement, no heavy styling. Swept behind both shoulders so
the neckline edge and both shoulder straps stay clearly visible for the
entire duration.`

// A third look — clean, bright, minimal-glam for a premium feel.
const cleanStyle = `SKIN — clean, bright, even complexion with a soft luminous finish. Fresh
and healthy with real skin texture preserved, a light natural radiance on
the high points of the face. No heavy contouring, no orange or golden
cast, cool-neutral shadows.

MAKEUP — a clean minimal-glam daytime look. Neutral matte base, softly
defined brows, a single wash of soft taupe eyeshadow, fine tightline
close to the lashes, and a muted rose satin lip. Understated and premium,
never heavy.

HAIR — Long dark brown hair, sleek and smooth with a soft centre part and
subtle shine, worn down. Swept behind both shoulders so the neckline edge
and both shoulder straps stay clearly visible for the entire duration.`

// BasePresets are the selectable base-prompt sets. The first is the default.
var BasePresets = []model.BasePreset{
	{
		ID:            "default",
		Name:          "ผู้หญิงไทย 25 · เกาหลีกลาสสกิน (ค่าเริ่มต้น)",
		Role:          defaultRole,
		ModelSettings: defaultModelSettings,
		Identity:      defaultIdentity,
		Style:         defaultStyle,
	},
	{
		ID:            "natural",
		Name:          "ผู้หญิงไทย 24 · ลุคธรรมชาติ แต่งหน้าน้อย",
		Role:          defaultRole,
		ModelSettings: defaultModelSettings,
		Identity:      naturalIdentity,
		Style:         naturalStyle,
	},
	{
		ID:            "clean",
		Name:          "ผู้หญิงไทย 25 · คลีน มินิมอล พรีเมียม",
		Role:          defaultRole,
		ModelSettings: defaultModelSettings,
		Identity:      defaultIdentity,
		Style:         cleanStyle,
	},
}

// GetBasePreset returns the preset with the given id.
func GetBasePreset(id string) (model.BasePreset, bool) {
	for _, b := range BasePresets {
		if b.ID == id {
			return b, true
		}
	}
	return model.BasePreset{}, false
}

// ResolveBase decides which base blocks Build should use: the selected preset,
// then any per-product custom edits overlaid on top (blank custom fields fall
// back to the selected preset's value).
func ResolveBase(p model.Product) model.BasePreset {
	base := BasePresets[0]
	if p.BasePresetID != "" {
		if b, ok := GetBasePreset(p.BasePresetID); ok {
			base = b
		}
	}
	if p.BaseCustom != nil {
		c := *p.BaseCustom
		if strings.TrimSpace(c.Role) == "" {
			c.Role = base.Role
		}
		if strings.TrimSpace(c.ModelSettings) == "" {
			c.ModelSettings = base.ModelSettings
		}
		if strings.TrimSpace(c.Identity) == "" {
			c.Identity = base.Identity
		}
		if strings.TrimSpace(c.Style) == "" {
			c.Style = base.Style
		}
		base = c
	}
	return base
}
