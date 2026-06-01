package resources

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func makeIngressRoute(host string, tls bool) *unstructured.Unstructured {
	return makeIngressRouteWithMatch("Host(`"+host+"`)", tls)
}

func makeIngressRouteWithMatch(match string, tls bool) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("traefik.io/v1alpha1")
	u.SetKind("IngressRoute")
	u.SetName("r")
	u.SetNamespace("default")
	routes := []any{
		map[string]any{"match": match},
	}
	_ = unstructured.SetNestedSlice(u.Object, routes, "spec", "routes")
	if tls {
		_ = unstructured.SetNestedMap(u.Object, map[string]any{"secretName": "s"}, "spec", "tls")
	}
	return u
}

func TestIngressRoute_URL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		obj  metav1.Object
		want string
	}{
		{"http", makeIngressRoute("example.com", false), "http://example.com"},
		{"https", makeIngressRoute("secure.example.com", true), "https://secure.example.com"},
		{"empty unstructured", &unstructured.Unstructured{}, ""},
		{
			name: "host with PathPrefix",
			obj:  makeIngressRouteWithMatch("Host(`api.example.com`) && PathPrefix(`/v1`)", false),
			want: "http://api.example.com/v1",
		},
		{
			name: "host with Path (exact)",
			obj:  makeIngressRouteWithMatch("Host(`api.example.com`) && Path(`/healthz`)", true),
			want: "https://api.example.com/healthz",
		},
		{
			name: "host with root path skipped",
			obj:  makeIngressRouteWithMatch("Host(`api.example.com`) && PathPrefix(`/`)", false),
			want: "http://api.example.com",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := (IngressRoute{}).URL(tt.obj); got != tt.want {
				t.Errorf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIngressRoute_DefaultConditionsAndGuardHost(t *testing.T) {
	t.Parallel()
	if got := (IngressRoute{}).DefaultConditions(); len(got) != 1 || got[0] != "[STATUS] == 200" {
		t.Errorf("DefaultConditions() = %v", got)
	}
	if got := (IngressRoute{}).GuardHost(makeIngressRoute("guarded.example.com", false)); got != "guarded.example.com" {
		t.Errorf("GuardHost() = %q", got)
	}
	if got := (IngressRoute{}).GuardHost(&unstructured.Unstructured{}); got != "" {
		t.Errorf("GuardHost(empty) = %q, want \"\"", got)
	}
}

func TestMatchTraefikHost(t *testing.T) {
	t.Parallel()
	cases := []struct {
		expr, want string
	}{
		{"Host(`example.com`)", "example.com"},
		{"Host(`a.example.com`) && PathPrefix(`/v1`)", "a.example.com"},
		{"Method(`GET`) && Host(`x.example.com`)", "x.example.com"},
		{"PathPrefix(`/api`)", ""},
		{"", ""},
	}
	for _, tt := range cases {
		t.Run(tt.expr, func(t *testing.T) {
			t.Parallel()
			if got := matchTraefikHost(tt.expr); got != tt.want {
				t.Errorf("matchTraefikHost(%q) = %q, want %q", tt.expr, got, tt.want)
			}
		})
	}
}

func TestIngressRouteHasTLS(t *testing.T) {
	t.Parallel()
	u := &unstructured.Unstructured{Object: map[string]any{}}
	if err := unstructured.SetNestedMap(u.Object, map[string]any{"secretName": "s"}, "spec", "tls"); err != nil {
		t.Fatalf("SetNestedMap: %v", err)
	}
	if !ingressRouteHasTLS(u) {
		t.Error("should detect tls")
	}
	if ingressRouteHasTLS(&unstructured.Unstructured{Object: map[string]any{}}) {
		t.Error("should be false without tls")
	}
}
