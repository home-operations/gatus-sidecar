package controller

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/handler"
)

// HTTPRouteHandler handles HTTPRoute resources
type HTTPRouteHandler struct{}

// Ensure HTTPRouteHandler implements the ResourceHandler interface
var _ handler.ResourceHandler = (*HTTPRouteHandler)(nil)

func (h *HTTPRouteHandler) ShouldProcess(obj metav1.Object, cfg *config.Config) bool {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return false
	}

	if cfg.GatewayName != "" && !referencesGateway(route, cfg.GatewayName) {
		return false
	}

	return true
}

func (h *HTTPRouteHandler) ExtractURL(obj metav1.Object) string {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return ""
	}

	url := firstHTTPRouteHostname(route)
	if url == "" {
		return ""
	}

	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	return url
}

func (h *HTTPRouteHandler) GetResourceName() string {
	return "route"
}

// Helper functions for HTTPRoute
func firstHTTPRouteHostname(route *gatewayv1.HTTPRoute) string {
	for _, h := range route.Spec.Hostnames {
		return string(h)
	}
	return ""
}

func referencesGateway(route *gatewayv1.HTTPRoute, gatewayName string) bool {
	for _, p := range route.Spec.ParentRefs {
		if p.Name == gatewayv1.ObjectName(gatewayName) {
			return true
		}
	}
	return false
}

func RunHTTPRoute(ctx context.Context, cfg *config.Config) error {
	handler := &HTTPRouteHandler{}
	ctrl := NewHTTPRouteController(handler)
	return ctrl.Run(ctx, cfg)
}
