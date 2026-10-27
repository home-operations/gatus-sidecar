package gatus

import (
	"reflect"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      string
		want    map[string]any
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"valid", "name: foo\ninterval: 30s\n", map[string]any{"name": "foo", "interval": "30s"}, false},
		{"invalid", ":\nbad", nil, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseTemplate(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTemplate err=%v wantErr=%v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestMergeTemplates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		parent, child map[string]any
		want          map[string]any
	}{
		{
			name:   "nil parent returns child",
			parent: nil,
			child:  map[string]any{"a": 1},
			want:   map[string]any{"a": 1},
		},
		{
			name:   "nil child returns parent",
			parent: map[string]any{"a": 1},
			child:  nil,
			want:   map[string]any{"a": 1},
		},
		{
			name:   "child overrides parent",
			parent: map[string]any{"a": 1, "b": 2},
			child:  map[string]any{"b": 99},
			want:   map[string]any{"a": 1, "b": 99},
		},
		{
			name:   "nested merge",
			parent: map[string]any{"dns": map[string]any{"q": "p"}},
			child:  map[string]any{"dns": map[string]any{"r": "c"}},
			want:   map[string]any{"dns": map[string]any{"q": "p", "r": "c"}},
		},
		{
			name:   "scalar child replaces map parent",
			parent: map[string]any{"x": map[string]any{"a": 1}},
			child:  map[string]any{"x": "replaced"},
			want:   map[string]any{"x": "replaced"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MergeTemplates(tt.parent, tt.child)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestIsGuarded(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		data map[string]any
		want bool
	}{
		{"nil template", nil, false},
		{"empty template", map[string]any{}, false},
		{"missing key", map[string]any{"group": "internal"}, false},
		{"true", map[string]any{"guarded": true}, true},
		{"false", map[string]any{"guarded": false}, false},
		{"null", map[string]any{"guarded": nil}, false},
		{"string true", map[string]any{"guarded": "true"}, true},
		{"string false", map[string]any{"guarded": "false"}, true},
		{"other string", map[string]any{"guarded": "any-value"}, true},
		{"empty string", map[string]any{"guarded": ""}, true},
		{"number", map[string]any{"guarded": 1}, true},
		{"zero", map[string]any{"guarded": 0}, true},
		{"empty map", map[string]any{"guarded": map[string]any{}}, true},
		{"empty slice", map[string]any{"guarded": []any{}}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsGuarded(tt.data); got != tt.want {
				t.Errorf("IsGuarded(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestPathOverride(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		data   map[string]any
		want   string
		wantOK bool
	}{
		{"absent", map[string]any{"url": "https://x"}, "", false},
		{"empty forces bare host", map[string]any{"path": ""}, "", true},
		{"custom path", map[string]any{"path": "/healthz"}, "/healthz", true},
		{"non-string ignored", map[string]any{"path": 42}, "", false},
		{"nil data", nil, "", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := PathOverride(tt.data)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("PathOverride() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
