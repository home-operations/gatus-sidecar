package gatus

import (
	"bytes"
	"cmp"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sync"

	"gopkg.in/yaml.v3"
)

// Writer aggregates endpoints and renders them to a YAML file atomically.
// Safe for concurrent use.
type Writer struct {
	path string

	mu        sync.Mutex
	endpoints map[string]*Endpoint
	// dirty signals that the in-memory state may differ from the on-disk
	// file (an unflushed change, a failed flush, or no flush yet). Cleared
	// only when Flush succeeds, so a transient write failure is retried on
	// the next Flush even when no endpoint changed.
	dirty bool
}

func NewWriter(path string) *Writer {
	return &Writer{
		path:      path,
		endpoints: make(map[string]*Endpoint),
		dirty:     true,
	}
}

// Upsert stores e under key and reports whether the stored value changed.
func (w *Writer) Upsert(key string, e *Endpoint) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if existing, ok := w.endpoints[key]; ok && reflect.DeepEqual(existing, e) {
		return false
	}
	w.endpoints[key] = e
	w.dirty = true
	return true
}

// Delete drops the endpoint stored under key and reports whether one existed.
func (w *Writer) Delete(key string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.endpoints[key]; !ok {
		return false
	}
	delete(w.endpoints, key)
	w.dirty = true
	return true
}

func (w *Writer) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.endpoints)
}

// Flush writes the endpoints to disk if they changed since the last
// successful Flush. The first call always writes, so the file exists even
// when there are no endpoints.
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.dirty {
		return nil
	}

	// Stable sort over sorted keys so same-named endpoints (different
	// namespaces or kinds) render in the same order on every flush.
	endpoints := make([]*Endpoint, 0, len(w.endpoints))
	for _, key := range slices.Sorted(maps.Keys(w.endpoints)) {
		endpoints = append(endpoints, w.endpoints[key])
	}
	slices.SortStableFunc(endpoints, func(a, b *Endpoint) int {
		return cmp.Compare(a.Name, b.Name)
	})

	data, err := yaml.Marshal(map[string]any{"endpoints": endpoints})
	if err != nil {
		return fmt.Errorf("gatus: marshal endpoints: %w", err)
	}
	// Gatus reloads on any mtime change, so leave identical content untouched.
	if cur, err := os.ReadFile(w.path); err != nil || !bytes.Equal(cur, data) {
		if err := writeAtomic(w.path, data); err != nil {
			return err
		}
	}
	w.dirty = false
	return nil
}

// writeAtomic writes data via tempfile+rename so a concurrent reader (Gatus)
// never observes a partial file.
func writeAtomic(path string, data []byte) (retErr error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("gatus: create output dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".gatus-sidecar-*.tmp")
	if err != nil {
		return fmt.Errorf("gatus: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if retErr != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("gatus: write temp file: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("gatus: chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("gatus: close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("gatus: rename to %s: %w", path, err)
	}
	return nil
}
