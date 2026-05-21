// Package gatus models Gatus configuration objects (endpoints, templates, writer).
package gatus

import "maps"

// Endpoint is a Gatus monitored endpoint. Extra holds template fields with no
// first-class representation and is inlined into the YAML output.
type Endpoint struct {
	Name       string         `yaml:"name"`
	Group      string         `yaml:"group,omitempty"`
	URL        string         `yaml:"url"`
	Conditions []string       `yaml:"conditions,omitempty"`
	Interval   string         `yaml:"interval"`
	DNS        map[string]any `yaml:"dns,omitempty"`
	Client     map[string]any `yaml:"client,omitempty"`
	UI         map[string]any `yaml:"ui,omitempty"`
	Extra      map[string]any `yaml:",inline,omitempty"`
}

// ApplyTemplate overlays data onto e. Known keys overwrite typed fields;
// everything else lands in Extra. "guarded" is consumed by the controller
// before this is called (see [IsGuarded]) and is not part of the output.
func (e *Endpoint) ApplyTemplate(data map[string]any) {
	for key, value := range data {
		switch key {
		case "name":
			assignString(&e.Name, value)
		case "group":
			assignString(&e.Group, value)
		case "url":
			assignString(&e.URL, value)
		case "interval":
			assignString(&e.Interval, value)
		case "conditions":
			e.Conditions = toStringSlice(value)
		case "dns":
			mergeMap(&e.DNS, value)
		case "client":
			mergeMap(&e.Client, value)
		case "ui":
			mergeMap(&e.UI, value)
		case "guarded":
			// consumed by the controller; never serialized
		default:
			e.setExtra(key, value)
		}
	}
}

func (e *Endpoint) setExtra(key string, value any) {
	if e.Extra == nil {
		e.Extra = make(map[string]any)
	}
	e.Extra[key] = value
}

func assignString(target *string, value any) {
	if s, ok := value.(string); ok {
		*target = s
	}
}

func toStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{v}
	default:
		return nil
	}
}

func mergeMap(target *map[string]any, value any) {
	src, ok := value.(map[string]any)
	if !ok {
		return
	}
	if *target == nil {
		*target = make(map[string]any, len(src))
	}
	maps.Copy(*target, src)
}
