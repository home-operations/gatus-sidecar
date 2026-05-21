package resources

import (
	"context"
	"slices"
	"strings"

	"github.com/home-operations/gatus-sidecar/internal/config"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	httpRouteGVR = schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "httproutes",
	}
	gatewayGVR = schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "gateways",
	}
)

type HTTPRoute struct{}

func (HTTPRoute) GVR() schema.GroupVersionResource { return httpRouteGVR }

func (HTTPRoute) Prefix(cfg *config.Config) string { return cfg.HTTPRoutePrefix }

func (HTTPRoute) Convert(u *unstructured.Unstructured) (metav1.Object, error) {
	return convertTo[gatewayv1.HTTPRoute](u)
}

func (HTTPRoute) Matches(obj metav1.Object, cfg *config.Config) bool {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return false
	}
	if len(cfg.GatewayNames) > 0 && !httpRouteReferencesAnyGateway(route, cfg.GatewayNames) {
		return false
	}
	if cfg.AutoHTTPRoute {
		return true
	}
	return hasGatusAnnotations(obj, cfg)
}

func (HTTPRoute) URL(obj metav1.Object) string {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return ""
	}
	host := firstHTTPRouteHostname(route)
	if host == "" {
		return ""
	}
	path := firstHTTPRoutePath(route)
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host + path
	}
	return "https://" + host + path
}

func (HTTPRoute) DefaultConditions() []string { return httpDefaultConditions }

func (HTTPRoute) GuardHost(obj metav1.Object) string {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return ""
	}
	return firstHTTPRouteHostname(route)
}

func (HTTPRoute) ParentAnnotations(ctx context.Context, obj metav1.Object, dc dynamic.Interface) map[string]string {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok || len(route.Spec.ParentRefs) == 0 {
		return nil
	}
	parent := route.Spec.ParentRefs[0]
	if parent.Kind != nil && *parent.Kind != "Gateway" {
		return nil
	}

	gvr := gatewayGVR
	if parent.Group != nil {
		gvr.Group = string(*parent.Group)
	}

	namespace := route.GetNamespace()
	if parent.Namespace != nil {
		namespace = string(*parent.Namespace)
	}

	gw, err := dc.Resource(gvr).Namespace(namespace).Get(ctx, string(parent.Name), metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return gw.GetAnnotations()
}

func firstHTTPRouteHostname(route *gatewayv1.HTTPRoute) string {
	if len(route.Spec.Hostnames) == 0 {
		return ""
	}
	return string(route.Spec.Hostnames[0])
}

// firstHTTPRoutePath returns the first Exact/PathPrefix match value. Regex
// matches are skipped — there's no concrete URL to probe.
func firstHTTPRoutePath(route *gatewayv1.HTTPRoute) string {
	for _, rule := range route.Spec.Rules {
		for _, match := range rule.Matches {
			if match.Path == nil || match.Path.Value == nil {
				continue
			}
			if match.Path.Type != nil && *match.Path.Type == gatewayv1.PathMatchRegularExpression {
				continue
			}
			if isProbablePath(*match.Path.Value) {
				return *match.Path.Value
			}
		}
	}
	return ""
}

func httpRouteReferencesAnyGateway(route *gatewayv1.HTTPRoute, names []string) bool {
	return slices.ContainsFunc(route.Spec.ParentRefs, func(p gatewayv1.ParentReference) bool {
		return slices.Contains(names, string(p.Name))
	})
}
