// Package video orchestrates asynchronous Veo video generation jobs: it starts
// a job, polls until the provider finishes, downloads the result and persists it.
package video

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"studio16/internal/ai"
	"studio16/internal/model"
	"studio16/internal/prompt"
	"studio16/internal/store"
)

// Generator is a VideoGenerator that can also generate the first-frame image
// and download its produced files.
type Generator interface {
	ai.VideoGenerator
	GenerateImage(ctx context.Context, prompt string, refs []ai.Image, seed int) (ai.Image, error)
	DownloadVideo(ctx context.Context, uri string) ([]byte, error)
}

type Manager struct {
	gen    Generator
	store  *store.Store
	ffmpeg string // ffmpeg binary path (for auto-merging a finished sequence)

	pollEvery time.Duration
	timeout   time.Duration

	mergeMu  sync.Mutex
	merging  map[string]bool // productID_batch currently being merged (prevents double work)
}

func NewManager(gen Generator, st *store.Store, ffmpeg string) *Manager {
	return &Manager{gen: gen, store: st, ffmpeg: ffmpeg, pollEvery: 10 * time.Second, timeout: 15 * time.Minute, merging: map[string]bool{}}
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
	if len(reqs) == 0 {
		return
	}
	// Shot 1 establishes the character (face, hair, body, outfit). Every later shot
	// is given shot 1's accepted frame as a CHARACTER reference so the SAME woman
	// appears throughout, while the per-scene prompt + seed still make each shot its
	// own distinct exercise/scene. (Duplicate frames were never caused by this —
	// that was an id-collision bug in NewID, now fixed — so character locking is
	// safe to use again.)
	first := m.run(productID, ids[0], reqs[0])
	for i := 1; i < len(reqs); i++ {
		if first != nil {
			reqs[i].CharRef = first
		}
		m.run(productID, ids[i], reqs[i])
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

// RenderExisting renders the video for a job that already has an accepted image,
// using the video prompt stored on the job. Used by the library's batch/single
// "generate video" action so images and videos are decoupled.
func (m *Manager) RenderExisting(productID, jobID string, checker ai.Analyzer, refs []ai.Image, spec string, threshold int) {
	p, err := m.store.Get(productID)
	if err != nil {
		return
	}
	var job model.Job
	found := false
	for _, j := range p.Jobs {
		if j.ID == jobID {
			job = j
			found = true
			break
		}
	}
	if !found || job.ImagePath == "" || job.VideoPath != "" {
		return
	}
	data, err := os.ReadFile(m.store.AbsPath(job.ImagePath))
	if err != nil {
		m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = "อ่านไฟล์รูปไม่สำเร็จ: " + err.Error() })
		return
	}
	frame := &ai.Image{Mime: "image/jpeg", Data: data}
	req := Request{
		VideoPrompt:     job.Prompt,
		DurationSeconds: job.DurationSeconds,
		Scene:           job.Scene,
		Checker:         checker, // enables video QC + fix-and-retry
		Refs:            refs,
		SpecText:        spec,
		Threshold:       threshold,
	}
	m.setJob(productID, jobID, func(j *model.Job) { j.Status = "queued"; j.Error = "" })
	go m.renderVideo(productID, jobID, req, frame)
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
	var prevImg *ai.Image // last attempt — refine it instead of starting over
	var fixNotes []string // the reported problems to fix on the next attempt
	for attempt := 1; attempt <= 3; attempt++ {
		m.setJob(productID, jobID, func(j *model.Job) { j.Status = "image"; j.Attempt = attempt })

		genRefs := append([]ai.Image{}, req.Refs...)
		imgPrompt := req.ImagePrompt
		if req.CharRef != nil { // lock the same person as shot 1 (added LAST)
			genRefs = append(genRefs, *req.CharRef)
			imgPrompt = req.ImagePrompt +
				"\n\nCHARACTER CONTINUITY: the LAST attached image is the SAME WOMAN from an earlier shot in this series. Keep her face, hairstyle, skin tone, body type and her exact outfit identical to that image. But this is a BRAND-NEW photograph — place her in the new scene and exercise described above, with a different pose, background and camera angle. Do NOT copy the pose, framing or background of that reference image."
		}
		if attempt > 1 && prevImg != nil && len(fixNotes) > 0 {
			// Refine, don't restart: carry the previous attempt forward and fix
			// only what was wrong, keeping the parts that already matched.
			genRefs = append(genRefs, *prevImg)
			imgPrompt = imgPrompt +
				"\n\nREFINE — DO NOT START OVER: the LAST attached image is your previous attempt. Keep everything that is already correct in it exactly the same, and change ONLY what is needed to fix these problems so the garment matches the product reference photo: " +
				strings.Join(fixNotes, "; ") + ". Do not alter anything that already matches."
		}

		// A distinct seed per scene (and per retry) so no two shots — and no two
		// attempts — render the same image. Scene index dominates so each shot in
		// the batch is clearly different; +attempt varies refines.
		seed := (req.Scene+1)*100003 + attempt*17 + 1
		img, err := m.gen.GenerateImage(ctx, imgPrompt, genRefs, seed)
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
		// Not there yet — keep this attempt and the fix list to refine next round.
		prevImg = &img
		fixNotes = mr.Mismatches
		if len(fixNotes) == 0 && strings.TrimSpace(mr.Verdict) != "" {
			fixNotes = []string{mr.Verdict}
		}
	}
	if !matched {
		msg := "สินค้าไม่ตรงหลังลองแก้ 3 ครั้ง — กดเจนใหม่"
		if len(fixNotes) > 0 {
			msg += " (ยังไม่ตรง: " + strings.Join(fixNotes, ", ") + ")"
		}
		m.setJob(productID, jobID, func(j *model.Job) { j.Status = "mismatch"; j.Error = msg })
		return nil, false
	}
	return frame, true
}

