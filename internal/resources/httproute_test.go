package resources

import (
	"context"
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func makeRoute(name string, hostnames []gatewayv1.Hostname, parentRefs []gatewayv1.ParentReference, annotations map[string]string) *gatewayv1.HTTPRoute {
	return &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Annotations: annotations},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{ParentRefs: parentRefs},
			Hostnames:       hostnames,
		},
	}
}

func TestHTTPRoute_URL(t *testing.T) {
	exact := gatewayv1.PathMatchExact
	prefix := gatewayv1.PathMatchPathPrefix
	regex := gatewayv1.PathMatchRegularExpression

	pathRoute := func(host string, ptype *gatewayv1.PathMatchType, value string) *gatewayv1.HTTPRoute {
		r := makeRoute("a", []gatewayv1.Hostname{gatewayv1.Hostname(host)}, nil, nil)
		v := value
		r.Spec.Rules = []gatewayv1.HTTPRouteRule{{
			Matches: []gatewayv1.HTTPRouteMatch{{
				Path: &gatewayv1.HTTPPathMatch{Type: ptype, Value: &v},
			}},
		}}
		return r
	}

	cases := []struct {
		name string
		in   metav1.Object
		want string
	}{
		{"https default", makeRoute("a", []gatewayv1.Hostname{"api.example.com"}, nil, nil), "https://api.example.com"},
		{"http prefix preserved", makeRoute("a", []gatewayv1.Hostname{"http://api"}, nil, nil), "http://api"},
		{"https prefix preserved", makeRoute("a", []gatewayv1.Hostname{"https://api"}, nil, nil), "https://api"},
		{"no hostnames", &gatewayv1.HTTPRoute{}, ""},
		{"wrong type", &corev1.Pod{}, ""},
		{"exact path appended", pathRoute("api.example.com", &exact, "/v1/health"), "https://api.example.com/v1/health"},
		{"pathprefix appended", pathRoute("api.example.com", &prefix, "/v1"), "https://api.example.com/v1"},
		{"regex skipped", pathRoute("api.example.com", &regex, "^/v1/.*"), "https://api.example.com"},
		{"root path skipped", pathRoute("api.example.com", &prefix, "/"), "https://api.example.com"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := (HTTPRoute{}).URL(tt.in); got != tt.want {
				t.Errorf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHTTPRoute_Matches(t *testing.T) {
	gw := gatewayv1.ObjectName("gw")
	cases := []struct {
		name string
		obj  metav1.Object
		cfg  *config.Config
		want bool
	}{
		{
			name: "auto + no filter",
			obj:  makeRoute("r", []gatewayv1.Hostname{"x"}, nil, nil),
			cfg:  &config.Config{AutoHTTPRoute: true},
			want: true,
		},
		{
			name: "gateway filter mismatch",
			obj:  makeRoute("r", []gatewayv1.Hostname{"x"}, []gatewayv1.ParentReference{{Name: gw}}, nil),
			cfg:  &config.Config{AutoHTTPRoute: true, GatewayNames: config.StringSet{"other"}},
			want: false,
		},
		{
			name: "gateway filter match",
			obj:  makeRoute("r", []gatewayv1.Hostname{"x"}, []gatewayv1.ParentReference{{Name: gw}}, nil),
			cfg:  &config.Config{AutoHTTPRoute: true, GatewayNames: config.StringSet{"gw"}},
			want: true,
		},
		{
			name: "matches any of multiple gateway names",
			obj:  makeRoute("r", []gatewayv1.Hostname{"x"}, []gatewayv1.ParentReference{{Name: gw}}, nil),
			cfg:  &config.Config{AutoHTTPRoute: true, GatewayNames: config.StringSet{"other", "gw", "third"}},
			want: true,
		},
		{
			name: "rejects when none of multiple gateway names match",
			obj:  makeRoute("r", []gatewayv1.Hostname{"x"}, []gatewayv1.ParentReference{{Name: gw}}, nil),
			cfg:  &config.Config{AutoHTTPRoute: true, GatewayNames: config.StringSet{"a", "b"}},
			want: false,
		},
		{
			name: "no auto, annotation present",
			obj:  makeRoute("r", []gatewayv1.Hostname{"x"}, nil, map[string]string{config.DefaultEnabledAnnotation: "true"}),
			cfg: &config.Config{
				EnabledAnnotation:  config.DefaultEnabledAnnotation,
				TemplateAnnotation: config.DefaultTemplateAnnotation,
			},
			want: true,
		},
		{
			name: "non-route",
			obj:  &corev1.Pod{},
			cfg:  &config.Config{AutoHTTPRoute: true},
			want: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := (HTTPRoute{}).Matches(tt.obj, tt.cfg); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPRoute_DefaultConditionsAndGuardHost(t *testing.T) {
	if got := (HTTPRoute{}).DefaultConditions(); len(got) != 1 || got[0] != "[STATUS] == 200" {
		t.Errorf("DefaultConditions() = %v", got)
	}
	if got := (HTTPRoute{}).GuardHost(makeRoute("a", []gatewayv1.Hostname{"guarded.example.com"}, nil, nil)); got != "guarded.example.com" {
		t.Errorf("GuardHost() = %q", got)
	}
	if got := (HTTPRoute{}).GuardHost(&corev1.Pod{}); got != "" {
		t.Errorf("GuardHost(non-route) = %q, want \"\"", got)
	}
}

func TestHTTPRoute_ParentAnnotations(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(gatewayGVR.GroupVersion().WithKind("Gateway"), &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(gatewayGVR.GroupVersion().WithKind("GatewayList"), &unstructured.UnstructuredList{})

	client := fake.NewSimpleDynamicClient(scheme)

	gw := &unstructured.Unstructured{}
	gw.SetAPIVersion("gateway.networking.k8s.io/v1")
	gw.SetKind("Gateway")
	gw.SetName("gw")
	gw.SetNamespace("default")
	gw.SetAnnotations(map[string]string{"parent": "annotation"})
	if _, err := client.Resource(gatewayGVR).Namespace("default").Create(context.Background(), gw, metav1.CreateOptions{}); err != nil {
		t.Fatalf("seed gateway: %v", err)
	}

	route := makeRoute("r", []gatewayv1.Hostname{"x"}, []gatewayv1.ParentReference{{Name: "gw"}}, nil)
	ann := (HTTPRoute{}).ParentAnnotations(context.Background(), route, k8s.NewFetcher(client))
	if ann["parent"] != "annotation" {
		t.Errorf("got %v", ann)
	}
}

func TestHTTPRoute_ParentAnnotations_NoParents(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	route := makeRoute("r", []gatewayv1.Hostname{"x"}, nil, nil)
	if ann := (HTTPRoute{}).ParentAnnotations(context.Background(), route, k8s.NewFetcher(client)); ann != nil {
		t.Errorf("got %v, want nil", ann)
	}
}

func TestHTTPRoute_ParentAnnotations_NonGatewayKind(t *testing.T) {
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)
	kind := gatewayv1.Kind("Service")
	route := makeRoute("r", []gatewayv1.Hostname{"x"}, []gatewayv1.ParentReference{{Name: "svc", Kind: &kind}}, nil)
	if ann := (HTTPRoute{}).ParentAnnotations(context.Background(), route, k8s.NewFetcher(client)); ann != nil {
		t.Errorf("got %v, want nil", ann)
	}
}
