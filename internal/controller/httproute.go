package controller

import (
	"context"
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

const (
	gatewayAPIGroup    = "gateway.networking.k8s.io"
	gatewayAPIVersion  = "v1"
	httproutesResource = "httproutes"
	gatewaysResource   = "gateways"
	gatewayKind        = "Gateway"
)

// HTTPRouteHandler handles HTTPRoute resources
type HTTPRouteHandler struct {
	dynamicClient dynamic.Interface
}

// Ensure HTTPRouteHandler implements the ResourceHandler interface
var _ handler.ResourceHandler = (*HTTPRouteHandler)(nil)

func (h *HTTPRouteHandler) ShouldProcess(obj metav1.Object, cfg *config.Config) bool {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return false
	}

	// Check gateway filter first (most restrictive)
	if cfg.GatewayName != "" && !h.referencesGateway(route, cfg.GatewayName) {
		return false
	}

	// If AutoHTTPRoute is enabled, process all routes (that passed gateway filter)
	if cfg.AutoHTTPRoute {
		return true
	}

	// If AutoHTTPRoute is disabled, only process if it has required annotations
	return hasRequiredAnnotations(obj, cfg)
}

func (h *HTTPRouteHandler) ExtractURL(obj metav1.Object) string {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return ""
	}

	hostname := h.getFirstHostname(route)
	if hostname == "" {
		return ""
	}

	if !strings.HasPrefix(hostname, httpPrefix) {
		return httpsPrefix + hostname
	}

	return hostname
}

func (h *HTTPRouteHandler) ApplyTemplate(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	if endpoint.Guarded {
		if route, ok := obj.(*gatewayv1.HTTPRoute); ok {
			applyGuardedTemplate(h.getFirstHostname(route), endpoint)
		}
	} else {
		endpoint.Conditions = []string{ingressCondition}
	}
}

func (h *HTTPRouteHandler) GetParentAnnotations(ctx context.Context, obj metav1.Object) map[string]string {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok || len(route.Spec.ParentRefs) == 0 {
		return nil
	}

	parent := route.Spec.ParentRefs[0]
	if parent.Kind != nil && *parent.Kind != gatewayKind {
		return nil
	}

	return h.fetchParentAnnotations(ctx, route, parent)
}

func (h *HTTPRouteHandler) fetchParentAnnotations(ctx context.Context, route *gatewayv1.HTTPRoute, parent gatewayv1.ParentReference) map[string]string {
	gvr := schema.GroupVersionResource{
		Group:    gatewayAPIGroup,
		Version:  gatewayAPIVersion,
		Resource: gatewaysResource,
	}
	if parent.Group != nil {
		gvr.Group = string(*parent.Group)
	}

	namespace := route.GetNamespace()
	if parent.Namespace != nil {
		namespace = string(*parent.Namespace)
	}

	parentResource, err := h.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, string(parent.Name), metav1.GetOptions{})
	if err != nil {
		return nil
	}

	return parentResource.GetAnnotations()
}

func (h *HTTPRouteHandler) getFirstHostname(route *gatewayv1.HTTPRoute) string {
	if len(route.Spec.Hostnames) == 0 {
		return ""
	}
	return string(route.Spec.Hostnames[0])
}

func (h *HTTPRouteHandler) referencesGateway(route *gatewayv1.HTTPRoute, gatewayName string) bool {
	for _, parent := range route.Spec.ParentRefs {
		if parent.Name == gatewayv1.ObjectName(gatewayName) {
			return true
		}
	}
	return false
}

// NewHTTPRouteController creates a controller for HTTPRoute resources
func NewHTTPRouteController(stateManager *manager.Manager, dynamicClient dynamic.Interface) *Controller {
	return &Controller{
		gvr: schema.GroupVersionResource{
			Group:    gatewayAPIGroup,
			Version:  gatewayAPIVersion,
			Resource: httproutesResource,
		},
		options:       metav1.ListOptions{},
		handler:       &HTTPRouteHandler{dynamicClient: dynamicClient},
		stateManager:  stateManager,
		dynamicClient: dynamicClient,
		convert:       convertUnstructuredToHTTPRoute,
	}
}

func convertUnstructuredToHTTPRoute(u *unstructured.Unstructured) (metav1.Object, error) {
	route := &gatewayv1.HTTPRoute{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, route); err != nil {
		return nil, fmt.Errorf("failed to convert to HTTPRoute: %w", err)
	}
	return route, nil
}
