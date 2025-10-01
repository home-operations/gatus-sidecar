package controller

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/handler"
	"github.com/home-operations/gatus-sidecar/internal/manager"
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

func (h *HTTPRouteHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) bool {
	if cfg.AutoGroup {
		route, ok := obj.(*gatewayv1.HTTPRoute)
		if !ok {
			return false
		}

		// If there are no ParentRefs, cannot group
		if len(route.Spec.ParentRefs) == 0 {
			return false
		}

		// Group by the first ParentRef (gateway) name
		endpoint.Group = string(route.Spec.ParentRefs[0].Name)
		return true
	}

	return false
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

// NewHTTPRouteController creates a controller for HTTPRoute resources
func NewHTTPRouteController(resourceHandler handler.ResourceHandler, stateManager *manager.Manager) *Controller {
	return &Controller{
		gvr:          gatewayv1.SchemeGroupVersion.WithResource("httproutes"),
		handler:      resourceHandler,
		stateManager: stateManager,
		convert: func(u *unstructured.Unstructured) (metav1.Object, error) {
			route := &gatewayv1.HTTPRoute{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, route); err != nil {
				return nil, fmt.Errorf("failed to convert to HTTPRoute: %w", err)
			}
			return route, nil
		},
	}
}

func RunHTTPRoute(ctx context.Context, cfg *config.Config) error {
	stateManager := manager.NewManager(cfg.Output)
	handler := &HTTPRouteHandler{}
	ctrl := NewHTTPRouteController(handler, stateManager)
	return ctrl.Run(ctx, cfg)
}
