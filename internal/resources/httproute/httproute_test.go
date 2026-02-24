package httproute

import (
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestURLExtractor(t *testing.T) {
	tests := []struct {
		name string
		obj  metav1.Object
		want string
	}{
		{
			name: "extracts HTTPS URL from HTTPRoute with hostname",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{
						"api.example.com",
						"api2.example.com",
					},
				},
			},
			want: "https://api.example.com",
		},
		{
			name: "returns empty for HTTPRoute without hostnames",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{},
				},
			},
			want: "",
		},
		{
			name: "uses first hostname when multiple",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name: "multi-route",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{
						"first.example.com",
						"second.example.com",
					},
				},
			},
			want: "https://first.example.com",
		},
		{
			name: "returns URL as-is if already has http prefix",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name: "prefixed-route",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{
						"http://already-prefixed.com",
					},
				},
			},
			want: "http://already-prefixed.com",
		},
		{
			name: "adds HTTPS prefix when hostname starts with 'http' but is not a URL",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name: "http-debug-route",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{
						"http-test.domain.com",
					},
				},
			},
			want: "https://http-test.domain.com",
		},
		{
			name: "returns empty for non-HTTPRoute object",
			obj:  &corev1.Pod{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urlExtractor(tt.obj)
			if got != tt.want {
				t.Errorf("urlExtractor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterFunc(t *testing.T) {
	gatewayName := gatewayv1.ObjectName("my-gateway")
	otherGateway := gatewayv1.ObjectName("other-gateway")

	tests := []struct {
		name string
		obj  metav1.Object
		cfg  *config.Config
		want bool
	}{
		{
			name: "no filter - allows all routes",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			},
			cfg:  &config.Config{},
			want: true,
		},
		{
			name: "filter by gateway name matches",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: gatewayv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayv1.CommonRouteSpec{
						ParentRefs: []gatewayv1.ParentReference{
							{Name: gatewayName},
						},
					},
				},
			},
			cfg:  &config.Config{GatewayName: "my-gateway"},
			want: true,
		},
		{
			name: "filter by gateway name does not match",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: gatewayv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayv1.CommonRouteSpec{
						ParentRefs: []gatewayv1.ParentReference{
							{Name: otherGateway},
						},
					},
				},
			},
			cfg:  &config.Config{GatewayName: "my-gateway"},
			want: false,
		},
		{
			name: "no parent refs passes filter when no gateway filter",
			obj: &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec:       gatewayv1.HTTPRouteSpec{},
			},
			cfg:  &config.Config{},
			want: true,
		},
		{
			name: "non-HTTPRoute object returns false",
			obj:  &corev1.Pod{},
			cfg:  &config.Config{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterFunc(tt.obj, tt.cfg)
			if got != tt.want {
				t.Errorf("filterFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConditionFunc(t *testing.T) {
	cfg := &config.Config{}
	obj := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}
	e := &endpoint.Endpoint{}

	conditionFunc(cfg, obj, e)

	if len(e.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(e.Conditions))
	}
	if e.Conditions[0] != "[STATUS] == 200" {
		t.Errorf("Condition = %v, want [STATUS] == 200", e.Conditions[0])
	}
}

func TestGuardedFunc(t *testing.T) {
	obj := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{"guarded.example.com"},
		},
	}
	e := &endpoint.Endpoint{Guarded: true}

	guardedFunc(obj, e)

	if e.URL != dnsTestURL {
		t.Errorf("URL = %v, want %v", e.URL, dnsTestURL)
	}
	if e.DNS == nil {
		t.Error("DNS config should not be nil")
	}
	if e.DNS["query-name"] != "guarded.example.com" {
		t.Errorf("DNS query-name = %v, want guarded.example.com", e.DNS["query-name"])
	}
	if e.DNS["query-type"] != dnsQueryType {
		t.Errorf("DNS query-type = %v, want %v", e.DNS["query-type"], dnsQueryType)
	}
	if len(e.Conditions) != 1 || e.Conditions[0] != dnsEmptyBodyCondition {
		t.Errorf("Conditions = %v, want [%v]", e.Conditions, dnsEmptyBodyCondition)
	}
}

func TestGuardedFuncNonHTTPRoute(t *testing.T) {
	obj := &corev1.Pod{}
	e := &endpoint.Endpoint{Guarded: true}

	guardedFunc(obj, e)

	if e.DNS != nil {
		t.Error("DNS should remain nil for non-HTTPRoute objects")
	}
}

