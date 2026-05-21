package gatus

import (
	"reflect"
	"testing"
)

func TestEndpoint_ApplyTemplate(t *testing.T) {
	tests := []struct {
		name string
		in   *Endpoint
		tmpl map[string]any
		want *Endpoint
	}{
		{
			name: "nil template is a no-op",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m"},
			tmpl: nil,
			want: &Endpoint{Name: "a", URL: "x", Interval: "1m"},
		},
		{
			name: "override string fields",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m", Group: "old"},
			tmpl: map[string]any{
				"name":     "new-name",
				"url":      "https://new",
				"interval": "30s",
				"group":    "new-group",
			},
			want: &Endpoint{Name: "new-name", URL: "https://new", Interval: "30s", Group: "new-group"},
		},
		{
			name: "conditions from []string, []any, and string",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m"},
			tmpl: map[string]any{"conditions": []any{"[STATUS] == 200", "[RESPONSE_TIME] < 500"}},
			want: &Endpoint{Name: "a", URL: "x", Interval: "1m", Conditions: []string{"[STATUS] == 200", "[RESPONSE_TIME] < 500"}},
		},
		{
			name: "conditions from single string",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m"},
			tmpl: map[string]any{"conditions": "[STATUS] == 200"},
			want: &Endpoint{Name: "a", URL: "x", Interval: "1m", Conditions: []string{"[STATUS] == 200"}},
		},
		{
			name: "dns merge preserves existing keys",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m", DNS: map[string]any{"query-name": "old"}},
			tmpl: map[string]any{"dns": map[string]any{"query-type": "AAAA"}},
			want: &Endpoint{Name: "a", URL: "x", Interval: "1m", DNS: map[string]any{"query-name": "old", "query-type": "AAAA"}},
		},
		{
			name: "guarded bool",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m"},
			tmpl: map[string]any{"guarded": true},
			want: &Endpoint{Name: "a", URL: "x", Interval: "1m", Guarded: true},
		},
		{
			name: "unknown keys go into Extra",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m"},
			tmpl: map[string]any{"alerts": []any{"slack"}},
			want: &Endpoint{Name: "a", URL: "x", Interval: "1m", Extra: map[string]any{"alerts": []any{"slack"}}},
		},
		{
			name: "ignore wrong type for string field",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m"},
			tmpl: map[string]any{"name": 123},
			want: &Endpoint{Name: "a", URL: "x", Interval: "1m"},
		},
		{
			name: "client and ui map merges",
			in:   &Endpoint{Name: "a", URL: "x", Interval: "1m"},
			tmpl: map[string]any{
				"client": map[string]any{"timeout": "5s"},
				"ui":     map[string]any{"hide-url": true},
			},
			want: &Endpoint{
				Name: "a", URL: "x", Interval: "1m",
				Client: map[string]any{"timeout": "5s"},
				UI:     map[string]any{"hide-url": true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.in.ApplyTemplate(tt.tmpl)
			if !reflect.DeepEqual(tt.in, tt.want) {
				t.Errorf("ApplyTemplate mismatch\n got=%+v\nwant=%+v", tt.in, tt.want)
			}
		})
	}
}

func TestToStringSlice_DropsNonStringElements(t *testing.T) {
	got := toStringSlice([]any{"a", 1, "b"})
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toStringSlice() = %v, want %v", got, want)
	}
}

func TestToStringSlice_NilForUnknown(t *testing.T) {
	if got := toStringSlice(12345); got != nil {
		t.Errorf("toStringSlice(12345) = %v, want nil", got)
	}
}
