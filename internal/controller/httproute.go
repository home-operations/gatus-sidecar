package controller

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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

	// If AutoRoutes is disabled, only process if it has the annotation
	if !cfg.AutoRoutes {
		annotations := route.GetAnnotations()
		if annotations == nil {
			return false
		}

		_, hasAnnotation := annotations[cfg.EnabledAnnotation]
		return hasAnnotation
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

func (h *HTTPRouteHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	if cfg.AutoGroup {
		route, ok := obj.(*gatewayv1.HTTPRoute)
		if !ok {
			return
		}

		// Group by first ParentRef (usually the Gateway)
		if len(route.Spec.ParentRefs) > 0 {
			endpoint.Group = string(route.Spec.ParentRefs[0].Name)
		}
	}

	endpoint.Conditions = []string{"[STATUS] == 200"}
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
func NewHTTPRouteController(stateManager *manager.Manager, dynamicClient dynamic.Interface) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    "gateway.networking.k8s.io",
			Version:  "v1",
			Resource: "httproutes",
		},
		options:       metav1.ListOptions{},
		handler:       &HTTPRouteHandler{},
		stateManager:  stateManager,
		dynamicClient: dynamicClient,
		convert: func(u *unstructured.Unstructured) (metav1.Object, error) {
			route := &gatewayv1.HTTPRoute{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, route); err != nil {
				return nil, fmt.Errorf("failed to convert to HTTPRoute: %w", err)
			}
			return route, nil
		},
	}
}
