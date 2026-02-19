package ingress

import (
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestURLExtractor(t *testing.T) {
	https := networkingv1.IngressTLS{
		Hosts: []string{"secure.example.com"},
	}

	tests := []struct {
		name string
		obj  metav1.Object
		want string
	}{
		{
			name: "extracts HTTP URL from ingress without TLS",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "example.com",
						},
					},
				},
			},
			want: "http://example.com",
		},
		{
			name: "extracts HTTPS URL from ingress with TLS",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secure-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{https},
					Rules: []networkingv1.IngressRule{
						{
							Host: "secure.example.com",
						},
					},
				},
			},
			want: "https://secure.example.com",
		},
		{
			name: "returns empty for ingress without rules",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{},
				},
			},
			want: "",
		},
		{
			name: "uses first hostname when multiple rules",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multi-ingress",
					Namespace: "default",
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "first.example.com"},
						{Host: "second.example.com"},
					},
				},
			},
			want: "http://first.example.com",
		},
		{
			name: "returns URL as-is if already has http prefix",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "prefixed-ingress",
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "http://already-prefixed.com"},
					},
				},
			},
			want: "http://already-prefixed.com",
		},
		{
			name: "returns empty for non-ingress object",
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
	nginxClass := "nginx"
	traefikClass := "traefik"

	tests := []struct {
		name string
		obj  metav1.Object
		cfg  *config.Config
		want bool
	}{
		{
			name: "no filter - allows all ingresses",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			},
			cfg:  &config.Config{},
			want: true,
		},
		{
			name: "filter by ingress class name matches",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: networkingv1.IngressSpec{
					IngressClassName: &nginxClass,
				},
			},
			cfg:  &config.Config{IngressClass: "nginx"},
			want: true,
		},
		{
			name: "filter by ingress class name does not match",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec: networkingv1.IngressSpec{
					IngressClassName: &nginxClass,
				},
			},
			cfg:  &config.Config{IngressClass: "traefik"},
			want: false,
		},
		{
			name: "filter by annotation matches (legacy)",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "traefik",
					},
				},
			},
			cfg:  &config.Config{IngressClass: "traefik"},
			want: true,
		},
		{
			name: "spec.ingressClassName takes precedence over annotation",
			obj: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: &traefikClass,
				},
			},
			cfg:  &config.Config{IngressClass: "traefik"},
			want: true,
		},
		{
			name: "non-ingress object returns false",
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
	obj := &networkingv1.Ingress{
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
	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{Host: "guarded.example.com"},
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

func TestGetFirstHostname(t *testing.T) {
	tests := []struct {
		name    string
		ingress *networkingv1.Ingress
		want    string
	}{
		{
			name: "returns first hostname",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: "first.example.com"},
						{Host: "second.example.com"},
					},
				},
			},
			want: "first.example.com",
		},
		{
			name: "returns empty string when no rules",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{},
				},
			},
			want: "",
		},
		{
			name: "skips empty hostnames",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{Host: ""},
						{Host: "valid.example.com"},
					},
				},
			},
			// FIX: The expectation here was "" which is incorrect.
			// The function is designed to skip empty ones and find the valid one.
			want: "valid.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFirstHostname(tt.ingress)
			if got != tt.want {
				t.Errorf("getFirstHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasTLS(t *testing.T) {
	tests := []struct {
		name     string
		ingress  *networkingv1.Ingress
		hostname string
		want     bool
	}{
		{
			name: "has TLS for hostname",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"secure.example.com", "another.example.com"}},
					},
				},
			},
			hostname: "secure.example.com",
			want:     true,
		},
		{
			name: "no TLS for hostname",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{
						{Hosts: []string{"other.example.com"}},
					},
				},
			},
			hostname: "secure.example.com",
			want:     false,
		},
		{
			name: "no TLS entries",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{},
				},
			},
			hostname: "secure.example.com",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasTLS(tt.ingress, tt.hostname)
			if got != tt.want {
				t.Errorf("hasTLS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIngressClass(t *testing.T) {
	nginxClass := "nginx"

	tests := []struct {
		name    string
		ingress *networkingv1.Ingress
		want    string
	}{
		{
			name: "returns spec.ingressClassName",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					IngressClassName: &nginxClass,
				},
			},
			want: "nginx",
		},
		{
			name: "returns annotation when no spec class",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "traefik",
					},
				},
			},
			want: "traefik",
		},
		{
			name: "spec takes precedence over annotation",
			ingress: &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "annotation-class",
					},
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: &nginxClass,
				},
			},
			want: "nginx",
		},
		{
			name:    "returns empty when no class",
			ingress: &networkingv1.Ingress{},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getIngressClass(tt.ingress)
			if got != tt.want {
				t.Errorf("getIngressClass() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetermineProtocol(t *testing.T) {
	https := networkingv1.IngressTLS{
		Hosts: []string{"secure.example.com"},
	}

	tests := []struct {
		name     string
		ingress  *networkingv1.Ingress
		hostname string
		want     string
	}{
		{
			name: "returns https when TLS present",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{https},
				},
			},
			hostname: "secure.example.com",
			want:     "https",
		},
		{
			name: "returns http when no TLS",
			ingress: &networkingv1.Ingress{
				Spec: networkingv1.IngressSpec{},
			},
			hostname: "insecure.example.com",
			want:     "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineProtocol(tt.ingress, tt.hostname)
			if got != tt.want {
				t.Errorf("determineProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefinition(t *testing.T) {
	def := Definition()

	if def.GVR.Group != "networking.k8s.io" {
		t.Errorf("GVR.Group = %v, want networking.k8s.io", def.GVR.Group)
	}
	if def.GVR.Version != "v1" {
		t.Errorf("GVR.Version = %v, want v1", def.GVR.Version)
	}
	if def.GVR.Resource != "ingresses" {
		t.Errorf("GVR.Resource = %v, want ingresses", def.GVR.Resource)
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
	if def.AutoConfigFunc == nil {
		t.Error("AutoConfigFunc should not be nil")
	}

	cfg := &config.Config{AutoIngress: true}
	if !def.AutoConfigFunc(cfg) {
		t.Error("AutoConfigFunc should return true when AutoIngress is enabled")
	}
}
