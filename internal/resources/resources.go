// Package resources implements [k8s.Resource] for Ingress, Service, Gateway
// API HTTPRoute, and Traefik IngressRoute.
package resources

import (
	"fmt"
	"reflect"

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

// All returns the Resource implementations enabled by cfg. With no flag set,
// all four kinds run in annotation-only mode.
func All(cfg *config.Config) []k8s.Resource {
	if !cfg.AnyExplicitlyEnabled() {
		return []k8s.Resource{Ingress{}, HTTPRoute{}, Service{}, IngressRoute{}}
	}
	var out []k8s.Resource
	if cfg.EnableIngress || cfg.AutoIngress {
		out = append(out, Ingress{})
	}
	if cfg.EnableHTTPRoute || cfg.AutoHTTPRoute {
		out = append(out, HTTPRoute{})
	}
	if cfg.EnableService || cfg.AutoService {
		out = append(out, Service{})
	}
	if cfg.EnableIngressRoute || cfg.AutoIngressRoute {
		out = append(out, IngressRoute{})
	}
	return out
}

func convertTo[T any](u *unstructured.Unstructured) (metav1.Object, error) {
	obj := new(T)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, obj); err != nil {
		return nil, fmt.Errorf("convert to %s: %w", reflect.TypeOf(*obj).Name(), err)
	}
	o, ok := any(obj).(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("type %s does not implement metav1.Object", reflect.TypeOf(*obj).Name())
	}
	return o, nil
}

// hasGatusAnnotations reports whether obj opts in via either gatus annotation
// — the fallback for annotation-only mode.
func hasGatusAnnotations(obj metav1.Object, cfg *config.Config) bool {
	ann := obj.GetAnnotations()
	if _, ok := ann[cfg.EnabledAnnotation]; ok {
		return true
	}
	_, ok := ann[cfg.TemplateAnnotation]
	return ok
}
