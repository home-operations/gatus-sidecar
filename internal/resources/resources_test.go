package resources

import (
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// autoEnabled returns a Kinds map with the named kind in auto-discovery mode.
func autoEnabled(name string) map[string]*config.KindConfig {
	return map[string]*config.KindConfig{name: {Auto: true}}
}

func TestAll_DefaultsToEverything(t *testing.T) {
	t.Parallel()
	got := All(&config.Config{})
	if len(got) != 4 {
		t.Errorf("got %d resources, want 4", len(got))
	}
}

func TestAll_HonorsExplicitFlags(t *testing.T) {
	t.Parallel()
	got := All(&config.Config{Kinds: map[string]*config.KindConfig{
		config.KindIngress: {Enable: true},
	}})
	if len(got) != 1 {
		t.Fatalf("got %d resources, want 1", len(got))
	}
	if got[0].GVR().Resource != "ingresses" {
		t.Errorf("got %s, want ingresses", got[0].GVR().Resource)
	}

	got = All(&config.Config{Kinds: map[string]*config.KindConfig{
		config.KindService:   {Auto: true},
		config.KindHTTPRoute: {Auto: true},
	}})
	names := map[string]bool{}
	for _, r := range got {
		names[r.GVR().Resource] = true
	}
	if !names["services"] || !names["httproutes"] {
		t.Errorf("got %v, want services & httproutes", names)
	}
	if names["ingresses"] || names["ingressroutes"] {
		t.Errorf("unexpected resources: %v", names)
	}
}

func TestConvertTo(t *testing.T) {
	t.Parallel()
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata":   map[string]any{"name": "s", "namespace": "n"},
			"spec":       map[string]any{},
		},
	}
	obj, err := convertTo[corev1.Service](u)
	if err != nil {
		t.Fatalf("convertTo err: %v", err)
	}
	if obj.GetName() != "s" || obj.GetNamespace() != "n" {
		t.Errorf("name=%q ns=%q", obj.GetName(), obj.GetNamespace())
	}
}

func TestPrefix(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Kinds: map[string]*config.KindConfig{
		config.KindIngress:      {Prefix: "ing-"},
		config.KindService:      {Prefix: "svc-"},
		config.KindHTTPRoute:    {Prefix: "route-"},
		config.KindIngressRoute: {Prefix: "traefik-"},
	}}
	cases := map[string]string{
		"ing-":     Ingress{}.Prefix(cfg),
		"svc-":     Service{}.Prefix(cfg),
		"route-":   HTTPRoute{}.Prefix(cfg),
		"traefik-": IngressRoute{}.Prefix(cfg),
	}
	for want, got := range cases {
		if got != want {
			t.Errorf("Prefix() = %q, want %q", got, want)
		}
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
			obj := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: tt.ann}}
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
