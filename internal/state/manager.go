package state

import (
	"log/slog"
	"os"
	"reflect"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/home-operations/gatus-sidecar/internal/endpoint"
)

// Manager maintains the global state of all endpoints
type Manager struct {
	mu         sync.Mutex
	endpoints  map[string]*endpoint.Endpoint // keyed by resource key (name-namespace)
	outputFile string
}

// NewManager creates a new state manager
func NewManager(outputFile string) *Manager {
	return &Manager{
		endpoints:  make(map[string]*endpoint.Endpoint),
		outputFile: outputFile,
	}
}

// AddOrUpdate adds or updates an endpoint and writes state if changed
func (m *Manager) AddOrUpdate(key string, e *endpoint.Endpoint, write bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if this is actually a change
	existing, exists := m.endpoints[key]
	if exists && reflect.DeepEqual(existing, e) {
		return false // No change
	}

	m.endpoints[key] = e

	// Write state if requested
	if write {
		m.writeState()
	}

	return true // Change detected
}

// Remove removes an endpoint and writes state if changed
func (m *Manager) Remove(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.endpoints[key]
	if !exists {
		return false // No change
	}

	delete(m.endpoints, key)
	m.writeState()
	return true // Change detected
}

// ForceWrite forces a write of the current state to disk
func (m *Manager) ForceWrite() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeState()
}

// writeState writes the current state to disk (must be called with mutex held)
func (m *Manager) writeState() {
	state := m.getCurrentState()

	yamlData, err := yaml.Marshal(state)
	if err != nil {
		slog.Error("failed to marshal state to yaml", "error", err)
		return
	}

	if err := os.WriteFile(m.outputFile, yamlData, 0o644); err != nil {
		slog.Error("failed to write state to file", "error", err)
		return
	}

	endpointCount := len(m.endpoints)
	slog.Info("wrote consolidated state file", "file", m.outputFile, "endpoints", endpointCount)
}

// getCurrentState returns the current state as a map suitable for YAML generation
// (must be called with mutex held)
func (m *Manager) getCurrentState() map[string]any {
	// Convert to slice and sort for consistent output
	endpoints := make([]*endpoint.Endpoint, 0, len(m.endpoints))
	for _, e := range m.endpoints {
		endpoints = append(endpoints, e)
	}

	// Sort by name for consistent ordering
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].Name < endpoints[j].Name
	})

	return map[string]any{
		"endpoints": endpoints,
	}
}
