package endpoint

import "maps"

// Endpoint represents the configuration for a single endpoint
type Endpoint struct {
	Name       string         `yaml:"name"`
	Group      string         `yaml:"group,omitempty"`
	URL        string         `yaml:"url"`
	Conditions []string       `yaml:"conditions,omitempty"`
	Interval   string         `yaml:"interval"`
	DNS        map[string]any `yaml:"dns,omitempty"`
	Client     map[string]any `yaml:"client,omitempty"`
	UI         map[string]any `yaml:"ui,omitempty"`
	Guarded    bool           `yaml:"-"`
	Extra      map[string]any `yaml:",inline,omitempty"` // For additional template fields
}

// ApplyTemplate applies template data to the endpoint, allowing overrides of default values
func (e *Endpoint) ApplyTemplate(templateData map[string]any) {
	if templateData == nil {
		return
	}

	// Apply template overrides
	for key, value := range templateData {
		switch key {
		case "name":
			e.setStringField(&e.Name, value)
		case "group":
			e.setStringField(&e.Group, value)
		case "url":
			e.setStringField(&e.URL, value)
		case "conditions":
			e.setConditionsField(value)
		case "interval":
			e.setStringField(&e.Interval, value)
		case "dns":
			e.setMapField(&e.DNS, value)
		case "client":
			e.setMapField(&e.Client, value)
		case "ui":
			e.setMapField(&e.UI, value)
		case "guarded":
			if guarded, ok := value.(bool); ok {
				e.Guarded = guarded
			}
		default:
			// Store other fields in Extra for inline YAML output
			e.AddExtraField(key, value)
		}
	}
}

func (e *Endpoint) AddExtraField(key string, value any) {
	if e.Extra == nil {
		e.Extra = make(map[string]any)
	}
	e.Extra[key] = value
}

// setStringField sets a string field if the value is a string
func (e *Endpoint) setStringField(field *string, value any) {
	if str, ok := value.(string); ok {
		*field = str
	}
}

// setConditionsField handles different condition formats
func (e *Endpoint) setConditionsField(value any) {
	switch v := value.(type) {
	case []string:
		e.Conditions = v
	case []any:
		conditions := make([]string, 0, len(v))
		for _, cond := range v {
			if str, ok := cond.(string); ok {
				conditions = append(conditions, str)
			}
		}
		e.Conditions = conditions
	case string:
		e.Conditions = []string{v}
	}
}

// setMapField merges map settings into the specified field
func (e *Endpoint) setMapField(field *map[string]any, value any) {
	if mapValue, ok := value.(map[string]any); ok {
		if *field == nil {
			*field = make(map[string]any)
		}
		maps.Copy(*field, mapValue)
	}
}
