package k8s

import (
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"

	corev1 "k8s.io/api/core/v1"
)

func TestMatchesAnnotation(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		EnabledAnnotation:  "enabled",
		TemplateAnnotation: "tpl",
	}
	cases := []struct {
		name string
		auto bool
		ann  map[string]string
		want bool
	}{
		{"auto allows anything", true, nil, true},
		{"auto still honors an explicit opt-out", true, map[string]string{"enabled": "false"}, false},
		{"no auto, no annotation", false, nil, false},
		{"no auto, enabled annotation", false, map[string]string{"enabled": "true"}, true},
		{"no auto, template annotation", false, map[string]string{"tpl": "x"}, true},
		{"no auto, template but disabled", false, map[string]string{"tpl": "x", "enabled": "false"}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obj := &corev1.Service{Annotations: tt.ann}
			if got := matchesAnnotation(obj, tt.auto, cfg); got != tt.want {
				t.Errorf("matchesAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasGatusAnnotations(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		EnabledAnnotation:  "enabled",
		TemplateAnnotation: "tpl",
	}
	cases := []struct {
		name string
		ann  map[string]string
		want bool
	}{
		{"none", nil, false},
		{"unrelated", map[string]string{"x": "y"}, false},
		{"enabled present", map[string]string{"enabled": "true"}, true},
		{"template present", map[string]string{"tpl": "x"}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obj := &corev1.Service{Annotations: tt.ann}
			if got := hasGatusAnnotations(obj, cfg); got != tt.want {
				t.Errorf("hasGatusAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExplicitlyDisabled(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ann  map[string]string
		want bool
	}{
		{"absent", nil, false},
		{"true", map[string]string{"enabled": "true"}, false},
		{"True", map[string]string{"enabled": "True"}, false},
		{"TRUE", map[string]string{"enabled": "TRUE"}, false},
		{"one", map[string]string{"enabled": "1"}, false},
		{"false", map[string]string{"enabled": "false"}, true},
		{"zero", map[string]string{"enabled": "0"}, true},
		{"empty", map[string]string{"enabled": ""}, true},
		{"unparseable", map[string]string{"enabled": "yes"}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isExplicitlyDisabled(tt.ann, "enabled"); got != tt.want {
				t.Errorf("isExplicitlyDisabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
