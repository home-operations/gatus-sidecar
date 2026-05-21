package gatus

import (
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
	// dirty signals that the in-memory state has diverged from the on-disk
	// file (either via an unflushed change or a failed flush). Cleared only
	// when flushLocked succeeds, so a transient write failure is retried on
	// the next flush even when the endpoint itself didn't change.
	dirty bool
}

func NewWriter(path string) *Writer {
	return &Writer{
		path:      path,
		endpoints: make(map[string]*Endpoint),
	}
}

// Upsert stores e under key. The bool reports whether the stored value
// changed. The file is rewritten when flush is true and either this call
// changed something or a previous flush failed.
func (w *Writer) Upsert(key string, e *Endpoint, flush bool) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	changed := false
	if existing, ok := w.endpoints[key]; !ok || !reflect.DeepEqual(existing, e) {
		w.endpoints[key] = e
		w.dirty = true
		changed = true
	}
	if flush && w.dirty {
		if err := w.flushLocked(); err != nil {
			return changed, err
		}
	}
	return changed, nil
}

// Delete drops the endpoint stored under key. The bool reports whether a
// deletion occurred. The file is rewritten when flush is true and either
// this call removed something or a previous flush failed.
func (w *Writer) Delete(key string, flush bool) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	removed := false
	if _, ok := w.endpoints[key]; ok {
		delete(w.endpoints, key)
		w.dirty = true
		removed = true
	}
	if flush && w.dirty {
		if err := w.flushLocked(); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

// Flush forces the current state to disk.
func (w *Writer) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flushLocked()
}

func (w *Writer) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.endpoints)
}

func (w *Writer) flushLocked() error {
	endpoints := slices.Collect(maps.Values(w.endpoints))
	slices.SortFunc(endpoints, func(a, b *Endpoint) int { return cmp.Compare(a.Name, b.Name) })

	data, err := yaml.Marshal(map[string]any{"endpoints": endpoints})
	if err != nil {
		return fmt.Errorf("marshal endpoints: %w", err)
	}
	if err := writeAtomic(w.path, data, 0o644); err != nil {
		return err
	}
	w.dirty = false
	return nil
}

// writeAtomic writes data via tempfile+rename so a concurrent reader (Gatus)
// never observes a partial file.
func writeAtomic(path string, data []byte, mode os.FileMode) (retErr error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".gatus-sidecar-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if retErr != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}
