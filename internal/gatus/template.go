package gatus

import (
	"fmt"
	"maps"

	"gopkg.in/yaml.v3"
)

// ParseTemplate decodes a YAML template annotation. An empty value yields a
// nil map (not an error) so callers can pass annotation values blindly.
func ParseTemplate(value string) (map[string]any, error) {
	if value == "" {
		return nil, nil
	}
	var out map[string]any
	if err := yaml.Unmarshal([]byte(value), &out); err != nil {
		return nil, fmt.Errorf("decode template annotation: %w", err)
	}
	return out, nil
}

// MergeTemplates deep-merges child into parent. Scalars from child win;
// nested map values are merged recursively.
func MergeTemplates(parent, child map[string]any) map[string]any {
	switch {
	case parent == nil:
		return child
	case child == nil:
		return parent
	}

	out := make(map[string]any, len(parent)+len(child))
	maps.Copy(out, parent)

	for key, childVal := range child {
		if parentVal, exists := out[key]; exists {
			if pm, ok := parentVal.(map[string]any); ok {
				if cm, ok := childVal.(map[string]any); ok {
					out[key] = MergeTemplates(pm, cm)
					continue
				}
			}
		}
		out[key] = childVal
	}
	return out
}

// IsGuarded reports whether data opts the endpoint into a DNS-only probe.
func IsGuarded(data map[string]any) bool {
	_, ok := data["guarded"]
	return ok
}
