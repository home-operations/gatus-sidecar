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

const (
	networkingAPIGroup     = "networking.k8s.io"
	networkingAPIVersion   = "v1"
	ingressesResource      = "ingresses"
	ingressClassAnnotation = "kubernetes.io/ingress.class"
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

	// Check ingress class filter first (most restrictive)
	if cfg.IngressClass != "" && !h.hasIngressClass(ingress, cfg.IngressClass) {
		return false
	}

	// If AutoIngress is enabled, process all ingresses (that passed class filter)
	if cfg.AutoIngress {
		return true
	}

	// If AutoIngress is disabled, only process if it has required annotations
	return hasRequiredAnnotations(ingress, cfg)
}

func (h *IngressHandler) ExtractURL(obj metav1.Object) string {
	ingress, ok := obj.(*networkingv1.Ingress)
	if !ok {
		return ""
	}

	hostname := h.getFirstHostname(ingress)
	if hostname == "" {
		return ""
	}

	protocol := h.determineProtocol(ingress, hostname)

	if !strings.HasPrefix(hostname, httpPrefix) {
		return fmt.Sprintf("%s://%s", protocol, hostname)
	}

	return hostname
}

func (h *IngressHandler) determineProtocol(ingress *networkingv1.Ingress, hostname string) string {
	if h.hasIngressTLS(ingress, hostname) {
		return httpsProtocol
	}
	return httpProtocol
}

func (h *IngressHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	if endpoint.Guarded {
		if ingress, ok := obj.(*networkingv1.Ingress); ok {
			applyGuardedTemplate(h.getFirstHostname(ingress), endpoint)
		}
	} else {
		endpoint.Conditions = []string{ingressCondition}
	}
}

func (h *IngressHandler) GetParentAnnotations(ctx context.Context, obj metav1.Object) map[string]string {
	return nil
}

func (h *IngressHandler) getFirstHostname(ingress *networkingv1.Ingress) string {
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			return rule.Host
		}
	}
	return ""
}

func (h *IngressHandler) hasIngressTLS(ingress *networkingv1.Ingress, hostname string) bool {
	for _, tls := range ingress.Spec.TLS {
		if slices.Contains(tls.Hosts, hostname) {
			return true
		}
	}
	return false
}

func (h *IngressHandler) hasIngressClass(ingress *networkingv1.Ingress, ingressClass string) bool {
	return h.getIngressClass(ingress) == ingressClass
}

func (h *IngressHandler) getIngressClass(ingress *networkingv1.Ingress) string {
	// Check spec.ingressClassName first (preferred)
	if ingress.Spec.IngressClassName != nil {
		return *ingress.Spec.IngressClassName
	}

	// Fallback to annotation (legacy)
	if ingress.Annotations != nil {
		if class, ok := ingress.Annotations[ingressClassAnnotation]; ok {
			return class
		}
	}

	return ""
}

// NewIngressController creates a controller for Ingress resources
func NewIngressController(stateManager *manager.Manager, dynamicClient dynamic.Interface) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    networkingAPIGroup,
			Version:  networkingAPIVersion,
			Resource: ingressesResource,
		},
		options:       metav1.ListOptions{},
		handler:       &IngressHandler{},
		stateManager:  stateManager,
		dynamicClient: dynamicClient,
		convert:       convertUnstructuredToIngress,
	}
}

func convertUnstructuredToIngress(u *unstructured.Unstructured) (metav1.Object, error) {
	ingress := &networkingv1.Ingress{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ingress); err != nil {
		return nil, fmt.Errorf("failed to convert to Ingress: %w", err)
	}
	return ingress, nil
}