func TestGetFirstHostname(t *testing.T) {
	tests := []struct {
		name  string
		route *gatewayv1.HTTPRoute
		want  string
	}{
		{
			name: "returns first hostname",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{
						"first.example.com",
						"second.example.com",
					},
				},
			},
			want: "first.example.com",
		},
		{
			name: "returns empty string when no hostnames",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					Hostnames: []gatewayv1.Hostname{},
				},
			},
			want: "",
		},
		{
			name: "returns empty string when nil hostnames",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFirstHostname(tt.route)
			if got != tt.want {
				t.Errorf("getFirstHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReferencesGateway(t *testing.T) {
	targetGateway := gatewayv1.ObjectName("target-gateway")
	otherGateway := gatewayv1.ObjectName("other-gateway")

	tests := []struct {
		name        string
		route       *gatewayv1.HTTPRoute
		gatewayName string
		want        bool
	}{
		{
			name: "references target gateway",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayv1.CommonRouteSpec{
						ParentRefs: []gatewayv1.ParentReference{
							{Name: targetGateway},
						},
					},
				},
			},
			gatewayName: "target-gateway",
			want:        true,
		},
		{
			name: "does not reference target gateway",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayv1.CommonRouteSpec{
						ParentRefs: []gatewayv1.ParentReference{
							{Name: otherGateway},
						},
					},
				},
			},
			gatewayName: "target-gateway",
			want:        false,
		},
		{
			name: "references gateway in multiple parents",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayv1.CommonRouteSpec{
						ParentRefs: []gatewayv1.ParentReference{
							{Name: otherGateway},
							{Name: targetGateway},
						},
					},
				},
			},
			gatewayName: "target-gateway",
			want:        true,
		},
		{
			name: "no parent refs",
			route: &gatewayv1.HTTPRoute{
				Spec: gatewayv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayv1.CommonRouteSpec{
						ParentRefs: []gatewayv1.ParentReference{},
					},
				},
			},
			gatewayName: "target-gateway",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := referencesGateway(tt.route, tt.gatewayName)
			if got != tt.want {
				t.Errorf("referencesGateway() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyGuardedTemplate(t *testing.T) {
	e := &endpoint.Endpoint{}
	hostname := "test.example.com"

	applyGuardedTemplate(hostname, e)

	if e.URL != dnsTestURL {
		t.Errorf("URL = %v, want %v", e.URL, dnsTestURL)
	}
	if e.DNS == nil {
		t.Error("DNS should not be nil")
	}
	if e.DNS["query-name"] != hostname {
		t.Errorf("DNS query-name = %v, want %v", e.DNS["query-name"], hostname)
	}
	if e.DNS["query-type"] != dnsQueryType {
		t.Errorf("DNS query-type = %v, want %v", e.DNS["query-type"], dnsQueryType)
	}
	if len(e.Conditions) != 1 || e.Conditions[0] != dnsEmptyBodyCondition {
		t.Errorf("Conditions = %v, want [%v]", e.Conditions, dnsEmptyBodyCondition)
	}
}

func TestDefinition(t *testing.T) {
	def := Definition()

	if def.GVR.Group != "gateway.networking.k8s.io" {
		t.Errorf("GVR.Group = %v, want gateway.networking.k8s.io", def.GVR.Group)
	}
	if def.GVR.Version != "v1" {
		t.Errorf("GVR.Version = %v, want v1", def.GVR.Version)
	}
	if def.GVR.Resource != "httproutes" {
		t.Errorf("GVR.Resource = %v, want httproutes", def.GVR.Resource)
	}
	if def.URLExtractor == nil {
		t.Error("URLExtractor should not be nil")
	}
	if def.ConditionFunc == nil {
		t.Error("ConditionFunc should not be nil")
	}
	if def.GuardedFunc == nil {
		t.Error("GuardedFunc should not be nil")
	}
	if def.FilterFunc == nil {
		t.Error("FilterFunc should not be nil")
	}
	if def.ParentExtractor == nil {
		t.Error("ParentExtractor should not be nil")
	}

	cfg := &config.Config{AutoHTTPRoute: true}
	if !def.AutoConfigFunc(cfg) {
		t.Error("AutoConfigFunc should return true when AutoHTTPRoute is enabled")
	}
}
