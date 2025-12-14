package ingressroute

import (
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/endpoint"
	"github.com/home-operations/gatus-sidecar/internal/resources"
)

const (
	dnsTestURL            = "1.1.1.1"
	dnsEmptyBodyCondition = "len([BODY]) == 0"
	dnsQueryType          = "A"
)

var hostRegex = regexp.MustCompile("Host\\(`([^`]+)`\\)")

// Definition creates a resource definition for Traefik IngressRoute resources
func Definition() *resources.ResourceDefinition {
	return &resources.ResourceDefinition{
		GVR: schema.GroupVersionResource{
			Group:    "traefik.io",
			Version:  "v1alpha1",
			Resource: "ingressroutes",
		},
		ConvertFunc:    convertFunc,
		AutoConfigFunc: func(cfg *config.Config) bool { return cfg.AutoIngressRoute },
		URLExtractor:   urlExtractor,
		ConditionFunc:  conditionFunc,
		GuardedFunc:    guardedFunc,
	}
}

func convertFunc(u *unstructured.Unstructured) (metav1.Object, error) {
	return u, nil
}

func urlExtractor(obj metav1.Object) string {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ""
	}

	hostname := getFirstHostname(u)
	if hostname == "" {
		return ""
	}

	if hasTLS(u) {
		return "https://" + hostname
	}
	return "http://" + hostname
}

func conditionFunc(cfg *config.Config, obj metav1.Object, endpoint *endpoint.Endpoint) {
	endpoint.Conditions = []string{"[STATUS] == 200"}
}

func guardedFunc(obj metav1.Object, endpoint *endpoint.Endpoint) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}

	hostname := getFirstHostname(u)
	if hostname == "" {
		return
	}

	endpoint.URL = dnsTestURL
	endpoint.DNS = map[string]any{
		"query-name": hostname,
		"query-type": dnsQueryType,
	}
	endpoint.Conditions = []string{dnsEmptyBodyCondition}
}

func getFirstHostname(u *unstructured.Unstructured) string {
	routes, found, err := unstructured.NestedSlice(u.Object, "spec", "routes")
	if err != nil || !found || len(routes) == 0 {
		return ""
	}

	for _, route := range routes {
		routeMap, ok := route.(map[string]any)
		if !ok {
			continue
		}

		match, ok := routeMap["match"].(string)
		if !ok {
			continue
		}

		if hostname := extractHostFromMatch(match); hostname != "" {
			return hostname
		}
	}

	return ""
}

func extractHostFromMatch(match string) string {
	matches := hostRegex.FindStringSubmatch(match)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func hasTLS(u *unstructured.Unstructured) bool {
	tls, found, err := unstructured.NestedMap(u.Object, "spec", "tls")
	return err == nil && found && tls != nil
}
