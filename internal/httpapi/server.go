// Package httpapi wires the store, providers, video manager and match scorer
// into a REST API over net/http (Go 1.22 method+pattern routing).
package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"studio16/internal/ai"
	"studio16/internal/ai/gemini"
	"studio16/internal/ai/openai"
	"studio16/internal/config"
	"studio16/internal/match"
	"studio16/internal/model"
	"studio16/internal/prompt"
	"studio16/internal/store"
	"studio16/internal/video"
)

type Server struct {
	cfg      config.Config
	store    *store.Store
	mu       sync.RWMutex // guards settings + the provider clients below
	settings Settings
	openai   *openai.Client
	gemini   *gemini.Client
	vid      *video.Manager
}

func New(cfg config.Config, st *store.Store) *Server {
	s := &Server{cfg: cfg, store: st}
	s.settings = s.loadSettings()
	s.rebuild()
	return s
}

// analyzer picks the vision provider by name, defaulting to OpenAI.
func (s *Server) analyzer(name string) ai.Analyzer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if strings.ToLower(name) == "gemini" {
		return s.gemini
	}
	return s.openai
}

func (s *Server) curVid() *video.Manager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vid
}

func (s *Server) geminiConfigured() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.GeminiKey != ""
}

func (s *Server) matchCfg() (provider string, threshold int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.MatchProvider, s.settings.MatchThreshold
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("POST /api/settings", s.updateSettings)
	mux.HandleFunc("GET /api/base-presets", s.listBasePresets)
	mux.HandleFunc("GET /api/products", s.listProducts)
	mux.HandleFunc("POST /api/products", s.createProduct)
	mux.HandleFunc("GET /api/products/{id}", s.getProduct)
	mux.HandleFunc("PATCH /api/products/{id}", s.patchProduct)
	mux.HandleFunc("DELETE /api/products/{id}", s.deleteProduct)

	mux.HandleFunc("POST /api/products/{id}/images", s.addImage)
	mux.HandleFunc("DELETE /api/products/{id}/images/{imgId}", s.deleteImage)

	mux.HandleFunc("POST /api/products/{id}/analyze", s.analyze)
	mux.HandleFunc("POST /api/products/{id}/script", s.makeScript)
	mux.HandleFunc("GET /api/products/{id}/prompt", s.buildPrompt)
	mux.HandleFunc("GET /api/products/{id}/scenes", s.buildScenes)
	mux.HandleFunc("GET /api/category-prompts", s.listCategoryPrompts)
	mux.HandleFunc("POST /api/category-prompts", s.saveCategoryPrompt)
	mux.HandleFunc("POST /api/products/{id}/generate", s.generate)
	mux.HandleFunc("POST /api/products/{id}/jobs/{jobId}/regen", s.regenJob)
	mux.HandleFunc("POST /api/products/{id}/render-videos", s.renderVideos)
	mux.HandleFunc("GET /api/products/{id}/report", s.getReport)
	mux.HandleFunc("POST /api/products/{id}/report", s.runReport)

	// Serve produced/uploaded assets.
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(s.store.AssetDir()))))

	// Serve the static web UI at the root (index.html and friends). More specific
	// /api/ and /assets/ patterns above take precedence in Go 1.22 routing.
	// no-cache so the browser always picks up UI updates (no stale index.html).
	mux.Handle("GET /", noCache(http.FileServer(http.Dir(s.cfg.WebDir))))

	return withCORS(mux)
}

// ================= helpers =================

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loadImages(p *model.Product, cap int) ([]ai.Image, error) {
	var out []ai.Image
	for _, im := range p.Images {
		if cap > 0 && len(out) >= cap {
			break
		}
		b, err := os.ReadFile(s.store.AbsPath(im.Path))
		if err != nil {
			return nil, err
		}
		mime := im.Mime
		if mime == "" {
			mime = "image/jpeg"
		}
		out = append(out, ai.Image{Mime: mime, Data: b})
	}
	return out, nil
}

// ================= handlers =================

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	resp := map[string]any{
		"ok":               true,
		"openaiConfigured": s.settings.OpenAIKey != "",
		"geminiConfigured": s.settings.GeminiKey != "",
		"veoModel":         s.settings.VeoModel,
		"matchThreshold":   s.settings.MatchThreshold,
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.List())
}

// listCategoryPrompts returns every product-review category with its (possibly
// edited) prompt, for the คลังพรอมต์ page's top-menu library.
func (s *Server) listCategoryPrompts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"categories": prompt.CategoryPromptList(s.store.CategoryPromptOverrides()),
	})
}

type categoryPromptReq struct {
	Category string `json:"category"`
	Prompt   string `json:"prompt"` // empty resets to the built-in default
}

