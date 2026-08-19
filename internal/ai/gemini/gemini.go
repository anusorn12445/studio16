// Package gemini implements ai.Analyzer (vision via generateContent) and
// ai.VideoGenerator (Google Veo via predictLongRunning) on the Gemini API.
//
// Endpoint shapes follow the public Generative Language API. Model ids and the
// exact Veo response layout evolve; they are configurable and isolated here so
// only this file changes if Google adjusts the contract.
package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"studio16/internal/ai"
	"studio16/internal/prompt"
)

const base = "https://generativelanguage.googleapis.com/v1beta"

type Client struct {
	key         string
	visionModel string
	imageModel  string
	veoModel    string
	http        *http.Client
}

func New(key, visionModel, imageModel, veoModel string) *Client {
	return &Client{
		key:         key,
		visionModel: visionModel,
		imageModel:  imageModel,
		veoModel:    veoModel,
		http:        &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) Name() string { return "gemini:" + c.visionModel }

// ================= image generation (Nano Banana / Gemini image) =================

type imgGenReq struct {
	Contents         []content `json:"contents"`
	GenerationConfig struct {
		ResponseModalities []string `json:"responseModalities"`
		Temperature        float64  `json:"temperature,omitempty"`
		Seed               *int     `json:"seed,omitempty"`
	} `json:"generationConfig"`
}

type imgGenResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text       string `json:"text"`
				InlineData *struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateImage creates a styled image from a text prompt plus optional
// reference photos (the garment), using the configured image model
// ("Nano Banana" — Gemini image generation). The result is the opening frame
// that Veo then animates.
func (c *Client) GenerateImage(ctx context.Context, promptText string, refs []ai.Image, seed int) (ai.Image, error) {
	if c.key == "" {
		return ai.Image{}, fmt.Errorf("gemini: GEMINI_API_KEY not set")
	}
	parts := make([]part, 0, len(refs)+1)
	for _, im := range refs {
		mime := im.Mime
		if mime == "" {
			mime = "image/jpeg"
		}
		parts = append(parts, part{InlineData: &inlineData{MimeType: mime, Data: base64.StdEncoding.EncodeToString(im.Data)}})
	}
	parts = append(parts, part{Text: promptText})

	var req imgGenReq
	req.Contents = []content{{Parts: parts}}
	req.GenerationConfig.ResponseModalities = []string{"IMAGE"}
	// Nano Banana is deterministic by default: given the same reference photo it
	// returns byte-identical output no matter how the text prompt differs, so
	// every shot in a batch collapsed to one image. A distinct seed per shot plus
	// a non-zero temperature forces each shot to be a genuinely different render.
	req.GenerationConfig.Temperature = 1.0
	if seed != 0 {
		s := seed
		req.GenerationConfig.Seed = &s
	}
	body, _ := json.Marshal(req)

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", base, c.imageModel, c.key)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return ai.Image{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var r imgGenResp
	_ = json.Unmarshal(raw, &r)
	if resp.StatusCode != http.StatusOK {
		if r.Error != nil {
			return ai.Image{}, fmt.Errorf("image: %s", r.Error.Message)
		}
		return ai.Image{}, fmt.Errorf("image: HTTP %d", resp.StatusCode)
	}
	for _, cand := range r.Candidates {
		for _, p := range cand.Content.Parts {
			if p.InlineData != nil && p.InlineData.Data != "" {
				data, err := base64.StdEncoding.DecodeString(p.InlineData.Data)
				if err != nil {
					return ai.Image{}, fmt.Errorf("image: bad base64: %w", err)
				}
				mime := p.InlineData.MimeType
				if mime == "" {
					mime = "image/png"
				}
				return ai.Image{Mime: mime, Data: data}, nil
			}
		}
	}
	return ai.Image{}, fmt.Errorf("image: no image returned by %s", c.imageModel)
}

// ================= vision / text (generateContent) =================

type inlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inline_data,omitempty"`
}

type content struct {
	Parts []part `json:"parts"`
}

type genReq struct {
	Contents         []content `json:"contents"`
	GenerationConfig struct {
		MaxOutputTokens int `json:"maxOutputTokens"`
	} `json:"generationConfig"`
}

