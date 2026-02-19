package ingressroute

import (
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestURLExtractor(t *testing.T) {
	tests := []struct {
		name string
		obj  metav1.Object
		want string
	}{
		{
			name: "extracts HTTP URL from IngressRoute without TLS",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "traefik.io/v1alpha1",
					"kind":       "IngressRoute",
					"metadata": map[string]any{
						"name":      "my-route",
						"namespace": "default",
					},
					"spec": map[string]any{
						"routes": []any{
							map[string]any{
								"match": "Host(`example.com`)",
							},
						},
					},
				},
			},
			want: "http://example.com",
		},
		{
			name: "extracts HTTPS URL from IngressRoute with TLS",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "traefik.io/v1alpha1",
					"kind":       "IngressRoute",
					"metadata": map[string]any{
						"name":      "secure-route",
						"namespace": "default",
					},
					"spec": map[string]any{
						"routes": []any{
							map[string]any{
								"match": "Host(`secure.example.com`)",
							},
						},
						"tls": map[string]any{
							"secretName": "tls-secret",
						},
					},
				},
			},
			want: "https://secure.example.com",
		},
		{
			name: "returns empty for IngressRoute without routes",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "traefik.io/v1alpha1",
					"kind":       "IngressRoute",
					"metadata": map[string]any{
						"name": "empty-route",
					},
					"spec": map[string]any{
						"routes": []any{},
					},
				},
			},
			want: "",
		},
		{
			name: "uses first host when multiple routes",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "traefik.io/v1alpha1",
					"kind":       "IngressRoute",
					"metadata": map[string]any{
						"name": "multi-route",
					},
					"spec": map[string]any{
						"routes": []any{
							map[string]any{
								"match": "Host(`first.example.com`)",
							},
							map[string]any{
								"match": "Host(`second.example.com`)",
							},
						},
					},
				},
			},
			want: "http://first.example.com",
		},
		{
			name: "returns empty for non-Unstructured object",
			obj:  &unstructured.Unstructured{},
			want: "",
		},
		{
			name: "handles complex match expression",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "traefik.io/v1alpha1",
					"kind":       "IngressRoute",
					"metadata": map[string]any{
						"name": "complex-route",
					},
					"spec": map[string]any{
						"routes": []any{
							map[string]any{
								"match": "Host(`api.example.com`) && PathPrefix(`/v1`)",
							},
						},
					},
				},
			},
			want: "http://api.example.com",
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

func TestConditionFunc(t *testing.T) {
	cfg := &config.Config{}
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"name": "test",
			},
		},
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
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"name": "test",
			},
			"spec": map[string]any{
				"routes": []any{
					map[string]any{
						"match": "Host(`guarded.example.com`)",
					},
				},
			},
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

func TestGuardedFuncNonUnstructured(t *testing.T) {
	obj := &unstructured.Unstructured{}
	e := &endpoint.Endpoint{Guarded: true}

	guardedFunc(obj, e)

	if e.DNS != nil {
		t.Error("DNS should remain nil for empty Unstructured objects")
	}
}

func TestGetFirstHostname(t *testing.T) {
	tests := []struct {
		name string
		obj  *unstructured.Unstructured
		want string
	}{
		{
			name: "extracts hostname from match",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"routes": []any{
							map[string]any{
								"match": "Host(`example.com`)",
							},
						},
					},
				},
			},
			want: "example.com",
		},
		{
			name: "returns empty when no routes",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"routes": []any{},
					},
				},
			},
			want: "",
		},
		{
			name: "returns empty when spec missing",
			obj: &unstructured.Unstructured{
				Object: map[string]any{},
			},
			want: "",
		},
		{
			name: "returns empty when routes missing",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{},
				},
			},
			want: "",
		},
		{
			name: "skips routes without match",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"routes": []any{
							map[string]any{},
							map[string]any{
								"match": "Host(`found.example.com`)",
							},
						},
					},
				},
			},
			want: "found.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFirstHostname(tt.obj)
			if got != tt.want {
				t.Errorf("getFirstHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractHostFromMatch(t *testing.T) {
	tests := []struct {
		name  string
		match string
		want  string
	}{
		{
			name:  "simple host match",
			match: "Host(`example.com`)",
			want:  "example.com",
		},
		{
			name:  "host with path prefix",
			match: "Host(`api.example.com`) && PathPrefix(`/v1`)",
			want:  "api.example.com",
		},
		{
			name:  "host with method",
			match: "Method(`GET`) && Host(`test.example.com`)",
			want:  "test.example.com",
		},
		{
			name:  "no host in match",
			match: "PathPrefix(`/api`)",
			want:  "",
		},
		{
			name:  "empty match",
			match: "",
			want:  "",
		},
		{
			name:  "host with port",
			match: "Host(`example.com:8080`)",
			want:  "example.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHostFromMatch(tt.match)
			if got != tt.want {
				t.Errorf("extractHostFromMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasTLS(t *testing.T) {
	tests := []struct {
		name string
		obj  *unstructured.Unstructured
		want bool
	}{
		{
			name: "has TLS config",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"tls": map[string]any{
							"secretName": "tls-secret",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "no TLS config",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{},
				},
			},
			want: false,
		},
		{
			name: "nil TLS config",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"tls": nil,
					},
				},
			},
			want: false,
		},
		{
			name: "missing spec",
			obj: &unstructured.Unstructured{
				Object: map[string]any{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasTLS(tt.obj)
			if got != tt.want {
				t.Errorf("hasTLS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefinition(t *testing.T) {
	def := Definition()

	if def.GVR.Group != "traefik.io" {
		t.Errorf("GVR.Group = %v, want traefik.io", def.GVR.Group)
	}
	if def.GVR.Version != "v1alpha1" {
		t.Errorf("GVR.Version = %v, want v1alpha1", def.GVR.Version)
	}
	if def.GVR.Resource != "ingressroutes" {
		t.Errorf("GVR.Resource = %v, want ingressroutes", def.GVR.Resource)
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
	if def.ConvertFunc == nil {
		t.Error("ConvertFunc should not be nil")
	}

	cfg := &config.Config{AutoIngressRoute: true}
	if !def.AutoConfigFunc(cfg) {
		t.Error("AutoConfigFunc should return true when AutoIngressRoute is enabled")
	}

	cfg.AutoIngressRoute = false
	if def.AutoConfigFunc(cfg) {
		t.Error("AutoConfigFunc should return false when AutoIngressRoute is disabled")
	}
}

func TestConvertFunc(t *testing.T) {
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "IngressRoute",
			"metadata": map[string]any{
				"name":      "test-route",
				"namespace": "default",
			},
		},
	}

	obj, err := convertFunc(u)
	if err != nil {
		t.Fatalf("convertFunc() error = %v", err)
	}

	if obj.GetName() != "test-route" {
		t.Errorf("GetName() = %v, want test-route", obj.GetName())
	}
	if obj.GetNamespace() != "default" {
		t.Errorf("GetNamespace() = %v, want default", obj.GetNamespace())
	}
}