// saveCategoryPrompt stores an edited prompt for one category (empty = reset).
func (s *Server) saveCategoryPrompt(w http.ResponseWriter, r *http.Request) {
	var req categoryPromptReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Category) == "" {
		writeErr(w, http.StatusBadRequest, "ต้องระบุ category")
		return
	}
	if err := s.store.SetCategoryPrompt(req.Category, strings.TrimSpace(req.Prompt)); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// listBasePresets returns the selectable base-prompt sets (full editable text),
// so the UI can offer a picker and pre-fill the editor.
func (s *Server) listBasePresets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, prompt.BasePresets)
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	p := model.Blank()
	// Optional overrides from the body.
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(p)
	}
	p.ID = store.NewID("p")
	if err := s.store.Create(p); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) patchProduct(w http.ResponseWriter, r *http.Request) {
	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	p, err := s.store.Update(r.PathValue("id"), func(p *model.Product) error {
		// Merge by re-marshalling the product, overlaying the patch keys, unmarshalling back.
		cur, _ := json.Marshal(p)
		var merged map[string]json.RawMessage
		_ = json.Unmarshal(cur, &merged)
		for k, v := range patch {
			merged[k] = v
		}
		out, _ := json.Marshal(merged)
		return json.Unmarshal(out, p)
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) deleteProduct(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Delete(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

type addImageReq struct {
	DataURL string `json:"dataUrl"` // e.g. data:image/jpeg;base64,....
	Data    string `json:"data"`    // raw base64
	Mime    string `json:"mime"`
}

func (s *Server) addImage(w http.ResponseWriter, r *http.Request) {
	var req addImageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	b64 := req.Data
	mime := req.Mime
	if req.DataURL != "" {
		if i := strings.Index(req.DataURL, ","); i >= 0 {
			meta := req.DataURL[:i]
			b64 = req.DataURL[i+1:]
			if strings.HasPrefix(meta, "data:") {
				if semi := strings.Index(meta, ";"); semi > 5 {
					mime = meta[5:semi]
				}
			}
		}
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil || len(data) == 0 {
		writeErr(w, http.StatusBadRequest, "invalid image data")
		return
	}
	if mime == "" {
		mime = "image/jpeg"
	}
	id := r.PathValue("id")
	imgID := store.NewID("img")
	ext := ".jpg"
	if strings.Contains(mime, "png") {
		ext = ".png"
	}
	path, err := s.store.SaveAsset(id, imgID+ext, data)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	p, err := s.store.Update(id, func(p *model.Product) error {
		p.Images = append(p.Images, model.Image{ID: imgID, Path: path, Mime: mime})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) deleteImage(w http.ResponseWriter, r *http.Request) {
	imgID := r.PathValue("imgId")
	p, err := s.store.Update(r.PathValue("id"), func(p *model.Product) error {
		kept := p.Images[:0]
		for _, im := range p.Images {
			if im.ID == imgID {
				_ = os.Remove(s.store.AbsPath(im.Path))
				continue
			}
			kept = append(kept, im)
		}
		p.Images = kept
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type analyzeReq struct {
	Focus string `json:"focus"`
}

func (s *Server) analyze(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.store.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if len(p.Images) == 0 {
		writeErr(w, http.StatusBadRequest, "อัปรูปสินค้าก่อนอย่างน้อย 1 รูป")
		return
	}
	var req analyzeReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	imgs, err := s.loadImages(p, 5)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 150*time.Second)
	defer cancel()

	jsonStr, err := s.analyzer(r.URL.Query().Get("provider")).AnalyzePhotos(ctx, imgs, p.Desc, req.Focus)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}

	// Overlay the returned fields (typeTh, type, spec, garment, heroColor) onto the product.
	updated, err := s.store.Update(id, func(p *model.Product) error {
		var got map[string]json.RawMessage
		if err := json.Unmarshal([]byte(jsonStr), &got); err != nil {
			return err
		}
		cur, _ := json.Marshal(p)
		var merged map[string]json.RawMessage
		_ = json.Unmarshal(cur, &merged)
		for _, k := range []string{"typeTh", "type", "spec", "garment", "heroColor"} {
			if v, ok := got[k]; ok {
				merged[k] = v
			}
		}
		out, _ := json.Marshal(merged)
		return json.Unmarshal(out, p)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// makeScript uses a text model to write the Thai spoken lines and saves them
// onto the product (Scripts[0]).
func (s *Server) makeScript(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.store.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	out, err := s.analyzer(r.URL.Query().Get("provider")).GenerateText(ctx, prompt.ScriptPrompt(*p))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	js, err := prompt.ExtractJSON(out)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	var got struct {
		Lines []string `json:"lines"`
	}
	if json.Unmarshal([]byte(js), &got) != nil || len(got.Lines) == 0 {
		writeErr(w, http.StatusBadGateway, "บทที่ได้ไม่ถูกต้อง")
		return
	}
	lines := got.Lines
	for len(lines) < 4 {
		lines = append(lines, "")
	}
	lines = lines[:4]

	updated, err := s.store.Update(id, func(p *model.Product) error {
		p.Scripts = []model.Script{{Lines: lines}}
		p.Pick = 0
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) buildPrompt(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	// Optional per-request overrides without persisting.
	pp := *p
	if f := r.URL.Query().Get("format"); f != "" {
		pp.Format = f
	}
	if a := r.URL.Query().Get("audio"); a != "" {
		pp.AudioMode = a
	}
	if b := r.URL.Query().Get("base"); b != "" {
		pp.BasePresetID = b
		pp.BaseCustom = nil
	}
	// ?veo=1 returns the compact single-video prompt used for the direct Veo API
	// call; the default is the full agent pipeline prompt (for copy into an agent).
	var text string
	if r.URL.Query().Get("img") == "1" {
		sc, _ := strconv.Atoi(r.URL.Query().Get("scene"))
		text = prompt.BuildVeoImage(pp, prompt.VeoOpts{Scene: sc})
	} else if r.URL.Query().Get("veo") == "1" {
		sc, _ := strconv.Atoi(r.URL.Query().Get("scene"))
		text = prompt.BuildVeo(pp, prompt.VeoOpts{Scene: sc})
	} else {
		text = prompt.Build(pp)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"prompt": text,
		"risks":  prompt.ScanRisk(text),
	})
}

// buildScenes returns one prompt PER shot (image + video), planned as a connected
// story exactly like generate. The UI shows these so the user can see (and edit)
// each scene separately before generating — each shot is a different scene.
func (s *Server) buildScenes(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	pp := *p
	if f := r.URL.Query().Get("format"); f != "" {
		pp.Format = f
	}
	if a := r.URL.Query().Get("audio"); a != "" {
		pp.AudioMode = a
	}
	shots, _ := strconv.Atoi(r.URL.Query().Get("shots"))
	if shots < 1 {
		shots = 1
	}
	if shots > 4 {
		shots = 4
	}
	beats := prompt.PlanBeats(pp, shots)
	scenes := make([]map[string]any, 0, len(beats))
	for i, beat := range beats {
		o := prompt.VeoOpts{Line: beat.Line, Role: beat.Role, Part: i + 1, Total: len(beats), Scene: i}
		img := prompt.BuildVeoImage(pp, o)
		scenes = append(scenes, map[string]any{
			"scene":       i,
			"label":       prompt.SceneLabel(pp.Format, i),
			"role":        beat.Role,
			"line":        beat.Line,
			"imagePrompt": img,
			"videoPrompt": prompt.BuildVeo(pp, o),
			"risks":       prompt.ScanRisk(img),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"scenes": scenes})
}

type sceneReq struct {
	ImagePrompt string `json:"imagePrompt"`
	VideoPrompt string `json:"videoPrompt"`
}

type generateReq struct {
	Format          string     `json:"format"`
	Audio           string     `json:"audio"`
	DurationSeconds int        `json:"durationSeconds"` // seconds per shot (Veo clamps to 4..8)
	Shots           int        `json:"shots"`           // how many clips to generate this batch
	Scenes          []sceneReq `json:"scenes"`          // optional per-shot prompt overrides (from the UI)
}

func (s *Server) generate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.store.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !s.geminiConfigured() {
		writeErr(w, http.StatusBadRequest, "ยังไม่ได้ตั้งค่า Gemini API key (จำเป็นสำหรับ Veo) — ตั้งได้ที่ปุ่ม ⚙️")
		return
	}
	var req generateReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	pp := *p
	if req.Format != "" {
		pp.Format = req.Format
	}
	if req.Audio != "" {
		pp.AudioMode = req.Audio
	}
	refs, _ := s.loadImages(p, 3)
	var firstFrame *ai.Image
	if len(refs) > 0 {
		firstFrame = &refs[0]
	}

	// Clamp options: seconds per shot 4..8 (default 8), shots 1..4 (default 1).
	dur := req.DurationSeconds
	if dur <= 0 {
		dur = 8
	}
	if dur < 4 {
		dur = 4
	}
	if dur > 8 {
		dur = 8
	}
	shots := req.Shots
	if len(req.Scenes) > 0 {
		shots = len(req.Scenes) // the UI sent one prompt per shot
	}
	if shots < 1 {
		shots = 1
	}
	if shots > 4 {
		shots = 4
	}

	// Pre-video match gate is ON: each shot's generated image must score >= 65 vs
	// the product photos before its video is rendered. Below that, makeImage refines
	// the image (fixing the reported problems) up to 3 times, then renders the video.
	const gateThreshold = 65
	provider, _ := s.matchCfg()
	spec := match.SpecText(p)

	// Plan the shots as a connected story (hook → body → close). Each shot gets
	// its own line + role (story) and its own scene index (a different exercise
	// for hyrox). Uploaded photos also gate the image: mismatch → no video.
	beats := prompt.PlanBeats(pp, shots)
	reqs := make([]video.Request, 0, len(beats))
	for i, beat := range beats {
		o := prompt.VeoOpts{Line: beat.Line, Role: beat.Role, Part: i + 1, Total: len(beats), Scene: i}
		imgPrompt := prompt.BuildVeoImage(pp, o)
		vidPrompt := prompt.BuildVeo(pp, o)
		// Honor per-shot prompts edited in the UI (fall back to the built ones).
		if i < len(req.Scenes) {
			if v := strings.TrimSpace(req.Scenes[i].ImagePrompt); v != "" {
				imgPrompt = req.Scenes[i].ImagePrompt
			}
			if v := strings.TrimSpace(req.Scenes[i].VideoPrompt); v != "" {
				vidPrompt = req.Scenes[i].VideoPrompt
			}
		}
		reqs = append(reqs, video.Request{
			Format:          pp.Format,
			AudioMode:       pp.AudioMode,
			VideoPrompt:     vidPrompt,
			ImagePrompt:     imgPrompt,
			Refs:            refs,
			FirstFrame:      firstFrame,
			DurationSeconds: dur,
			Scene:           i,
			Checker:         s.analyzer(provider), // gate on: verify image matches product
			Threshold:       gateThreshold,
			SpecText:        spec,
		})
	}
	jobs, err := s.curVid().StartBatch(id, reqs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, jobs)
}

// regenJob re-runs an existing shot (after a mismatch): it regenerates the image
// (with the same scene/prompt) and, if it now matches, continues to the video.
func (s *Server) regenJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	jobID := r.PathValue("jobId")
	p, err := s.store.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	var job *model.Job
	for i := range p.Jobs {
		if p.Jobs[i].ID == jobID {
			job = &p.Jobs[i]
			break
		}
	}
	if job == nil {
		writeErr(w, http.StatusNotFound, "job not found")
		return
	}
	refs, _ := s.loadImages(p, 3)
	var firstFrame *ai.Image
	if len(refs) > 0 {
		firstFrame = &refs[0]
	}
	provider, threshold := s.matchCfg()
	s.curVid().Rerun(id, jobID, video.Request{
		Format:          job.Format,
		AudioMode:       job.AudioMode,
		VideoPrompt:     job.Prompt,
		ImagePrompt:     job.ImagePrompt,
		Refs:            refs,
		FirstFrame:      firstFrame,
		DurationSeconds: job.DurationSeconds,
		Scene:           job.Scene,
		Checker:         s.analyzer(provider),
		Threshold:       threshold,
		SpecText:        match.SpecText(p),
	})
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "regenerating"})
}

type renderVideosReq struct {
	JobIDs []string `json:"jobIds"` // empty = every eligible image in the library
}

// renderVideos is the recovery action: for images that already passed the gate but
// never got a video (Veo quota hit, a crash, a timeout), render the video now from
// the stored frame + the job's own video prompt. It skips images that already have
// a video and anything still in progress.
func (s *Server) renderVideos(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.store.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !s.geminiConfigured() {
		writeErr(w, http.StatusBadRequest, "ยังไม่ได้ตั้งค่า Gemini API key (จำเป็นสำหรับ Veo) — ตั้งได้ที่ปุ่ม ⚙️")
		return
	}
	var req renderVideosReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	pick := map[string]bool{}
	for _, jid := range req.JobIDs {
		pick[jid] = true
	}

	const libThreshold = 60 // library shows/renders images scoring >= 60
	started := 0
	for _, j := range p.Jobs {
		if j.ImagePath == "" || j.VideoPath != "" {
			continue // no image, or already has a video
		}
		if j.MatchScore < libThreshold {
			continue // below the library bar
		}
		if j.Status == "queued" || j.Status == "running" || j.Status == "image" || j.Status == "checking" {
			continue // already working on it
		}
		if len(pick) > 0 && !pick[j.ID] {
			continue // a selection was given and this isn't in it
		}
		s.curVid().RenderExisting(id, j.ID)
		started++
	}
	writeJSON(w, http.StatusAccepted, map[string]int{"started": started})
}

func (s *Server) getReport(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if p.Report == nil {
		writeErr(w, http.StatusNotFound, "ยังไม่มีรีพอท — สั่งตรวจก่อน")
		return
	}
	writeJSON(w, http.StatusOK, p.Report)
}

func (s *Server) runReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.store.Get(id); err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	provider, threshold := s.matchCfg()
	scorer := match.NewScorer(s.analyzer(provider), s.store, threshold, s.cfg.FFmpegPath)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	report, err := scorer.Run(ctx, id)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}
