// Package video orchestrates asynchronous Veo video generation jobs: it starts
// a job, polls until the provider finishes, downloads the result and persists it.
package video

import (
	"context"
	"time"

	"studio16/internal/ai"
	"studio16/internal/model"
	"studio16/internal/store"
)

// Generator is a VideoGenerator that can also generate the first-frame image
// and download its produced files.
type Generator interface {
	ai.VideoGenerator
	GenerateImage(ctx context.Context, prompt string, refs []ai.Image) (ai.Image, error)
	DownloadVideo(ctx context.Context, uri string) ([]byte, error)
}

type Manager struct {
	gen   Generator
	store *store.Store

	pollEvery time.Duration
	timeout   time.Duration
}

func NewManager(gen Generator, st *store.Store) *Manager {
	return &Manager{gen: gen, store: st, pollEvery: 10 * time.Second, timeout: 15 * time.Minute}
}

// Request bundles everything one shot needs, including the optional image
// quality gate (Checker + Threshold + SpecText).
type Request struct {
	Format          string
	AudioMode       string
	VideoPrompt     string
	ImagePrompt     string
	Refs            []ai.Image
	FirstFrame      *ai.Image
	DurationSeconds int
	Scene           int

	Checker   ai.Analyzer // scores the generated image vs the product; nil = no gate
	Threshold int
	SpecText  string

	CharRef *ai.Image // extra image-gen reference (shot 1's image) to lock the same face
}

