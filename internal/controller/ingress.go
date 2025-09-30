package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/handler"
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
	// Check spec.ingressClassName first (preferred)
	if ingress.Spec.IngressClassName != nil && *ingress.Spec.IngressClassName == ingressClass {
		return true
	}

	// Fallback to annotation (legacy)
	if ingress.Annotations != nil {
		if class, ok := ingress.Annotations["kubernetes.io/ingress.class"]; ok {
			return class == ingressClass
		}
	}

	return false
}

func RunIngress(ctx context.Context, cfg *config.Config) error {
	handler := &IngressHandler{}
	ctrl := NewIngressController(handler)
	return ctrl.Run(ctx, cfg)
}
