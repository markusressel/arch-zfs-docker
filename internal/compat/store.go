package compat

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	// incompatibleTTL bounds how long a failure blocks automatic retries of the same
	// (kernel, ZFS version) pair; the cause may have been fixed in the packaging.
	incompatibleTTL = 7 * 24 * time.Hour
	maxSeenJobs     = 500
)

// Entry is a learned incompatibility.
type Entry struct {
	Kernel     string    `json:"kernel"` // dotted, e.g. 7.2.9.arch1.1
	ZfsVersion string    `json:"zfsVersion"`
	Reason     string    `json:"reason"`
	Evidence   string    `json:"evidence,omitempty"`
	Job        string    `json:"job"`
	At         time.Time `json:"at"`
}

type storeData struct {
	Entries map[string]Entry     `json:"entries"`
	Seen    map[string]time.Time `json:"seen"` // failed jobs that were already classified
}

// Store persists learned incompatibilities as JSON next to the repository.
type Store struct {
	path string

	mu   sync.Mutex
	data storeData
}

// NewStore loads (or creates) the store at path. Problems only disable persistence.
func NewStore(path string) *Store {
	s := &Store{path: path, data: storeData{Entries: map[string]Entry{}, Seen: map[string]time.Time{}}}
	if b, err := os.ReadFile(path); err == nil {
		var d storeData
		if json.Unmarshal(b, &d) == nil {
			if d.Entries != nil {
				s.data.Entries = d.Entries
			}
			if d.Seen != nil {
				s.data.Seen = d.Seen
			}
		}
	}
	return s
}

func key(kernel, zfs string) string { return kernel + "|" + zfs }

// claim marks a job as being processed; false means it was already handled.
func (s *Store) claim(job string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data.Seen[job]; ok {
		return false
	}
	s.data.Seen[job] = time.Now()
	return true
}

func (s *Store) release(job string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Seen, job)
}

// record stores an incompatibility and persists the store.
func (s *Store) record(e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Entries[key(e.Kernel, e.ZfsVersion)] = e
	s.saveLocked()
}

// persist saves the seen set (called after a job was classified).
func (s *Store) persist() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveLocked()
}

// lookup returns a non-expired incompatibility for the pair.
func (s *Store) lookup(kernel, zfs string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data.Entries[key(kernel, zfs)]
	if !ok || time.Since(e.At) > incompatibleTTL {
		return Entry{}, false
	}
	return e, true
}

func (s *Store) saveLocked() {
	if len(s.data.Seen) > maxSeenJobs {
		type kv struct {
			job string
			at  time.Time
		}
		all := make([]kv, 0, len(s.data.Seen))
		for j, t := range s.data.Seen {
			all = append(all, kv{j, t})
		}
		sort.Slice(all, func(i, j int) bool { return all[i].at.After(all[j].at) })
		for _, old := range all[maxSeenJobs:] {
			delete(s.data.Seen, old.job)
		}
	}
	if s.path == "" {
		return
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return
	}
	tmp := s.path + ".tmp"
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err == nil {
		if err = os.WriteFile(tmp, b, 0o644); err == nil {
			err = os.Rename(tmp, s.path)
		}
		if err != nil {
			log.Printf("[compat] Could not persist %s: %v\n", s.path, err)
		}
	}
}
