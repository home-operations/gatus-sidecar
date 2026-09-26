// Package resources implements [k8s.Resource] for Ingress, Service, Gateway
// API HTTPRoute, and Traefik IngressRoute.
package resources

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/home-operations/gatus-sidecar/internal/config"
	"github.com/home-operations/gatus-sidecar/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	conditionHTTPOK    = "[STATUS] == 200"
	conditionConnected = "[CONNECTED] == true"
)

var (
	httpDefaultConditions = []string{conditionHTTPOK}
	tcpDefaultConditions  = []string{conditionConnected}
)

// formatURL composes scheme://host/path, honoring an embedded scheme on host
// (e.g. host = "http://example.com" yields host+path unchanged).
func formatURL(host, path string, useTLS bool) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host + path
	}
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	return scheme + "://" + host + path
}

// registry is the single source of truth for which kinds exist and the
// order they're created in.
var registry = []k8s.Resource{Ingress{}, HTTPRoute{}, Service{}, IngressRoute{}}

// All returns the Resource implementations enabled by cfg. With no flag set,
// all kinds run in annotation-only mode.
func All(cfg *config.Config) []k8s.Resource {
	annotationOnly := !cfg.AnyExplicitlyEnabled()
	out := make([]k8s.Resource, 0, len(registry))
	for _, r := range registry {
		if annotationOnly || cfg.KindEnabled(r.Kind()) {
			out = append(out, r)
		}
	}
	return out
}

func convertTo[T any, PT interface {
	*T
	metav1.Object
}](u *unstructured.Unstructured) (metav1.Object, error) {
	obj := PT(new(T))
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, obj); err != nil {
		return nil, fmt.Errorf("resources: convert to %s: %w", reflect.TypeFor[T]().Name(), err)
	}
	return obj, nil
}