// Start creates a job on the product, kicks off generation and returns the job
// immediately. The job is polled to completion in a background goroutine.
func (m *Manager) Start(productID string, req Request) (*model.Job, error) {
	now := time.Now().Unix()
	job := &model.Job{
		ID:              store.NewID("job"),
		Format:          req.Format,
		AudioMode:       req.AudioMode,
		DurationSeconds: req.DurationSeconds,
		Prompt:          req.VideoPrompt,
		ImagePrompt:     req.ImagePrompt,
		Scene:           req.Scene,
		Provider:        m.gen.Name(),
		Status:          "queued",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if _, err := m.store.Update(productID, func(p *model.Product) error {
		p.Jobs = append([]model.Job{*job}, p.Jobs...)
		return nil
	}); err != nil {
		return nil, err
	}

	go m.run(productID, job.ID, req)
	return job, nil
}

// StartBatch runs a sequence of shots as one story. Shot 1's accepted image
// becomes the character reference for the rest, so the same woman appears in
// every shot. Image steps run sequentially; each video render happens in the
// background so the batch does not wait for renders.
func (m *Manager) StartBatch(productID string, reqs []Request) ([]*model.Job, error) {
	now := time.Now().Unix()
	jobs := make([]*model.Job, 0, len(reqs))
	ids := make([]string, 0, len(reqs))
	for _, req := range reqs {
		job := &model.Job{
			ID:              store.NewID("job"),
			Format:          req.Format,
			AudioMode:       req.AudioMode,
			DurationSeconds: req.DurationSeconds,
			Prompt:          req.VideoPrompt,
			ImagePrompt:     req.ImagePrompt,
			Scene:           req.Scene,
			Provider:        m.gen.Name(),
			Status:          "queued",
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		jobs = append(jobs, job)
		ids = append(ids, job.ID)
	}
	if _, err := m.store.Update(productID, func(p *model.Product) error {
		batch := make([]model.Job, len(jobs))
		for i, j := range jobs {
			batch[i] = *j
		}
		p.Jobs = append(batch, p.Jobs...) // keep story order (part 1 first)
		return nil
	}); err != nil {
		return nil, err
	}
	go m.runBatch(productID, ids, reqs)
	return jobs, nil
}

func (m *Manager) runBatch(productID string, ids []string, reqs []Request) {
	var charRef *ai.Image
	for i := range reqs {
		req := reqs[i]
		if charRef != nil {
			req.CharRef = charRef
		}
		img := m.run(productID, ids[i], req)
		if charRef == nil && img != nil {
			charRef = img
		}
	}
}

// Rerun re-runs an existing job (used by the regenerate button after a mismatch).
func (m *Manager) Rerun(productID, jobID string, req Request) {
	m.setJob(productID, jobID, func(j *model.Job) {
		j.Status = "queued"
		j.Error = ""
		j.VideoPath = ""
		j.MatchScore = 0
		j.Attempt = 0
	})
	go m.run(productID, jobID, req)
}

func (m *Manager) setJob(productID, jobID string, mutate func(*model.Job)) {
	_, _ = m.store.Update(productID, func(p *model.Product) error {
		for i := range p.Jobs {
			if p.Jobs[i].ID == jobID {
				mutate(&p.Jobs[i])
				p.Jobs[i].UpdatedAt = time.Now().Unix()
			}
		}
		return nil
	})
}

// run does the image step (with gate/retry), then kicks off the video render in
// the background. It returns the accepted first-frame image (the character), so
// a batch can lock the same person across shots; nil on mismatch/error.
func (m *Manager) run(productID, jobID string, req Request) *ai.Image {
	frame, ok := m.makeImage(productID, jobID, req)
	if !ok {
		return nil
	}
	go m.renderVideo(productID, jobID, req, frame)
	return frame
}

// makeImage generates the opening-frame image, forces 9:16, and — when a Checker
// is set — verifies it matches the product, retrying up to 3 times. If it still
// doesn't match, it sets status "mismatch" and returns ok=false (no video).
func (m *Manager) makeImage(productID, jobID string, req Request) (*ai.Image, bool) {
	if req.ImagePrompt == "" {
		return req.FirstFrame, true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	frame := req.FirstFrame
	gate := req.Checker != nil && len(req.Refs) > 0
	matched := !gate
	for attempt := 1; attempt <= 3; attempt++ {
		m.setJob(productID, jobID, func(j *model.Job) { j.Status = "image"; j.Attempt = attempt })

		genRefs := req.Refs
		if req.CharRef != nil { // lock the same face as an earlier shot
			genRefs = append(append([]ai.Image{}, req.Refs...), *req.CharRef)
		}
		img, err := m.gen.GenerateImage(ctx, req.ImagePrompt, genRefs)
		if err != nil {
			if attempt >= 3 {
				if frame == nil {
					m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = "สร้างภาพไม่สำเร็จ: " + err.Error() })
					return nil, false
				}
				break // fall back to the uploaded reference photo
			}
			continue
		}
		img.Data = cropTo916(img.Data) // force exact 9:16, no black bars
		img.Mime = "image/jpeg"
		frame = &img
		if path, e := m.store.SaveAsset(productID, jobID+".frame.jpg", img.Data); e == nil {
			m.setJob(productID, jobID, func(j *model.Job) { j.ImagePath = path })
		}
		if !gate {
			matched = true
			break
		}
		m.setJob(productID, jobID, func(j *model.Job) { j.Status = "checking" })
		mr, e := req.Checker.ScoreMatch(ctx, req.Refs, img, req.SpecText) // gate uses product refs only
		if e != nil {
			matched = true // don't block on a failed check
			break
		}
		m.setJob(productID, jobID, func(j *model.Job) { j.MatchScore = mr.Score })
		if mr.Score >= req.Threshold {
			matched = true
			break
		}
		// else: not matching — loop and regenerate the image
	}
	if !matched {
		m.setJob(productID, jobID, func(j *model.Job) {
			j.Status = "mismatch"
			j.Error = "สินค้าไม่ตรงหลังลองสร้างภาพ 3 ครั้ง — กดเจนใหม่"
		})
		return nil, false
	}
	return frame, true
}

// renderVideo animates the frame into video with Veo and polls to completion.
func (m *Manager) renderVideo(productID, jobID string, req Request, frame *ai.Image) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	opName, err := m.gen.StartVideo(ctx, req.VideoPrompt, frame, req.DurationSeconds)
	if err != nil {
		m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = err.Error() })
		return
	}
	m.setJob(productID, jobID, func(j *model.Job) { j.Status = "running"; j.OpName = opName })

	ticker := time.NewTicker(m.pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = "timed out" })
			return
		case <-ticker.C:
			st, err := m.gen.PollVideo(ctx, opName)
			if err != nil {
				continue // transient; keep polling until timeout
			}
			if !st.Done {
				continue
			}
			if st.Error != "" {
				m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = st.Error })
				return
			}
			data := st.Inline
			if len(data) == 0 && st.VideoURL != "" {
				data, err = m.gen.DownloadVideo(ctx, st.VideoURL)
				if err != nil {
					m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = err.Error() })
					return
				}
			}
			path, err := m.store.SaveAsset(productID, jobID+".mp4", data)
			if err != nil {
				m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = err.Error() })
				return
			}
			m.setJob(productID, jobID, func(j *model.Job) { j.Status = "done"; j.VideoPath = path })
			return
		}
	}
}
