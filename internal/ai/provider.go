// Package ai defines provider-agnostic interfaces for the vision, text and video
// capabilities studio16-go needs, plus the shared request/response types.
package ai

import "context"

// Image is one binary image passed to a provider.
type Image struct {
	Mime string
	Data []byte
}

// MatchResult is one asset scored against the product reference photos.
type MatchResult struct {
	Score      int      `json:"score"` // 0-100, how well the asset matches the real product
	Verdict    string   `json:"verdict"`
	Mismatches []string `json:"mismatches"`
}

// Analyzer reads product photos and returns garment spec JSON, and scores matches.
type Analyzer interface {
	Name() string
	// AnalyzePhotos returns a raw JSON object string describing the garment
	// (typeTh, type, spec{...}) following the STUDIO 16 schema.
	AnalyzePhotos(ctx context.Context, imgs []Image, shopDesc, focus string) (string, error)
	// ScoreMatch compares a candidate image against reference photos and the
	// written spec, returning a 0-100 match score and any mismatches.
	ScoreMatch(ctx context.Context, refs []Image, candidate Image, specText string) (MatchResult, error)
}

// VideoStatus is the state of an async video generation operation.
type VideoStatus struct {
	Done     bool
	VideoURL string // download URL, if the provider returns one
	Inline   []byte // inline bytes, if the provider returns them
	Mime     string
	Error    string
}

// VideoGenerator generates video from a text prompt and an optional first frame.
type VideoGenerator interface {
	Name() string
	StartVideo(ctx context.Context, prompt string, firstFrame *Image, durationSeconds int) (opName string, err error)
	PollVideo(ctx context.Context, opName string) (VideoStatus, error)
}