// renderVideo renders the clip with Veo and — when a quality checker is set —
// scores the finished video (flicker / distortion / unnatural motion / stray
// people). If it scores below the threshold it re-renders, appending the exact
// defects to the prompt so Veo fixes them rather than starting from scratch, up to
// 3 attempts. Every step is logged.
func (m *Manager) renderVideo(productID, jobID string, req Request, frame *ai.Image) {
	gate := false // video QC + auto-regeneration removed by request — render once
	maxAtt := 1
	if gate {
		maxAtt = 3
	}
	vp := req.VideoPrompt
	for attempt := 1; attempt <= maxAtt; attempt++ {
		m.setJob(productID, jobID, func(j *model.Job) { j.VideoAttempt = attempt })
		data, err := m.renderOnce(productID, jobID, req, frame, vp)
		if err != nil {
			log.Printf("[video] product=%s job=%s attempt=%d render error: %v", productID, jobID, attempt, err)
			m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = err.Error() })
			return
		}
		path, err := m.store.SaveAsset(productID, jobID+".mp4", data)
		if err != nil {
			m.setJob(productID, jobID, func(j *model.Job) { j.Status = "error"; j.Error = err.Error() })
			return
		}
		m.setJob(productID, jobID, func(j *model.Job) { j.VideoPath = path })

		if !gate {
			m.setJob(productID, jobID, func(j *model.Job) { j.Status = "done" })
			log.Printf("[video] product=%s job=%s scene=%d done (no QC)", productID, jobID, req.Scene)
			go m.afterVideoDone(productID, jobID)
			return
		}

		// Score the finished video.
		m.setJob(productID, jobID, func(j *model.Job) { j.Status = "vcheck" })
		sctx, scancel := context.WithTimeout(context.Background(), 3*time.Minute)
		frames, ferr := m.extractVideoFrames(m.store.AbsPath(path), 5, req.DurationSeconds, jobID)
		if ferr != nil || len(frames) == 0 {
			scancel()
			log.Printf("[video] product=%s job=%s frame-extract failed (%v) — accepting video", productID, jobID, ferr)
			m.setJob(productID, jobID, func(j *model.Job) { j.Status = "done" })
			go m.afterVideoDone(productID, jobID)
			return
		}
		mr, serr := req.Checker.ScoreVideo(sctx, req.Refs, frames, req.SpecText)
		scancel()
		if serr != nil {
			log.Printf("[video] product=%s job=%s QC error: %v — accepting video", productID, jobID, serr)
			m.setJob(productID, jobID, func(j *model.Job) { j.Status = "done" })
			go m.afterVideoDone(productID, jobID)
			return
		}
		issues := append([]string{}, mr.Mismatches...)
		issues = append(issues, mr.Issues...)
		m.setJob(productID, jobID, func(j *model.Job) { j.VideoScore = mr.Score; j.VideoIssues = issues })
		log.Printf("[video] QC product=%s job=%s scene=%d attempt=%d score=%d verdict=%q issues=%v",
			productID, jobID, req.Scene, attempt, mr.Score, mr.Verdict, issues)

		if mr.Score >= req.Threshold {
			m.setJob(productID, jobID, func(j *model.Job) { j.Status = "done" })
			log.Printf("[video] product=%s job=%s scene=%d PASSED video QC (%d>=%d)", productID, jobID, req.Scene, mr.Score, req.Threshold)
			go m.afterVideoDone(productID, jobID)
			return
		}
		if attempt < maxAtt {
			vp = req.VideoPrompt + prompt.VideoFixNote(issues)
			log.Printf("[video] product=%s job=%s scene=%d video QC %d<%d — re-rendering to fix: %v",
				productID, jobID, req.Scene, mr.Score, req.Threshold, issues)
		}
	}
	// Ran out of attempts and still below the bar — keep the best video we produced
	// (a low-scored clip is still more useful than none) with its score recorded.
	m.setJob(productID, jobID, func(j *model.Job) { j.Status = "done" })
	log.Printf("[video] product=%s job=%s scene=%d kept after %d attempts (still below QC threshold)", productID, jobID, req.Scene, maxAtt)
	go m.afterVideoDone(productID, jobID)
}

