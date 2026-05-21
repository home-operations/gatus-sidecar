package resources

import (
	"context"
	"regexp"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	ingressRouteGVR = schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}
	traefikHostMatcher = regexp.MustCompile("Host\\(`([^`]+)`\\)")
	traefikPathMatcher = regexp.MustCompile("(?:PathPrefix|Path)\\(`([^`]+)`\\)")
)

// IngressRoute keeps Traefik's CRD as unstructured — Traefik's typed Go
// definitions live outside k8s.io and aren't worth the dependency.
type IngressRoute struct{}

func (IngressRoute) GVR() schema.GroupVersionResource { return ingressRouteGVR }

func (IngressRoute) Prefix(cfg *config.Config) string { return cfg.IngressRoutePrefix }

func (IngressRoute) Convert(u *unstructured.Unstructured) (metav1.Object, error) {
	return u, nil
}

func (IngressRoute) Matches(obj metav1.Object, cfg *config.Config) bool {
	if _, ok := obj.(*unstructured.Unstructured); !ok {
		return false
	}
	return matchesAnnotation(obj, cfg.AutoIngressRoute, cfg)
}

func (IngressRoute) URL(obj metav1.Object) string {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ""
	}
	host, path := firstIngressRouteHostAndPath(u)
	if host == "" {
		return ""
	}
	return formatURL(host, path, ingressRouteHasTLS(u))
}

func (IngressRoute) DefaultConditions() []string { return httpDefaultConditions }

func (IngressRoute) GuardHost(obj metav1.Object) string {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ""
	}
	return firstIngressRouteHostname(u)
}

func (IngressRoute) ParentAnnotations(context.Context, metav1.Object, k8s.Fetcher) map[string]string {
	return nil
}

func firstIngressRouteHostname(u *unstructured.Unstructured) string {
	host, _ := firstIngressRouteHostAndPath(u)
	return host
}

// firstIngressRouteHostAndPath scans the route list for the first match
// expression containing Host(), returning that host plus any Path()/
// PathPrefix() in the same expression.
func firstIngressRouteHostAndPath(u *unstructured.Unstructured) (host, path string) {
	routes, found, err := unstructured.NestedSlice(u.Object, "spec", "routes")
	if err != nil || !found {
		return "", ""
	}
	for _, raw := range routes {
		route, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		match, ok := route["match"].(string)
		if !ok {
			continue
		}
		h := matchTraefikHost(match)
		if h == "" {
			continue
		}
		return h, matchTraefikPath(match)
	}
	return "", ""
}

func matchTraefikHost(expr string) string {
	matches := traefikHostMatcher.FindStringSubmatch(expr)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func matchTraefikPath(expr string) string {
	matches := traefikPathMatcher.FindStringSubmatch(expr)
	if len(matches) >= 2 && isProbablePath(matches[1]) {
		return matches[1]
	}
	return ""
}

func ingressRouteHasTLS(u *unstructured.Unstructured) bool {
	tls, found, err := unstructured.NestedMap(u.Object, "spec", "tls")
	return err == nil && found && tls != nil
}
