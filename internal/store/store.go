package store

import (
	"errors"
	"sync"
)

// ErrExists is returned by Create when a flag with the same key already exists.
var ErrExists = errors.New("flag already exists")

// Flag is a feature flag stored in memory.
type Flag struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent int    `json:"rollout_percent"`
}

// Store is a thread-safe in-memory flag store.
type Store struct {
	mu    sync.RWMutex
	flags map[string]Flag
}

// New returns an empty, ready-to-use Store.
func New() *Store {
	return &Store{flags: make(map[string]Flag)}
}

// Create inserts a new flag. It returns ErrExists if the key already exists.
func (s *Store) Create(f Flag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[f.Key]; ok {
		return ErrExists
	}
	s.flags[f.Key] = f
	return nil
}

// Get returns the flag with the given key and whether it was found.
func (s *Store) Get(key string) (Flag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flags[key]
	return f, ok
}

// List returns all flags. The result is never nil; an empty store yields [].
func (s *Store) List() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		out = append(out, f)
	}
	return out
}

// Update replaces the flag with the given key. It reports whether the key existed.
func (s *Store) Update(key string, f Flag) (Flag, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[key]; !ok {
		return Flag{}, false
	}
	s.flags[key] = f
	return f, true
}

// Delete removes the flag with the given key and reports whether it existed.
func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[key]; !ok {
		return false
	}
	delete(s.flags, key)
	return true
}
