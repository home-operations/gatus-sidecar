package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/handler"
	"github.com/home-operations/gatus-sidecar/internal/manager"
)

// IngressHandler handles Ingress resources
type IngressHandler struct{}

// Ensure IngressHandler implements the ResourceHandler interface
var _ handler.ResourceHandler = (*IngressHandler)(nil)

func (h *IngressHandler) ShouldProcess(obj metav1.Object, cfg *config.Config) bool {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return false
	}

	if cfg.IngressClass != "" && !hasIngressClass(ingress, cfg.IngressClass) {
		return false
	}

	return true
}

func (h *IngressHandler) ExtractURL(obj metav1.Object) string {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return ""
	}

	url := firstIngressHostname(ingress)
	if url == "" {
		return ""
	}

	// Determine protocol based on TLS configuration
	protocol := "http"
	if hasIngressTLS(ingress, url) {
		protocol = "https"
	}

	if !strings.HasPrefix(url, "http") {
		url = fmt.Sprintf("%s://%s", protocol, url)
	}

	return url
}

func (h *IngressHandler) GetResourceName() string {
	return "ingress"
}

func (h *IngressHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) bool {
	if cfg.AutoGroup {
		ingress, ok := obj.(*networkingv1.Ingress)
		if !ok {
			return false
		}

		// Group by the first ParentRef (gateway) name
		ingressClass := getIngressClass(ingress)
		if ingressClass == "" {
			return false
		}

		// Group by IngressClass
		endpoint.Group = ingressClass
		return true
	}

	return false
}

// Helper functions for Ingress
func firstIngressHostname(ingress *networkingv1.Ingress) string {
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			return rule.Host
		}
	}

	return ""
}

func hasIngressTLS(ingress *networkingv1.Ingress, hostname string) bool {
	for _, tls := range ingress.Spec.TLS {
		if slices.Contains(tls.Hosts, hostname) {
			return true
		}
	}

	return false
}

func hasIngressClass(ingress *networkingv1.Ingress, ingressClass string) bool {
	return getIngressClass(ingress) == ingressClass
}

func getIngressClass(ingress *networkingv1.Ingress) string {
	// Check spec.ingressClassName first (preferred)
	if ingress.Spec.IngressClassName != nil {
		return *ingress.Spec.IngressClassName
	}

	// Fallback to annotation (legacy)
	if ingress.Annotations != nil {
		if class, ok := ingress.Annotations["kubernetes.io/ingress.class"]; ok {
			return class
		}
	}

	return ""
}

// NewIngressController creates a controller for Ingress resources
func NewIngressController(resourceHandler handler.ResourceHandler, stateManager *manager.Manager) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "ingresses",
		},
		handler:      resourceHandler,
		stateManager: stateManager,
		convert: func(u *unstructured.Unstructured) (metav1.Object, error) {
			ingress := &networkingv1.Ingress{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ingress); err != nil {
				return nil, fmt.Errorf("failed to convert to Ingress: %w", err)
			}
			return ingress, nil
		},
	}
}

func RunIngress(ctx context.Context, cfg *config.Config) error {
	stateManager := manager.NewManager(cfg.Output)
	handler := &IngressHandler{}
	ctrl := NewIngressController(handler, stateManager)
	return ctrl.Run(ctx, cfg)
}
