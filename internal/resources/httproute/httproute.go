package httproute

import (
	"context"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/resources"
)

const (
	dnsTestURL            = "1.1.1.1"
	dnsEmptyBodyCondition = "len([BODY]) == 0"
	dnsQueryType          = "A"
)

// Definition creates a resource definition for HTTPRoute resources
func Definition() *resources.ResourceDefinition {
	return &resources.ResourceDefinition{
		GVR: schema.GroupVersionResource{
			Group:    "gateway.networking.k8s.io",
			Version:  "v1",
			Resource: "httproutes",
		},
		TargetType:      reflect.TypeOf(gatewayv1.HTTPRoute{}),
		ConvertFunc:     resources.CreateConvertFunc(reflect.TypeOf(gatewayv1.HTTPRoute{})),
		AutoConfigFunc:  func(cfg *config.Config) bool { return cfg.AutoHTTPRoute },
		FilterFunc:      filterFunc,
		URLExtractor:    urlExtractor,
		ConditionFunc:   conditionFunc,
		GuardedFunc:     guardedFunc,
		ParentExtractor: parentExtractor,
	}
}

func filterFunc(obj metav1.Object, cfg *config.Config) bool {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return false
	}

	// Check gateway filter if configured
	if cfg.GatewayName != "" {
		return referencesGateway(route, cfg.GatewayName)
	}

	return true
}

func urlExtractor(obj metav1.Object) string {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok {
		return ""
	}

	hostname := getFirstHostname(route)
	if hostname == "" {
		return ""
	}

	if !strings.HasPrefix(hostname, "http") {
		return "https://" + hostname
	}

	return hostname
}

func conditionFunc(cfg *config.Config, obj metav1.Object, e *endpoint.Endpoint) {
	e.Conditions = []string{"[STATUS] == 200"}
}

func guardedFunc(obj metav1.Object, e *endpoint.Endpoint) {
	if route, ok := obj.(*gatewayv1.HTTPRoute); ok {
		applyGuardedTemplate(getFirstHostname(route), e)
	}
}

func applyGuardedTemplate(dnsQueryName string, e *endpoint.Endpoint) {
	e.URL = dnsTestURL
	e.DNS = map[string]any{
		"query-name": dnsQueryName,
		"query-type": dnsQueryType,
	}
	e.Conditions = []string{dnsEmptyBodyCondition}
}

func parentExtractor(ctx context.Context, obj metav1.Object, client dynamic.Interface) map[string]string {
	route, ok := obj.(*gatewayv1.HTTPRoute)
	if !ok || len(route.Spec.ParentRefs) == 0 {
		return nil
	}

	parent := route.Spec.ParentRefs[0]
	if parent.Kind != nil && *parent.Kind != "Gateway" {
		return nil
	}

	gvr := schema.GroupVersionResource{
		Group:    "gateway.networking.k8s.io",
		Version:  "v1",
		Resource: "gateways",
	}
	if parent.Group != nil {
		gvr.Group = string(*parent.Group)
	}

	namespace := route.GetNamespace()
	if parent.Namespace != nil {
		namespace = string(*parent.Namespace)
	}

	parentResource, err := client.Resource(gvr).Namespace(namespace).Get(ctx, string(parent.Name), metav1.GetOptions{})
	if err != nil {
		return nil
	}

	return parentResource.GetAnnotations()
}

// Helper functions

func getFirstHostname(route *gatewayv1.HTTPRoute) string {
	if len(route.Spec.Hostnames) == 0 {
		return ""
	}
	return string(route.Spec.Hostnames[0])
}

func referencesGateway(route *gatewayv1.HTTPRoute, gatewayName string) bool {
	for _, parent := range route.Spec.ParentRefs {
		if parent.Name == gatewayv1.ObjectName(gatewayName) {
			return true
		}
	}
	return false
}
