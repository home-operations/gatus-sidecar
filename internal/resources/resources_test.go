package resources

import (
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestAll_DefaultsToEverything(t *testing.T) {
	got := All(&config.Config{})
	if len(got) != 4 {
		t.Errorf("got %d resources, want 4", len(got))
	}
}

func TestAll_HonorsExplicitFlags(t *testing.T) {
	got := All(&config.Config{EnableIngress: true})
	if len(got) != 1 {
		t.Fatalf("got %d resources, want 1", len(got))
	}
	if got[0].GVR().Resource != "ingresses" {
		t.Errorf("got %s, want ingresses", got[0].GVR().Resource)
	}

	got = All(&config.Config{AutoService: true, AutoHTTPRoute: true})
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
	cfg := &config.Config{
		IngressPrefix:      "ing-",
		ServicePrefix:      "svc-",
		HTTPRoutePrefix:    "route-",
		IngressRoutePrefix: "traefik-",
	}
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
			obj := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Annotations: tt.ann}}
			if got := hasGatusAnnotations(obj, cfg); got != tt.want {
				t.Errorf("hasGatusAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}
