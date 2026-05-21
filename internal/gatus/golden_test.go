package gatus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriter_GoldenOutput pins the exact YAML shape gatus consumes. Any change
// to the marshalled output requires updating the golden string here, so that
// downstream gatus configuration breakage is caught at compile-test time.
func TestWriter_GoldenOutput(t *testing.T) {
	w := NewWriter(filepath.Join(t.TempDir(), "out.yaml"))

	e := &Endpoint{
		Name:       "demo",
		Group:      "prod",
		URL:        "https://demo.example.com",
		Conditions: []string{"[STATUS] == 200", "[RESPONSE_TIME] < 500"},
		Interval:   "30s",
		Extra: map[string]any{
			"alerts": []any{
				map[string]any{"type": "slack", "webhook-url": "https://example.com/hook"},
			},
		},
	}
	if _, err := w.Upsert("demo", e, true); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := os.ReadFile(w.path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	const want = `endpoints:
    - name: demo
      group: prod
      url: https://demo.example.com
      conditions:
        - '[STATUS] == 200'
        - '[RESPONSE_TIME] < 500'
      interval: 30s
      alerts:
        - type: slack
          webhook-url: https://example.com/hook
`
	if strings.TrimSpace(string(got)) != strings.TrimSpace(want) {
		t.Errorf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
