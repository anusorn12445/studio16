package video

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"sync"
	"testing"
	"time"

	"studio16/internal/ai"
	"studio16/internal/model"
	"studio16/internal/prompt"
	"studio16/internal/store"
)

// stubGen records every GenerateImage call and returns a UNIQUE image per call,
// so we can prove whether the manager sends distinct prompts/seeds per shot and
// saves distinct files. No network — this is free and deterministic.
type stubGen struct {
	mu    sync.Mutex
	calls []stubCall
}
type stubCall struct {
	prompt string
	seed   int
}

func (g *stubGen) Name() string { return "stub" }
func (g *stubGen) GenerateImage(ctx context.Context, promptText string, refs []ai.Image, seed int) (ai.Image, error) {
	g.mu.Lock()
	n := len(g.calls)
	g.calls = append(g.calls, stubCall{prompt: promptText, seed: seed})
	g.mu.Unlock()
	// unique solid-colour JPEG per call
	img := image.NewRGBA(image.Rect(0, 0, 90, 160))
	c := color.RGBA{R: uint8(30 + n*40), G: uint8(60 + n*20), B: uint8(90 + n*10), A: 255}
	for y := 0; y < 160; y++ {
		for x := 0; x < 90; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return ai.Image{Mime: "image/jpeg", Data: buf.Bytes()}, nil
}
func (g *stubGen) StartVideo(ctx context.Context, p string, f *ai.Image, d int) (string, error) {
	return "op", nil
}
func (g *stubGen) PollVideo(ctx context.Context, op string) (ai.VideoStatus, error) {
	return ai.VideoStatus{Done: true, Inline: []byte("mp4"), Mime: "video/mp4"}, nil
}
func (g *stubGen) DownloadVideo(ctx context.Context, uri string) ([]byte, error) { return []byte("mp4"), nil }

func TestBatchProducesDistinctImages(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := &model.Product{ID: "ptest", Name: "t", Format: "hyrox", AudioMode: "talk",
		HeroColor: "black", Garment: "a black sports tee",
		Images: []model.Image{{ID: "i1", Path: "x", Mime: "image/jpeg"}},
		Scripts: []model.Script{{Lines: []string{"หนึ่ง", "สอง", "สาม", "สี่"}}},
	}
	if err := st.Create(p); err != nil {
		t.Fatal(err)
	}

	g := &stubGen{}
	m := NewManager(g, st)

	// Build reqs EXACTLY like the generate handler does for 3 shots.
	refs := []ai.Image{{Mime: "image/jpeg", Data: []byte("ref")}}
	first := &refs[0]
	beats := prompt.PlanBeats(*p, 3)
	var reqs []Request
	for i, beat := range beats {
		o := prompt.VeoOpts{Line: beat.Line, Role: beat.Role, Part: i + 1, Total: len(beats), Scene: i}
		reqs = append(reqs, Request{
			Format: p.Format, AudioMode: p.AudioMode,
			VideoPrompt: prompt.BuildVeo(*p, o), ImagePrompt: prompt.BuildVeoImage(*p, o),
			Refs: refs, FirstFrame: first, DurationSeconds: 8, Scene: i,
		})
	}
	if _, err := m.StartBatch("ptest", reqs); err != nil {
		t.Fatal(err)
	}

	// wait for all 3 image files to appear
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		pp, _ := st.Get("ptest")
		done := 0
		for _, j := range pp.Jobs {
			if j.ImagePath != "" {
				done++
			}
		}
		if done >= 3 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Report what the manager actually sent to GenerateImage.
	g.mu.Lock()
	for i, c := range g.calls {
		ph := md5.Sum([]byte(c.prompt))
		t.Logf("call %d: seed=%d promptMd5=%x len=%d", i, c.seed, ph[:4], len(c.prompt))
	}
	nCalls := len(g.calls)
	g.mu.Unlock()

	// Report saved image files.
	pp, _ := st.Get("ptest")
	seen := map[string]int{}
	for _, j := range pp.Jobs {
		if j.ImagePath == "" {
			continue
		}
		b, _ := os.ReadFile(st.AbsPath(j.ImagePath))
		h := fmt.Sprintf("%x", md5.Sum(b))
		seen[h]++
		t.Logf("job scene=%d imgMd5=%s len=%d", j.Scene, h[:12], len(b))
	}
	t.Logf("GenerateImage calls=%d, distinct saved images=%d", nCalls, len(seen))
	if len(seen) < 3 {
		t.Errorf("BUG REPRODUCED: expected 3 distinct images, got %d distinct", len(seen))
	}
}
