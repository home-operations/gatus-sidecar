package service

import (
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestURLExtractor(t *testing.T) {
	tests := []struct {
		name string
		obj  metav1.Object
		want string
	}{
		{
			name: "extracts URL from service with TCP port",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-service",
					Namespace: "production",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port:     8080,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
			want: "tcp://my-service.production.svc:8080",
		},
		{
			name: "extracts URL from service with UDP port",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dns-service",
					Namespace: "kube-system",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port:     53,
							Protocol: corev1.ProtocolUDP,
						},
					},
				},
			},
			want: "udp://dns-service.kube-system.svc:53",
		},
		{
			name: "returns empty for service without ports",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "headless-service",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{},
				},
			},
			want: "",
		},
		{
			name: "uses first port when multiple ports",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multi-port",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port:     80,
							Protocol: corev1.ProtocolTCP,
						},
						{
							Port:     443,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
			want: "tcp://multi-port.default.svc:80",
		},
		{
			name: "returns empty for non-service object",
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

func TestConditionFunc(t *testing.T) {
	cfg := &config.Config{}
	obj := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}
	e := &endpoint.Endpoint{}

	conditionFunc(cfg, obj, e)

	if len(e.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(e.Conditions))
	}
	if e.Conditions[0] != "[CONNECTED] == true" {
		t.Errorf("Condition = %v, want [CONNECTED] == true", e.Conditions[0])
	}
}

func TestDefinition(t *testing.T) {
	def := Definition()

	if def.GVR.Group != "" {
		t.Errorf("GVR.Group = %v, want empty string", def.GVR.Group)
	}
	if def.GVR.Version != "v1" {
		t.Errorf("GVR.Version = %v, want v1", def.GVR.Version)
	}
	if def.GVR.Resource != "services" {
		t.Errorf("GVR.Resource = %v, want services", def.GVR.Resource)
	}
	if def.URLExtractor == nil {
		t.Error("URLExtractor should not be nil")
	}
	if def.ConditionFunc == nil {
		t.Error("ConditionFunc should not be nil")
	}
	if def.AutoConfigFunc == nil {
		t.Error("AutoConfigFunc should not be nil")
	}

	cfg := &config.Config{AutoService: true}
	if !def.AutoConfigFunc(cfg) {
		t.Error("AutoConfigFunc should return true when AutoService is enabled")
	}

	cfg.AutoService = false
	if def.AutoConfigFunc(cfg) {
		t.Error("AutoConfigFunc should return false when AutoService is disabled")
	}
}
