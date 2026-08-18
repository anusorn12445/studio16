// Package store is a small, dependency-free JSON-file-backed product store.
// Images and generated videos are written as files under <dataDir>/assets.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"studio16/internal/model"
)

// idCounter guarantees IDs are unique even when several are minted within the
// same clock tick. On Windows time.Now().UnixNano() has coarse (~1ms+) resolution,
// so a tight loop (e.g. one batch creating several jobs) would otherwise mint the
// SAME id for every job — collapsing them onto one asset file.
var idCounter uint64

var ErrNotFound = errors.New("not found")

type Store struct {
	mu       sync.RWMutex
	dataDir  string
	dbPath   string
	assetDir string
	db       db
}

type db struct {
	Products        []*model.Product  `json:"products"`
	CategoryPrompts map[string]string `json:"categoryPrompts"` // category path -> edited prompt (overrides the default)
}

func Open(dataDir string) (*Store, error) {
	assetDir := filepath.Join(dataDir, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{
		dataDir:  dataDir,
		dbPath:   filepath.Join(dataDir, "db.json"),
		assetDir: assetDir,
	}
	if b, err := os.ReadFile(s.dbPath); err == nil {
		_ = json.Unmarshal(b, &s.db)
	}
	if s.db.Products == nil {
		s.db.Products = []*model.Product{}
	}
	if s.db.CategoryPrompts == nil {
		s.db.CategoryPrompts = map[string]string{}
	}
	return s, nil
}

// CategoryPromptOverrides returns a copy of the saved per-category prompt edits.
func (s *Store) CategoryPromptOverrides() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.db.CategoryPrompts))
	for k, v := range s.db.CategoryPrompts {
		out[k] = v
	}
	return out
}

// SetCategoryPrompt saves an edited prompt for a category. An empty prompt clears
// the override (the category falls back to its built-in default).
func (s *Store) SetCategoryPrompt(category, prompt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db.CategoryPrompts == nil {
		s.db.CategoryPrompts = map[string]string{}
	}
	if prompt == "" {
		delete(s.db.CategoryPrompts, category)
	} else {
		s.db.CategoryPrompts[category] = prompt
	}
	return s.flush()
}

func (s *Store) AssetDir() string { return s.assetDir }

// flush writes the db to disk atomically. Caller holds the write lock.
func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.db, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.dbPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.dbPath)
}

func (s *Store) List() []*model.Product {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.Product, len(s.db.Products))
	copy(out, s.db.Products)
	return out
}

func (s *Store) Get(id string) (*model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.db.Products {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) Create(p *model.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db.Products = append([]*model.Product{p}, s.db.Products...)
	return s.flush()
}

// Update applies fn to the product with the given id under the write lock, then persists.
func (s *Store) Update(id string, fn func(*model.Product) error) (*model.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.db.Products {
		if p.ID == id {
			if err := fn(p); err != nil {
				return nil, err
			}
			if err := s.flush(); err != nil {
				return nil, err
			}
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.db.Products[:0]
	found := false
	for _, p := range s.db.Products {
		if p.ID == id {
			found = true
			continue
		}
		kept = append(kept, p)
	}
	if !found {
		return ErrNotFound
	}
	s.db.Products = kept
	_ = os.RemoveAll(filepath.Join(s.assetDir, id))
	return s.flush()
}

// SaveAsset writes bytes to <assets>/<productID>/<name> and returns the path relative to dataDir.
func (s *Store) SaveAsset(productID, name string, data []byte) (string, error) {
	dir := filepath.Join(s.assetDir, productID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	rel, err := filepath.Rel(s.dataDir, full)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// AbsPath resolves a stored relative asset path to an absolute filesystem path.
func (s *Store) AbsPath(rel string) string {
	return filepath.Join(s.dataDir, filepath.FromSlash(rel))
}

// NewID returns a compact unique id (mirrors the STUDIO 16 uid() shape). The
// atomic counter suffix keeps ids distinct even when many are created in the same
// nanosecond-resolution-limited tick (Windows), so batched jobs never collide.
func NewID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s%d%03d", prefix, time.Now().UnixNano(), n%1000)
}
