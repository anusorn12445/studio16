// Package model holds the core data structures for a product, ported field-for-field
// from the STUDIO 16 HTML app (the blank() object) so the prompt engine stays faithful.
package model

// BasePreset holds the editable "base prompt" scaffolding — the fixed blocks
// that surround the per-product content. Users pick a preset and/or edit it.
type BasePreset struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Role          string `json:"role"`          // the ROLE block body
	ModelSettings string `json:"modelSettings"` // the MODEL SETTINGS block body
	Identity      string `json:"identity"`      // IDENTITY_BLOCK body (the person)
	Style         string `json:"style"`         // STYLE_BLOCK body (skin/makeup/hair)
}

// Spec is the detailed garment reading produced by the vision AI.
// Keys match the STUDIO 16 spec object exactly.
type Spec struct {
	Neckline string `json:"neckline"`
	Straps   string `json:"straps"`
	Armhole  string `json:"armhole"`
	Fabric   string `json:"fabric"`
	Print    string `json:"print"`
	Trim     string `json:"trim"`
	Hem      string `json:"hem"`
	Fit      string `json:"fit"`
	Details  string `json:"details"`
}

// Script is one set of four spoken lines (2 lines per scene, 2 scenes).
type Script struct {
	Lines []string `json:"lines"`
}

// Clip is a saved reference/output link.
type Clip struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Image is one uploaded product reference photo, stored on disk.
type Image struct {
	ID   string `json:"id"`
	Path string `json:"path"` // relative path under the assets dir
	Mime string `json:"mime"`
}

// Product mirrors the STUDIO 16 product object one-to-one.
type Product struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	TypeTh   string `json:"typeTh"`
	Price    string `json:"price"`
	OldPrice string `json:"oldPrice"`

	HeroColor string   `json:"heroColor"`
	Colors    []string `json:"colors"`
	ColorsEn  []string `json:"colorsEn"`
	Features  []string `json:"features"`
	SizeNote  string   `json:"sizeNote"`
	Desc      string   `json:"desc"`

	Format    string `json:"format"`    // mirror | cafe | pair
	AudioMode string `json:"audioMode"` // talk | laugh | silent

	Wm    bool   `json:"wm"`
	WmText string `json:"wmText"`
	WmPos  string `json:"wmPos"`

	PairA string `json:"pairA"`
	PairB string `json:"pairB"`
	PairC string `json:"pairC"`
	PairD string `json:"pairD"`

	StyleA string `json:"styleA"`
	StyleB string `json:"styleB"`
	StyleC string `json:"styleC"`
	StyleD string `json:"styleD"`

	Luck string `json:"luck"`

	Bottom         string `json:"bottom"`
	BottomTh       string `json:"bottomTh"`
	Shoes          string `json:"shoes"`
	ShoesTh        string `json:"shoesTh"`
	Accessories    string `json:"accessories"`
	AccessoriesTh  string `json:"accessoriesTh"`

	LookTh   string   `json:"lookTh"`
	LookWhy  string   `json:"lookWhy"`
	LookRefs []string `json:"lookRefs"`

	CafeTh   string `json:"cafeTh"`
	CafeItem string `json:"cafeItem"`
	VoiceTh  string `json:"voiceTh"`
	Voice    string `json:"voice"`

	Spec    Spec   `json:"spec"`
	Garment string `json:"garment"`

	// Base prompt selection: which preset, plus optional user edits.
	BasePresetID string      `json:"basePresetId"`
	BaseCustom   *BasePreset `json:"baseCustom"`

	Scripts []Script `json:"scripts"`
	Pick    int      `json:"pick"`
	Notes   string   `json:"notes"`
	Clips   []Clip   `json:"clips"`

	// studio16-go additions (not in the original HTML):
	Images []Image `json:"images"` // uploaded reference photos
	Jobs   []Job   `json:"jobs"`   // video generation jobs
	Report *Report `json:"report"` // latest product-match report
}

// Job tracks one Veo video generation request.
type Job struct {
	ID              string `json:"id"`
	Format          string `json:"format"`
	AudioMode       string `json:"audioMode"`
	DurationSeconds int    `json:"durationSeconds"`
	Prompt          string `json:"prompt"`
	Provider  string `json:"provider"` // "gemini-veo"
	OpName    string `json:"opName"`   // provider operation id for polling
	Status    string `json:"status"`   // queued | running | done | error
	VideoPath string `json:"videoPath"`
	Error     string `json:"error"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// Report is the AI product-match audit over reference photos and generated clips.
type Report struct {
	CreatedAt int64        `json:"createdAt"`
	Threshold int          `json:"threshold"`
	Items     []MatchItem  `json:"items"`
	PassCount int          `json:"passCount"`
	FailCount int          `json:"failCount"`
}

// MatchItem is one scored asset (an uploaded image or a generated clip frame).
type MatchItem struct {
	Kind       string   `json:"kind"` // "image" | "clip"
	RefID      string   `json:"refId"`
	Path       string   `json:"path"`
	Score      int      `json:"score"` // 0-100
	Pass       bool     `json:"pass"`
	Verdict    string   `json:"verdict"`
	Mismatches []string `json:"mismatches"`
}