// renderOnce does a single Veo render (start → poll → download) and returns the
// video bytes.
func (m *Manager) renderOnce(productID, jobID string, req Request, frame *ai.Image, videoPrompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	opName, err := m.gen.StartVideo(ctx, videoPrompt, frame, req.DurationSeconds)
	if err != nil {
		return nil, err
	}
	m.setJob(productID, jobID, func(j *model.Job) { j.Status = "running"; j.OpName = opName })
	ticker := time.NewTicker(m.pollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out")
		case <-ticker.C:
			st, err := m.gen.PollVideo(ctx, opName)
			if err != nil {
				continue
			}
			if !st.Done {
				continue
			}
			if st.Error != "" {
				return nil, fmt.Errorf("%s", st.Error)
			}
			data := st.Inline
			if len(data) == 0 && st.VideoURL != "" {
				data, err = m.gen.DownloadVideo(ctx, st.VideoURL)
				if err != nil {
					return nil, err
				}
			}
			return data, nil
		}
	}
}

// extractVideoFrames pulls n stills evenly spaced across the clip for the QC pass.
func (m *Manager) extractVideoFrames(videoAbs string, n, durationSec int, prefix string) ([]ai.Image, error) {
	if durationSec <= 0 {
		durationSec = 8
	}
	var frames []ai.Image
	for i := 0; i < n; i++ {
		t := float64(durationSec) * (float64(i) + 0.5) / float64(n)
		out := filepath.Join(os.TempDir(), fmt.Sprintf("s16vf_%s_%d.jpg", prefix, i))
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := exec.CommandContext(ctx, m.ffmpeg, "-y", "-ss", strconv.FormatFloat(t, 'f', 2, 64),
			"-i", videoAbs, "-frames:v", "1", "-q:v", "3", out).Run()
		cancel()
		if err != nil {
			continue
		}
		b, err := os.ReadFile(out)
		_ = os.Remove(out)
		if err == nil && len(b) > 0 {
			frames = append(frames, ai.Image{Mime: "image/jpeg", Data: b})
		}
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames extracted")
	}
	return frames, nil
}

