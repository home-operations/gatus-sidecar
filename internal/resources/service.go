package resources

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var serviceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "services",
}

type Service struct{}

func (Service) GVR() schema.GroupVersionResource { return serviceGVR }

func (Service) Prefix(cfg *config.Config) string { return cfg.ServicePrefix }

func (Service) Convert(u *unstructured.Unstructured) (metav1.Object, error) {
	return convertTo[corev1.Service](u)
}

func (Service) Matches(obj metav1.Object, cfg *config.Config) bool {
	if _, ok := obj.(*corev1.Service); !ok {
		return false
	}
	return matchesAnnotation(obj, cfg.AutoService, cfg)
}

func (Service) URL(obj metav1.Object) string {
	svc, ok := obj.(*corev1.Service)
	if !ok || len(svc.Spec.Ports) == 0 {
		return ""
	}
	port := svc.Spec.Ports[0]
	protocol := strings.ToLower(string(cmp.Or(port.Protocol, corev1.ProtocolTCP)))
	return fmt.Sprintf("%s://%s.%s.svc:%d", protocol, svc.Name, svc.Namespace, port.Port)
}

func (Service) DefaultConditions() []string { return tcpDefaultConditions }

// Services have no meaningful guarded mode.
func (Service) GuardHost(metav1.Object) string { return "" }

func (Service) ParentAnnotations(context.Context, metav1.Object, k8s.Fetcher) map[string]string {
	return nil
}
