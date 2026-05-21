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
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	w := NewWriter(path)

	e := &Endpoint{Name: "a", URL: "https://a", Interval: "1m"}

	changed, err := w.Upsert("k1", e, true)
	if err != nil {
		t.Fatalf("Upsert err: %v", err)
	}
	if !changed {
		t.Error("first Upsert should report changed=true")
	}

	changed, err = w.Upsert("k1", &Endpoint{Name: "a", URL: "https://a", Interval: "1m"}, true)
	if err != nil {
		t.Fatalf("Upsert err: %v", err)
	}
	if changed {
		t.Error("equal Upsert should report changed=false")
	}

	changed, err = w.Upsert("k1", &Endpoint{Name: "a", URL: "https://b", Interval: "1m"}, true)
	if err != nil {
		t.Fatalf("Upsert err: %v", err)
	}
	if !changed {
		t.Error("Upsert with new URL should report changed=true")
	}

	removed, err := w.Delete("k1", true)
	if err != nil {
		t.Fatalf("Delete err: %v", err)
	}
	if !removed {
		t.Error("Delete should report removed=true")
	}
	removed, err = w.Delete("k1", true)
	if err != nil {
		t.Fatalf("Delete err: %v", err)
	}
	if removed {
		t.Error("Delete of absent key should report removed=false")
	}
}

func TestWriter_Flush_SortsAndMatchesYAMLShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	w := NewWriter(path)

	endpoints := []*Endpoint{
		{Name: "zebra", URL: "z", Interval: "1m"},
		{Name: "alpha", URL: "a", Interval: "1m", Conditions: []string{"[STATUS] == 200"}},
		{Name: "mid", URL: "m", Interval: "1m"},
	}
	for _, e := range endpoints {
		if _, err := w.Upsert(e.Name, e, false); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
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
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	w := NewWriter(path)
	if _, err := w.Upsert("k", &Endpoint{Name: "a", URL: "x", Interval: "1m"}, true); err != nil {
		t.Fatalf("Upsert: %v", err)
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

func TestWriter_Concurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	w := NewWriter(path)

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_, _ = w.Upsert("k", &Endpoint{Name: "a", URL: "x", Interval: "1m"}, true)
		})
	}
	wg.Wait()

	if w.Len() != 1 {
		t.Errorf("Len() = %d, want 1", w.Len())
	}
}

func TestWriter_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "out.yaml")
	w := NewWriter(path)
	if _, err := w.Upsert("k", &Endpoint{Name: "a", URL: "x", Interval: "1m"}, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected output file: %v", err)
	}
}