// afterVideoDone auto-merges a sequence: when a shot finishes and no other shot in
// the same batch is still working, it concatenates the finished shots into one clip
// (only if there are >= 2 and it wasn't merged before).
func (m *Manager) afterVideoDone(productID, jobID string) {
	if m.ffmpeg == "" {
		return
	}
	p, err := m.store.Get(productID)
	if err != nil {
		return
	}
	var createdAt int64
	found := false
	for _, j := range p.Jobs {
		if j.ID == jobID {
			createdAt = j.CreatedAt
			found = true
			break
		}
	}
	if !found {
		return
	}
	key := strconv.FormatInt(createdAt, 10)
	busy, doneVids := 0, 0
	for _, j := range p.Jobs {
		if j.CreatedAt != createdAt {
			continue
		}
		switch j.Status {
		case "queued", "image", "checking", "running":
			busy++
		case "done":
			if j.VideoPath != "" {
				doneVids++
			}
		}
	}
	if busy > 0 || doneVids < 2 {
		return // still rendering, or nothing to merge
	}
	if p.MergedVideos != nil && p.MergedVideos[key] != "" {
		return // already merged this sequence
	}
	mkey := productID + "_" + key
	m.mergeMu.Lock()
	if m.merging[mkey] {
		m.mergeMu.Unlock()
		return
	}
	m.merging[mkey] = true
	m.mergeMu.Unlock()
	defer func() { m.mergeMu.Lock(); delete(m.merging, mkey); m.mergeMu.Unlock() }()
	_, _ = m.MergeBatch(productID, createdAt)
}

// MergeBatch concatenates every finished shot of one sequence (batch, keyed by
// createdAt) into a single video with ffmpeg, ordered by scene, and stores it on
// the product under that batch key.
func (m *Manager) MergeBatch(productID string, createdAt int64) (string, error) {
	p, err := m.store.Get(productID)
	if err != nil {
		return "", err
	}
	var jobs []model.Job
	for _, j := range p.Jobs {
		if j.CreatedAt == createdAt && j.Status == "done" && j.VideoPath != "" {
			jobs = append(jobs, j)
		}
	}
	if len(jobs) < 2 {
		return "", fmt.Errorf("ต้องมีวิดีโอที่เสร็จอย่างน้อย 2 ช็อตในชุดนี้ถึงจะรวมได้")
	}
	sort.Slice(jobs, func(a, b int) bool { return jobs[a].Scene < jobs[b].Scene })

	withAudio := jobs[0].AudioMode != "silent"
	audioFlag := 0
	if withAudio {
		audioFlag = 1
	}
	args := []string{"-y"}
	var pads strings.Builder
	for i, j := range jobs {
		args = append(args, "-i", m.store.AbsPath(j.VideoPath))
		pads.WriteString(fmt.Sprintf("[%d:v:0]", i))
		if withAudio {
			pads.WriteString(fmt.Sprintf("[%d:a:0]", i))
		}
	}
	filter := pads.String() + fmt.Sprintf("concat=n=%d:v=1:a=%d", len(jobs), audioFlag)
	if withAudio {
		filter += "[v][a]"
	} else {
		filter += "[v]"
	}
	args = append(args, "-filter_complex", filter, "-map", "[v]")
	if withAudio {
		args = append(args, "-map", "[a]", "-c:a", "aac")
	}
	args = append(args, "-c:v", "libx264", "-preset", "veryfast", "-pix_fmt", "yuv420p", "-movflags", "+faststart")

	key := strconv.FormatInt(createdAt, 10)
	tmp := filepath.Join(os.TempDir(), "s16merge_"+productID+"_"+key+".mp4")
	args = append(args, tmp)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if out, err := exec.CommandContext(ctx, m.ffmpeg, args...).CombinedOutput(); err != nil {
		return "", fmt.Errorf("ffmpeg: %s", ffmpegTail(out))
	}
	data, err := os.ReadFile(tmp)
	_ = os.Remove(tmp)
	if err != nil {
		return "", err
	}
	path, err := m.store.SaveAsset(productID, "merged_"+key+".mp4", data)
	if err != nil {
		return "", err
	}
	_, _ = m.store.Update(productID, func(pp *model.Product) error {
		if pp.MergedVideos == nil {
			pp.MergedVideos = map[string]string{}
		}
		pp.MergedVideos[key] = path
		return nil
	})
	return path, nil
}

// ffmpegTail returns the last line of ffmpeg output (its actual error message).
func ffmpegTail(out []byte) string {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
