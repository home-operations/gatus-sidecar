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
	"k8s.io/client-go/dynamic"

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

	// If AutoIngresses is disabled, only process if it has the annotation
	if !cfg.AutoIngresses {
		annotations := ingress.GetAnnotations()
		if annotations == nil {
			return false
		}

		_, hasEnabledAnnotation := annotations[cfg.EnabledAnnotation]
		_, hasTemplateAnnotation := annotations[cfg.TemplateAnnotation]

		return hasEnabledAnnotation || hasTemplateAnnotation
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

func (h *IngressHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return
	}

	if cfg.AutoGroup {
		ingressClass := getIngressClass(ingress)
		if ingressClass != "" {
			endpoint.Group = ingressClass
		}
	}

	endpoint.Conditions = []string{"[STATUS] == 200"}
}

func (h *IngressHandler) GetParentAnnotations(ctx context.Context, obj metav1.Object) map[string]string {
	return nil
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
func NewIngressController(stateManager *manager.Manager, dynamicClient dynamic.Interface) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "ingresses",
		},
		options:       metav1.ListOptions{},
		handler:       &IngressHandler{},
		stateManager:  stateManager,
		dynamicClient: dynamicClient,
		convert: func(u *unstructured.Unstructured) (metav1.Object, error) {
			ingress := &networkingv1.Ingress{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ingress); err != nil {
				return nil, fmt.Errorf("failed to convert to Ingress: %w", err)
			}
			return ingress, nil
		},
	}
}