type genResp struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) generate(ctx context.Context, imgs []ai.Image, text string, maxTokens int) (string, error) {
	if c.key == "" {
		return "", fmt.Errorf("gemini: GEMINI_API_KEY not set")
	}
	parts := make([]part, 0, len(imgs)+1)
	for _, im := range imgs {
		mime := im.Mime
		if mime == "" {
			mime = "image/jpeg"
		}
		parts = append(parts, part{InlineData: &inlineData{
			MimeType: mime,
			Data:     base64.StdEncoding.EncodeToString(im.Data),
		}})
	}
	parts = append(parts, part{Text: text})

	var req genReq
	req.Contents = []content{{Parts: parts}}
	req.GenerationConfig.MaxOutputTokens = maxTokens
	body, _ := json.Marshal(req)

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", base, c.visionModel, c.key)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var r genResp
	_ = json.Unmarshal(raw, &r)
	if resp.StatusCode != http.StatusOK {
		if r.Error != nil {
			return "", fmt.Errorf("gemini: %s", r.Error.Message)
		}
		return "", fmt.Errorf("gemini: HTTP %d", resp.StatusCode)
	}
	if len(r.Candidates) == 0 || len(r.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}
	out := ""
	for _, p := range r.Candidates[0].Content.Parts {
		out += p.Text
	}
	return out, nil
}

func (c *Client) AnalyzePhotos(ctx context.Context, imgs []ai.Image, shopDesc, focus string) (string, error) {
	out, err := c.generate(ctx, imgs, prompt.AnalyzePrompt(shopDesc, focus), 2000)
	if err != nil {
		return "", err
	}
	return prompt.ExtractJSON(out)
}

func (c *Client) GenerateText(ctx context.Context, promptText string) (string, error) {
	return c.generate(ctx, nil, promptText, 2048)
}

func (c *Client) ScoreMatch(ctx context.Context, refs []ai.Image, candidate ai.Image, specText string) (ai.MatchResult, error) {
	imgs := append([]ai.Image{}, refs...)
	imgs = append(imgs, candidate)
	out, err := c.generate(ctx, imgs, prompt.MatchPrompt(len(refs), specText), 2048)
	if err != nil {
		return ai.MatchResult{}, err
	}
	js, err := prompt.ExtractJSON(out)
	if err != nil {
		return ai.MatchResult{}, err
	}
	var mr ai.MatchResult
	if err := json.Unmarshal([]byte(js), &mr); err != nil {
		return ai.MatchResult{}, fmt.Errorf("gemini: bad match json: %w", err)
	}
	return mr, nil
}

func (c *Client) ScoreVideo(ctx context.Context, refs []ai.Image, frames []ai.Image, specText string) (ai.MatchResult, error) {
	imgs := append([]ai.Image{}, refs...)
	imgs = append(imgs, frames...)
	out, err := c.generate(ctx, imgs, prompt.VideoQualityPrompt(len(refs), len(frames), specText), 2048)
	if err != nil {
		return ai.MatchResult{}, err
	}
	js, err := prompt.ExtractJSON(out)
	if err != nil {
		return ai.MatchResult{}, err
	}
	var mr ai.MatchResult
	if err := json.Unmarshal([]byte(js), &mr); err != nil {
		return ai.MatchResult{}, fmt.Errorf("gemini: bad video json: %w", err)
	}
	return mr, nil
}

// ================= video (Veo, predictLongRunning) =================

type veoImage struct {
	BytesBase64Encoded string `json:"bytesBase64Encoded"`
	MimeType           string `json:"mimeType"`
}

type veoInstance struct {
	Prompt string    `json:"prompt"`
	Image  *veoImage `json:"image,omitempty"`
}

type veoReq struct {
	Instances  []veoInstance  `json:"instances"`
	Parameters map[string]any `json:"parameters"`
}

