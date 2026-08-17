// Package config loads runtime settings from environment variables.
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port    string
	DataDir string // where db.json and assets live
	WebDir  string // static web UI served at "/"

	OpenAIKey   string
	OpenAIModel string // vision/text model, e.g. gpt-4o

	GeminiKey         string
	GeminiVisionModel string // e.g. gemini-2.5-flash
	GeminiImageModel  string // e.g. gemini-2.5-flash-image (Nano Banana)
	VeoModel          string // e.g. veo-3.1-generate-preview

	MatchThreshold int    // 0-100 pass/fail cutoff
	MatchProvider  string // which provider scores the match: openai | gemini
	FFmpegPath     string // path to ffmpeg binary (frame extraction)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func Load() Config {
	return Config{
		Port:    env("PORT", "8080"),
		DataDir: env("DATA_DIR", "./data"),
		WebDir:  env("WEB_DIR", "./web"),

		OpenAIKey:   os.Getenv("OPENAI_API_KEY"),
		OpenAIModel: env("OPENAI_MODEL", "gpt-4o"),

		GeminiKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiVisionModel: env("GEMINI_VISION_MODEL", "gemini-2.5-flash"),
		GeminiImageModel:  env("GEMINI_IMAGE_MODEL", "gemini-2.5-flash-image"),
		VeoModel:          env("VEO_MODEL", "veo-3.1-generate-preview"),

		MatchThreshold: envInt("MATCH_THRESHOLD", 60),
		MatchProvider:  env("MATCH_PROVIDER", "openai"),
		FFmpegPath:     env("FFMPEG_PATH", "ffmpeg"),
	}
}
