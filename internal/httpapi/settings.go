package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"studio16/internal/ai/gemini"
	"studio16/internal/ai/openai"
	"studio16/internal/video"
)

// Settings holds runtime-configurable provider settings, editable from the UI
// (the ⚙️ dialog) and persisted to <dataDir>/settings.json. Saved values
// override the .env defaults, so keys can be set without touching .env or
// restarting the server.
type Settings struct {
	OpenAIKey         string `json:"openaiKey"`
	OpenAIModel       string `json:"openaiModel"`
	GeminiKey         string `json:"geminiKey"`
	GeminiVisionModel string `json:"geminiVisionModel"`
	GeminiImageModel  string `json:"geminiImageModel"`
	VeoModel          string `json:"veoModel"`
	MatchProvider     string `json:"matchProvider"`
	MatchThreshold    int    `json:"matchThreshold"`
}

func (s *Server) settingsPath() string {
	return filepath.Join(s.cfg.DataDir, "settings.json")
}

// loadSettings starts from the .env defaults (cfg) and overlays any saved
// settings.json values on top.
func (s *Server) loadSettings() Settings {
	set := Settings{
		OpenAIKey:         s.cfg.OpenAIKey,
		OpenAIModel:       s.cfg.OpenAIModel,
		GeminiKey:         s.cfg.GeminiKey,
		GeminiVisionModel: s.cfg.GeminiVisionModel,
		GeminiImageModel:  s.cfg.GeminiImageModel,
		VeoModel:          s.cfg.VeoModel,
		MatchProvider:     s.cfg.MatchProvider,
		MatchThreshold:    s.cfg.MatchThreshold,
	}
	if b, err := os.ReadFile(s.settingsPath()); err == nil {
		var v Settings
		if json.Unmarshal(b, &v) == nil {
			if v.OpenAIKey != "" {
				set.OpenAIKey = v.OpenAIKey
			}
			if v.OpenAIModel != "" {
				set.OpenAIModel = v.OpenAIModel
			}
			if v.GeminiKey != "" {
				set.GeminiKey = v.GeminiKey
			}
			if v.GeminiVisionModel != "" {
				set.GeminiVisionModel = v.GeminiVisionModel
			}
			if v.GeminiImageModel != "" {
				set.GeminiImageModel = v.GeminiImageModel
			}
			if v.VeoModel != "" {
				set.VeoModel = v.VeoModel
			}
			if v.MatchProvider != "" {
				set.MatchProvider = v.MatchProvider
			}
			if v.MatchThreshold != 0 {
				set.MatchThreshold = v.MatchThreshold
			}
		}
	}
	return set
}

// saveSettings persists the current settings (file mode 0600 — it holds keys).
// Caller holds s.mu.
func (s *Server) saveSettings() error {
	b, _ := json.MarshalIndent(s.settings, "", "  ")
	return os.WriteFile(s.settingsPath(), b, 0o600)
}

// rebuild recreates the provider clients from the current settings.
// Caller holds s.mu.
func (s *Server) rebuild() {
	s.openai = openai.New(s.settings.OpenAIKey, s.settings.OpenAIModel)
	s.gemini = gemini.New(s.settings.GeminiKey, s.settings.GeminiVisionModel, s.settings.GeminiImageModel, s.settings.VeoModel)
	s.vid = video.NewManager(s.gemini, s.store, s.cfg.FFmpegPath)
}

// getSettings returns non-secret settings plus whether each key is configured.
// Keys themselves are never sent back to the browser.
func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"openaiKeySet":      s.settings.OpenAIKey != "",
		"openaiModel":       s.settings.OpenAIModel,
		"geminiKeySet":      s.settings.GeminiKey != "",
		"geminiVisionModel": s.settings.GeminiVisionModel,
		"geminiImageModel":  s.settings.GeminiImageModel,
		"veoModel":          s.settings.VeoModel,
		"matchProvider":     s.settings.MatchProvider,
		"matchThreshold":    s.settings.MatchThreshold,
	})
}

// settingsReq uses pointers so an omitted field is left unchanged; a blank key
// is treated as "keep the existing key".
type settingsReq struct {
	OpenAIKey         *string `json:"openaiKey"`
	OpenAIModel       *string `json:"openaiModel"`
	GeminiKey         *string `json:"geminiKey"`
	GeminiVisionModel *string `json:"geminiVisionModel"`
	GeminiImageModel  *string `json:"geminiImageModel"`
	VeoModel          *string `json:"veoModel"`
	MatchProvider     *string `json:"matchProvider"`
	MatchThreshold    *int    `json:"matchThreshold"`
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	s.mu.Lock()
	if req.OpenAIKey != nil && strings.TrimSpace(*req.OpenAIKey) != "" {
		s.settings.OpenAIKey = strings.TrimSpace(*req.OpenAIKey)
	}
	if req.GeminiKey != nil && strings.TrimSpace(*req.GeminiKey) != "" {
		s.settings.GeminiKey = strings.TrimSpace(*req.GeminiKey)
	}
	if req.OpenAIModel != nil && *req.OpenAIModel != "" {
		s.settings.OpenAIModel = *req.OpenAIModel
	}
	if req.GeminiVisionModel != nil && *req.GeminiVisionModel != "" {
		s.settings.GeminiVisionModel = *req.GeminiVisionModel
	}
	if req.GeminiImageModel != nil && *req.GeminiImageModel != "" {
		s.settings.GeminiImageModel = *req.GeminiImageModel
	}
	if req.VeoModel != nil && *req.VeoModel != "" {
		s.settings.VeoModel = *req.VeoModel
	}
	if req.MatchProvider != nil && *req.MatchProvider != "" {
		s.settings.MatchProvider = *req.MatchProvider
	}
	if req.MatchThreshold != nil && *req.MatchThreshold > 0 {
		s.settings.MatchThreshold = *req.MatchThreshold
	}
	s.rebuild()
	err := s.saveSettings()
	s.mu.Unlock()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "บันทึกไม่สำเร็จ: "+err.Error())
		return
	}
	s.getSettings(w, r)
}
