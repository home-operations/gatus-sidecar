package resources

import (
	"context"
	"testing"

	"github.com/home-operations/gatus-sidecar/internal/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeService(name, ns string, port int32, protocol corev1.Protocol) *corev1.Service {
	return &corev1.Service{
		Name: name, Namespace: ns,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: port, Protocol: protocol}},
		},
	}
}

func TestService_URL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		svc    metav1.Object
		domain string
		want   string
	}{
		{"tcp", makeService("a", "ns", 8080, corev1.ProtocolTCP), "cluster.local", "tcp://a.ns.svc.cluster.local.:8080"},
		{"udp", makeService("dns", "kube-system", 53, corev1.ProtocolUDP), "cluster.local", "udp://dns.kube-system.svc.cluster.local.:53"},
		{"custom cluster domain", makeService("a", "ns", 80, corev1.ProtocolTCP), "k8s.example", "tcp://a.ns.svc.k8s.example.:80"},
		{"empty cluster domain falls back to the default", makeService("a", "ns", 80, corev1.ProtocolTCP), "", "tcp://a.ns.svc.cluster.local.:80"},
		{"dots-only cluster domain falls back to the default", makeService("a", "ns", 80, corev1.ProtocolTCP), ".", "tcp://a.ns.svc.cluster.local.:80"},
		{"trailing dot on the configured domain is not doubled", makeService("a", "ns", 80, corev1.ProtocolTCP), "cluster.local.", "tcp://a.ns.svc.cluster.local.:80"},
		{"default protocol", &corev1.Service{
			Name: "a", Namespace: "n",
			Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}},
		}, "cluster.local", "tcp://a.n.svc.cluster.local.:80"},
		{"no ports", &corev1.Service{Name: "a"}, "cluster.local", ""},
		{"wrong type", &corev1.Pod{}, "cluster.local", ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &config.Config{ClusterDomain: tt.domain}
			if got := (Service{}).URL(tt.svc, cfg); got != tt.want {
				t.Errorf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestService_DefaultConditionsAndMatches(t *testing.T) {
	t.Parallel()
	if got := (Service{}).DefaultConditions(); len(got) != 1 || got[0] != "[CONNECTED] == true" {
		t.Errorf("DefaultConditions() = %v", got)
	}

	if !(Service{}).Matches(makeService("a", "n", 80, corev1.ProtocolTCP), &config.Config{Kinds: autoEnabled(config.KindService)}) {
		t.Error("auto mode should match")
	}
	if (Service{}).Matches(makeService("a", "n", 80, corev1.ProtocolTCP), &config.Config{EnabledAnnotation: "x", TemplateAnnotation: "y"}) {
		t.Error("no auto + no annotations should not match")
	}
}

func TestService_GuardHostAndParentAnnotations_NoOps(t *testing.T) {
	t.Parallel()
	if got := (Service{}).GuardHost(makeService("a", "n", 80, corev1.ProtocolTCP)); got != "" {
		t.Errorf("GuardHost() = %q, want \"\"", got)
	}
	if ann := (Service{}).ParentAnnotations(context.Background(), makeService("a", "n", 80, corev1.ProtocolTCP), nil); ann != nil {
		t.Errorf("ParentAnnotations should always return nil, got %v", ann)
	}
}
