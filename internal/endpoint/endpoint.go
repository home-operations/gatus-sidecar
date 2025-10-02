package endpoint

import "maps"

// Endpoint represents the configuration for a single endpoint
type Endpoint struct {
	Name       string         `yaml:"name"`
	Group      string         `yaml:"group,omitempty"`
	URL        string         `yaml:"url"`
	Interval   string         `yaml:"interval"`
	Client     map[string]any `yaml:"client,omitempty"`
	Conditions []string       `yaml:"conditions"`
	Extra      map[string]any `yaml:",inline,omitempty"` // For additional template fields
}

// ApplyTemplate applies template data to the endpoint, allowing overrides of default values
func (e *Endpoint) ApplyTemplate(templateData map[string]any) {
	if templateData == nil {
		return
	}

	if e.Extra == nil {
		e.Extra = make(map[string]any)
	}

	// Apply template overrides
	for key, value := range templateData {
		switch key {
		case "name":
			e.setField(&e.Name, value)
		case "group":
			e.setField(&e.Group, value)
		case "url":
			e.setField(&e.URL, value)
		case "interval":
			e.setField(&e.Interval, value)
		case "client":
			e.setField(&e.Client, value)
		case "conditions":
			e.setField(&e.Conditions, value)
		default:
			// Store other fields in Extra for inline YAML output
			e.Extra[key] = value
		}
	}
}

// setField is a generic setter that handles different field types
func (e *Endpoint) setField(field any, value any) {
	switch f := field.(type) {
	case *string:
		if str, ok := value.(string); ok {
			*f = str
		}
	case *[]string:
		if list, ok := value.([]string); ok {
			*f = list
		}
	case *map[string]any:
		if mapValue, ok := value.(map[string]any); ok {
			if *f == nil {
				*f = make(map[string]any)
			}
			maps.Copy(*f, mapValue)
		}
	}
}
