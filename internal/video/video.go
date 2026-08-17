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

// Start creates a job on the product, kicks off generation and returns the job
// immediately. The job is polled to completion in a background goroutine.
func (m *Manager) Start(productID, format, audioMode, videoPrompt, imagePrompt string, refs []ai.Image, firstFrame *ai.Image, durationSeconds int) (*model.Job, error) {
	now := time.Now().Unix()
	job := &model.Job{
		ID:              store.NewID("job"),
		Format:          format,
		AudioMode:       audioMode,
		DurationSeconds: durationSeconds,
		Prompt:          videoPrompt,
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

	go m.run(productID, job.ID, videoPrompt, imagePrompt, refs, firstFrame, durationSeconds)
	return job, nil
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

func (m *Manager) run(productID, jobID, videoPrompt, imagePrompt string, refs []ai.Image, firstFrame *ai.Image, durationSeconds int) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	// Step 1 — generate the opening-frame image (if requested), then use it as
	// Veo's first frame. Fall back to any uploaded reference photo on failure.
	frame := firstFrame
	if imagePrompt != "" {
		m.setJob(productID, jobID, func(j *model.Job) { j.Status = "image" })
		img, err := m.gen.GenerateImage(ctx, imagePrompt, refs)
		if err != nil {
			if frame == nil {
				m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = "สร้างภาพเฟรมแรกไม่สำเร็จ: " + err.Error() })
				return
			}
			// else: keep going with the uploaded reference photo as the frame
		} else {
			frame = &img
			if path, e := m.store.SaveAsset(productID, jobID+".frame.png", img.Data); e == nil {
				m.setJob(productID, jobID, func(j *model.Job) { j.ImagePath = path })
			}
		}
	}

	// Step 2 — animate the frame into video with Veo.
	opName, err := m.gen.StartVideo(ctx, videoPrompt, frame, durationSeconds)
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
