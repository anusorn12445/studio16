// Package match produces the product-match report: it scores each uploaded
// photo and each generated clip against the real product, using a vision model,
// and marks pass/fail against a configurable threshold.
package match

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"studio16/internal/ai"
	"studio16/internal/model"
	"studio16/internal/store"
)

type Scorer struct {
	analyzer  ai.Analyzer
	store     *store.Store
	threshold int
	ffmpeg    string
}

func NewScorer(analyzer ai.Analyzer, st *store.Store, threshold int, ffmpeg string) *Scorer {
	return &Scorer{analyzer: analyzer, store: st, threshold: threshold, ffmpeg: ffmpeg}
}

// specText renders the garment spec as plain text for the match instruction.
func specText(p *model.Product) string {
	var b strings.Builder
	add := func(label, v string) {
		if strings.TrimSpace(v) != "" {
			fmt.Fprintf(&b, "- %s: %s\n", label, v)
		}
	}
	add("Type", p.Type)
	add("Main colour", p.HeroColor)
	add("Neckline", p.Spec.Neckline)
	add("Straps", p.Spec.Straps)
	add("Armhole", p.Spec.Armhole)
	add("Fabric", p.Spec.Fabric)
	add("Print", p.Spec.Print)
	add("Trim", p.Spec.Trim)
	add("Hem", p.Spec.Hem)
	add("Fit", p.Spec.Fit)
	add("Details", p.Spec.Details)
	add("Garment", p.Garment)
	return b.String()
}

func (s *Scorer) readImage(rel, mime string) (ai.Image, error) {
	b, err := os.ReadFile(s.store.AbsPath(rel))
	if err != nil {
		return ai.Image{}, err
	}
	if mime == "" {
		mime = "image/jpeg"
	}
	return ai.Image{Mime: mime, Data: b}, nil
}

// extractFrame grabs a still (~1s in) from a video and returns it as an image.
func (s *Scorer) extractFrame(productID, videoRel string) (ai.Image, string, error) {
	framRel := strings.TrimSuffix(videoRel, filepath.Ext(videoRel)) + ".frame.jpg"
	frameAbs := s.store.AbsPath(framRel)
	if err := os.MkdirAll(filepath.Dir(frameAbs), 0o755); err != nil {
		return ai.Image{}, "", err
	}
	cmd := exec.Command(s.ffmpeg, "-y", "-ss", "1", "-i", s.store.AbsPath(videoRel),
		"-frames:v", "1", "-q:v", "3", frameAbs)
	if out, err := cmd.CombinedOutput(); err != nil {
		return ai.Image{}, "", fmt.Errorf("ffmpeg: %v: %s", err, string(out))
	}
	img, err := s.readImage(framRel, "image/jpeg")
	return img, framRel, err
}

// Run scores every uploaded image (against the other uploads) and every finished
// clip (against the uploads), building a fresh Report.
func (s *Scorer) Run(ctx context.Context, productID string) (*model.Report, error) {
	p, err := s.store.Get(productID)
	if err != nil {
		return nil, err
	}
	spec := specText(p)

	// Load all uploaded reference images.
	type ref struct {
		id  string
		rel string
		img ai.Image
	}
	var refs []ref
	for _, im := range p.Images {
		img, err := s.readImage(im.Path, im.Mime)
		if err != nil {
			continue
		}
		refs = append(refs, ref{id: im.ID, rel: im.Path, img: img})
	}

	report := &model.Report{
		CreatedAt: time.Now().Unix(),
		Threshold: s.threshold,
	}

	imgsOf := func(rs []ref) []ai.Image {
		out := make([]ai.Image, 0, len(rs))
		for _, r := range rs {
			out = append(out, r.img)
		}
		if len(out) > 3 {
			out = out[:3]
		}
		return out
	}

	// 1) Score each uploaded image against the other uploads (flags an odd one out).
	for i, r := range refs {
		var others []ref
		for j, o := range refs {
			if j != i {
				others = append(others, o)
			}
		}
		if len(others) == 0 {
			continue // nothing to compare a single photo against
		}
		mr, err := s.analyzer.ScoreMatch(ctx, imgsOf(others), r.img, spec)
		item := model.MatchItem{Kind: "image", RefID: r.id, Path: r.rel}
		if err != nil {
			item.Verdict = "ตรวจไม่สำเร็จ: " + err.Error()
		} else {
			item.Score = mr.Score
			item.Verdict = mr.Verdict
			item.Mismatches = mr.Mismatches
			item.Pass = mr.Score >= s.threshold
		}
		s.tally(report, item)
	}

	// 2) Score each finished clip's frame against the uploaded references.
	for _, job := range p.Jobs {
		if job.Status != "done" || job.VideoPath == "" {
			continue
		}
		item := model.MatchItem{Kind: "clip", RefID: job.ID, Path: job.VideoPath}
		if len(refs) == 0 {
			item.Verdict = "ไม่มีรูปสินค้าอ้างอิงให้เทียบ"
			s.tally(report, item)
			continue
		}
		frame, frameRel, err := s.extractFrame(productID, job.VideoPath)
		if err != nil {
			item.Verdict = "แยกเฟรมไม่สำเร็จ: " + err.Error()
			s.tally(report, item)
			continue
		}
		item.Path = frameRel
		mr, err := s.analyzer.ScoreMatch(ctx, imgsOf(refs), frame, spec)
		if err != nil {
			item.Verdict = "ตรวจไม่สำเร็จ: " + err.Error()
		} else {
			item.Score = mr.Score
			item.Verdict = mr.Verdict
			item.Mismatches = mr.Mismatches
			item.Pass = mr.Score >= s.threshold
		}
		s.tally(report, item)
	}

	// Persist the report onto the product.
	_, _ = s.store.Update(productID, func(pp *model.Product) error {
		pp.Report = report
		return nil
	})
	return report, nil
}

func (s *Scorer) tally(r *model.Report, item model.MatchItem) {
	if item.Pass {
		r.PassCount++
	} else {
		r.FailCount++
	}
	r.Items = append(r.Items, item)
}
