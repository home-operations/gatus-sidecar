package endpoint

import (
	"testing"
)

func TestEndpoint_ApplyTemplate(t *testing.T) {
	tests := []struct {
		name     string
		endpoint *Endpoint
		template map[string]any
		want     *Endpoint
	}{
		{
			name: "nil template does nothing",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
			},
			template: nil,
			want: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
			},
		},
		{
			name: "override string fields",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
				Group:    "old-group",
			},
			template: map[string]any{
				"name":     "new-name",
				"url":      "https://new.example.com",
				"interval": "30s",
				"group":    "new-group",
			},
			want: &Endpoint{
				Name:     "new-name",
				URL:      "https://new.example.com",
				Interval: "30s",
				Group:    "new-group",
			},
		},
		{
			name: "set conditions from string slice",
			endpoint: &Endpoint{
				Name:       "test",
				URL:        "https://example.com",
				Interval:   "1m",
				Conditions: []string{"[STATUS] == 200"},
			},
			template: map[string]any{
				"conditions": []string{"[STATUS] == 200", "[RESPONSE_TIME] < 500"},
			},
			want: &Endpoint{
				Name:       "test",
				URL:        "https://example.com",
				Interval:   "1m",
				Conditions: []string{"[STATUS] == 200", "[RESPONSE_TIME] < 500"},
			},
		},
		{
			name: "set conditions from any slice",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
			},
			template: map[string]any{
				"conditions": []any{"[STATUS] == 200", "[RESPONSE_TIME] < 500"},
			},
			want: &Endpoint{
				Name:       "test",
				URL:        "https://example.com",
				Interval:   "1m",
				Conditions: []string{"[STATUS] == 200", "[RESPONSE_TIME] < 500"},
			},
		},
		{
			name: "set conditions from single string",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
			},
			template: map[string]any{
				"conditions": "[STATUS] == 200",
			},
			want: &Endpoint{
				Name:       "test",
				URL:        "https://example.com",
				Interval:   "1m",
				Conditions: []string{"[STATUS] == 200"},
			},
		},
		{
			name: "set dns config",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
			},
			template: map[string]any{
				"dns": map[string]any{
					"query-name": "example.com",
					"query-type": "A",
				},
			},
			want: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
				DNS: map[string]any{
					"query-name": "example.com",
					"query-type": "A",
				},
			},
		},
		{
			name: "merge dns config",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
				DNS: map[string]any{
					"query-name": "old.example.com",
				},
			},
			template: map[string]any{
				"dns": map[string]any{
					"query-type": "AAAA",
				},
			},
			want: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
				DNS: map[string]any{
					"query-name": "old.example.com",
					"query-type": "AAAA",
				},
			},
		},
		{
			name: "set guarded flag",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
				Guarded:  false,
			},
			template: map[string]any{
				"guarded": true,
			},
			want: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
				Guarded:  true,
			},
		},
		{
			name: "extra fields stored in Extra",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
			},
			template: map[string]any{
				"alerts": []any{
					map[string]any{
						"type":        "slack",
						"webhook-url": "https://hooks.slack.com/...",
					},
				},
			},
			want: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
				Extra: map[string]any{
					"alerts": []any{
						map[string]any{
							"type":        "slack",
							"webhook-url": "https://hooks.slack.com/...",
						},
					},
				},
			},
		},
		{
			name: "ignore invalid string type",
			endpoint: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
			},
			template: map[string]any{
				"name": 123,
			},
			want: &Endpoint{
				Name:     "test",
				URL:      "https://example.com",
				Interval: "1m",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.endpoint.ApplyTemplate(tt.template)
			if tt.endpoint.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", tt.endpoint.Name, tt.want.Name)
			}
			if tt.endpoint.URL != tt.want.URL {
				t.Errorf("URL = %v, want %v", tt.endpoint.URL, tt.want.URL)
			}
			if tt.endpoint.Interval != tt.want.Interval {
				t.Errorf("Interval = %v, want %v", tt.endpoint.Interval, tt.want.Interval)
			}
			if tt.endpoint.Group != tt.want.Group {
				t.Errorf("Group = %v, want %v", tt.endpoint.Group, tt.want.Group)
			}
			if tt.endpoint.Guarded != tt.want.Guarded {
				t.Errorf("Guarded = %v, want %v", tt.endpoint.Guarded, tt.want.Guarded)
			}
			if !equalStringSlices(tt.endpoint.Conditions, tt.want.Conditions) {
				t.Errorf("Conditions = %v, want %v", tt.endpoint.Conditions, tt.want.Conditions)
			}
			if !equalMaps(tt.endpoint.DNS, tt.want.DNS) {
				t.Errorf("DNS = %v, want %v", tt.endpoint.DNS, tt.want.DNS)
			}
		})
	}
}

func TestEndpoint_AddExtraField(t *testing.T) {
	e := &Endpoint{}
	e.AddExtraField("key1", "value1")

	if e.Extra == nil {
		t.Error("Extra should not be nil")
	}
	if e.Extra["key1"] != "value1" {
		t.Errorf("Extra[key1] = %v, want value1", e.Extra["key1"])
	}

	e.AddExtraField("key2", 123)
	if e.Extra["key2"] != 123 {
		t.Errorf("Extra[key2] = %v, want 123", e.Extra["key2"])
	}
}

func TestEndpoint_setStringField(t *testing.T) {
	e := &Endpoint{}
	var field string

	e.setStringField(&field, "test")
	if field != "test" {
		t.Errorf("field = %v, want test", field)
	}

	field = ""
	e.setStringField(&field, 123)
	if field != "" {
		t.Errorf("field should remain empty for invalid type, got %v", field)
	}
}

func TestEndpoint_setConditionsField(t *testing.T) {
	e := &Endpoint{}

	e.setConditionsField([]string{"cond1", "cond2"})
	if !equalStringSlices(e.Conditions, []string{"cond1", "cond2"}) {
		t.Errorf("Conditions = %v, want [cond1, cond2]", e.Conditions)
	}

	e.setConditionsField([]any{"cond3", "cond4"})
	if !equalStringSlices(e.Conditions, []string{"cond3", "cond4"}) {
		t.Errorf("Conditions = %v, want [cond3, cond4]", e.Conditions)
	}

	e.setConditionsField("single-condition")
	if !equalStringSlices(e.Conditions, []string{"single-condition"}) {
		t.Errorf("Conditions = %v, want [single-condition]", e.Conditions)
	}
}

func TestEndpoint_setMapField(t *testing.T) {
	e := &Endpoint{}
	var field map[string]any

	e.setMapField(&field, map[string]any{"key1": "value1"})
	if field == nil || field["key1"] != "value1" {
		t.Errorf("field = %v, want {key1: value1}", field)
	}

	e.setMapField(&field, map[string]any{"key2": "value2"})
	if field["key1"] != "value1" || field["key2"] != "value2" {
		t.Errorf("field should merge, got %v", field)
	}

	e.setMapField(&field, "invalid")
	if len(field) != 2 {
		t.Errorf("field should remain unchanged for invalid type, got %v", field)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalMaps(a, b map[string]any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
