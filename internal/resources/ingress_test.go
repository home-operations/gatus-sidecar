package resources

import (
	"context"
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func makeIngress(host string, tls bool, class *string, annotations map[string]string) *networkingv1.Ingress {
	return makeIngressWithPaths(host, tls, class, annotations, nil)
}

func makeIngressWithPaths(host string, tls bool, class *string, annotations map[string]string, paths []string) *networkingv1.Ingress {
	rule := networkingv1.IngressRule{Host: host}
	if len(paths) > 0 {
		http := &networkingv1.HTTPIngressRuleValue{}
		for _, p := range paths {
			http.Paths = append(http.Paths, networkingv1.HTTPIngressPath{Path: p})
		}
		rule.HTTP = http
	}
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "ing",
			Namespace:   "default",
			Annotations: annotations,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: class,
			Rules:            []networkingv1.IngressRule{rule},
		},
	}
	if tls {
		ing.Spec.TLS = []networkingv1.IngressTLS{{Hosts: []string{host}}}
	}
	return ing
}

func TestIngress_URL(t *testing.T) {
	cases := []struct {
		name string
		in   metav1.Object
		want string
	}{
		{"http", makeIngress("example.com", false, nil, nil), "http://example.com"},
		{"https with tls", makeIngress("example.com", true, nil, nil), "https://example.com"},
		{"already prefixed", makeIngress("http://x.com", false, nil, nil), "http://x.com"},
		{"http-prefix-not-url", makeIngress("http-debug.com", false, nil, nil), "http://http-debug.com"},
		{"no rules", &networkingv1.Ingress{}, ""},
		{"wrong type", &corev1.Pod{}, ""},
		{
			name: "first non-trivial path appended",
			in:   makeIngressWithPaths("example.com", true, nil, nil, []string{"/api"}),
			want: "https://example.com/api",
		},
		{
			name: "root path skipped",
			in:   makeIngressWithPaths("example.com", false, nil, nil, []string{"/"}),
			want: "http://example.com",
		},
		{
			name: "empty path skipped",
			in:   makeIngressWithPaths("example.com", false, nil, nil, []string{""}),
			want: "http://example.com",
		},
		{
			name: "non-rooted path skipped",
			in:   makeIngressWithPaths("example.com", false, nil, nil, []string{"api"}),
			want: "http://example.com",
		},
		{
			name: "first probable path among multiple wins",
			in:   makeIngressWithPaths("example.com", false, nil, nil, []string{"/", "/healthz", "/api"}),
			want: "http://example.com/healthz",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Ingress{}).URL(tt.in); got != tt.want {
				t.Errorf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIngress_Matches(t *testing.T) {
	nginx := "nginx"
	cfg := &config.Config{
		AutoIngress:        false,
		EnabledAnnotation:  config.DefaultEnabledAnnotation,
		TemplateAnnotation: config.DefaultTemplateAnnotation,
	}

	cases := []struct {
		name string
		obj  metav1.Object
		cfg  *config.Config
		want bool
	}{
		{
			name: "auto-mode allows anything",
			obj:  makeIngress("x", false, nil, nil),
			cfg:  &config.Config{AutoIngress: true, EnabledAnnotation: cfg.EnabledAnnotation, TemplateAnnotation: cfg.TemplateAnnotation},
			want: true,
		},
		{
			name: "annotation gate without auto",
			obj:  makeIngress("x", false, nil, map[string]string{cfg.EnabledAnnotation: "true"}),
			cfg:  cfg,
			want: true,
		},
		{
			name: "no annotation, no auto = false",
			obj:  makeIngress("x", false, nil, nil),
			cfg:  cfg,
			want: false,
		},
		{
			name: "ingress class mismatch rejects",
			obj:  makeIngress("x", false, &nginx, map[string]string{cfg.EnabledAnnotation: "true"}),
			cfg:  &config.Config{AutoIngress: true, IngressClasses: config.StringSet{"traefik"}},
			want: false,
		},
		{
			name: "ingress class match accepts",
			obj:  makeIngress("x", false, &nginx, nil),
			cfg:  &config.Config{AutoIngress: true, IngressClasses: config.StringSet{"nginx"}},
			want: true,
		},
		{
			name: "matches any of multiple ingress classes",
			obj:  makeIngress("x", false, &nginx, nil),
			cfg:  &config.Config{AutoIngress: true, IngressClasses: config.StringSet{"traefik", "nginx", "haproxy"}},
			want: true,
		},
		{
			name: "rejects when none of multiple ingress classes match",
			obj:  makeIngress("x", false, &nginx, nil),
			cfg:  &config.Config{AutoIngress: true, IngressClasses: config.StringSet{"traefik", "haproxy"}},
			want: false,
		},
		{
			name: "non-ingress",
			obj:  &corev1.Pod{},
			cfg:  cfg,
			want: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Ingress{}).Matches(tt.obj, tt.cfg); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIngress_DefaultConditions(t *testing.T) {
	got := (Ingress{}).DefaultConditions()
	if len(got) != 1 || got[0] != "[STATUS] == 200" {
		t.Errorf("DefaultConditions() = %v", got)
	}
}

func TestIngress_GuardHost(t *testing.T) {
	if got := (Ingress{}).GuardHost(makeIngress("host.example.com", false, nil, nil)); got != "host.example.com" {
		t.Errorf("GuardHost() = %q", got)
	}
	if got := (Ingress{}).GuardHost(&corev1.Pod{}); got != "" {
		t.Errorf("GuardHost(non-ingress) = %q, want \"\"", got)
	}
}

func TestIngressClassOf(t *testing.T) {
	nginx := "nginx"
	cases := []struct {
		name string
		ing  *networkingv1.Ingress
		want string
	}{
		{"spec wins over legacy", &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{legacyIngressClassAnnotation: "ignored"}},
			Spec:       networkingv1.IngressSpec{IngressClassName: &nginx},
		}, "nginx"},
		{"legacy fallback", &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{legacyIngressClassAnnotation: "traefik"}},
		}, "traefik"},
		{"none", &networkingv1.Ingress{}, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := ingressClassOf(tt.ing); got != tt.want {
				t.Errorf("ingressClassOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIngress_ParentAnnotations(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(ingressClassGVR.GroupVersion().WithKind("IngressClass"), &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(ingressClassGVR.GroupVersion().WithKind("IngressClassList"), &unstructured.UnstructuredList{})
	client := fake.NewSimpleDynamicClient(scheme)

	className := "nginx"
	parent := &unstructured.Unstructured{}
	parent.SetAPIVersion("networking.k8s.io/v1")
	parent.SetKind("IngressClass")
	parent.SetName(className)
	parent.SetAnnotations(map[string]string{"parent": "annotation"})
	if _, err := client.Resource(ingressClassGVR).Create(context.Background(), parent, metav1.CreateOptions{}); err != nil {
		t.Fatalf("seed ingressclass: %v", err)
	}

	ing := makeIngress("x", false, &className, nil)
	ann := (Ingress{}).ParentAnnotations(context.Background(), ing, k8s.NewFetcher(client))
	if ann["parent"] != "annotation" {
		t.Errorf("ParentAnnotations = %v, want {parent: annotation}", ann)
	}
}

func TestIngress_ParentAnnotations_Missing(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	ing := makeIngress("x", false, nil, nil)

	if ann := (Ingress{}).ParentAnnotations(context.Background(), ing, k8s.NewFetcher(client)); ann != nil {
		t.Errorf("ParentAnnotations(no class) = %v, want nil", ann)
	}
}
