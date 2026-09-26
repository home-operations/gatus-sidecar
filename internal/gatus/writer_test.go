package gatus

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWriter_UpsertAndDelete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	w := NewWriter(path)

	if !w.Upsert("k1", &Endpoint{Name: "a", URL: "https://a", Interval: "1m"}) {
		t.Error("first Upsert should report changed=true")
	}
	if w.Upsert("k1", &Endpoint{Name: "a", URL: "https://a", Interval: "1m"}) {
		t.Error("equal Upsert should report changed=false")
	}
	if !w.Upsert("k1", &Endpoint{Name: "a", URL: "https://b", Interval: "1m"}) {
		t.Error("Upsert with new URL should report changed=true")
	}
	if !w.Delete("k1") {
		t.Error("Delete should report removed=true")
	}
	if w.Delete("k1") {
		t.Error("Delete of absent key should report removed=false")
	}
}

func TestWriter_FirstFlushWritesEmptyFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "out.yaml")
	if err := NewWriter(path).Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != "endpoints: []" {
		t.Errorf("empty flush = %q, want %q", got, "endpoints: []")
	}
}

func TestWriter_Flush_SortsAndMatchesYAMLShape(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	w := NewWriter(path)

	endpoints := []*Endpoint{
		{Name: "zebra", URL: "z", Interval: "1m"},
		{Name: "alpha", URL: "a", Interval: "1m", Conditions: []string{"[STATUS] == 200"}},
		{Name: "mid", URL: "m", Interval: "1m"},
	}
	for _, e := range endpoints {
		w.Upsert(e.Name, e)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "alpha") || strings.Index(out, "alpha") > strings.Index(out, "mid") {
		t.Error("alpha should appear before mid")
	}
	if strings.Index(out, "mid") > strings.Index(out, "zebra") {
		t.Error("mid should appear before zebra")
	}

	var doc struct {
		Endpoints []map[string]any `yaml:"endpoints"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("YAML unmarshal: %v", err)
	}
	if len(doc.Endpoints) != 3 {
		t.Errorf("got %d endpoints, want 3", len(doc.Endpoints))
	}
}

func TestWriter_FlushIsAtomic(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	w := NewWriter(path)
	w.Upsert("k", &Endpoint{Name: "a", URL: "x", Interval: "1m"})
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".gatus-sidecar-") {
			t.Errorf("temp file left in output dir: %s", entry.Name())
		}
	}
}

func TestWriter_FlushSkipsIdenticalContent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "out.yaml")
	upsertAll := func(w *Writer) {
		t.Helper()
		// Same name under different keys exercises the tie-break ordering.
		for _, key := range []string{"services/a/web", "services/b/web", "ingresses/a/web"} {
			w.Upsert(key, &Endpoint{Name: "web", URL: key, Interval: "1m"})
		}
	}

	w := NewWriter(path)
	upsertAll(w)
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	// A fresh writer with the same state mirrors a sidecar restart.
	for range 5 {
		w := NewWriter(path)
		upsertAll(w)
		if err := w.Flush(); err != nil {
			t.Fatalf("Flush: %v", err)
		}
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !os.SameFile(before, after) {
		t.Error("identical content should not replace the output file")
	}
}

func TestWriter_Concurrent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	w := NewWriter(path)

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			w.Upsert("k", &Endpoint{Name: "a", URL: "x", Interval: "1m"})
			if err := w.Flush(); err != nil {
				t.Errorf("Flush: %v", err)
			}
		})
	}
	wg.Wait()

	if w.Len() != 1 {
		t.Errorf("Len() = %d, want 1", w.Len())
	}
}

func TestWriter_CreatesDirectories(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "out.yaml")
	w := NewWriter(path)
	w.Upsert("k", &Endpoint{Name: "a", URL: "x", Interval: "1m"})
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected output file: %v", err)
	}
}
