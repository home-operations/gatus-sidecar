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
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: port, Protocol: protocol}},
		},
	}
}

func TestService_URL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		svc  metav1.Object
		want string
	}{
		{"tcp", makeService("a", "ns", 8080, corev1.ProtocolTCP), "tcp://a.ns.svc:8080"},
		{"udp", makeService("dns", "kube-system", 53, corev1.ProtocolUDP), "udp://dns.kube-system.svc:53"},
		{"default protocol", &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "n"},
			Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}},
		}, "tcp://a.n.svc:80"},
		{"no ports", &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "a"}}, ""},
		{"wrong type", &corev1.Pod{}, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := (Service{}).URL(tt.svc); got != tt.want {
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
