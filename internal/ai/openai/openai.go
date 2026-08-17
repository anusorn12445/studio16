// Package openai implements the ai.Analyzer interface using the OpenAI
// Chat Completions API with vision (e.g. gpt-4o).
package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"studio16/internal/ai"
	"studio16/internal/prompt"
)

const endpoint = "https://api.openai.com/v1/chat/completions"

type Client struct {
	key   string
	model string
	http  *http.Client
}

func New(key, model string) *Client {
	return &Client{key: key, model: model, http: &http.Client{Timeout: 120 * time.Second}}
}

func (c *Client) Name() string { return "openai:" + c.model }

// ---- request/response shapes ----

type message struct {
	Role    string        `json:"role"`
	Content []contentPart `json:"content"`
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatReq struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type chatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func dataURL(img ai.Image) string {
	mime := img.Mime
	if mime == "" {
		mime = "image/jpeg"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(img.Data)
}

func (c *Client) chat(ctx context.Context, imgs []ai.Image, text string, maxTokens int) (string, error) {
	if c.key == "" {
		return "", fmt.Errorf("openai: OPENAI_API_KEY not set")
	}
	parts := make([]contentPart, 0, len(imgs)+1)
	for _, im := range imgs {
		parts = append(parts, contentPart{Type: "image_url", ImageURL: &imageURL{URL: dataURL(im)}})
	}
	parts = append(parts, contentPart{Type: "text", Text: text})

	body, _ := json.Marshal(chatReq{
		Model:     c.model,
		Messages:  []message{{Role: "user", Content: parts}},
		MaxTokens: maxTokens,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var r chatResp
	_ = json.Unmarshal(raw, &r)
	if resp.StatusCode != http.StatusOK {
		if r.Error != nil {
			return "", fmt.Errorf("openai: %s", r.Error.Message)
		}
		return "", fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("openai: empty response")
	}
	return r.Choices[0].Message.Content, nil
}

func (c *Client) AnalyzePhotos(ctx context.Context, imgs []ai.Image, shopDesc, focus string) (string, error) {
	out, err := c.chat(ctx, imgs, prompt.AnalyzePrompt(shopDesc, focus), 2000)
	if err != nil {
		return "", err
	}
	return prompt.ExtractJSON(out)
}

func (c *Client) GenerateText(ctx context.Context, promptText string) (string, error) {
	return c.chat(ctx, nil, promptText, 700)
}

func (c *Client) ScoreMatch(ctx context.Context, refs []ai.Image, candidate ai.Image, specText string) (ai.MatchResult, error) {
	// Reference photos first, then the candidate to be judged.
	imgs := append([]ai.Image{}, refs...)
	imgs = append(imgs, candidate)
	out, err := c.chat(ctx, imgs, prompt.MatchPrompt(len(refs), specText), 900)
	if err != nil {
		return ai.MatchResult{}, err
	}
	js, err := prompt.ExtractJSON(out)
	if err != nil {
		return ai.MatchResult{}, err
	}
	var mr ai.MatchResult
	if err := json.Unmarshal([]byte(js), &mr); err != nil {
		return ai.MatchResult{}, fmt.Errorf("openai: bad match json: %w", err)
	}
	return mr, nil
}