type opResp struct {
	Name  string `json:"name"`
	Done  bool   `json:"done"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Response json.RawMessage `json:"response"`
}

func (c *Client) StartVideo(ctx context.Context, promptText string, firstFrame *ai.Image, durationSeconds int) (string, error) {
	if c.key == "" {
		return "", fmt.Errorf("gemini: GEMINI_API_KEY not set")
	}
	// Veo accepts a limited duration range; clamp and default to 8s.
	if durationSeconds <= 0 {
		durationSeconds = 8
	}
	if durationSeconds < 4 {
		durationSeconds = 4
	}
	if durationSeconds > 8 {
		durationSeconds = 8
	}
	inst := veoInstance{Prompt: promptText}
	if firstFrame != nil {
		mime := firstFrame.Mime
		if mime == "" {
			mime = "image/jpeg"
		}
		inst.Image = &veoImage{
			BytesBase64Encoded: base64.StdEncoding.EncodeToString(firstFrame.Data),
			MimeType:           mime,
		}
	}
	body, _ := json.Marshal(veoReq{
		Instances: []veoInstance{inst},
		Parameters: map[string]any{
			"aspectRatio":     "9:16",
			"sampleCount":     1,
			"durationSeconds": durationSeconds,
		},
	})

	url := fmt.Sprintf("%s/models/%s:predictLongRunning?key=%s", base, c.veoModel, c.key)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var r opResp
	_ = json.Unmarshal(raw, &r)
	if resp.StatusCode != http.StatusOK || r.Name == "" {
		if r.Error != nil {
			return "", fmt.Errorf("veo: %s", r.Error.Message)
		}
		return "", fmt.Errorf("veo: start failed HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return r.Name, nil
}

// videoURIFromResponse digs the produced video uri out of the operation payload.
// Veo has shipped a few layouts; try the known ones.
func videoURIFromResponse(resp json.RawMessage) string {
	if len(resp) == 0 {
		return ""
	}
	var probe map[string]any
	if json.Unmarshal(resp, &probe) != nil {
		return ""
	}
	// walk common paths: generateVideoResponse.generatedSamples[].video.uri
	//                    predictions[].videoUri / bytesBase64Encoded
	find := func(m map[string]any, keys ...string) any {
		var cur any = m
		for _, k := range keys {
			mm, ok := cur.(map[string]any)
			if !ok {
				return nil
			}
			cur = mm[k]
		}
		return cur
	}
	if v := find(probe, "generateVideoResponse"); v != nil {
		if gv, ok := v.(map[string]any); ok {
			if samples, ok := gv["generatedSamples"].([]any); ok && len(samples) > 0 {
				if s0, ok := samples[0].(map[string]any); ok {
					if vid, ok := s0["video"].(map[string]any); ok {
						if uri, _ := vid["uri"].(string); uri != "" {
							return uri
						}
					}
				}
			}
		}
	}
	if preds, ok := probe["predictions"].([]any); ok && len(preds) > 0 {
		if p0, ok := preds[0].(map[string]any); ok {
			if uri, _ := p0["videoUri"].(string); uri != "" {
				return uri
			}
		}
	}
	return ""
}

func (c *Client) PollVideo(ctx context.Context, opName string) (ai.VideoStatus, error) {
	url := fmt.Sprintf("%s/%s?key=%s", base, opName, c.key)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return ai.VideoStatus{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var r opResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return ai.VideoStatus{}, fmt.Errorf("veo: bad poll json: %w", err)
	}
	if r.Error != nil {
		return ai.VideoStatus{Done: true, Error: r.Error.Message}, nil
	}
	if !r.Done {
		return ai.VideoStatus{Done: false}, nil
	}
	uri := videoURIFromResponse(r.Response)
	if uri == "" {
		return ai.VideoStatus{Done: true, Error: "veo: operation done but no video uri found"}, nil
	}
	return ai.VideoStatus{Done: true, VideoURL: uri, Mime: "video/mp4"}, nil
}

// DownloadVideo fetches a produced video uri, appending the API key which the
// Gemini Files API requires for access.
func (c *Client) DownloadVideo(ctx context.Context, uri string) ([]byte, error) {
	sep := "?"
	if bytes.ContainsRune([]byte(uri), '?') {
		sep = "&"
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, uri+sep+"key="+c.key, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("veo: download HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
