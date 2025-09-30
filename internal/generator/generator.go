package generator

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func Write(endpoint map[string]any, dir string, filename string, template map[string]any) error {
	finalEndpoint := endpoint

	// If template is provided, merge it with the endpoint
	if template != nil {
		finalEndpoint = mergeMaps(endpoint, template)
	}

	// Wrap in config structure
	config := map[string]any{
		"endpoints": []any{finalEndpoint},
	}

	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, yamlData, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func Delete(dir string, filename string) error {
	path := filepath.Join(dir, filename)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}

	return nil
}

// mergeMaps recursively merges src into dst, with src taking precedence
func mergeMaps(dst, src map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy dst first
	maps.Copy(result, dst)

	// Merge src, handling nested maps
	for k, v := range src {
		if dstMap, dstOk := result[k].(map[string]any); dstOk {
			if srcMap, srcOk := v.(map[string]any); srcOk {
				result[k] = mergeMaps(dstMap, srcMap)
				continue
			}
		}
		result[k] = v
	}

	return result
}
