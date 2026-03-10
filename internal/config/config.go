package config

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/0x-JP/tmux-todo/internal/fileio"
)

type Data struct {
	Version     int                 `json:"version"`
	Tags        []string            `json:"tags"`
	UI          UIState             `json:"ui,omitempty"`
	Keybindings map[string][]string `json:"keybindings,omitempty"`
}

type UIState struct {
	MainMode string `json:"main_mode,omitempty"`
	Selected struct {
		Scope      string `json:"scope,omitempty"`
		ContextKey string `json:"context_key,omitempty"`
		ID         string `json:"id,omitempty"`
	} `json:"selected,omitempty"`
}

type Store struct {
	mu   sync.Mutex
	path string
	data Data
}

func New(path string, defaultTags []string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(defaultTags); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Keybindings() map[string][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.Keybindings
}

func (s *Store) Tags() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.data.Tags...)
}

func (s *Store) UI() UIState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.UI
}

func (s *Store) SaveUI(ui UIState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.UI = ui
	return s.saveLocked()
}

func (s *Store) AddTag(tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tag = normalize(tag)
	if tag == "" {
		return nil
	}
	set := map[string]struct{}{}
	for _, t := range s.data.Tags {
		set[t] = struct{}{}
	}
	set[tag] = struct{}{}
	s.data.Tags = mapToSorted(set)
	return s.saveLocked()
}

func (s *Store) RemoveTag(tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tag = normalize(tag)
	if tag == "" {
		return nil
	}
	out := make([]string, 0, len(s.data.Tags))
	for _, t := range s.data.Tags {
		if t == tag {
			continue
		}
		out = append(out, t)
	}
	s.data.Tags = out
	return s.saveLocked()
}

func (s *Store) load(defaultTags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = Data{Version: 2, Tags: normalizeAll(defaultTags)}
			return s.saveLocked()
		}
		return err
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	if d.Version == 0 {
		d.Version = 1
	}
	if d.Version < 2 {
		d.Version = 2
	}
	if len(d.Tags) == 0 {
		d.Tags = normalizeAll(defaultTags)
	}
	d.Tags = normalizeAll(d.Tags)
	s.data = d
	return nil
}

func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return fileio.WriteFileAtomic(s.path, b, 0o644)
}

func normalize(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.TrimPrefix(v, "@")
	return v
}

func normalizeAll(in []string) []string {
	set := map[string]struct{}{}
	for _, t := range in {
		if v := normalize(t); v != "" {
			set[v] = struct{}{}
		}
	}
	return mapToSorted(set)
}

func mapToSorted(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
