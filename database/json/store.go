package json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// file is the on-disk shape: one file holds the whole database, keyed by table
// name and then by primary key.
//
//	{"users": {"1": {...}, "2": {...}}, "posts": {"7": {...}}}
//
// Primary keys become object keys, so a lookup by key needs no scan.
type file map[string]map[string]json.RawMessage

// store owns one JSON file. Every mutation rewrites the whole file, so all
// access is serialised and writes go through a temporary file plus rename to
// keep a crash or a full disk from leaving truncated JSON behind.
type store struct {
	mu   sync.Mutex
	path string
	data file

	// transient marks a store owned by an enclosing transaction: it mutates its
	// document in place and never persists, letting the outer update commit once.
	transient bool
}

func newStore(filename string) (*store, error) {
	if filename == "" {
		return nil, fmt.Errorf("json: filename is required")
	}
	s := &store{path: filename}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// load reads the file into memory. A missing file is an empty database, not an
// error, so the first write creates it.
func (s *store) load() error {
	raw, err := os.ReadFile(s.path)
	switch {
	case os.IsNotExist(err):
		s.data = make(file)
		return nil
	case err != nil:
		return fmt.Errorf("json: read %s: %w", s.path, err)
	}

	if len(raw) == 0 {
		s.data = make(file)
		return nil
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		return fmt.Errorf("json: parse %s: %w", s.path, err)
	}
	if s.data == nil {
		s.data = make(file)
	}
	return nil
}

// update applies fn to the whole document and persists the result. fn receives
// the live map; returning an error leaves both the map and the file untouched.
func (s *store) update(fn func(file) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Inside a transaction the document is already a private draft owned by the
	// enclosing update, so edit it in place and let that call persist it. Cloning
	// here would strand the changes in a copy nothing else can see.
	if s.transient {
		return fn(s.data)
	}

	// fn works on a scratch copy so a failure cannot leave a half-applied edit
	// in memory.
	draft, err := clone(s.data)
	if err != nil {
		return err
	}
	if err := fn(draft); err != nil {
		return err
	}
	if err := s.write(draft); err != nil {
		return err
	}
	s.data = draft
	return nil
}

// read runs fn against a consistent snapshot. update never mutates the current
// document in place — it works on a clone and swaps the pointer at the end — so
// the snapshot handed out here stays immutable for the duration of fn.
func (s *store) read(fn func(file)) {
	s.mu.Lock()
	snapshot := s.data
	s.mu.Unlock()
	fn(snapshot)
}

func clone(in file) (file, error) {
	out := make(file, len(in))
	for name, rows := range in {
		copied := make(map[string]json.RawMessage, len(rows))
		for k, v := range rows {
			// RawMessage is a byte slice, so the copy must own its bytes.
			copied[k] = append(json.RawMessage(nil), v...)
		}
		out[name] = copied
	}
	return out, nil
}

// write replaces the file atomically: a temporary file in the same directory is
// written and synced, then renamed over the target. Rename is atomic within a
// filesystem, so a reader never observes a partial document.
func (s *store) write(data file) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("json: create %s: %w", dir, err)
	}

	// Marshal with indent: the file is meant to be read and edited by hand.
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("json: encode: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(s.path)+".tmp*")
	if err != nil {
		return fmt.Errorf("json: temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("json: write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("json: sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("json: close: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("json: chmod: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("json: rename: %w", err)
	}
	return nil
}

// tables lists the table names in a stable order, which keeps generated files
// deterministic.
func (f file) tables() []string {
	out := make([]string, 0, len(f))
	for name := range f {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
