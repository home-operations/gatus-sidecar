package resources

import (
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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

func TestKind(t *testing.T) {
	t.Parallel()
	cases := map[string]k8s.Resource{
		config.KindIngress:      Ingress{},
		config.KindService:      Service{},
		config.KindHTTPRoute:    HTTPRoute{},
		config.KindIngressRoute: IngressRoute{},
	}
	for want, r := range cases {
		if got := r.Kind(); got != want {
			t.Errorf("%T.Kind() = %q, want %q", r, got, want)
		}
	}
}
