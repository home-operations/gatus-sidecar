package gatus

import (
	"reflect"
	"testing"
)

func TestParseTemplate(t *testing.T) {
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
			got := MergeTemplates(tt.parent, tt.child)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestIsGuarded(t *testing.T) {
	if IsGuarded(nil) {
		t.Error("nil data should not be guarded")
	}
	if !IsGuarded(map[string]any{"guarded": true}) {
		t.Error("explicit guarded should be true")
	}
	if !IsGuarded(map[string]any{"guarded": "any-value"}) {
		t.Error("any value at guarded key should be treated as guarded")
	}
}
