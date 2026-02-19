package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/endpoint"
)

func TestManager_AddOrUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.yaml")
	m := NewManager(outputFile)

	e1 := &endpoint.Endpoint{
		Name:     "test-endpoint",
		URL:      "https://example.com",
		Interval: "1m",
	}

	tests := []struct {
		name       string
		key        string
		endpoint   *endpoint.Endpoint
		write      bool
		wantChange bool
	}{
		{
			name:       "add new endpoint with write",
			key:        "test.default.services",
			endpoint:   e1,
			write:      false,
			wantChange: true,
		},
		{
			name:       "update existing endpoint",
			key:        "test.default.services",
			endpoint:   &endpoint.Endpoint{Name: "test-endpoint", URL: "https://updated.example.com", Interval: "30s"},
			write:      false,
			wantChange: true,
		},
		{
			name:       "no change for same endpoint",
			key:        "test.default.services",
			endpoint:   &endpoint.Endpoint{Name: "test-endpoint", URL: "https://updated.example.com", Interval: "30s"},
			write:      false,
			wantChange: false,
		},
		{
			name:       "add another endpoint",
			key:        "test2.default.services",
			endpoint:   &endpoint.Endpoint{Name: "test-endpoint-2", URL: "https://example2.com", Interval: "1m"},
			write:      false,
			wantChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := m.AddOrUpdate(tt.key, tt.endpoint, tt.write)
			if changed != tt.wantChange {
				t.Errorf("AddOrUpdate() changed = %v, want %v", changed, tt.wantChange)
			}
		})
	}
}

func TestManager_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.yaml")
	m := NewManager(outputFile)

	e1 := &endpoint.Endpoint{
		Name:     "test-endpoint",
		URL:      "https://example.com",
		Interval: "1m",
	}

	m.AddOrUpdate("test.default.services", e1, false)

	tests := []struct {
		name       string
		key        string
		wantChange bool
	}{
		{
			name:       "remove existing endpoint",
			key:        "test.default.services",
			wantChange: true,
		},
		{
			name:       "remove non-existent endpoint",
			key:        "nonexistent.default.services",
			wantChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := m.Remove(tt.key)
			if changed != tt.wantChange {
				t.Errorf("Remove() changed = %v, want %v", changed, tt.wantChange)
			}
		})
	}
}

func TestManager_ForceWrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.yaml")
	m := NewManager(outputFile)

	e1 := &endpoint.Endpoint{
		Name:       "test-endpoint",
		URL:        "https://example.com",
		Interval:   "1m",
		Conditions: []string{"[STATUS] == 200"},
	}
	e2 := &endpoint.Endpoint{
		Name:       "another-endpoint",
		URL:        "tcp://database.default.svc:5432",
		Interval:   "30s",
		Conditions: []string{"[CONNECTED] == true"},
	}

	m.AddOrUpdate("test.default.services", e1, false)
	m.AddOrUpdate("another.default.services", e2, false)
	m.ForceWrite()

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Output file is empty")
	}

	content := string(data)
	if content == "" {
		t.Error("Output file content is empty")
	}
}

func TestManager_WriteStateWithMultipleEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.yaml")
	m := NewManager(outputFile)

	endpoints := []*endpoint.Endpoint{
		{Name: "z-endpoint", URL: "https://z.example.com", Interval: "1m"},
		{Name: "a-endpoint", URL: "https://a.example.com", Interval: "1m"},
		{Name: "m-endpoint", URL: "https://m.example.com", Interval: "1m"},
	}

	for _, e := range endpoints {
		m.AddOrUpdate(e.Name+".default.services", e, false)
	}

	m.ForceWrite()

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	content := string(data)
	if content == "" {
		t.Fatal("Output file content is empty")
	}

	if !containsInOrder(content, "a-endpoint", "m-endpoint", "z-endpoint") {
		t.Error("Endpoints should be sorted alphabetically by name")
	}
}

func TestManager_NoWriteOnNoChange(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.yaml")
	m := NewManager(outputFile)

	e1 := &endpoint.Endpoint{
		Name:     "test-endpoint",
		URL:      "https://example.com",
		Interval: "1m",
	}

	m.AddOrUpdate("test.default.services", e1, true)

	data1, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	m.AddOrUpdate("test.default.services", e1, true)

	data2, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if string(data1) != string(data2) {
		t.Error("File should not change when endpoint is unchanged")
	}
}

func TestManager_ConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "test.yaml")
	m := NewManager(outputFile)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(_ int) {
			e := &endpoint.Endpoint{
				Name:     "concurrent-test",
				URL:      "https://example.com",
				Interval: "1m",
			}
			m.AddOrUpdate("concurrent.default.services", e, false)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	m.ForceWrite()

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Output file is empty after concurrent operations")
	}
}

func containsInOrder(s string, substrs ...string) bool {
	idx := 0
	for _, substr := range substrs {
		pos := indexOf(s[idx:], substr)
		if pos == -1 {
			return false
		}
		idx += pos + len(substr)
	}
	return true
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
